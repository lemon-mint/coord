package coord

import (
	"sort"
	"sync"

	"github.com/lemon-mint/coord/provider"
)

var (
	providersMu sync.RWMutex
	providers   = make(map[string]provider.Provider)
)

// Providers returns the names of the registered providers.
func Providers() []string {
	providersMu.RLock()
	defer providersMu.RUnlock()
	list := make([]string, 0, len(providers))
	for name := range providers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

// RegisterProvider registers a provider.
func RegisterProvider(name string, p provider.Provider) {
	providersMu.Lock()
	defer providersMu.Unlock()
	if p == nil {
		panic("coord: Register provider is nil")
	}
	providers[name] = p
}
