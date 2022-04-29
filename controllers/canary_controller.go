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
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CanaryReconciler reconciles a Canary object
type CanaryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=*
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=*
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
	//return ctrl.Result{}, nil
	return ctrl.Result{}, reconcileCanary(ctx, r, req)
}
func reconcileCanary(ctx context.Context, r *CanaryReconciler, req ctrl.Request) error {
	list := &cdv1alpha1.CanaryList{}
	err2 := r.List(context.TODO(), list)
	if err2 == nil {
		for _, i := range list.Items {
			CanaryStoreInstance().update(&i)
		}
	} else {
		return err2
	}
	if KClientSet == nil {
		Init(r.Client)
	}
	canary, err := updateLocalCache(ctx, req, r)
	if err != nil {
		return err
	}
	if canary == nil {
		klog.Infof("Canary is deleted %s/%s", req.NamespacedName, req.Name)
		return nil
	}
	if canary.IsPaused() {
		klog.Infof("Canary %s/%s is paused.", canary.Namespace, canary.Name)
		return nil
	}
	//if canary.Status.Finished {
	//	klog.Infof("Canary %s/%s is finished.", canary.Namespace, canary.Name)
	//	return nil
	//}
	go func() {
		// stable version not exist.
		if !canary.Status.Scaling {
			stableDeploy, err := FindStableDeployment(ctx, r.Client, canary)
			if err == nil {
				if isSameContainers(stableDeploy.Spec.Template.Spec.Containers, canary.Spec.Template.Spec.Containers) {
					if *stableDeploy.Spec.Replicas != *canary.Spec.Replicas {
						stableDeploy.Spec.Replicas = canary.Spec.Replicas
						updateDeployment(ctx, r.Client, stableDeploy)
					} else {
						go applyDeployment(ctx, r.Client, canary, Canary, &canary.Status.CanaryReplicasSize)
						klog.Infof("same version")
					}
					return
				}
			}

			canary.Status.Scaling = true
			canary.Status.CanaryReplicasSize = 1
			canary.Status.StableReplicasSize = 0

			if stableDeploy.Spec.Replicas == nil {
				canary.Status.OldStableReplicasSize = 0
			} else {
				canary.Status.OldStableReplicasSize = *stableDeploy.Spec.Replicas
			}
			updateCanaryStatus(ctx, r.Client, *canary)
			return
		}
		//canaryMaxReplicas := calcCanaryReplicas(canary)

		serviceReconcile(ctx, r.Client, canary)

		//klog.Infof("ingressReconcile CanaryReplicasSize %d,"+
		//	"\nStableReplicasSize %d,\nOldStableReplicasSize %d",
		//	canary.Status.CanaryReplicasSize,
		//	canary.Status.StableReplicasSize,
		//	canary.Status.OldStableReplicasSize)
		//canaryVersionIsZero := canary.Status.CanaryReplicasSize == 0
		//stableVersionIsAllReady := canary.Status.StableReplicasSize == *canary.Spec.Replicas
		//if canaryVersionIsZero || stableVersionIsAllReady {
		//	klog.Infof("canaryVersionIsZero %v, stableVersionIsAllReady %v",
		//		canaryVersionIsZero, stableVersionIsAllReady)
		//	go ingressReconcile(ctx, r.Client, canary)
		//} else {
		//	if canary.Status.OldStableReplicasSize == 0 {
		//		klog.Infof("Not have old stable version, setting all traffic to canary.")
		//		go ingressReconcile(ctx, r.Client, canary)
		//	} else if canary.Status.OldStableReplicasSize != 0 {
		//		formatFloat := calcCanaryPodWeight(canary)
		//		klog.Infof("Have old stable version, %d/%d canary version is ready, setting %f traffic to canary.",
		//			canary.Status.CanaryReplicasSize,
		//			canaryMaxReplicas,
		//			formatFloat)
		//		ingressReconcile(ctx, r.Client, canary)
		//	}
		//}

		klog.Infof("scaling ---------------")
		go applyDeployment(ctx, r.Client, canary, Canary, &canary.Status.CanaryReplicasSize)
		if canary.Status.StableReplicasSize != 0 {
			go applyDeployment(ctx, r.Client, canary, Stable, &canary.Status.StableReplicasSize)
		}
	}()
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CanaryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cdv1alpha1.Canary{}).
		Complete(r)
}
