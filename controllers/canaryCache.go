package controllers

import (
	"context"
	"encoding/json"
	"github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	canary := store[genKey(namespace, name)]
	deepCopy := canary.DeepCopy()
	return deepCopy
}
func genKey(namespace string, name string) string {
	return namespace + string(Separator) + name
}

func initCache(c client.Client) {
	canaryList := &v1alpha1.CanaryList{}
	err := c.List(context.TODO(), canaryList, &client.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, item := range canaryList.Items {
		marshal, err := json.Marshal(item)
		if err != nil {
			panic(err)
		}
		canary := v1alpha1.Canary{}
		err1 := json.Unmarshal(marshal, &canary)
		if err1 != nil {
			panic(err1)
		}
		CanaryStoreInstance().update(&canary)
	}
}
