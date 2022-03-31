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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
		CanaryStoreInstance().del(canary.Namespace, canary.Name)
		return ctrl.Result{}, nil
	} else {
		CanaryStoreInstance().apply(canary)
	}
	stableDeploy, err := findStableDeployment(canary)
	if err == nil && isSameWithStable(stableDeploy) {
		klog.Info("same version with current stable version %s %s %s",
			stableDeploy.Namespace,
			stableDeploy.Name,
			stableDeploy.Spec.Template.Spec.Containers[0].Name)
		if stableDeploy.Spec.Replicas == canary.Spec.Replicas {
			klog.Infof("Replicas also same. Nothing change for this canary. %s %s", canary.Namespace, canary.Name)
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
			val := canary.Spec.Strategy.ScaleInterval
			if val == nil {
				// default 10s ready
				time.Sleep(time.Duration(10) * time.Second)
			} else {
				time.Sleep(time.Duration(val.IntVal) * time.Second)
			}

			err = applyDeployment(canary, Canary, &i)
			if err != nil {
				klog.Error("Scaling canary version error", err)
			}
		}()
	} else if canaryTargetSize == canarySize {
		klog.Infof("canary deployment is ready, setting network.")
		//serviceReconcile(canary)
		//ingressReconcile(canary)
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&cdv1alpha1.Canary{}).
		Complete(r)
}
