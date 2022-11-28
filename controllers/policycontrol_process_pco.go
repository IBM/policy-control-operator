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

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"

	kcptoolsv1alpha1 "github.com/IBM/policy-control-operator/api/v1alpha1"
)

func (r *PolicyControlReconciler) syncPCO(
	ctx context.Context,
	req ctrl.Request,
	logger logr.Logger,
	pc kcptoolsv1alpha1.PolicyControl,
	kcpKubeConfig string,
	namespace string,
) (ctrl.Result, error) {

	syncerManfests, err := syncWorkspace(kcpKubeConfig, pc.Spec.Workspace, pc.Spec.PolicyControlCluster.IngressName, SYNCER_IMAGE, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	config, err := getInClusterConfig()
	if err != nil || config == nil {
		config, _ = getOutOfClusterConfig()
	}
	c := discovery.NewDiscoveryClientForConfigOrDie(config)
	groupResources, _ := restmapper.GetAPIGroupResources(c)
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	dyClient, _ := dynamic.NewForConfig(config)
	for _, str := range strings.Split(syncerManfests, "---") {
		var obj unstructured.Unstructured
		if err := yaml.Unmarshal([]byte(str), &obj); err != nil {
			logger.Error(err, "invalid yaml")
			continue
		}
		if _, err := r.createOrUpdateResource(ctx, req, logger, pc, dyClient, mapper, obj); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *PolicyControlReconciler) createOrUpdateResource(
	ctx context.Context,
	req ctrl.Request,
	logger logr.Logger,
	pc kcptoolsv1alpha1.PolicyControl,
	dyClient dynamic.Interface,
	restMapper meta.RESTMapper,
	obj unstructured.Unstructured,
) (ctrl.Result, error) {

	gvk := obj.GetObjectKind().GroupVersionKind()
	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := restMapper.RESTMapping(gk)

	if err != nil {
		logger.Error(err, "Failed to map gk to resource")
		return ctrl.Result{}, err
	}

	if mapping.Scope == meta.RESTScopeNamespace && obj.GetNamespace() == "" {
		obj.SetNamespace(req.Namespace)
	}

	switch mapping.Resource.Resource {
	case corev1.ResourceConfigMaps.String():
		return r.createOrUpdateTypedResourceByUnstructured(ctx, logger, pc, obj, &corev1.ConfigMap{}, false)
	case corev1.ResourceSecrets.String():
		return r.createOrUpdateTypedResourceByUnstructured(ctx, logger, pc, obj, &corev1.Secret{}, false)
	case corev1.ResourceServices.String():
		return r.createOrUpdateTypedResourceByUnstructured(ctx, logger, pc, obj, &corev1.Service{}, false)
	case "clusterroles":
		if strings.Contains(obj.GetName(), "kcp-syncer-") {
			clusterRole := &rbacv1.ClusterRole{}
			gvk := clusterRole.GetObjectKind().GroupVersionKind().String()
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), clusterRole)
			if err != nil {
				logger.Error(err, fmt.Sprintf("failed to convert resource to %s", gvk))
				return ctrl.Result{}, err
			}
			rules := []rbacv1.PolicyRule{
				{
					Verbs:     []string{"*"},
					Resources: []string{"policies"},
					APIGroups: []string{"kyverno.io"},
				},
				{
					Verbs:     []string{"*"},
					Resources: []string{"kyvernoes"},
					APIGroups: []string{"operator.kyverno.io"},
				},
			}
			clusterRole.Rules = append(clusterRole.Rules, rules...)
			return r.createOrUpdateTypedResource(ctx, logger, pc, clusterRole, false)
		}
		return r.createOrUpdateTypedResourceByUnstructured(ctx, logger, pc, obj, &rbacv1.ClusterRole{}, false)
	case "serviceaccounts":
		return r.createOrUpdateTypedResourceByUnstructured(ctx, logger, pc, obj, &corev1.ServiceAccount{}, false)
	case "customresourcedefinitions":
		return r.createOrUpdateTypedResourceByUnstructured(ctx, logger, pc, obj, &apiextensions.CustomResourceDefinition{}, false)
	default:
		return r.createOrUpdateUnstructuredResource(ctx, logger, dyClient, *mapping, obj, false)
	}
}
