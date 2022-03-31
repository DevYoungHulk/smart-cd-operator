/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"time"

	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
)

// CanaryReconciler reconciles a Canary object
type CanaryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=*
//+kubebuilder:rbac:groups="",resources=configmaps;endpoints;events;persistentvolumeclaims;pods;namespaces;secrets;serviceaccounts;services;services/finalizers,verbs=*
//+kubebuilder:rbac:groups=apps,resources=deployments;replicasets;daemonsets;statefulsets,verbs=*
//+kubebuilder:rbac:groups=cd.org.smart,resources=canaries,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cd.org.smart,resources=canaries/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cd.org.smart,resources=canaries/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Canary object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *CanaryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	canary := getCanary(&ctx, req.Namespace, req.Name)
	if canary == nil {
		CanaryStoreInstance().del(req.Namespace, req.Name)
		return ctrl.Result{}, nil
	} else {
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
			return ctrl.Result{}, nil
		} else {
			stableDeploy.Spec.Replicas = canary.Spec.Replicas
			updateDeployment(*stableDeploy)
			return ctrl.Result{}, nil
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
		return ctrl.Result{}, err1
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

			err = applyDeployment(canary, Canary, &i)
			if err != nil {
				klog.Error("Scaling canary version error", err)
			}
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
				ingressReconcile(canary, 1)
				applyDeployment(canary, Stable, &canary.Status.StableTargetReplicasSize)
				return
			}
			//canaryReplicasSize := canary.Status.CanaryReplicasSize
			//canaryWeight := float64(canaryReplicasSize)/float64(stableReplicasSize) + float64(canaryReplicasSize)
			//if canaryWeight > float {
			//	canaryWeight = float
			//}
			if canary.Status.CanaryReplicasSize != canary.Status.CanaryTargetReplicasSize {
				if canary.Status.CanaryTargetReplicasSize == 0 {
					ingressReconcile(canary, 0)
				}
				applyDeployment(canary, Canary, &canary.Status.CanaryTargetReplicasSize)
			} else {
				float, _ := strconv.ParseFloat(canary.Spec.Strategy.Traffic.Weight, 64)
				ingressReconcile(canary, float)
				applyDeployment(canary, Stable, &canary.Status.StableTargetReplicasSize)
			}
		}()

	}

	//go deploymentReconcile(canary, req)
	//go serviceReconcile(canary)
	//go serviceMonitorReconcile(canary)
	//go ingressReconcile(canary)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CanaryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	Init()
	initInformers(context.Background())
	list, err := DClientSet.Resource(canaryGVR).Namespace("").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, item := range list.Items {
		marshal, err := json.Marshal(item.Object)
		if err != nil {
			panic(err)
		}
		canary := cdv1alpha1.Canary{}
		err1 := json.Unmarshal(marshal, &canary)
		if err1 != nil {
			panic(err1)
		}
		CanaryStoreInstance().update(&canary)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&cdv1alpha1.Canary{}).
		Complete(r)
}
