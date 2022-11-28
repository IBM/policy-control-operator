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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/IBM/policy-control-operator/api/v1alpha1"
)

func BuildServiceForKyverno(cr *v1alpha1.PolicyControl) *corev1.Service {
	normalizedWorkspace := normalizeWorkdpaceName(cr)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      normalizedWorkspace,
			Namespace: cr.Spec.PolicyControlCluster.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "kyverno-controller", "workspace": normalizedWorkspace},
			Ports: []corev1.ServicePort{{
				Protocol:   "TCP",
				Port:       int32(cr.Spec.PolicyControlCluster.IngressPort),
				TargetPort: intstr.FromInt(9443),
			}},
		},
	}
	return service
}
