package controllers

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sync"
)

var canaryGVR = schema.GroupVersionResource{
	Group:    "cd.org.smart",
	Version:  "v1alpha1",
	Resource: "canaries",
}
var deployGVR = schema.GroupVersionResource{
	Group:    "apps",
	Version:  "v1",
	Resource: "deployments",
}

var ClientSet dynamic.Interface
var once sync.Once

func Init() {
	once.Do(func() {
		ClientSet = initClientSet()
	})
}
func initClientSet() dynamic.Interface {
	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	clientset, err := dynamic.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	return clientset
}
