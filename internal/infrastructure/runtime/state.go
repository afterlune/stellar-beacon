package runtime

import (
	"sync"
	"time"
)

type State struct {
	Ready      bool              `json:"ready"`
	Components map[string]string `json:"components"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}

var (
	stateMu    sync.RWMutex
	ready      bool
	components = map[string]string{}
	updatedAt  = time.Now()
)

func SetReady(value bool) {
	stateMu.Lock()
	ready = value
	updatedAt = time.Now()
	stateMu.Unlock()
}

func SetComponent(name, status string) {
	stateMu.Lock()
	components[name] = status
	updatedAt = time.Now()
	stateMu.Unlock()
}

func Snapshot() State {
	stateMu.RLock()
	defer stateMu.RUnlock()
	copy := make(map[string]string, len(components))
	for name, status := range components {
		copy[name] = status
	}
	return State{Ready: ready, Components: copy, UpdatedAt: updatedAt}
}
