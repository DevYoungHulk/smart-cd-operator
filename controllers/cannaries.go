package controllers

import (
	"context"
	"encoding/json"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func getCanary(ctx *context.Context, namespace string, name string) *cdv1alpha1.Canary {
	list, err := DClientSet.Resource(canaryGVR).Namespace(namespace).Get(*ctx, name, metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		return nil
	}
	data, err := list.MarshalJSON()
	if err != nil {
		klog.Error(err)
		return nil
	}
	var canary cdv1alpha1.Canary
	if err = json.Unmarshal(data, &canary); err != nil {
		klog.Error(err)
		return nil
	}
	return &canary
}
