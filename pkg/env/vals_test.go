package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEnvSecrets_Echo(t *testing.T) {
	// Clean up
	defer os.Unsetenv("TEST_SECRET")

	// Set env var with ref+echo:// - returns the path as value
	os.Setenv("TEST_SECRET", "ref+echo://my-test-secret")

	// Resolve
	err := ResolveEnvSecrets()
	if err != nil {
		t.Fatalf("ResolveEnvSecrets() error = %v", err)
	}

	// Check resolved value
	got := os.Getenv("TEST_SECRET")
	want := "my-test-secret"
	if got != want {
		t.Errorf("TEST_SECRET = %q, want %q", got, want)
	}
}

func TestResolveEnvSecrets_File(t *testing.T) {
	// Create temp file with secret
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("file-secret-value"), 0644); err != nil {
		t.Fatalf("creating secret file: %v", err)
	}

	// Clean up
	defer os.Unsetenv("TEST_FILE_SECRET")

	// Set env var with ref+file://
	os.Setenv("TEST_FILE_SECRET", "ref+file://"+secretFile)

	// Resolve
	err := ResolveEnvSecrets()
	if err != nil {
		t.Fatalf("ResolveEnvSecrets() error = %v", err)
	}

	// Check resolved value
	got := os.Getenv("TEST_FILE_SECRET")
	want := "file-secret-value"
	if got != want {
		t.Errorf("TEST_FILE_SECRET = %q, want %q", got, want)
	}
}

func TestResolveEnvSecrets_NoRefs(t *testing.T) {
	// Set regular env var (no ref+)
	os.Setenv("TEST_PLAIN", "plain-value")
	defer os.Unsetenv("TEST_PLAIN")

	// Resolve should be no-op
	err := ResolveEnvSecrets()
	if err != nil {
		t.Fatalf("ResolveEnvSecrets() error = %v", err)
	}

	// Value should be unchanged
	got := os.Getenv("TEST_PLAIN")
	if got != "plain-value" {
		t.Errorf("TEST_PLAIN = %q, want %q", got, "plain-value")
	}
}

func TestResolveEnvSecrets_MultipleRefs(t *testing.T) {
	// Clean up
	defer func() {
		os.Unsetenv("TEST_A")
		os.Unsetenv("TEST_B")
		os.Unsetenv("TEST_PLAIN")
	}()

	// Set multiple env vars
	os.Setenv("TEST_A", "ref+echo://secret-a")
	os.Setenv("TEST_B", "ref+echo://secret-b")
	os.Setenv("TEST_PLAIN", "plain-value")

	// Resolve
	err := ResolveEnvSecrets()
	if err != nil {
		t.Fatalf("ResolveEnvSecrets() error = %v", err)
	}

	// Check all values
	tests := []struct {
		key  string
		want string
	}{
		{"TEST_A", "secret-a"},
		{"TEST_B", "secret-b"},
		{"TEST_PLAIN", "plain-value"}, // unchanged
	}

	for _, tt := range tests {
		got := os.Getenv(tt.key)
		if got != tt.want {
			t.Errorf("%s = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestHasSecretRefs(t *testing.T) {
	// Clean up
	defer os.Unsetenv("TEST_REF")

	// No refs initially
	os.Unsetenv("TEST_REF")
	if HasSecretRefs() {
		// This might be true if other tests left refs - skip
		t.Skip("environment has other refs")
	}

	// Add a ref
	os.Setenv("TEST_REF", "ref+echo://test")
	if !HasSecretRefs() {
		t.Error("HasSecretRefs() = false, want true")
	}
}

func TestListSecretRefs(t *testing.T) {
	// Clean up
	defer func() {
		os.Unsetenv("TEST_REF_A")
		os.Unsetenv("TEST_REF_B")
	}()

	// Add refs
	os.Setenv("TEST_REF_A", "ref+echo://a")
	os.Setenv("TEST_REF_B", "ref+echo://b")

	refs := ListSecretRefs()

	// Should contain our refs
	hasA, hasB := false, false
	for _, r := range refs {
		if r == "TEST_REF_A" {
			hasA = true
		}
		if r == "TEST_REF_B" {
			hasB = true
		}
	}

	if !hasA || !hasB {
		t.Errorf("ListSecretRefs() = %v, want to contain TEST_REF_A and TEST_REF_B", refs)
	}
}

func TestResolveString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "plain value",
			input: "plain-value",
			want:  "plain-value",
		},
		{
			name:  "echo ref",
			input: "ref+echo://my-secret",
			want:  "my-secret",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveString_File(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("file-content"), 0644); err != nil {
		t.Fatalf("creating secret file: %v", err)
	}

	got, err := ResolveString("ref+file://" + secretFile)
	if err != nil {
		t.Fatalf("ResolveString() error = %v", err)
	}

	want := "file-content"
	if got != want {
		t.Errorf("ResolveString() = %q, want %q", got, want)
	}
}
