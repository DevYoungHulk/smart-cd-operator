package controllers

import (
	"context"
	"encoding/json"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"time"
)

func reconcileCanary(ctx context.Context, req ctrl.Request, r *CanaryReconciler) error {
	canary := &cdv1alpha1.Canary{}
	c, err2 := getCanaryByNamespacedName(r, &ctx, req.NamespacedName)
	if err2 != nil {
		return err2
	} else if c != nil {
		CanaryStoreInstance().del(req.Namespace, req.Name)
		return nil
	} else {
		//canary := getCanary(&ctx, req.Namespace, req.Name)
		oldCanary := CanaryStoreInstance().get(req.Namespace, req.Name)
		if oldCanary != nil {
			diff := cmp.Diff(oldCanary.Spec, canary.Spec)
			if len(diff) > 0 {
				//canary.Status.Finished = false
				klog.Infof("canary spec is changed -> %s", diff)
			}
			diff2 := cmp.Diff(oldCanary.Status, canary.Status)
			if len(diff2) > 0 {
				klog.Infof("canary status is changed -> %s", diff2)
			}
		}
		CanaryStoreInstance().update(canary)
	}

	stableDeploy, err := findStableDeployment(canary)
	if err == nil && isSameWithStable(stableDeploy.Spec.Template.Spec.Containers, canary.Spec.Template.Spec.Containers) {
		klog.Info("same version with current stable version %s %s %s",
			stableDeploy.Namespace,
			stableDeploy.Name,
			stableDeploy.Spec.Template.Spec.Containers[0].Name)
		if *stableDeploy.Spec.Replicas == *canary.Spec.Replicas {
			klog.Infof("Replicas also same. Nothing change for this canary. %s %s", canary.Namespace, canary.Name)
			if canary.Status.CanaryReplicasSize != canary.Status.CanaryTargetReplicasSize {
				ingressReconcile(canary, 0)
				applyDeployment(canary, Canary, &canary.Status.CanaryTargetReplicasSize)
			}
			return nil
		} else {
			stableDeploy.Spec.Replicas = canary.Spec.Replicas
			updateDeployment(*stableDeploy)
			return nil
		}
	}
	// stable version not exist.
	if !canary.Status.Scaling {
		canary.Status.Scaling = true
		canary.Status.Finished = false
		canary.Status.Pause = false
		canary.Status.CanaryReplicasSize = 0
		canary.Status.StableReplicasSize = 0
		replicas := calcCanaryReplicas(canary)
		canary.Status.StableTargetReplicasSize = *canary.Spec.Replicas
		canary.Status.CanaryTargetReplicasSize = replicas
		err1 := updateCanaryStatus(*canary)
		return err1
	}
	klog.Infof("scaling")
	canaryTargetSize := canary.Status.CanaryTargetReplicasSize
	canarySize := canary.Status.CanaryReplicasSize
	if canaryTargetSize > canarySize {
		i := canarySize + 1
		go func() {
			if i != 1 {
				val := canary.Spec.Strategy.ScaleInterval
				if val == nil {
					// default 10s ready
					time.Sleep(time.Duration(10) * time.Second)
				} else {
					time.Sleep(time.Duration(val.IntVal) * time.Second)
				}
			}
			go applyDeployment(canary, Canary, &i)
		}()
	} else {
		go func() {
			val := canary.Spec.Strategy.ScaleInterval
			if val == nil {
				// default 10s ready
				time.Sleep(time.Duration(10) * time.Second)
			} else {
				time.Sleep(time.Duration(val.IntVal) * time.Second)
			}
			klog.Infof("canary deployment is ready, setting network.")

			serviceReconcile(canary)
			stableReplicasSize := canary.Status.StableReplicasSize
			if stableReplicasSize == 0 {
				go ingressReconcile(canary, 1)
				go applyDeployment(canary, Stable, &canary.Status.StableTargetReplicasSize)
				return
			}
			//canaryReplicasSize := canary.Status.CanaryReplicasSize
			//canaryWeight := float64(canaryReplicasSize)/float64(stableReplicasSize) + float64(canaryReplicasSize)
			//if canaryWeight > float {
			//	canaryWeight = float
			//}
			if canary.Status.CanaryReplicasSize != canary.Status.CanaryTargetReplicasSize {
				if canary.Status.CanaryTargetReplicasSize == 0 {
					go ingressReconcile(canary, 0)
				}
				go applyDeployment(canary, Canary, &canary.Status.CanaryTargetReplicasSize)
			} else {
				float, _ := strconv.ParseFloat(canary.Spec.Strategy.Traffic.Weight, 64)
				ingressReconcile(canary, float)
				go applyDeployment(canary, Stable, &canary.Status.StableTargetReplicasSize)
			}
		}()
	}
	return nil
}

func getCanary(client client.Client, ctx *context.Context, namespace string, name string) (*cdv1alpha1.Canary, error) {
	return getCanaryByNamespacedName(client, ctx, types.NamespacedName{Namespace: namespace, Name: name})
}
func getCanaryByNamespacedName(client client.Client, ctx *context.Context, namespacedName client.ObjectKey) (*cdv1alpha1.Canary, error) {
	canary := &cdv1alpha1.Canary{}
	err := client.Get(*ctx, namespacedName, canary)
	if err != nil {
		if errors2.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return canary, nil
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
