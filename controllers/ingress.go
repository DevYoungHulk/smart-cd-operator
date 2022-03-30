package controllers

import (
	"context"
	"github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func ingressReconcile(canary *cdv1alpha1.Canary) {
	if canary.Spec.Strategy.Traffic.TType == Nginx {
		genIngress(canary, Stable)
		genIngress(canary, Canary)
	} else {
		klog.Warning("istio & traefik support is building....")
	}
}

func genIngress(canary *v1alpha1.Canary, side string) {
	var pathPrefix = v1.PathTypePrefix
	i := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: canary.Namespace,
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
			},
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					Host: "test.nginx.local",
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{{
								Path:     "/",
								PathType: &pathPrefix,
								Backend: v1.IngressBackend{
									Service: &v1.IngressServiceBackend{
										Port: v1.ServiceBackendPort{
											Name: "http",
										},
									},
								},
							},
							},
						},
					},
				},
			},
		},
	}
	if side == "" {
		i.ObjectMeta.Name = canary.Name
	} else {
		i.ObjectMeta.Name = canary.Name + "--" + side
		i.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name = canary.Name + "--" + side
		if side == Canary {
			i.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/canary"] = "true"
			//float, _ := strconv.ParseFloat(canary.Spec.Strategy.Traffic.Weight, 64)
			//weight := strconv.Itoa(int(float * 100))
			i.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/canary-weight"] = "0"
		}
	}
	namespaced := KClientSet.NetworkingV1().Ingresses(canary.Namespace)
	get, err := namespaced.Get(context.TODO(), i.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		klog.Error(err)
		return
	}
	if get == nil || get.Name == "" {
		_, err1 := namespaced.Create(context.TODO(), &i, metav1.CreateOptions{})
		if err1 != nil {
			klog.Errorf("Create ingress failed. %v", err)
			return
		} else {
			klog.Infof("Create ingress success. %s %s", i.Name, i.Namespace)
		}
	} else {
		update, err1 := namespaced.Update(context.TODO(), &i, metav1.UpdateOptions{})
		if err1 != nil {
			klog.Errorf("Update ingress failed. %v", update)
			return
		} else {
			klog.Infof("Update ingress success. %s %s", i.Name, i.Namespace)
		}
	}
	return
}
