package security

import (
	"io/ioutil"
	"os"
	"strings"
)

// GetSecret returns a secret value by checking environment variable first
// and falling back to Docker secrets at /run/secrets/<name> if present.
func GetSecret(name string) string {
	// Try environment first
	if v := os.Getenv(name); v != "" {
		return v
	}

	// Docker secrets are mounted at /run/secrets/<name>
	path := "/run/secrets/" + strings.ToLower(name)
	if b, err := ioutil.ReadFile(path); err == nil {
		return strings.TrimSpace(string(b))
	}

	// Try uppercase filename variant (some setups keep uppercase)
	path2 := "/run/secrets/" + name
	if b, err := ioutil.ReadFile(path2); err == nil {
		return strings.TrimSpace(string(b))
	}

	return ""
}
