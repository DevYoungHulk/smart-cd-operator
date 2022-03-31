package controllers

import (
	"context"
	"encoding/json"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"strconv"
	"strings"
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
	canary.Kind = Canary
	canary.APIVersion = canaryGVR.Group + "/" + canaryGVR.Version
	return &canary
}

func updateCanaryStatus(canary cdv1alpha1.Canary) error {
	marshal, err := json.Marshal(&canary)
	if err != nil {
		klog.Error("Update Canary failed", err)
		return nil
	}
	utd := &unstructured.Unstructured{}
	err = json.Unmarshal(marshal, utd)
	if err != nil {
		klog.Error("Update Canary failed", err)
		return nil
	}
	_, err = DClientSet.Resource(canaryGVR).Namespace(canary.Namespace).UpdateStatus(context.TODO(), utd, metav1.UpdateOptions{})
	if err != nil {
		klog.Error("Update Canary failed", err)
		return err
	} else {
		klog.Infof("Update Canary success")
	}
	return nil
}

func updateCanaryStatusVals(deployment *appsv1.Deployment) {
	name := deployment.GetName()
	namespace := deployment.Namespace
	filter := labels.Set(deployment.Spec.Selector.MatchLabels).String()
	list, err2 := KClientSet.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filter})
	if err2 != nil {
		return
	}
	readyReplicas := deployment.Status.ReadyReplicas
	if strings.HasSuffix(name, "--"+Canary) {
		replace := strings.Replace(name, "--"+Canary, "", 1)
		canary := CanaryStoreInstance().get(namespace, replace)
		canary.Status.CanaryReplicasSize = readyReplicas
		if canary.Status.CanaryReplicasSize == 0 &&
			canary.Status.CanaryTargetReplicasSize == 0 &&
			canary.Status.StableTargetReplicasSize == canary.Status.StableReplicasSize {
			canary.Status.Finished = true
			canary.Status.Scaling = false
		}
		err := updateCanaryStatus(*canary)
		if err != nil {
			klog.Error("updateCanaryStatusVals Canary failed.", err)
		}
	} else if strings.HasSuffix(name, "--"+Stable) {
		replace := strings.Replace(name, "--"+Stable, "", 1)
		canary := CanaryStoreInstance().get(namespace, replace)
		canary.Status.StableReplicasSize = getStableReadyCount(list, canary)
		notUpdatedSize := canary.Status.StableTargetReplicasSize - canary.Status.StableReplicasSize
		if notUpdatedSize < canary.Status.CanaryTargetReplicasSize {
			canary.Status.CanaryTargetReplicasSize = notUpdatedSize
			updateCanaryStatus(*canary)
		}
		err := updateCanaryStatus(*canary)
		if err != nil {
			klog.Error("updateCanaryStatusVals Stable failed.", err)
		}
	}
	// all replicas are ready
	if strings.HasSuffix(name, "--"+Canary) {
		go func() {
			// All canary version is ready, start to scaling stable deployment
			//time.Sleep(time.Second * canary.Spec.Strategy.)
		}()
	} else if strings.HasSuffix(name, "--"+Stable) {
		// All canary version is ready, scaling stable deployment is in progress
	}

}

func getStableReadyCount(list *v1.PodList, canary *cdv1alpha1.Canary) int32 {
	i := int32(0)
	for _, pod := range list.Items {
		if isSameWithStable(pod.Spec.Containers, canary.Spec.Template.Spec.Containers) &&
			allContainerReady(pod.Status.ContainerStatuses) {
			i++
		}
	}
	return i
}
func allContainerReady(s []v1.ContainerStatus) bool {
	for _, i := range s {
		if !i.Ready {
			return false
		}
	}
	return true
}

func calcCanaryReplicas(canary *cdv1alpha1.Canary) int32 {
	replicas := *canary.Spec.Replicas
	float, _ := strconv.ParseFloat(canary.Spec.Strategy.PodWeight, 64)
	i := int32(float64(replicas) * float)
	return i
}
