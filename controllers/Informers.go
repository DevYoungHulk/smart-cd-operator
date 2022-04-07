package controllers

import (
	"context"
	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func initInformers(c client.Client) {
	ctx := context.Background()
	podInformer := newInformer()
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*v1.Pod)
			s := pod.Labels[Canary]
			if len(s) == 0 {
				return
			}
			allReady := false
			for _, i := range pod.Status.ContainerStatuses {
				if !i.Ready {
					allReady = true
				}
			}
			if allReady {
				updateCanaryStatusVales(ctx, c, pod)
				klog.Infof("podInformer AddFunc %v", pod.GetName())
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newPod := newObj.(*v1.Pod)
			s := newPod.Labels[Canary]
			if len(s) == 0 {
				return
			}
			oldPod := oldObj.(*v1.Pod)
			oName := oldPod.GetName()
			diff := cmp.Diff(oldPod.Status, newPod.Status) + cmp.Diff(oldPod.Spec, newPod.Spec)
			if len(diff) > 0 {
				allReady := false
				for _, i := range newPod.Status.ContainerStatuses {
					if !i.Ready {
						allReady = true
					}
				}
				if allReady {
					klog.Infof("podInformer UpdateFunc  %s", oName)
					updateCanaryStatusVales(ctx, c, newPod)
				}
			} else {
				klog.Infof("podInformer UpdateFunc nothing changed. name-> %v", oName)
			}
		},
		DeleteFunc: func(obj interface{}) {
			//name := obj.GetName()
			//klog.Infof("podInformer DeleteFunc %v", name)
		},
	})
	go podInformer.Run(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced)
}

func newInformer() cache.SharedIndexInformer {
	var resClient v12.PodInterface
	resClient = KClientSet.CoreV1().Pods("")
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (object runtime.Object, err error) {
				return resClient.List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return resClient.Watch(context.Background(), options)
			},
		},
		&v1.Pod{},
		reSyncPeriod,
		cache.Indexers{},
	)
	return informer
}
