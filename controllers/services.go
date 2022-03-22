package controllers

import (
	"context"
	"encoding/json"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

func serviceReconcile(canary *cdv1alpha1.Canary, side string) error {
	labels := canary.Spec.Deployment.Selector.MatchLabels
	labels["canary"] = side
	s := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      canary.Name + "--" + side,
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
			Selector:  labels,
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
	namespaced := ClientSet.Resource(serviceGVR).Namespace(canary.Namespace)
	get, err := namespaced.Get(context.TODO(), s.Name, metav1.GetOptions{})
	if get == nil {
		create, err := namespaced.Create(context.TODO(), utd, metav1.CreateOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Created svc %q.\n", create.GetName())
	} else {
		patch, err := namespaced.Patch(context.TODO(), s.Name, types.MergePatchType, marshal, metav1.PatchOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Patched svc %q.\n", patch.GetName())
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
	namespaced := ClientSet.Resource(serviceAccountGVR).Namespace(canary.Namespace)
	get, _ := namespaced.Get(context.TODO(), canary.Name, metav1.GetOptions{})
	if get == nil {
		create, err := namespaced.Create(context.TODO(), utd, metav1.CreateOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Created serviceAccount %q.\n", create.GetName())
	} else {
		patch, err := namespaced.Patch(context.TODO(), s.Name, types.MergePatchType, marshal, metav1.PatchOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Patched serviceAccount %q.\n", patch.GetName())
	}
	return nil
}
