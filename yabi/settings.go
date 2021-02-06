package yabi

import (
	"strings"
	"sync"
)

// YabiCookieName is the default cookie name for the yabi auth system
const YabiCookieName = "yabi"

// ExpireCookieInDays is the default user's cookie expiration in 30 days if not provided
var ExpireCookieInDays int = 30 // number of days

// InitYabi initializes the common configurations that yabi package use
type InitYabi struct {
	BaseURL string     // e.g http://127.0.0.1:8081/ or https://maharlikanscode.com/ with the trailing "/" slash
	mu      sync.Mutex // ensures atomic writes; protect the following fields
}

// YB is the pointer for InitYabi configuration
var YB *InitYabi

// InitYabiConfig initialize all the necessary yabi configurations and its default values
func InitYabiConfig(b *InitYabi) *InitYabi {
	// Check all the required configurations are in place or not
	if len(strings.TrimSpace(b.BaseURL)) == 0 {
		b.BaseURL = "http://127.0.0.1:8081/"
	}
	return &InitYabi{
		BaseURL: b.BaseURL,
	}
}

func init() {
	// Set initial default yabi config
	YB = InitYabiConfig(&InitYabi{})
}

// SetYabiConfig sets the custom config values from the user
func SetYabiConfig(b *InitYabi) *InitYabi {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check all the required configurations are in place or not
	if len(strings.TrimSpace(b.BaseURL)) == 0 {
		b.BaseURL = "http://127.0.0.1:8081/"
	}

	// Re-configure the yabi configurations
	b = InitYabiConfig(b)
	return b
}
