package controllers

import (
	"context"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
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

func reconcileCanary(ctx context.Context, req ctrl.Request, r *CanaryReconciler) error {
	canary := &cdv1alpha1.Canary{}
	err2 := getCanaryByNamespacedName(ctx, r.Client, req.NamespacedName, canary)
	if errors2.IsNotFound(err2) {
		CanaryStoreInstance().del(req.Namespace, req.Name)
		return nil
	} else if err2 != nil {
		return err2
	} else {
		oldCanary := CanaryStoreInstance().get(req.Namespace, req.Name)
		if oldCanary != nil {
			diff := cmp.Diff(oldCanary.Spec, canary.Spec)
			diff2 := cmp.Diff(oldCanary.Status, canary.Status)
			if len(diff) > 0 || len(diff2) > 0 {
				//canary.Status.Finished = false
				klog.Infof("canary spec is changed -> spec %v, status %v", len(diff) > 0, len(diff2) > 0)
				CanaryStoreInstance().update(canary)
			}
		} else {
			CanaryStoreInstance().update(canary)
		}
	}
	if canary.Spec.Paused {
		klog.Infof("Canary %s %s is paused.", canary.Namespace, canary.Name)
		return nil
	}
	if canary.Status.Finished {
		klog.Infof("Canary %s %s is finished.", canary.Namespace, canary.Name)
		return nil
	}

	stableDeploy, err := findStableDeployment(ctx, r.Client, canary)
	if err == nil && isSameWithStable(stableDeploy.Spec.Template.Spec.Containers, canary.Spec.Template.Spec.Containers) {
		klog.Info("same version with current stable version %s %s %s",
			stableDeploy.Namespace,
			stableDeploy.Name,
			stableDeploy.Spec.Template.Spec.Containers[0].Name)
		if *stableDeploy.Spec.Replicas == *canary.Spec.Replicas {
			klog.Infof("Replicas also same. Nothing change for this canary. %s %s", canary.Namespace, canary.Name)
			if canary.Status.CanaryReplicasSize != canary.Status.CanaryTargetReplicasSize {
				ingressReconcile(ctx, r.Client, canary, 0)
				applyDeployment(ctx, r.Client, canary, Canary, &canary.Status.CanaryTargetReplicasSize)
			}
			return nil
		} else {
			stableDeploy.Spec.Replicas = canary.Spec.Replicas
			updateDeployment(ctx, r.Client, stableDeploy)
			return nil
		}
	}
	// stable version not exist.
	if !canary.Status.Scaling {
		canary.Status.Scaling = true
		canary.Status.Finished = false
		canary.Status.CanaryReplicasSize = 0
		canary.Status.StableReplicasSize = 0
		replicas := calcCanaryReplicas(canary)
		canary.Status.StableTargetReplicasSize = *canary.Spec.Replicas
		canary.Status.CanaryTargetReplicasSize = replicas
		err1 := updateCanaryStatus(ctx, r.Client, *canary)
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
			go applyDeployment(ctx, r.Client, canary, Canary, &i)
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

			serviceReconcile(ctx, r.Client, canary)
			stableReplicasSize := canary.Status.StableReplicasSize
			if stableReplicasSize == 0 {
				go ingressReconcile(ctx, r.Client, canary, 1)
				go applyDeployment(ctx, r.Client, canary, Stable, &canary.Status.StableTargetReplicasSize)
				return
			}
			//canaryReplicasSize := canary.Status.CanaryReplicasSize
			//canaryWeight := float64(canaryReplicasSize)/float64(stableReplicasSize) + float64(canaryReplicasSize)
			//if canaryWeight > float {
			//	canaryWeight = float
			//}
			if canary.Status.CanaryReplicasSize != canary.Status.CanaryTargetReplicasSize {
				if canary.Status.CanaryTargetReplicasSize == 0 {
					go ingressReconcile(ctx, r.Client, canary, 0)
				}
				go applyDeployment(ctx, r.Client, canary, Canary, &canary.Status.CanaryTargetReplicasSize)
			} else {
				podWeight, err := strconv.ParseFloat(canary.Spec.Strategy.PodWeight, 64)
				if err != nil {
					klog.Error("Pod Weight parse error, %s", canary.Spec.Strategy.PodWeight)
				}
				canaryMaxReplicas := int32(podWeight * float64(*canary.Spec.Replicas))
				if canary.Status.CanaryReplicasSize >= canaryMaxReplicas {
					podWeight = 1
				} else {
					podWeight = float64(canary.Status.CanaryReplicasSize) / float64(canaryMaxReplicas)
				}
				trafficWeight, err := strconv.ParseFloat(canary.Spec.Strategy.Traffic.Weight, 64)
				if err != nil {
					klog.Error("Traffic Weight parse error, %s", canary.Spec.Strategy.Traffic.Weight)
					trafficWeight = 0
				}
				formatFloat, _ := FormatFloat(trafficWeight*podWeight, 2)
				ingressReconcile(ctx, r.Client, canary, formatFloat)
				go applyDeployment(ctx, r.Client, canary, Stable, &canary.Status.StableTargetReplicasSize)
			}
		}()
	}
	return nil
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
		klog.Infof("Update Canary success")
	}
	return nil
}

func updateCanaryStatusVales(ctx context.Context, c client.Client, deployment *appsv1.Deployment) {
	namespace := deployment.Namespace
	name := deployment.Name

	list := &v1.PodList{}
	selector := labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels)
	options := &client.ListOptions{LabelSelector: selector, Namespace: namespace}
	err2 := c.List(ctx, list, options)
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
		err := updateCanaryStatus(ctx, c, *canary)
		if err != nil {
			klog.Error("updateCanaryStatusVales Canary failed.", err)
		}
	} else if strings.HasSuffix(name, "--"+Stable) {
		replace := strings.Replace(name, "--"+Stable, "", 1)
		canary := CanaryStoreInstance().get(namespace, replace)
		canary.Status.StableReplicasSize = getStableReadyCount(list, canary)
		notUpdatedSize := canary.Status.StableTargetReplicasSize - canary.Status.StableReplicasSize
		if notUpdatedSize < canary.Status.CanaryTargetReplicasSize {
			canary.Status.CanaryTargetReplicasSize = notUpdatedSize
			err := updateCanaryStatus(ctx, c, *canary)
			if err != nil {
				klog.Error("updateCanaryStatusVales Stable failed.", err)
			}
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
