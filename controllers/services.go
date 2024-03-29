package controllers

import (
	"context"
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
		createService(ctx, c, canary, "")
		klog.Warning("istio support is building....")
	}
}
func createService(ctx context.Context, c client.Client, canary *cdv1alpha1.Canary, side string) {
	labels := canary.Spec.Selector.MatchLabels
	var serviceName string
	if len(side) > 0 {
		labels[Canary] = side
		serviceName = canary.Name + "--" + side
	} else {
		serviceName = canary.Name
	}

	s := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
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
	namespacedName := types.NamespacedName{Namespace: canary.Namespace, Name: canary.Name}
	err := c.Get(ctx, namespacedName, get)
	if nil != err && !errors.IsNotFound(err) {
		klog.Error(err)
		return
	}
	if errors.IsNotFound(err) {
		err1 := c.Create(ctx, &s)
		if err1 != nil {
			klog.Error(err1)
			return
		}
		klog.Infof("Created svc %q.\n", s.GetName())
	} else if err == nil {
		err1 := c.Patch(ctx, &s, client.Apply)
		if err1 != nil {
			klog.Error(err1)
			return
		}
		klog.Infof("Patched svc %q.\n", s.GetName())
	}
	return
}

func createServiceAccount(ctx context.Context, c client.Client, canary *cdv1alpha1.Canary) error {
	s := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      canary.Name,
			Namespace: canary.Namespace,
			Labels: map[string]string{
				"creator": "smart.cd",
			},
		},
	}

	name := types.NamespacedName{}
	get := &v1.ServiceAccount{}
	err := c.Get(ctx, name, get)
	if errors.IsNotFound(err) {
		err1 := c.Create(context.TODO(), s)
		if err1 != nil {
			klog.Error(err1)
			return err1
		}
		klog.Infof("Created serviceAccount %q.\n", s.GetName())
	} else if err == nil {
		err1 := c.Patch(ctx, s, client.Apply)
		if err1 != nil {
			klog.Error(err1)
			return err1
		}
		klog.Infof("Patched serviceAccount %q.\n", s.GetName())
	} else {
		return err
	}
	return nil
}
