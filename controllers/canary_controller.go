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
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	result, err3, done := reconcileCanary(ctx, req, r)
	if done {
		return result, err3
	}
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
