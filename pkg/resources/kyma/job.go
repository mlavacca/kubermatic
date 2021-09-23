package kyma

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	jobInstallationName   = resources.KymaJobInstallationName
	kymaContainerName     = "kyma"
	kymaContainerImage    = "piotrkpc/kyma_cli"
	jobUninstallationName = resources.KymaJobUninstallationName
)

func getResourceRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1Gi"),
			corev1.ResourceCPU:    resource.MustParse("200m"),
		},
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.KymaInstallerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.KymaInstallerKubeconfigSecretName,
				},
			},
		},
	}
}

func getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      resources.KymaInstallerKubeconfigSecretName,
			ReadOnly:  true,
			MountPath: "/etc/kubernetes/kubeconfig",
		},
	}
}

func getEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "KUBECONFIG",
			Value: "/etc/kubernetes/kubeconfig/kubeconfig",
		},
	}
}

// InstallationJobCreator returns the function to create and update the Kyma installation job
func InstallationJobCreator(m func(job *batchv1.Job)) reconciling.NamedJobCreatorGetter {
	return func() (string, reconciling.JobCreator) {
		return jobInstallationName, func(job *batchv1.Job) (*batchv1.Job, error) {
			job.Name = jobInstallationName
			job.Labels = resources.BaseAppLabels(jobInstallationName, nil)

			job.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  kymaContainerName,
					Image: kymaContainerImage,
					Args: []string{
						"deploy",
						"--non-interactive",
					},
					Env:          getEnvVars(),
					VolumeMounts: getVolumeMounts(),
					Resources:    getResourceRequirements(),
				},
			}
			job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
			job.Spec.Template.Spec.Volumes = getVolumes()

			if m != nil {
				m(job)
			}

			return job, nil
		}
	}
}

// UninstallationJobCreator returns the function to create and update the Kyma uninstallation job
func UninstallationJobCreator(m func(job *batchv1.Job)) reconciling.NamedJobCreatorGetter {
	return func() (string, reconciling.JobCreator) {
		return jobUninstallationName, func(job *batchv1.Job) (*batchv1.Job, error) {
			job.Name = jobUninstallationName
			job.Labels = resources.BaseAppLabels(jobUninstallationName, nil)

			if job.Annotations == nil {
				job.Annotations = make(map[string]string)
			}
			job.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  kymaContainerName,
					Image: kymaContainerImage,
					Args: []string{
						"delete",
						"--non-interactive",
					},
					Env:          getEnvVars(),
					VolumeMounts: getVolumeMounts(),
					Resources:    getResourceRequirements(),
				},
			}
			job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
			job.Spec.Template.Spec.Volumes = getVolumes()

			if m != nil {
				m(job)
			}

			return job, nil
		}
	}
}
