package kyma

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	jobInstallationName   = resources.JobInstallationName
	kymaContainerName     = "kyma"
	kymaContainerImage    = "nginx"
	jobUninstallationName = resources.JobUninstallationName
)

// ControllerJobCreator returns the function to create and update the Kyma installation job
func ControllerJobCreator() reconciling.NamedJobCreatorGetter {
	return func() (string, reconciling.JobCreator) {
		return jobInstallationName, func(job *batchv1.Job) (*batchv1.Job, error) {
			job.Name = jobInstallationName
			//job.Labels = resources.BaseAppLabels(jobInstallationName, gatekeeperControllerLabels)

			if job.Annotations == nil {
				job.Annotations = make(map[string]string)
			}
			//job.Annotations["container.seccomp.security.alpha.kubernetes.io/manager"] = "runtime/default"

			/*job.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(controllerName, gatekeeperControllerLabels),
			}*/

			/*job.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: resources.BaseAppLabels(controllerName, gatekeeperControllerLabels),
			}*/

			//job.Spec.Template.Spec.TerminationGracePeriodSeconds = pointer.Int64Ptr(60)
			//job.Spec.Template.Spec.NodeSelector = map[string]string{"kubernetes.io/os": "linux"}
			//job.Spec.Template.Spec.ServiceAccountName = serviceAccountName
			//job.Spec.Template.Spec.PriorityClassName = "system-cluster-critical"
			job.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  kymaContainerName,
					Image: kymaContainerImage,
				},
			}
			job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
			err := resources.SetResourceRequirements(job.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, job.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			/*job.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: resources.GatekeeperWebhookServerCertSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.GatekeeperWebhookServerCertSecretName,
						},
					},
				},
			}*/

			return job, nil
		}
	}
}

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		jobInstallationName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("1"),
			},
		},
		jobUninstallationName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("1"),
			},
		},
	}
)
