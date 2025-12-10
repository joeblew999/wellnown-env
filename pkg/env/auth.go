// auth.go: NATS authentication configuration for all lifecycle phases
//
// Security Lifecycle:
//
//	Dev/Local:   NATS_AUTH=none   - No auth, fast iteration
//	Test/CI:     NATS_AUTH=token  - Shared token via env var
//	Staging:     NATS_AUTH=nkey   - NKey public/private keypairs
//	Production:  NATS_AUTH=jwt    - Full NSC accounts with revocation
//
// Files read from .auth/ directory:
//
//	.auth/mode         - Current auth mode (token/nkey/jwt)
//	.auth/token        - Shared token for token mode
//	.auth/user.pub     - NKey public key for nkey mode
//	.auth/user.nk      - NKey seed for client auth (nkey mode)
//	.auth/creds/       - JWT credentials directory (jwt mode)
//	.auth/creds/user.creds - User credentials file (jwt mode)
package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

// Auth file paths (relative to working directory)
const (
	authDir       = ".auth"
	authModeFile  = ".auth/mode"
	authTokenFile = ".auth/token"
	authNKeyPub   = ".auth/user.pub"
	authNKeySeed  = ".auth/user.nk"
	authCredsDir  = ".auth/creds"
	authCredsFile = ".auth/creds/user.creds"
)

// readAuthFile reads and trims a file from the auth directory
func readAuthFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// preloadedResolver is a custom AccountResolver that holds preloaded account JWTs
// This is needed because MemAccResolver.Store() can only be called after Start()
type preloadedResolver struct {
	sync.RWMutex
	accounts map[string]string
}

func newPreloadedResolver() *preloadedResolver {
	return &preloadedResolver{
		accounts: make(map[string]string),
	}
}

func (r *preloadedResolver) Fetch(name string) (string, error) {
	r.RLock()
	defer r.RUnlock()
	if jwt, ok := r.accounts[name]; ok {
		return jwt, nil
	}
	return "", fmt.Errorf("account %s not found", name)
}

func (r *preloadedResolver) Store(name, jwt string) error {
	r.Lock()
	defer r.Unlock()
	r.accounts[name] = jwt
	return nil
}

func (r *preloadedResolver) IsReadOnly() bool {
	return false
}

func (r *preloadedResolver) Start(s *server.Server) error {
	return nil
}

func (r *preloadedResolver) IsTrackingUpdate() bool {
	return false
}

func (r *preloadedResolver) Reload() error {
	return nil
}

func (r *preloadedResolver) Close() {
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Mode     string // none, token, nkey, jwt
	Token    string // for token mode
	NKeyPub  string // for nkey mode (user public key)
	CredsDir string // for jwt mode
}

// LoadAuthConfig reads auth configuration from environment and .auth/ directory
func LoadAuthConfig() (*AuthConfig, error) {
	cfg := &AuthConfig{
		Mode: GetEnv("NATS_AUTH", "none"),
	}

	// Check if .auth/mode file exists and use it (takes precedence over env)
	if mode, err := readAuthFile(authModeFile); err == nil && mode != "" {
		cfg.Mode = mode
	}

	switch cfg.Mode {
	case "none":
		// No additional config needed

	case "token":
		// Try environment variable first, then file
		cfg.Token = os.Getenv("NATS_TOKEN")
		if cfg.Token == "" {
			cfg.Token, _ = readAuthFile(authTokenFile)
		}
		if cfg.Token == "" {
			return nil, fmt.Errorf("token auth requires NATS_TOKEN env var or %s file", authTokenFile)
		}

	case "nkey":
		// Read the public key
		var err error
		cfg.NKeyPub, err = readAuthFile(authNKeyPub)
		if err != nil {
			return nil, fmt.Errorf("nkey auth requires %s file: %w", authNKeyPub, err)
		}
		// Validate it's a valid user public key (starts with U)
		if !nkeys.IsValidPublicUserKey(cfg.NKeyPub) {
			return nil, fmt.Errorf("invalid NKey in %s (must start with U)", authNKeyPub)
		}
		// Verify seed file exists (needed for client auth)
		if _, err := os.Stat(authNKeySeed); os.IsNotExist(err) {
			return nil, fmt.Errorf("nkey auth requires seed file: %s", authNKeySeed)
		}

	case "jwt":
		// Check for credentials directory
		cfg.CredsDir = authCredsDir
		if dir := os.Getenv("NATS_CREDS_DIR"); dir != "" {
			cfg.CredsDir = dir
		}
		if _, err := os.Stat(cfg.CredsDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("jwt auth requires credentials directory: %s", cfg.CredsDir)
		}

	default:
		return nil, fmt.Errorf("unknown auth mode: %s (use: none, token, nkey, jwt)", cfg.Mode)
	}

	return cfg, nil
}

// ConfigureAuth applies authentication settings to NATS server options
func ConfigureAuth(opts *server.Options, cfg *AuthConfig) error {
	switch cfg.Mode {
	case "none":
		return nil

	case "token":
		opts.Authorization = cfg.Token
		return nil

	case "nkey":
		return configureNKeyAuth(opts, cfg)

	case "jwt":
		return configureJWTAuth(opts, cfg)

	default:
		return fmt.Errorf("unknown auth mode: %s", cfg.Mode)
	}
}

// configureNKeyAuth sets up NKey-based authentication
func configureNKeyAuth(opts *server.Options, cfg *AuthConfig) error {
	// Create NKey user with full permissions
	nkeyUser := &server.NkeyUser{
		Nkey: cfg.NKeyPub,
		Permissions: &server.Permissions{
			Publish: &server.SubjectPermission{
				Allow: []string{">"},
			},
			Subscribe: &server.SubjectPermission{
				Allow: []string{">"},
			},
		},
	}

	opts.Nkeys = []*server.NkeyUser{nkeyUser}
	return nil
}

// configureJWTAuth sets up JWT/Account-based authentication
// Uses memory resolver with preloaded account JWTs from NSC store
func configureJWTAuth(opts *server.Options, cfg *AuthConfig) error {
	// Find the operator JWT from NSC store
	nscStore := os.Getenv("NATS_NSC_STORE")
	if nscStore == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot find home directory: %w", err)
		}
		nscStore = filepath.Join(homeDir, ".local", "share", "nats", "nsc", "stores")
	}

	// Look for wellnown operator
	operatorDir := filepath.Join(nscStore, "wellnown")
	if _, err := os.Stat(operatorDir); os.IsNotExist(err) {
		return fmt.Errorf("NSC operator not found at %s - run 'task auth:jwt' first", operatorDir)
	}

	// Find the operator JWT file
	operatorJWTFile := filepath.Join(operatorDir, "wellnown.jwt")
	if _, err := os.Stat(operatorJWTFile); os.IsNotExist(err) {
		return fmt.Errorf("operator JWT not found at %s", operatorJWTFile)
	}

	// Read and parse the operator JWT
	operatorJWTBytes, err := os.ReadFile(operatorJWTFile)
	if err != nil {
		return fmt.Errorf("reading operator JWT: %w", err)
	}

	operatorClaims, err := jwt.DecodeOperatorClaims(string(operatorJWTBytes))
	if err != nil {
		return fmt.Errorf("decoding operator JWT: %w", err)
	}

	// Configure the server to use the operator
	opts.TrustedOperators = []*jwt.OperatorClaims{operatorClaims}

	// Load account JWTs into memory resolver
	accountsDir := filepath.Join(operatorDir, "accounts")
	preloads := make(map[string]string)

	// Walk accounts directory and load all account JWTs
	entries, err := os.ReadDir(accountsDir)
	if err != nil {
		return fmt.Errorf("reading accounts dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		accountName := entry.Name()
		accountJWTFile := filepath.Join(accountsDir, accountName, accountName+".jwt")
		if jwtBytes, err := os.ReadFile(accountJWTFile); err == nil {
			accountClaims, err := jwt.DecodeAccountClaims(string(jwtBytes))
			if err == nil {
				preloads[accountClaims.Subject] = string(jwtBytes)
			}
		}
	}

	// Create our custom resolver with preloaded accounts
	resolver := newPreloadedResolver()
	for pubKey, jwtStr := range preloads {
		resolver.Store(pubKey, jwtStr)
	}
	opts.AccountResolver = resolver

	// System account is configured in the operator claims
	if operatorClaims.SystemAccount != "" {
		opts.SystemAccount = operatorClaims.SystemAccount
	}

	return nil
}

// GetClientConnectOptions returns NATS client connection options for the current auth mode
func GetClientConnectOptions(cfg *AuthConfig) ([]nats.Option, error) {
	switch cfg.Mode {
	case "none":
		return nil, nil

	case "token":
		return []nats.Option{nats.Token(cfg.Token)}, nil

	case "nkey":
		return getNKeyClientOptions()

	case "jwt":
		return getJWTClientOptions(cfg.CredsDir)

	default:
		return nil, fmt.Errorf("unknown auth mode: %s", cfg.Mode)
	}
}

// getNKeyClientOptions loads NKey seed and returns client options
func getNKeyClientOptions() ([]nats.Option, error) {
	seed, err := readAuthFile(authNKeySeed)
	if err != nil {
		return nil, fmt.Errorf("reading NKey seed %s: %w", authNKeySeed, err)
	}

	kp, err := nkeys.FromSeed([]byte(seed))
	if err != nil {
		return nil, fmt.Errorf("parsing NKey seed: %w", err)
	}

	pubKey, err := kp.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("getting public key: %w", err)
	}

	sigCB := func(nonce []byte) ([]byte, error) {
		return kp.Sign(nonce)
	}

	return []nats.Option{nats.Nkey(pubKey, sigCB)}, nil
}

// getJWTClientOptions returns client options using JWT credentials file
func getJWTClientOptions(credsDir string) ([]nats.Option, error) {
	credsFile := filepath.Join(credsDir, "user.creds")
	if _, err := os.Stat(credsFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("credentials file not found: %s", credsFile)
	}
	return []nats.Option{nats.UserCredentials(credsFile)}, nil
}
