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
	"fmt"
	"os"
	"os/exec"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

func getEnv(name, defaultValue string) string {
	value, exist := os.LookupEnv(name)
	if exist {
		return value
	}
	return defaultValue
}

func getInClusterConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}

func getOutOfClusterConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	return getClusterConfigFromFile(kubeconfigPath)
}

func getClusterConfigFromFile(kubeconfigPath string) (*rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// KUBECONFIG=$KUBECONFIG_KCP_ADMIN kubectl kcp ws use $workspace
func switchWorkspace(kcpKubeConfig string, workspace string, logger logr.Logger) {
	command := fmt.Sprintf("KUBECONFIG=%s kubectl kcp ws use %s", kcpKubeConfig, workspace)
	logger.V(4).Info(command)
	result, err := exec.Command("/bin/sh", "-c", command).Output()
	if err != nil {
		logger.Error(err, "Failed to switch workspace")

	}
	logger.V(4).Info(string(result))
}

// KUBECONFIG=$KUBECONFIG_KCP_ADMIN kubectl kcp workload sync $PG_CLUSTER --syncer-image $syncer_image -o $kustomize_dir/syncer.yaml --resources=kyvernoes,policies
func syncWorkspace(kcpKubeConfig string, workspace string, targetCluster string, syncerImage string, logger logr.Logger) (string, error) {
	switchWorkspace(kcpKubeConfig, workspace, logger)
	command := fmt.Sprintf("KUBECONFIG=%s kubectl kcp workload sync %s --syncer-image %s -o - --resources=kyvernoes,policies", kcpKubeConfig, targetCluster, syncerImage)
	logger.V(4).Info(command)
	result, err := exec.Command("/bin/sh", "-c", command).Output()
	return string(result), err
}

func getWorkspaceKubeConfig(kcpKubeConfig string, workspace string, logger logr.Logger) (string, error) {
	switchWorkspace(kcpKubeConfig, workspace, logger)
	command := fmt.Sprintf("KUBECONFIG=%s kubectl config view --minify --raw", kcpKubeConfig)
	logger.V(4).Info(command)
	result, err := exec.Command("/bin/sh", "-c", command).Output()
	return string(result), err
}

func getWorkspaceConfigs(
	kcpKubeConfig string,
	workspace string,
	logger logr.Logger,
) (*rest.Config, meta.RESTMapper, error) {
	switchWorkspace(kcpKubeConfig, workspace, logger)
	config, err := getClusterConfigFromFile(kcpKubeConfig)
	if err != nil {
		return nil, nil, err
	}

	c := discovery.NewDiscoveryClientForConfigOrDie(config)
	groupResources, err := restmapper.GetAPIGroupResources(c)
	if err != nil {
		return nil, nil, err
	}

	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	return config, mapper, nil
}

func getUnstructuredFromFile(
	logger logr.Logger,
	path string,
	restMapper meta.RESTMapper,
) (unstructured.Unstructured, *meta.RESTMapping, error) {
	var obj unstructured.Unstructured
	objBytes, err := os.ReadFile(path)
	if err != nil {
		return obj, nil, err
	}
	if err := yaml.Unmarshal(objBytes, &obj); err != nil {
		return obj, nil, err
	}
	mapping, err := getMapping(logger, obj, restMapper)
	return obj, mapping, err
}

func getMapping(
	logger logr.Logger,
	obj unstructured.Unstructured,
	restMapper meta.RESTMapper,
) (*meta.RESTMapping, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := restMapper.RESTMapping(gk)
	if err != nil {
		logger.Error(err, "Failed to map gk to resource")
	}
	return mapping, err
}
