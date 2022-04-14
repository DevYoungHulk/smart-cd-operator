package controllers

import (
	"context"
	"encoding/json"
	cdv1alpha1 "github.com/DevYoungHulk/smart-cd-operator/api/v1alpha1"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//func deploymentReconcile(ctx context.Context, c client.Client, canary *cdv1alpha1.Canary) {
//	if canary == nil {
//		klog.Info("Canary deleted, but deployment not effected. -> %s: %s", canary.Namespace, canary.Name)
//		return
//		//return deleteDeployment(req.Namespace, req.Name)
//	}
//	if canary.Status.Scaling {
//		klog.Infof("Last canary is running waiting finished.")
//		return
//	}
//	stableDeploy, err := findStableDeployment(ctx, c, canary)
//	if err == nil && isSameContainers(stableDeploy.Spec.Template.Spec.Containers, canary.Spec.Template.Spec.Containers) {
//		return
//	}
//	i := calcCanaryReplicas(canary)
//	// create canary version
//	applyDeployment(ctx, c, canary, Canary, &i)
//
//	return
//}

func isSameContainers(containers1 []v1.Container, containers2 []v1.Container) bool {
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

func findStableDeployment(ctx context.Context, c client.Client, canary *cdv1alpha1.Canary) (*appsv1.Deployment, error) {
	stableDeploy := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{Namespace: canary.Namespace, Name: canary.Name + "--" + Stable}
	err := c.Get(ctx, namespacedName, stableDeploy)
	return stableDeploy, err
}

func deleteDeployment(ctx context.Context, c client.Client, namespace string, name string) error {
	klog.Infof("Deleting Deployment namespace:%s name:%s\n", namespace, name)
	deployment := &appsv1.Deployment{}
	deployment.Namespace = namespace
	deployment.Name = name
	err := c.Delete(ctx, deployment)
	if err != nil {
		klog.Infof("Delete Deployment failed namespace:%s name:%s\n", namespace, name)
		return err
	} else {
		klog.Infof("Delete Deployment succesed namespace:%s name:%s\n", namespace, name)
		return nil
	}
}
func updateDeployment(ctx context.Context, c client.Client, deployment *appsv1.Deployment) {
	err := c.Update(ctx, deployment)
	if err != nil {
		klog.Error("UpdateReplicas fail %s %s.", deployment.Namespace, deployment.Name)
	}
}
func applyDeployment(ctx context.Context, c client.Client, canary *cdv1alpha1.Canary, side string, replicas *int32) {
	klog.Infof("Creating Or Updating deployment... namespace:%s name:%s\n", canary.Namespace, canary.Name)

	//namespaced := KClientSet.AppsV1().Deployments(canary.Namespace)
	app := &appsv1.Deployment{}
	name := types.NamespacedName{Namespace: canary.Namespace, Name: canary.Name + "--" + side}
	err := c.Get(ctx, name, app)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorf("ApplyDeployment failed %v", err)
		return
	}
	deploy, err2 := genDeployment(canary, side, replicas)
	if err2 != nil {
		klog.Errorf("ApplyDeployment failed %v", err2)
		return
	}
	if errors.IsNotFound(err) {
		err1 := c.Create(ctx, deploy)
		if err1 != nil {
			klog.Errorf("ApplyDeployment failed %v", err1)
			return
		}
		klog.Infof("Created deployment %q.\n", deploy.GetName())
	} else if err == nil && len(cmp.Diff(app.Spec, deploy.Spec)) > 0 {
		err1 := c.Patch(ctx, deploy, client.Merge)
		if err1 != nil {
			klog.Error(err1)
			return
		}
		klog.Infof("Updated deployment %q.\n", deploy.GetName())
	} else {
		klog.Infof("Deployment replicas not change.")
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
