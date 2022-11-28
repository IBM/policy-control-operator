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

	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/typed/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/typed/operators/v1alpha1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	kcptoolsv1alpha1 "github.com/IBM/policy-control-operator/api/v1alpha1"
	"github.com/IBM/policy-control-operator/resources"
)

func (r *PolicyControlReconciler) installKyvernoOnEdge(
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

	// create namespace in the workspace to be installed in-cluster kyverno
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	namespace := pc.Spec.KyvernoInCluster.InstallNamespace
	nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	_, err = clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		_, err = clientset.CoreV1().Namespaces().Create(ctx, nsSpec, metav1.CreateOptions{})
		if err != nil {
			logger.Error(err, "failed to create Resource")
			return ctrl.Result{}, err
		}
	}

	// create OperatorGroup
	operatorGroupObj := resources.BuildOperatorGroupForKyverno(&pc)
	operatorClientset, err := operatorsv1.NewForConfig(config)
	if err != nil {
		logger.Error(err, "failed to create k8s client for OperatorGroup")
		return ctrl.Result{}, err
	}
	_, err = operatorClientset.OperatorGroups(operatorGroupObj.GetNamespace()).Get(ctx, operatorGroupObj.GetName(), metav1.GetOptions{})
	if err != nil {
		if _, err := operatorClientset.OperatorGroups(operatorGroupObj.GetNamespace()).Create(ctx, operatorGroupObj, metav1.CreateOptions{}); err != nil {
			logger.Error(err, "failed to create OperatorGroup")
			return ctrl.Result{}, err
		}
	}

	// create Subscription
	subscriptionObj := resources.BuildSubscriptionForKyverno(&pc)
	subscriptionClientset, err := operatorsv1alpha1.NewForConfig(config)
	if err != nil {
		logger.Error(err, "failed to create k8s client for Subscription")
		return ctrl.Result{}, err
	}
	_, err = subscriptionClientset.Subscriptions(subscriptionObj.GetNamespace()).Get(ctx, subscriptionObj.GetName(), metav1.GetOptions{})
	if err != nil {
		if _, err := subscriptionClientset.Subscriptions(subscriptionObj.GetNamespace()).Create(ctx, subscriptionObj, metav1.CreateOptions{}); err != nil {
			logger.Error(err, "failed to create Subscription")
			return ctrl.Result{}, err
		}
	}

	// create KyvernoCR
	dyClient, _ := dynamic.NewForConfig(config)
	kyvernoCRObj := resources.BuildKyvernoCR(&pc)
	mapping, err := getMapping(logger, *kyvernoCRObj, mapper)
	if err != nil {
		logger.Error(err, "failed to map KyvernoCR (Kyverno) to registered Kinds")
		return ctrl.Result{}, err
	}
	_, err = r.createOrUpdateUnstructuredResource(ctx, logger, dyClient, *mapping, *kyvernoCRObj, true)
	if err != nil {
		logger.Error(err, "failed to create KyvernoCR")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
