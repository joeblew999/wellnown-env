package env

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal string
		envVal     string
		want       string
	}{
		{
			name:       "returns default when not set",
			key:        "TEST_GET_ENV_NOT_SET",
			defaultVal: "default-value",
			envVal:     "",
			want:       "default-value",
		},
		{
			name:       "returns env value when set",
			key:        "TEST_GET_ENV_SET",
			defaultVal: "default-value",
			envVal:     "env-value",
			want:       "env-value",
		},
		{
			name:       "returns default when empty string",
			key:        "TEST_GET_ENV_EMPTY",
			defaultVal: "default-value",
			envVal:     "",
			want:       "default-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			os.Unsetenv(tt.key)
			defer os.Unsetenv(tt.key)

			if tt.envVal != "" {
				os.Setenv(tt.key, tt.envVal)
			}

			got := GetEnv(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("GetEnv(%q, %q) = %q, want %q", tt.key, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal int
		envVal     string
		want       int
	}{
		{
			name:       "returns default when not set",
			key:        "TEST_GET_ENV_INT_NOT_SET",
			defaultVal: 42,
			envVal:     "",
			want:       42,
		},
		{
			name:       "returns env value when set",
			key:        "TEST_GET_ENV_INT_SET",
			defaultVal: 42,
			envVal:     "123",
			want:       123,
		},
		{
			name:       "returns default on invalid int",
			key:        "TEST_GET_ENV_INT_INVALID",
			defaultVal: 42,
			envVal:     "not-a-number",
			want:       42,
		},
		{
			name:       "handles zero",
			key:        "TEST_GET_ENV_INT_ZERO",
			defaultVal: 42,
			envVal:     "0",
			want:       0,
		},
		{
			name:       "handles negative",
			key:        "TEST_GET_ENV_INT_NEG",
			defaultVal: 42,
			envVal:     "-5",
			want:       -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			defer os.Unsetenv(tt.key)

			if tt.envVal != "" {
				os.Setenv(tt.key, tt.envVal)
			}

			got := GetEnvInt(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("GetEnvInt(%q, %d) = %d, want %d", tt.key, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal bool
		envVal     string
		want       bool
	}{
		{
			name:       "returns default when not set",
			key:        "TEST_GET_ENV_BOOL_NOT_SET",
			defaultVal: false,
			envVal:     "",
			want:       false,
		},
		{
			name:       "true for 'true'",
			key:        "TEST_GET_ENV_BOOL_TRUE",
			defaultVal: false,
			envVal:     "true",
			want:       true,
		},
		{
			name:       "true for 'TRUE'",
			key:        "TEST_GET_ENV_BOOL_TRUE_UPPER",
			defaultVal: false,
			envVal:     "TRUE",
			want:       true,
		},
		{
			name:       "true for '1'",
			key:        "TEST_GET_ENV_BOOL_ONE",
			defaultVal: false,
			envVal:     "1",
			want:       true,
		},
		{
			name:       "true for 'yes'",
			key:        "TEST_GET_ENV_BOOL_YES",
			defaultVal: false,
			envVal:     "yes",
			want:       true,
		},
		{
			name:       "false for 'false'",
			key:        "TEST_GET_ENV_BOOL_FALSE",
			defaultVal: true,
			envVal:     "false",
			want:       false,
		},
		{
			name:       "false for '0'",
			key:        "TEST_GET_ENV_BOOL_ZERO",
			defaultVal: true,
			envVal:     "0",
			want:       false,
		},
		{
			name:       "false for invalid value",
			key:        "TEST_GET_ENV_BOOL_INVALID",
			defaultVal: true,
			envVal:     "invalid",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			defer os.Unsetenv(tt.key)

			if tt.envVal != "" {
				os.Setenv(tt.key, tt.envVal)
			}

			got := GetEnvBool(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("GetEnvBool(%q, %v) = %v, want %v", tt.key, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestGetProcessComposeURL(t *testing.T) {
	// Clean up all env vars
	cleanup := func() {
		os.Unsetenv("PC_URL")
		os.Unsetenv("PC_ADDRESS")
		os.Unsetenv("PC_PORT")
	}
	cleanup()
	defer cleanup()

	t.Run("returns default URL", func(t *testing.T) {
		cleanup()
		got := GetProcessComposeURL()
		want := "http://localhost:8181"
		if got != want {
			t.Errorf("GetProcessComposeURL() = %q, want %q", got, want)
		}
	})

	t.Run("uses PC_URL override", func(t *testing.T) {
		cleanup()
		os.Setenv("PC_URL", "http://custom:9999")
		got := GetProcessComposeURL()
		want := "http://custom:9999"
		if got != want {
			t.Errorf("GetProcessComposeURL() = %q, want %q", got, want)
		}
	})

	t.Run("uses PC_ADDRESS and PC_PORT", func(t *testing.T) {
		cleanup()
		os.Setenv("PC_ADDRESS", "192.168.1.100")
		os.Setenv("PC_PORT", "8080")
		got := GetProcessComposeURL()
		want := "http://192.168.1.100:8080"
		if got != want {
			t.Errorf("GetProcessComposeURL() = %q, want %q", got, want)
		}
	})
}

func TestGetViaAddr(t *testing.T) {
	cleanup := func() {
		os.Unsetenv("VIA_ADDR")
		os.Unsetenv("VIA_PORT")
	}
	cleanup()
	defer cleanup()

	t.Run("returns default addr", func(t *testing.T) {
		cleanup()
		got := GetViaAddr()
		want := ":3000"
		if got != want {
			t.Errorf("GetViaAddr() = %q, want %q", got, want)
		}
	})

	t.Run("uses VIA_ADDR override", func(t *testing.T) {
		cleanup()
		os.Setenv("VIA_ADDR", "0.0.0.0:8080")
		got := GetViaAddr()
		want := "0.0.0.0:8080"
		if got != want {
			t.Errorf("GetViaAddr() = %q, want %q", got, want)
		}
	})

	t.Run("builds from VIA_PORT", func(t *testing.T) {
		cleanup()
		os.Setenv("VIA_PORT", "4000")
		got := GetViaAddr()
		want := ":4000"
		if got != want {
			t.Errorf("GetViaAddr() = %q, want %q", got, want)
		}
	})
}

func TestGetViaURL(t *testing.T) {
	cleanup := func() {
		os.Unsetenv("VIA_HOST")
		os.Unsetenv("VIA_PORT")
	}
	cleanup()
	defer cleanup()

	t.Run("returns default URL", func(t *testing.T) {
		cleanup()
		got := GetViaURL()
		want := "http://localhost:3000"
		if got != want {
			t.Errorf("GetViaURL() = %q, want %q", got, want)
		}
	})

	t.Run("uses VIA_HOST and VIA_PORT", func(t *testing.T) {
		cleanup()
		os.Setenv("VIA_HOST", "example.com")
		os.Setenv("VIA_PORT", "8080")
		got := GetViaURL()
		want := "http://example.com:8080"
		if got != want {
			t.Errorf("GetViaURL() = %q, want %q", got, want)
		}
	})
}

func TestMustGetEnv(t *testing.T) {
	t.Run("returns value when set", func(t *testing.T) {
		os.Setenv("TEST_MUST_GET_SET", "value")
		defer os.Unsetenv("TEST_MUST_GET_SET")

		got := MustGetEnv("TEST_MUST_GET_SET")
		if got != "value" {
			t.Errorf("MustGetEnv() = %q, want %q", got, "value")
		}
	})

	t.Run("panics when not set", func(t *testing.T) {
		os.Unsetenv("TEST_MUST_GET_NOT_SET")

		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGetEnv() did not panic for unset var")
			}
		}()

		MustGetEnv("TEST_MUST_GET_NOT_SET")
	})
}

func TestLoadConfig(t *testing.T) {
	// Clean up all env vars
	cleanup := func() {
		os.Unsetenv("PC_ADDRESS")
		os.Unsetenv("PC_PORT")
		os.Unsetenv("PC_URL")
		os.Unsetenv("VIA_ADDR")
		os.Unsetenv("VIA_HOST")
		os.Unsetenv("VIA_PORT")
		os.Unsetenv("APP_NAME")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("DEBUG")
	}
	cleanup()
	defer cleanup()

	t.Run("loads default values", func(t *testing.T) {
		cleanup()
		cfg := LoadConfig()

		if cfg.PCAddress != "localhost" {
			t.Errorf("PCAddress = %q, want %q", cfg.PCAddress, "localhost")
		}
		if cfg.PCPort != "8181" {
			t.Errorf("PCPort = %q, want %q", cfg.PCPort, "8181")
		}
		if cfg.PCURL != "http://localhost:8181" {
			t.Errorf("PCURL = %q, want %q", cfg.PCURL, "http://localhost:8181")
		}
		if cfg.ViaHost != "localhost" {
			t.Errorf("ViaHost = %q, want %q", cfg.ViaHost, "localhost")
		}
		if cfg.ViaPort != "3000" {
			t.Errorf("ViaPort = %q, want %q", cfg.ViaPort, "3000")
		}
		if cfg.ViaAddr != ":3000" {
			t.Errorf("ViaAddr = %q, want %q", cfg.ViaAddr, ":3000")
		}
		if cfg.ViaURL != "http://localhost:3000" {
			t.Errorf("ViaURL = %q, want %q", cfg.ViaURL, "http://localhost:3000")
		}
		if cfg.AppName != "wellnown-env" {
			t.Errorf("AppName = %q, want %q", cfg.AppName, "wellnown-env")
		}
		if cfg.LogLevel != "info" {
			t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
		}
		if cfg.Debug != false {
			t.Errorf("Debug = %v, want %v", cfg.Debug, false)
		}
	})

	t.Run("loads from environment", func(t *testing.T) {
		cleanup()
		os.Setenv("PC_ADDRESS", "custom-pc")
		os.Setenv("PC_PORT", "9999")
		os.Setenv("VIA_HOST", "custom-via")
		os.Setenv("VIA_PORT", "4000")
		os.Setenv("APP_NAME", "my-app")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("DEBUG", "true")

		cfg := LoadConfig()

		if cfg.PCAddress != "custom-pc" {
			t.Errorf("PCAddress = %q, want %q", cfg.PCAddress, "custom-pc")
		}
		if cfg.PCPort != "9999" {
			t.Errorf("PCPort = %q, want %q", cfg.PCPort, "9999")
		}
		if cfg.PCURL != "http://custom-pc:9999" {
			t.Errorf("PCURL = %q, want %q", cfg.PCURL, "http://custom-pc:9999")
		}
		if cfg.ViaHost != "custom-via" {
			t.Errorf("ViaHost = %q, want %q", cfg.ViaHost, "custom-via")
		}
		if cfg.ViaPort != "4000" {
			t.Errorf("ViaPort = %q, want %q", cfg.ViaPort, "4000")
		}
		if cfg.ViaURL != "http://custom-via:4000" {
			t.Errorf("ViaURL = %q, want %q", cfg.ViaURL, "http://custom-via:4000")
		}
		if cfg.AppName != "my-app" {
			t.Errorf("AppName = %q, want %q", cfg.AppName, "my-app")
		}
		if cfg.LogLevel != "debug" {
			t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
		}
		if cfg.Debug != true {
			t.Errorf("Debug = %v, want %v", cfg.Debug, true)
		}
	})
}
