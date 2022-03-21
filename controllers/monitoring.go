package controllers

import (
	"context"
	"encoding/json"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	"time"
)

func createServiceMonitor(canary *cdv1alpha1.Canary) error {
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
				MatchLabels: canary.Spec.Deployment.Selector.MatchLabels,
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
		return err
	}
	if err = json.Unmarshal(marshal, &utd.Object); err != nil {
		klog.Error(err)
		return err
	}
	namespace := ClientSet.Resource(serviceMonitorGVR).Namespace(canary.Namespace)
	get, _ := namespace.Get(context.TODO(), canary.Name, metav1.GetOptions{})
	if get == nil {
		create, err := namespace.Create(context.TODO(), utd, metav1.CreateOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Created monitoring %q.\n", create.GetName())
	} else {
		//update, err := namespace.Update(context.TODO(), utd, metav1.UpdateOptions{})
		//if err != nil {
		//	klog.Error(err)
		//	return err
		//}
		klog.Infof("Monitoring exist %q.\n", get.GetName())
	}
	return nil
}

func getStartTime() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, warnings, err := pClient.Query(
		ctx,
		"max(container_start_time_seconds{namespace=\"canary-sample\",name=~\".*my-nginx-app.*\"})",
		time.Now(),
	)
	if err != nil {
		klog.Errorf("Error querying Prometheus: %v\n", err)
	}
	klog.Infof("result -> %s , %v\n", result.String(), warnings)
}
