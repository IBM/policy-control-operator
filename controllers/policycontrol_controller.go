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
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kcptoolsv1alpha1 "github.com/IBM/policy-control-operator/api/v1alpha1"
)

// PolicyControlReconciler reconciles a PolicyControl object
type PolicyControlReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var WORKSPACE_KYVERNO_INSTALL_MANIFESTS_DIR string = os.Getenv("WORKSPACE_KYVERNO_INSTALL_MANIFESTS_DIR")
var WORKSPACE_APIBINDINGS_MANIFEST string = os.Getenv("WORKSPACE_APIBINDINGS_MANIFEST")

// TODO: Remove syncer dependency once required CRDs can get installed through APIResourceSchemas not syncer
var SYNCER_IMAGE string = getEnv("SYNCER_IMAGE", "ghcr.io/kcp-dev/kcp/syncer:554c247")

//+kubebuilder:rbac:groups=ibm.github.com,resources=policycontrols,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ibm.github.com,resources=policycontrols/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ibm.github.com,resources=policycontrols/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles;rolebindings;clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// TODO: As of now, need following RBACs for policy-control-operator to install syncer in the PCO cluster.
//       Once we find a different approach to import CRDs instead of syncing, we can remove the following RBACs.
//+kubebuilder:rbac:groups="",resources=configmaps;secrets;serviceaccounts;services,verbs="*"
//+kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs="*"
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs="*"
//+kubebuilder:rbac:groups="apiextensions.k8s.io",resources=customresourcedefinitions,verbs=get;watch;list
//+kubebuilder:rbac:groups="",resources=kyvernoes;policies,verbs="*"
//+kubebuilder:rbac:groups="kyverno.io",resources=policies,verbs="*"
//+kubebuilder:rbac:groups="operator.kyverno.io",resources=kyvernoes,verbs="*"
//+kubebuilder:rbac:groups="operators.coreos.com",resources=catalogsources;operatorgroups;subscriptions,verbs="*"

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PolicyControl object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *PolicyControlReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(1).Info(fmt.Sprintf("namespace=%s, name=%s", req.NamespacedName.Namespace, req.NamespacedName.Name))

	var pc kcptoolsv1alpha1.PolicyControl
	err := r.Get(ctx, req.NamespacedName, &pc)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}

	// TODO: Until swithing to use KCP Go library or REST API, we use kcp command with kubeconfig file specified.
	//		 Once switched, remove this part.
	kcpKubeConfigSecret := pc.Spec.PolicyControlCluster.KcpKubeConfigSecret
	var kcpSecret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: pc.Spec.PolicyControlCluster.Namespace, Name: kcpKubeConfigSecret.Name}, &kcpSecret); err != nil {
		return ctrl.Result{}, err
	}
	file, err := os.CreateTemp("", "*")
	if err != nil {
		return ctrl.Result{}, err
	}
	if _, err := file.Write(kcpSecret.Data[kcpKubeConfigSecret.Key]); err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Error(err, "failed to close "+file.Name())
		}
		if err := os.Remove(file.Name()); err != nil {
			logger.Error(err, "failed to remove "+file.Name())
		}
	}()

	/*
		Sync pcc

		export SOURCE=$(pwd)/scripts/const.sh
		pushd ../kyverno-at-edge
		./scripts/sync-pcc.sh $SYNCER_IMAGE cluster1
		popd
	*/

	if _, err := r.syncPCO(ctx, req, logger, pc, file.Name(), req.NamespacedName.Namespace); err != nil {
		return ctrl.Result{}, err
	}

	/*
			Install Kyverno on Edge side through KCP

			export SOURCE=$(pwd)/scripts/const.sh
		    pushd ../kyverno-at-edge
		    ./scripts/install-kyverno-on-edge.sh $WORKSPACE
		    popd
	*/

	if _, err := r.installKyvernoOnEdge(ctx, req, logger, pc, file.Name()); err != nil {
		return ctrl.Result{}, err
	}

	/*
			Enable workspace policy governance

			export SOURCE=$(pwd)/scripts/const.sh
		    pushd ../kyverno-at-workspace
		    ./scripts/enable-workspace-level-kyverno.sh $WORKSPACE
		    popd
	*/

	if _, err := r.installKyvernoOnWorkspace(ctx, req, logger, pc, file.Name()); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PolicyControlReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kcptoolsv1alpha1.PolicyControl{}).
		Complete(r)
}
