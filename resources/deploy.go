//
// Copyright 2022 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package resources

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/IBM/policy-control-operator/api/v1alpha1"
)

func BuildDeploymentForKyverno(cr *v1alpha1.PolicyControl) *appsv1.Deployment {
	normalizedWorkspace := normalizeWorkdpaceName(cr)
	advertisedUrl := fmt.Sprintf("%s:%d/%s", cr.Spec.PolicyControlCluster.IngressHost, cr.Spec.PolicyControlCluster.IngressPort, normalizedWorkspace)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      normalizedWorkspace,
			Namespace: cr.Spec.PolicyControlCluster.Namespace,
			Labels: map[string]string{
				"app":       "kyverno-controller",
				"workspace": normalizedWorkspace,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "kyverno-controller",
					"workspace": normalizedWorkspace,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "kyverno-controller",
						"workspace": normalizedWorkspace,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "kyverno",
							Image: cr.Spec.KyvernoInWorkspace.KyvernoImage,
							Args: []string{
								"-v=4",
								"--kubeconfig=/tmp/kyverno-runtime-credentials/target-kubeconfig.yaml",
								"--serverIP=" + advertisedUrl,
							},
							Env: []corev1.EnvVar{{
								Name:  "KYVERNO_SVC",
								Value: "kyverno-svc-remote",
							}},
							Ports: []corev1.ContainerPort{{
								Name:          "http",
								Protocol:      corev1.ProtocolTCP,
								ContainerPort: 9443,
							}},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      "kyverno-runtime-credentials",
								MountPath: "/tmp/kyverno-runtime-credentials",
								ReadOnly:  true,
							}},
						},
					},
					Volumes: []corev1.Volume{{
						Name: "kyverno-runtime-credentials",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: normalizedWorkspace,
							},
						},
					}},
				},
			},
		},
	}
	return deployment
}
