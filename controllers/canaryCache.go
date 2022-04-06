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

const (
	Separator = '/'
)

var store = make(map[string]v1alpha1.Canary)
var lock = sync.RWMutex{}

func (c *CanaryStore) update(canary *v1alpha1.Canary) {
	lock.Lock()
	defer lock.Unlock()
	store[genKey(canary.Namespace, canary.Name)] = *canary
}

func (c *CanaryStore) del(namespace string, name string) {
	lock.Lock()
	defer lock.Unlock()
	delete(store, genKey(namespace, name))
}

func (c *CanaryStore) get(namespace string, name string) *v1alpha1.Canary {
	if val, ok := store[genKey(namespace, name)]; ok {
		return val.DeepCopy()
	} else {
		return nil
	}
}
func genKey(namespace string, name string) string {
	return namespace + string(Separator) + name
}
