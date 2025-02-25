package psact

import (
	"fmt"
	"time"

	"github.com/rancher/rancher/tests/framework/clients/rancher"
	steveV1 "github.com/rancher/rancher/tests/framework/clients/rancher/v1"
	"github.com/rancher/rancher/tests/framework/extensions/workloads"
	namegenerator "github.com/rancher/rancher/tests/framework/pkg/namegenerator"
	"github.com/sirupsen/logrus"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kwait "k8s.io/apimachinery/pkg/util/wait"
)

const (
	containerName     = "nginx"
	deploymentName    = "nginx"
	imageName         = "nginx"
	namespace         = "default"
	rancherPrivileged = "rancher-privileged"
	rancherRestricted = "rancher-restricted"
	workload          = "workload"
)

// CreateTestDeployment will create an nginx deployment into the default namespace. If the PSACT value is rancher-privileged, then the
// deployment should successfully create. If the PSACT value is rancher-unprivileged, then the deployment should fail to create.
func CreateNginxDeployment(client *rancher.Client, clusterID string, psact string) (*steveV1.SteveAPIObject, error) {
	labels := map[string]string{}
	labels["workload.user.cattle.io/workloadselector"] = fmt.Sprintf("apps.deployment-%v-%v", namespace, workload)

	containerTemplate := workloads.NewContainer(containerName, imageName, v1.PullAlways, []v1.VolumeMount{}, []v1.EnvFromSource{})
	podTemplate := workloads.NewPodTemplate([]v1.Container{containerTemplate}, []v1.Volume{}, []v1.LocalObjectReference{}, labels)
	deploymentTemplate := workloads.NewDeploymentTemplate(deploymentName, namespace, podTemplate, true, labels)

	steveclient, err := client.Steve.ProxyDownstream(clusterID)
	if err != nil {
		return nil, err
	}

	// If the deployment already exists, then create a new deployment with a different name to avoid a naming conflict.
	if _, err := steveclient.SteveType(workloads.DeploymentSteveType).ByID(deploymentTemplate.Namespace + "/" + deploymentTemplate.Name); err == nil {
		deploymentTemplate.Name = deploymentTemplate.Name + "-" + namegenerator.RandStringLower(5)
	}

	_, err = steveclient.SteveType(workloads.DeploymentSteveType).Create(deploymentTemplate)
	if err != nil {
		return nil, err
	}

	err = kwait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		steveclient, err := client.Steve.ProxyDownstream(clusterID)
		if err != nil {
			return false, err
		}

		deploymentResp, err := steveclient.SteveType(workloads.DeploymentSteveType).ByID(deploymentTemplate.Namespace + "/" + deploymentTemplate.Name)
		if err != nil {
			// We don't want to return the error so we don't exit the poll too soon.
			// There could be delay of when the deployment is created.
			return false, nil
		}

		deployment := &appv1.Deployment{}
		err = steveV1.ConvertToK8sType(deploymentResp.JSONResp, deployment)
		if err != nil {
			return false, err
		}

		if *deployment.Spec.Replicas == deployment.Status.AvailableReplicas && (psact == "" || psact == rancherPrivileged) {
			logrus.Infof("Deployment %s successfully created; this is expected for %s!", deployment.Name, psact)
			return true, nil
		} else if *deployment.Spec.Replicas != deployment.Status.AvailableReplicas && psact == rancherRestricted {
			logrus.Infof("Deployment %s failed to create; this is expected for %s!", deployment.Name, psact)
			return true, nil
		}

		return false, nil
	})

	return nil, err
}
