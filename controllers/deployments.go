package controllers

import (
	"context"
	"encoding/json"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

func deploymentReconcile(canary *cdv1alpha1.Canary, req ctrl.Request) {
	if canary == nil {
		klog.Info("Canary deleted, but deployment not effected. -> %s: %s", canary.Namespace, canary.Name)
		return
		//return deleteDeployment(req.Namespace, req.Name)
	}
	if canary.Status.Scaling {
		klog.Infof("Last canary is running waiting finished.")
		return
	}
	stableDeploy, err := findStableDeployment(canary)
	if err == nil && isSameWithStable(stableDeploy.Spec.Template.Spec.Containers, canary.Spec.Template.Spec.Containers) {
		return
	}
	i := calcCanaryReplicas(canary)
	// create canary version
	applyDeployment(canary, Canary, &i)

	return
}

func isSameWithStable(containers1 []v1.Container, containers2 []v1.Container) bool {
	if len(containers1) == len(containers2) {
		for i := range containers1 {
			if containers1[i].Image != containers2[i].Image {
				return false
			}
		}
		return true
	}
	return false
}

func findStableDeployment(canary *cdv1alpha1.Canary) (*appsv1.Deployment, error) {
	namespaced := KClientSet.AppsV1().Deployments(canary.Namespace)
	stableDeploy, err := namespaced.Get(context.TODO(), canary.Name+"--"+Stable, metav1.GetOptions{})
	return stableDeploy, err
}

func deleteDeployment(namespace string, name string) error {
	klog.Infof("Deleting Deployment namespace:%s name:%s\n", namespace, name)
	err := KClientSet.AppsV1().Deployments(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		klog.Infof("Delete Deployment failed namespace:%s name:%s\n", namespace, name)
		return err
	} else {
		klog.Infof("Delete Deployment succesed namespace:%s name:%s\n", namespace, name)
		return nil
	}
}
func updateDeployment(deployment appsv1.Deployment) {
	namespaced := KClientSet.AppsV1().Deployments(deployment.Namespace)
	_, err := namespaced.Update(context.TODO(), &deployment, metav1.UpdateOptions{})
	if err != nil {
		klog.Error("UpdateReplicas fail %s %s.", deployment.Namespace, deployment.Name)
	}
}
func applyDeployment(canary *cdv1alpha1.Canary, side string, replicas *int32) {
	klog.Infof("Creating Or Updating deployment... namespace:%s name:%s\n", canary.Namespace, canary.Name)

	namespaced := KClientSet.AppsV1().Deployments(canary.Namespace)
	deploy, err := genDeployment(canary, side, replicas)
	if err != nil {
		klog.Errorf("ApplyDeployment failed %v", err)
		return
	}
	app, err := namespaced.Get(context.TODO(), canary.Name+"--"+side, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf("ApplyDeployment failed %v", err)
		return
	}
	if app == nil || app.Name == "" {
		created, err1 := namespaced.Create(context.TODO(), deploy, metav1.CreateOptions{})
		if err1 != nil {
			klog.Errorf("ApplyDeployment failed %v", err1)
			return
		}
		klog.Infof("Created deployment %q.\n", created.GetName())
	} else {
		updated, err1 := namespaced.
			Update(context.TODO(), deploy, metav1.UpdateOptions{})
		if err1 != nil {
			klog.Error(err1)
			return
		}
		klog.Infof("Updated deployment %q.\n", updated.GetName())
	}
	getStartTime()
	return
}

func genDeployment(canary *cdv1alpha1.Canary, side string, targetReplicas *int32) (*appsv1.Deployment, error) {
	bytes, err2 := json.Marshal(canary.Spec)
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
	matchLabels[Canary] = side
	labels[Canary] = side
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
