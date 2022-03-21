package controllers

import (
	"context"
	"encoding/json"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
)

func createService(canary *cdv1alpha1.Canary) error {
	s := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      canary.Name,
			Namespace: canary.Namespace,
			//Labels:    canary.Spec.Deployment.Selector.MatchLabels,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
			Selector:  canary.Spec.Deployment.Selector.MatchLabels,
			ClusterIP: "",
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
	namespace := ClientSet.Resource(serviceGVR).Namespace(canary.Namespace)
	get, _ := namespace.Get(context.TODO(), canary.Name, metav1.GetOptions{})
	if get == nil {
		create, err := namespace.Create(context.TODO(), utd, metav1.CreateOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Created svc %q.\n", create.GetName())
	} else {
		update, err := namespace.Update(context.TODO(), utd, metav1.UpdateOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Updated svc %q.\n", update.GetName())
	}
	return nil
}

func createServiceAccount(canary *cdv1alpha1.Canary) error {
	s := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      canary.Name,
			Namespace: canary.Namespace,
			Labels: map[string]string{
				"creator": "smart.cd",
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
	namespace := ClientSet.Resource(serviceAccountGVR).Namespace(canary.Namespace)
	get, _ := namespace.Get(context.TODO(), canary.Name, metav1.GetOptions{})
	if get == nil {
		create, err := namespace.Create(context.TODO(), utd, metav1.CreateOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Created serviceAccount %q.\n", create.GetName())
	} else {
		update, err := namespace.Update(context.TODO(), utd, metav1.UpdateOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Updated serviceAccount %q.\n", update.GetName())
	}
	return nil
}
