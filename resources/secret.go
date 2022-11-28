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

	"github.com/IBM/policy-control-operator/api/v1alpha1"
)

func BuildSecretForKyverno(cr *v1alpha1.PolicyControl, kcpKubeConfig string) *corev1.Secret {
	normalizedWorkspace := normalizeWorkdpaceName(cr)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      normalizedWorkspace,
			Namespace: cr.Spec.PolicyControlCluster.Namespace,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"target-kubeconfig.yaml": kcpKubeConfig},
	}
	return secret
}

func BuildTLSCASecretForKyverno(cr *v1alpha1.PolicyControl, tlsCACrt string) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kyverno-svc-remote.kyverno.svc.kyverno-tls-ca",
			Namespace: cr.Spec.KyvernoInWorkspace.NamespaceForAPIResources,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"rootCA.crt": tlsCACrt},
	}
	return secret
}

func BuildTLSKeyCertSecretForKyverno(cr *v1alpha1.PolicyControl, tlsKey string, tlsCrt string) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kyverno-svc-remote.kyverno.svc.kyverno-tls-pair",
			Namespace: cr.Spec.KyvernoInWorkspace.NamespaceForAPIResources,
		},
		Type:       corev1.SecretTypeTLS,
		StringData: map[string]string{"tls.key": tlsKey, "tls.crt": tlsCrt},
	}
	return secret
}

func BuildTLSKeyCertSecretForIngress(cr *v1alpha1.PolicyControl, tlsKey string, tlsCrt string) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kyverno-ingress",
			Namespace: cr.Spec.PolicyControlCluster.Namespace,
		},
		Type:       corev1.SecretTypeTLS,
		StringData: map[string]string{"tls.key": tlsKey, "tls.crt": tlsCrt},
	}
	return secret
}
