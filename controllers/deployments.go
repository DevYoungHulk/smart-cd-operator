package controllers

import (
	"context"
	"encoding/json"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

func deploymentReconcile(canary *cdv1alpha1.Canary, req ctrl.Request) error {
	if canary == nil {
		return deleteDeployment(req.Namespace, req.Name)
	}
	return createOrUpdateDeployment(canary)
}

func deleteDeployment(namespace string, name string) error {
	klog.Infof("Deleting Deployment namespace:%s name:%s\n", namespace, name)
	err := ClientSet.Resource(deployGVR).Namespace(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		klog.Infof("Delete Deployment failed namespace:%s name:%s\n", namespace, name)
		return err
	} else {
		klog.Infof("Delete Deployment succesed namespace:%s name:%s\n", namespace, name)
		return nil
	}
}

func createOrUpdateDeployment(canary *cdv1alpha1.Canary) error {
	klog.Infof("Creating Or Updating deployment... namespace:%s name:%s\n", canary.Namespace, canary.Name)
	bytes, err2 := json.Marshal(canary.Spec.Deployment)
	if err2 != nil {
		klog.Error(err2)
		return err2
	}
	deploymentSpec := &appsv1.DeploymentSpec{}
	err2 = json.Unmarshal(bytes, deploymentSpec)
	if err2 != nil {
		klog.Error(err2)
		return err2
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      canary.Name,
			Namespace: canary.Namespace,
		},
		Spec: *deploymentSpec,
	}
	marshal, err := json.Marshal(&deployment)
	if err != nil {
		klog.Error(err)
		return err
	}
	utd := &unstructured.Unstructured{}
	if err = json.Unmarshal(marshal, &utd.Object); err != nil {
		klog.Error(err)
		return err
	}
	namespace := ClientSet.Resource(deployGVR).Namespace(canary.Namespace)
	get, _ := namespace.Get(context.TODO(), canary.Name, metav1.GetOptions{})

	if get == nil {
		create, err := namespace.
			Create(context.TODO(), utd, metav1.CreateOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Created deployment %q.\n", create.GetName())

	} else {
		update, err := namespace.
			Update(context.TODO(), utd, metav1.UpdateOptions{})
		if err != nil {
			klog.Error(err)
			return err
		}
		klog.Infof("Updated deployment %q.\n", update.GetName())
	}

	return nil
}
