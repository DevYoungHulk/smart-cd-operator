package controllers

import (
	"encoding/json"
	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
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
var serviceGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "services",
}
var serviceAccountGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "serviceaccounts",
}

var serviceMonitorGVR = schema.GroupVersionResource{
	Group:    "monitoring.coreos.com",
	Version:  "v1",
	Resource: "servicemonitors",
}

var ClientSet dynamic.Interface
var once sync.Once

func Init() {
	once.Do(func() {
		ClientSet = initClientSet()
		pClient = initPrometheus()
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

var pClient prometheusv1.API

func initPrometheus() prometheusv1.API {
	client, err := api.NewClient(api.Config{
		Address: "http://localhost:9091",
		//Address: "http://prometheus-k8s.monitoring.svc.cluster.local",
	})
	if err != nil {
		klog.Errorf("Error creating client: %v\n", err)
	}
	return prometheusv1.NewAPI(client)
}

func objectToJsonData(deployment *appsv1.Deployment) ([]byte, error) {
	return json.Marshal(&deployment)
}

func objectToJsonUtd(deployment *appsv1.Deployment) (*unstructured.Unstructured, error) {
	marshal, err := objectToJsonData(deployment)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	utd := &unstructured.Unstructured{}
	if err = json.Unmarshal(marshal, &utd.Object); err != nil {
		klog.Error(err)
		return nil, err
	}
	return utd, nil
}
