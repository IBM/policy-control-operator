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
	"path/filepath"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kcptoolsv1alpha1 "github.com/IBM/policy-control-operator/api/v1alpha1"
	"github.com/IBM/policy-control-operator/resources"
)

func (r *PolicyControlReconciler) installKyvernoOnWorkspace(
	ctx context.Context,
	req ctrl.Request,
	logger logr.Logger,
	pc kcptoolsv1alpha1.PolicyControl,
	kcpKubeConfig string,
) (ctrl.Result, error) {

	config, mapper, err := getWorkspaceConfigs(kcpKubeConfig, pc.Spec.Workspace, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	// create namespace in the workspace to be installed workspace kyverno
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	namespace := pc.Spec.KyvernoInWorkspace.NamespaceForAPIResources
	nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	_, err = clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		_, err = clientset.CoreV1().Namespaces().Create(ctx, nsSpec, metav1.CreateOptions{})
		if err != nil {
			logger.Error(err, "failed to create Resource")
			return ctrl.Result{}, err
		}
	}

	logger.V(4).Info("install Kyverno related manifests")
	dyClient, _ := dynamic.NewForConfig(config)
	files, _ := filepath.Glob(fmt.Sprintf("%s/*.yaml", WORKSPACE_KYVERNO_INSTALL_MANIFESTS_DIR))
	for _, f := range files {
		if err := r.createOrUpdateUnstructuredResourceFromFile(ctx, logger, mapper, dyClient, f, true); err != nil {
			logger.Error(err, fmt.Sprintf("failed to create manifest %s", f))
			return ctrl.Result{}, err
		}
	}

	logger.V(4).Info("import k8s basic resource definitions (Pod and Daemonset fow now) so that a standalone Kyverno can run")
	if err := r.createOrUpdateUnstructuredResourceFromFile(ctx, logger, mapper, dyClient, WORKSPACE_APIBINDINGS_MANIFEST, true); err != nil {
		logger.Error(err, fmt.Sprintf("failed to create manifest %s", WORKSPACE_APIBINDINGS_MANIFEST))
		return ctrl.Result{}, err
	}

	logger.V(4).Info("create secret for PCO cluster's TLS Key and cert that will be loaded by a standalone Kyverno")
	crTlsSecret := pc.Spec.PolicyControlCluster.IngressTLSSecret
	var tlsSecret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: pc.Spec.PolicyControlCluster.Namespace, Name: crTlsSecret.Name}, &tlsSecret); err != nil {
		return ctrl.Result{}, err
	}
	tlsKey := string(tlsSecret.Data[crTlsSecret.KeyForPrivKey])
	tlsCert := string(tlsSecret.Data[crTlsSecret.KeyForCert])
	tlsCACrt := string(tlsSecret.Data[crTlsSecret.KeyForCacert])

	tlsKeyCertSecret := resources.BuildTLSKeyCertSecretForKyverno(&pc, tlsKey, tlsCert)
	_, err = clientset.CoreV1().Secrets(tlsKeyCertSecret.GetNamespace()).Get(ctx, tlsKeyCertSecret.GetName(), metav1.GetOptions{})
	if err != nil {
		_, err = clientset.CoreV1().Secrets(tlsKeyCertSecret.GetNamespace()).Create(ctx, tlsKeyCertSecret, metav1.CreateOptions{})
		if err != nil {
			logger.Error(err, "failed to create Resource")
			return ctrl.Result{}, err
		}
	}

	logger.V(4).Info("create secret for PCO cluster's CA cert that will be set in webhook configurations by a standalone Kyverno")
	tlsCaSecret := resources.BuildTLSCASecretForKyverno(&pc, tlsCACrt)
	_, err = clientset.CoreV1().Secrets(tlsCaSecret.GetNamespace()).Get(ctx, tlsCaSecret.GetName(), metav1.GetOptions{})
	if err != nil {
		_, err = clientset.CoreV1().Secrets(tlsCaSecret.GetNamespace()).Create(ctx, tlsCaSecret, metav1.CreateOptions{})
		if err != nil {
			logger.Error(err, "failed to create Resource")
			return ctrl.Result{}, err
		}
	}

	logger.V(4).Info("create Ingress TLS Key Cert pair secret")
	ingressSecret := &corev1.Secret{}
	err = r.Get(ctx, client.ObjectKey{Namespace: pc.Spec.PolicyControlCluster.Namespace, Name: "kyverno-ingress"}, ingressSecret)
	if err != nil {
		ingressSecret = resources.BuildTLSKeyCertSecretForIngress(&pc, tlsKey, tlsCert)
		if err := r.Create(ctx, ingressSecret); err != nil {
			logger.Error(err, fmt.Sprintf("failed to create ingress %s", ingressSecret.GetName()))
			return ctrl.Result{}, err
		}
	}

	logger.V(4).Info("create ingress or add route to an existing ingress")
	ingress := &networkingv1.Ingress{}
	err = r.Get(ctx, client.ObjectKey{Namespace: pc.Spec.PolicyControlCluster.Namespace, Name: "kyverno-ingress"}, ingress)
	if err != nil {
		ingress = resources.BuildIngressForKyverno(&pc)
		if err := r.Create(ctx, ingress); err != nil {
			logger.Error(err, fmt.Sprintf("failed to create ingress %s", ingress.GetName()))
			return ctrl.Result{}, err
		}
	} else {
		ingress, _ = resources.AddIngressRuleForKyverno(&pc, ingress)
		if err := r.Update(ctx, ingress); err != nil {
			logger.Error(err, fmt.Sprintf("failed to add ingress rule %s", ingress.GetName()))
			return ctrl.Result{}, err
		}
	}

	// KUBECONFIG=$KUBECONFIG_PG_CLUSTER kubectl -n $PG_NAMESPACE  create secret generic kyverno-runtime-credentials-$norm_workspace --from-file=target-kubeconfig.yaml=$kcp_ws_kubeconfig
	logger.V(4).Info("create secret for the target workspace kubeconfig that's consumed by a standalone Kyverno")
	kubeConfig, err := getWorkspaceKubeConfig(kcpKubeConfig, pc.Spec.Workspace, logger)
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to generate workspace (%s) kubeconfig", pc.Spec.Workspace))
		return ctrl.Result{}, err
	}
	secret := resources.BuildSecretForKyverno(&pc, kubeConfig)
	if err := r.createOrUpdate(ctx, logger, secret); err != nil {
		logger.Error(err, fmt.Sprintf("failed to create secrets for target workspace kubeconfig %s", secret.GetName()))
		return ctrl.Result{}, err
	}

	// WORKSPACE=$norm_workspace envsubst < ./manifests/policy-control-cluster/kyverno-controller/service-template.yaml | KUBECONFIG=$KUBECONFIG_PG_CLUSTER kubectl -n $PG_NAMESPACE apply -f -
	logger.V(4).Info("create service for standalone Kyverno")
	service := resources.BuildServiceForKyverno(&pc)
	if err := r.createOrUpdate(ctx, logger, service); err != nil {
		logger.Error(err, fmt.Sprintf("failed to create service for for standalone Kyverno %s", service.GetName()))
		return ctrl.Result{}, err
	}

	// WORKSPACE=$norm_workspace envsubst < ./manifests/policy-control-cluster/kyverno-controller/deployment-template.yaml | KUBECONFIG=$KUBECONFIG_PG_CLUSTER kubectl -n $PG_NAMESPACE apply -f -
	logger.V(4).Info("create deployment for standalone Kyverno")
	deployment := resources.BuildDeploymentForKyverno(&pc)
	if err := r.createOrUpdate(ctx, logger, deployment); err != nil {
		logger.Error(err, fmt.Sprintf("failed to create deployment for for standalone Kyverno %s", deployment.GetName()))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
