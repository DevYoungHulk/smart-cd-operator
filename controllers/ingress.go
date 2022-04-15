package controllers

import (
	"context"
	"github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

func ingressReconcile(ctx context.Context, c client.Client, canary *cdv1alpha1.Canary) {
	canaryTraffic, err := strconv.ParseFloat(canary.Status.CanaryTraffic, 64)
	if err != nil {
		canaryTraffic = 0
	}
	klog.Infof("canary traffic (%v)", canaryTraffic)
	if canary.Spec.Strategy.Traffic.TType == Nginx {

		genIngress(ctx, c, canary, Stable, 1-canaryTraffic)
		genIngress(ctx, c, canary, Canary, canaryTraffic)
	} else {
		klog.Warning("istio & traefik support is building....")
	}
}

func genIngress(ctx context.Context, c client.Client, canary *v1alpha1.Canary, side string, weight float64) {
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
					Host: canary.Spec.Strategy.Traffic.Host,
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
			if side == Canary {
				i.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/canary-weight"] = strconv.Itoa(int(weight * 100))
			}
		}
	}
	get := &v1.Ingress{}
	err := c.Get(ctx, types.NamespacedName{Namespace: i.Namespace, Name: i.Name}, get)
	if err != nil && !errors.IsNotFound(err) {
		klog.Error(err)
		return
	}
	if get == nil || get.Name == "" {
		err1 := c.Create(ctx, &i)
		if err1 != nil {
			klog.Errorf("Create ingress failed. %v", err)
			return
		} else {
			klog.Infof("Create ingress success. %s %s", i.Name, i.Namespace)
		}
	} else if i.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/canary-weight"] !=
		get.ObjectMeta.Annotations["nginx.ingress.kubernetes.io/canary-weight"] {
		err1 := c.Update(ctx, &i)
		if err1 != nil {
			klog.Errorf("Update ingress failed. %v", i)
			return
		} else {
			klog.Infof("Update ingress success. %s %s", i.Name, i.Namespace)
		}
	} else {
		klog.Infof("Ingress weight not change.")
	}
	return
}
