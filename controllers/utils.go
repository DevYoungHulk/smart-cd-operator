package controllers

import (
	"encoding/json"
	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sync"
)

var KClientSet *kubernetes.Clientset
var DClientSet dynamic.Interface
var once sync.Once

func Init() {
	once.Do(func() {
		KClientSet = initClientSet()
		DClientSet = initDClientSet()
		pClient = initPrometheus()
	})
}
func initClientSet() *kubernetes.Clientset {
	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	return clientSet
}

func initDClientSet() dynamic.Interface {
	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	clientSet, err := dynamic.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	return clientSet
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
