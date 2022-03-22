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
		klog.Info("Canary deleted, but deployment not effected. -> %s: %s", canary.Namespace, canary.Name)
		return nil
		//return deleteDeployment(req.Namespace, req.Name)
	}
	replicas := *canary.Spec.Deployment.Replicas
	float := canary.Spec.Strategy.PodWeight

	i := int32(float32(replicas) * float)
	return applyDeployment(canary, "canary", &i)
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

func applyDeployment(canary *cdv1alpha1.Canary, side string, targetReplicas *int32) error {
	klog.Infof("Creating Or Updating deployment... namespace:%s name:%s\n", canary.Namespace, canary.Name)

	namespaced := ClientSet.Resource(deployGVR).Namespace(canary.Namespace)
	canaryApp, _ := namespaced.Get(context.TODO(), canary.Name+"--"+side, metav1.GetOptions{})

	utdCanary, err := genDeploymentUtd(canary, "canary", targetReplicas)
	if err != nil {
		klog.Error(err)
		return err
	}
	if canaryApp == nil {
		created, err1 := namespaced.Create(context.TODO(), utdCanary, metav1.CreateOptions{})
		if err1 != nil {
			klog.Error(err1)
			return err1
		}
		klog.Infof("Created deployment %q.\n", created.GetName())
	} else {
		updated, err1 := namespaced.
			Update(context.TODO(), utdCanary, metav1.UpdateOptions{})
		if err1 != nil {
			klog.Error(err1)
			return err1
		}
		klog.Infof("Updated deployment %q.\n", updated.GetName())
	}
	err = serviceReconcile(canary, side)
	if err != nil {
		return err
	}
	//return createServiceAccount(canary)
	getStartTime()
	return serviceMonitorReconcile(canary)
}

func genDeployment(canary *cdv1alpha1.Canary, side string, targetReplicas *int32) (*appsv1.Deployment, error) {
	bytes, err2 := json.Marshal(canary.Spec.Deployment)
	if err2 != nil {
		klog.Error(err2)
		return nil, err2
	}
	deploymentSpec := &appsv1.DeploymentSpec{}
	err2 = json.Unmarshal(bytes, deploymentSpec)
	if err2 != nil {
		klog.Error(err2)
		return nil, err2
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      canary.Name + "--" + side,
			Namespace: canary.Namespace,
		},
		Spec: *deploymentSpec,
	}
	matchLabels := deploymentSpec.Selector.MatchLabels
	labels := deploymentSpec.Template.ObjectMeta.Labels

	deployment.Spec.Replicas = targetReplicas
	matchLabels["canary"] = side
	labels["canary"] = side
	return deployment, nil
}

func genDeploymentUtd(canary *cdv1alpha1.Canary, side string, targetReplicas *int32) (*unstructured.Unstructured, error) {
	deployment, err := genDeployment(canary, side, targetReplicas)
	if err != nil {
		return nil, err
	}
	return objectToJsonUtd(deployment)
}

func genDeploymentPatch(canary *cdv1alpha1.Canary, side string, targetReplicas *int32) ([]byte, error) {
	deployment, err := genDeployment(canary, side, targetReplicas)
	if err != nil {
		return nil, err
	}
	return objectToJsonData(deployment)
}
