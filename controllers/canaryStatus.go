package controllers

import (
	"github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	"sync"
)

var instance *CanaryStore
var o sync.Once

func CanaryStoreInstance() *CanaryStore {
	o.Do(func() {
		instance = &CanaryStore{}
	})
	return instance
}

type CanaryStore struct {
}

var store = make(map[string]v1alpha1.Canary)
var lock = sync.RWMutex{}

func (c *CanaryStore) apply(namespace string, name string, canary *v1alpha1.Canary) {
	lock.Lock()
	defer lock.Unlock()
	store[namespace+":"+name] = *canary
}

func (c *CanaryStore) del(namespace string, name string) {
	lock.Lock()
	defer lock.Unlock()
	delete(store, namespace+":"+name)
}

func (c *CanaryStore) get(namespace string, name string) v1alpha1.Canary {
	return store[namespace+":"+name]
}
