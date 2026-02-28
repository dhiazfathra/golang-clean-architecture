package rbac

import "sync"

type ModuleDefinition struct {
	Name         string
	Fields       []string
	DefaultPerms []Permission
}

var (
	mu       sync.RWMutex
	registry = map[string]ModuleDefinition{}
)

func RegisterModule(def ModuleDefinition) {
	mu.Lock()
	defer mu.Unlock()
	registry[def.Name] = def
}

func Modules() map[string]ModuleDefinition {
	mu.RLock()
	defer mu.RUnlock()
	out := make(map[string]ModuleDefinition, len(registry))
	for k, v := range registry {
		out[k] = v
	}
	return out
}
