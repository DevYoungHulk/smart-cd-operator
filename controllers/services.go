package controllers

import (
	"context"
	"encoding/json"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func serviceReconcile(ctx context.Context, c client.Client, canary *cdv1alpha1.Canary) {
	if canary.Spec.Strategy.Traffic.TType != Istio {
		createService(ctx, c, canary, "canary")
		createService(ctx, c, canary, "stable")
	} else {
		klog.Warning("istio support is building....")
	}
}
func createService(ctx context.Context, c client.Client, canary *cdv1alpha1.Canary, side string) {
	labels := canary.Spec.Selector.MatchLabels
	labels[Canary] = side
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

	get := &v1.Service{}
	name := types.NamespacedName{Namespace: canary.Namespace, Name: canary.Name}
	err := c.Get(ctx, name, get)
	if nil != err && !errors.IsNotFound(err) {
		klog.Error(err)
		return
	}
	if get == nil || get.Name == "" {
		err1 := c.Create(ctx, &s)
		if err1 != nil {
			klog.Error(err1)
			return
		}
		klog.Infof("Created svc %q.\n", s.GetName())
	} else {
		err1 := c.Patch(ctx, &s, client.Apply)
		if err1 != nil {
			klog.Error(err1)
			return
		}
		klog.Infof("Patched svc %q.\n", s.GetName())
	}
	return
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

	namespaced := KClientSet.CoreV1().ServiceAccounts(canary.Namespace)
	get, _ := namespaced.Get(context.TODO(), canary.Name, metav1.GetOptions{})
	if get == nil {
		create, err := namespaced.Create(context.TODO(), &s, metav1.CreateOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Created serviceAccount %q.\n", create.GetName())
	} else {
		marshal, err := json.Marshal(&s)
		if err != nil {
			klog.Error(err)
			return err
		}
		patch, err := namespaced.Patch(context.TODO(), s.Name, types.MergePatchType, marshal, metav1.PatchOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Patched serviceAccount %q.\n", patch.GetName())
	}
	return nil
}
