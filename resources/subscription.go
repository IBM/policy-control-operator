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
	"github.com/IBM/policy-control-operator/api/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BuildSubscriptionForKyverno(cr *v1alpha1.PolicyControl) *operatorsv1alpha1.Subscription {
	crSubscription := cr.Spec.KyvernoInCluster.Subscription
	obj := &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Spec.KyvernoInCluster.Subscription.Name,
			Namespace: cr.Spec.KyvernoInCluster.InstallNamespace,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			Package:                "kyverno-operator",
			InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
			Channel:                "alpha",
			CatalogSource:          "kyverno-operator",
			CatalogSourceNamespace: crSubscription.OLMNamespace,
		},
	}
	return obj
}
