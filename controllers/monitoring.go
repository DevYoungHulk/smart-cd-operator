package controllers

import (
	"context"
	"encoding/json"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func serviceMonitorReconcile(ctx context.Context, c client.Client, canary *cdv1alpha1.Canary) {
	s := monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceMonitor",
			APIVersion: "monitoring.coreos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      canary.Name,
			Namespace: canary.Namespace,
			Labels: map[string]string{
				"creator": "smart.cd",
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: canary.Spec.Selector.MatchLabels,
			},
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "http",
				},
			},
		},
	}

	utd := &unstructured.Unstructured{}

	marshal, err := json.Marshal(&s)
	if err != nil {
		klog.Error(err)
		return
	}
	if err = json.Unmarshal(marshal, &utd.Object); err != nil {
		klog.Error(err)
		return
	}
	get := monitoringv1.ServiceMonitor{}
	name := types.NamespacedName{Namespace: canary.Name, Name: canary.Namespace}
	err = c.Get(ctx, name, &get)
	if errors2.IsNotFound(err) {
		err1 := c.Create(ctx, &s)
		if err1 != nil {
			klog.Error(err1)
			return
		}
		klog.Infof("Created monitoring %q.\n", s.GetName())
	} else if err == nil {
		err1 := c.Patch(ctx, &s, client.Apply)
		if err1 != nil {
			klog.Error(err1)
			return
		}
		klog.Infof("Patched Monitoring %q.\n", s.GetName())
	} else {
		klog.Errorf("monitoringv1.ServiceMonitor %v", err)
	}
	return
}

func getStartTime() {
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	//result, warnings, err := pClient.Query(
	//	ctx,
	//	"max(container_start_time_seconds{namespace=\"canary-sample\",name=~\".*my-nginx-app.*\"})",
	//	time.Now(),
	//)
	//if err != nil {
	//	klog.Errorf("Error querying Prometheus: %v\n", err)
	//} else {
	//	klog.Infof("result -> %s , %v\n", result.String(), warnings)
	//}
}
