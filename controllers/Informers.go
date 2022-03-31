package controllers

import (
	"context"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func initInformers(ctx context.Context) {
	namespace := KClientSet.AppsV1().Deployments("")
	deployInformer := newInformer(namespace, "")
	deployInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment := obj.(*appsv1.Deployment)
			s := deployment.Spec.Selector.MatchLabels[Canary]
			if len(s) == 0 {
				return
			}
			updateCanaryStatusVals(deployment)
			klog.Infof("deployInformer AddFunc %v", deployment.GetName())
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newDeployment := newObj.(*appsv1.Deployment)
			s := newDeployment.Spec.Selector.MatchLabels[Canary]
			if len(s) == 0 {
				return
			}
			oldDeployment := oldObj.(*appsv1.Deployment)
			oname := oldDeployment.GetName()
			diff := cmp.Diff(oldObj, newObj)
			if len(diff) > 0 {
				klog.Infof("deployInformer UpdateFunc  %s", oname)
				updateCanaryStatusVals(newDeployment)
			} else {
				klog.Infof("deployInformer UpdateFunc nothing changed. name-> %v", oname)
			}
		},
		DeleteFunc: func(obj interface{}) {
			name := obj.(*appsv1.Deployment).GetName()
			klog.Infof("deployInformer DeleteFunc %v", name)
		},
	})
	go deployInformer.Run(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), deployInformer.HasSynced)
}

func newInformer(resClient v1.DeploymentInterface, selector string) cache.SharedIndexInformer {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (object runtime.Object, err error) {
				options.LabelSelector = selector
				return resClient.List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = selector
				return resClient.Watch(context.Background(), options)
			},
		},
		&appsv1.Deployment{},
		reSyncPeriod,
		cache.Indexers{},
	)
	return informer
}
