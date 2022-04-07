package controllers

import (
	"encoding/json"
	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"math"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"strconv"
	"sync"
)

var KClientSet *kubernetes.Clientset
var PClient prometheusv1.API

var once sync.Once

func Init(c client.Client) {
	once.Do(func() {
		KClientSet = initClientSet()
		//PClient = initPrometheus()
		initInformers(c)
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

func FormatFloat(num float64, decimal int) (float64, error) {
	// 默认乘1
	d := float64(1)
	if decimal > 0 {
		// 10的N次方
		d = math.Pow10(decimal)
	}
	// math.trunc作用就是返回浮点数的整数部分
	// 再除回去，小数点后无效的0也就不存在了
	res := strconv.FormatFloat(math.Trunc(num*d)/d, 'f', -1, 64)
	return strconv.ParseFloat(res, 64)
}
