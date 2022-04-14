package controllers

import (
	"context"
	"fmt"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"time"
)

const defaultScaleInterval = time.Duration(30) * time.Second

func canaryTest(canary *cdv1alpha1.Canary) {
	klog.Infof("Canary test sample waiting start .... we can also query prometheus or kibana here")
	val := canary.Spec.Strategy.ScaleInterval
	waitTime := defaultScaleInterval
	if val != nil {
		duration, _ := time.ParseDuration(val.StrVal)
		waitTime = duration
	}
	klog.Infof("Sleeping... Waiting for scaling canary version, interval is %f s", waitTime.Seconds())
	time.Sleep(waitTime)
}

func calcCanaryWeight(canary *cdv1alpha1.Canary) string {
	if canary.Status.OldStableReplicasSize == 0 && canary.Status.StableReplicasSize == 0 {
		return "1"
	}
	var canaryPodWeight float64
	canaryMaxReplicas := calcCanaryReplicas(canary)
	if canary.Status.CanaryReplicasSize >= canaryMaxReplicas {
		canaryPodWeight = 1
	} else {
		canaryPodWeight = float64(canary.Status.CanaryReplicasSize) / float64(canaryMaxReplicas)
	}
	trafficWeight, err := strconv.ParseFloat(canary.Spec.Strategy.Traffic.Weight, 64)
	if err != nil {
		klog.Error("Traffic Weight parse error, %s", canary.Spec.Strategy.Traffic.Weight)
		trafficWeight = 0
	}
	formatFloat, _ := FormatFloat(trafficWeight*canaryPodWeight, 2)
	return fmt.Sprintf("%v", formatFloat)
}

func updateLocalCache(ctx context.Context, req ctrl.Request, r *CanaryReconciler) (*cdv1alpha1.Canary, error) {
	canary := &cdv1alpha1.Canary{}
	err := getCanaryByNamespacedName(ctx, r.Client, req.NamespacedName, canary)
	if errors2.IsNotFound(err) {
		CanaryStoreInstance().del(req.Namespace, req.Name)
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		oldCanary := CanaryStoreInstance().get(req.Namespace, req.Name)
		if oldCanary != nil {
			diff := cmp.Diff(oldCanary.Spec, canary.Spec)
			diff2 := cmp.Diff(oldCanary.Status, canary.Status)
			diff3 := cmp.Diff(oldCanary.Labels, canary.Labels)
			diff4 := cmp.Diff(oldCanary.Annotations, canary.Annotations)
			if len(diff) > 0 || len(diff2) > 0 || len(diff3) > 0 || len(diff4) > 0 {
				//canary.Status.Finished = false
				klog.Infof("canary spec is changed -> spec %v, status %v, labels %v, annotations %v",
					len(diff) > 0, len(diff2) > 0, len(diff3) > 0, len(diff4) > 0)
				CanaryStoreInstance().update(canary)
			} else {
				klog.Infof("Canary not change.")
			}
		} else {
			CanaryStoreInstance().update(canary)
		}
	}
	return canary, nil
}

func getCanary(ctx context.Context, client client.Client, namespace string, name string, canary *cdv1alpha1.Canary) error {
	return getCanaryByNamespacedName(ctx, client, types.NamespacedName{Namespace: namespace, Name: name}, canary)
}
func getCanaryByNamespacedName(ctx context.Context, client client.Client, namespacedName client.ObjectKey, canary *cdv1alpha1.Canary) error {
	return client.Get(ctx, namespacedName, canary)
}

func updateCanaryStatus(ctx context.Context, client client.Client, canary cdv1alpha1.Canary) error {
	err := client.Status().Update(ctx, &canary)
	if err != nil {
		klog.Error("Update Canary failed", err)
		return err
	} else {
		klog.Infof("Update Canary success, status %v", canary.Status)
	}
	return nil
}

func updateCanaryStatusVales(ctx context.Context, c client.Client, pod *v1.Pod) {

	namespace := pod.Namespace
	podName := pod.Name
	list := &v1.PodList{}
	delete(pod.Labels, Canary)
	selector := labels.SelectorFromSet(pod.Labels)
	options := &client.ListOptions{LabelSelector: selector, Namespace: namespace}
	err2 := c.List(ctx, list, options)
	if err2 != nil {
		return
	}
	var canary *cdv1alpha1.Canary
	split := strings.Split(podName, "--")

	canary = CanaryStoreInstance().get(namespace, split[0])
	if canary == nil {
		return
	}

	stableCount, canaryCount := getReadyPodsCount(list, canary)
	canaryMaxReplicas := calcCanaryReplicas(canary)

	if canaryCount >= canaryMaxReplicas {
		canary.Status.CanaryTraffic = calcCanaryWeight(canary)
		canary.Status.StableReplicasSize = *canary.Spec.Replicas
	} else if canary.Status.StableReplicasSize == 0 {
		canary.Status.CanaryTraffic = calcCanaryWeight(canary)
		canary.Status.CanaryReplicasSize = canaryCount + 1
	} else {
		notUpdatedSize := *canary.Spec.Replicas - stableCount
		if notUpdatedSize < canary.Status.CanaryReplicasSize {
			canary.Status.CanaryReplicasSize = notUpdatedSize
			canary.Status.CanaryTraffic = calcCanaryWeight(canary)
			if notUpdatedSize == 0 {
				canary.Status.CanaryTraffic = "0"
				canary.Status.OldStableReplicasSize = 0
				canary.Status.Scaling = false
			}
		}
	}
	ingressReconcile(ctx, c, canary)

	if len(cmp.Diff(CanaryStoreInstance().get(canary.Namespace, canary.Name), canary)) == 0 {
		// nothing change
		return
	}
	canaryTest(canary)
	if canary.IsPaused() {
		klog.Infof("Canary is paused......")
		return
	}
	err := updateCanaryStatus(ctx, c, *canary)
	if err != nil {
		klog.Error("updateCanaryStatusVales Canary failed.", err)
	}
}

func getReadyPodsCount(list *v1.PodList, canary *cdv1alpha1.Canary) (int32, int32) {
	stableCount := int32(0)
	canaryCount := int32(0)
	for _, pod := range list.Items {

		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.ContainersReady && condition.Status == v1.ConditionTrue {
				if isSameContainers(pod.Spec.Containers, canary.Spec.Template.Spec.Containers) {
					if pod.Labels[Canary] == Canary {
						canaryCount++
					} else if pod.Labels[Canary] == Stable {
						stableCount++
					}
				}
			}
		}

	}
	return stableCount, canaryCount
}

func calcCanaryReplicas(canary *cdv1alpha1.Canary) int32 {
	replicas := *canary.Spec.Replicas
	float, _ := strconv.ParseFloat(canary.Spec.Strategy.PodWeight, 64)
	i := int32(float64(replicas) * float)
	return i
}
