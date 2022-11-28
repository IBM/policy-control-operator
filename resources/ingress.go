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

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/IBM/policy-control-operator/api/v1alpha1"
)

func BuildIngressForKyverno(cr *v1alpha1.PolicyControl) *networkingv1.Ingress {

	ingressPath := buildIngressHTTPIngressPath(cr, getPath(cr))
	ingresClass := "nginx"
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kyverno-ingress",
			Namespace: cr.Spec.PolicyControlCluster.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/backend-protocol": "HTTPS",
				"nginx.ingress.kubernetes.io/rewrite-target":   "/$2",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingresClass,
			TLS: []networkingv1.IngressTLS{{
				Hosts:      []string{cr.Spec.PolicyControlCluster.IngressHost},
				SecretName: "kyverno-ingress",
			}},
			Rules: []networkingv1.IngressRule{{
				Host: cr.Spec.PolicyControlCluster.IngressHost,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{*ingressPath},
					},
				},
			}},
		},
	}
	return ingress
}

func AddIngressRuleForKyverno(cr *v1alpha1.PolicyControl, ingress *networkingv1.Ingress) (*networkingv1.Ingress, error) {

	logger := log.WithValues("AddIngressRuleForKyverno", cr.GetName())

	path := getPath(cr)

	// find "host" entry matching with cr.Spec.PolicyControlCluster.Host from the existing rules
	var idxHost int = -1
	var idxPath int = -1
	for i, r := range ingress.Spec.Rules {
		if r.Host == cr.Spec.PolicyControlCluster.IngressHost {
			idxHost = i
			for j, p := range r.HTTP.Paths {
				if p.Path == path {
					idxPath = j
				}
			}
		}
	}
	if idxHost == -1 { // TODO: support to add Host entry instead of returning an error
		err := fmt.Errorf("ingress doesn't contain a route for %s", cr.Spec.PolicyControlCluster.IngressHost)
		logger.Error(err, "")
		return nil, err
	}

	// find "path" entry matching with path from the existing host entry
	for i, p := range ingress.Spec.Rules[idxHost].HTTP.Paths {
		if p.Path == path {
			idxPath = i
		}
	}

	if idxPath != -1 {
		logger.Info(fmt.Sprintf("Path (%s) already exists in ingress (%s)", path, ingress.GetName()))
		return ingress, nil
	}

	ingressPath := buildIngressHTTPIngressPath(cr, path)
	ingress.Spec.Rules[idxHost].HTTP.Paths = append(ingress.Spec.Rules[idxHost].HTTP.Paths, *ingressPath)

	return ingress, nil
}

func buildIngressHTTPIngressPath(cr *v1alpha1.PolicyControl, path string) *networkingv1.HTTPIngressPath {
	normalizedWorkspace := normalizeWorkdpaceName(cr)
	pathPrefix := networkingv1.PathTypePrefix
	ingressPath := &networkingv1.HTTPIngressPath{
		Path:     path,
		PathType: &pathPrefix,
		Backend: networkingv1.IngressBackend{
			Service: &networkingv1.IngressServiceBackend{
				Name: normalizedWorkspace,
				Port: networkingv1.ServiceBackendPort{
					Number: cr.Spec.PolicyControlCluster.IngressPort,
				},
			},
		},
	}
	return ingressPath
}

func getPath(cr *v1alpha1.PolicyControl) string {
	return fmt.Sprintf("/%s(/|$)(.*)", normalizeWorkdpaceName(cr))
}
