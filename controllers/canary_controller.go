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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
)

var canaryGVR = schema.GroupVersionResource{
	Group:    "cd.org.smart",
	Version:  "v1alpha1",
	Resource: "canaries",
}
var deployGVR = schema.GroupVersionResource{
	Group:    "apps",
	Version:  "v1",
	Resource: "deployments",
}

// CanaryReconciler reconciles a Canary object
type CanaryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

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
		deleteDeployment(req.Namespace, req.Name)
		return ctrl.Result{}, nil
	}
	createOrUpdateDeployment(canary)

	return ctrl.Result{}, nil
}

func deleteDeployment(namespace string, name string) {
	klog.Infof("Deleting Deployment namespace:%s name:%s\n", namespace, name)
	err := clientset.Resource(deployGVR).Namespace(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		klog.Infof("Delete Deployment failed namespace:%s name:%s\n", namespace, name)
	} else {
		klog.Infof("Delete Deployment succesed namespace:%s name:%s\n", namespace, name)
	}
}

func createOrUpdateDeployment(canary *cdv1alpha1.Canary) {
	klog.Infof("Creating Or Updating deployment... namespace:%s name:%s\n", canary.Namespace, canary.Name)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      canary.Name,
			Namespace: canary.Namespace,
		},
		Spec: canary.Spec.Deployment,
	}
	deployment.Spec.Template.ObjectMeta.Labels = canary.Spec.Deployment.Selector.MatchLabels
	marshal, err := json.Marshal(&deployment)
	if err != nil {
		klog.Error(err)
		return
	}
	utd := &unstructured.Unstructured{}
	if err = json.Unmarshal(marshal, &utd.Object); err != nil {
		klog.Error(err)
		return
	}
	namespace := clientset.Resource(deployGVR).Namespace(canary.Namespace)
	get, _ := namespace.Get(context.TODO(), canary.Name, metav1.GetOptions{})

	if get == nil {
		create, err := namespace.
			Create(context.TODO(), utd, metav1.CreateOptions{})
		if err != nil {
			klog.Error(err)
			return
		}
		klog.Infof("Created deployment %q.\n", create.GetName())

	} else {
		update, err := namespace.
			Update(context.TODO(), utd, metav1.UpdateOptions{})
		if err != nil {
			klog.Error(err)
			return
		}
		klog.Infof("Updated deployment %q.\n", update.GetName())
	}

}

func getCanary(ctx *context.Context, namespace string, name string) *cdv1alpha1.Canary {
	list, err := clientset.Resource(canaryGVR).Namespace(namespace).Get(*ctx, name, metav1.GetOptions{})
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
	return &canary
}

func initClientSet() dynamic.Interface {
	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	clientset, err := dynamic.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	return clientset
}

var clientset dynamic.Interface
var once sync.Once

// SetupWithManager sets up the controller with the Manager.
func (r *CanaryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	once.Do(func() {
		clientset = initClientSet()
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&cdv1alpha1.Canary{}).
		Complete(r)
}
