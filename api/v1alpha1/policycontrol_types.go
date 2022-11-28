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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PolicyControlSpec defines the desired state of PolicyControl
type PolicyControlSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Workspace            string               `json:"workspace,omitempty"`
	PolicyControlCluster PolicyControlCluster `json:"policy_control_cluster,omitempty"`
	KyvernoInWorkspace   KyvernoInWorkspace   `json:"kyverno_in_workspace,omitempty"`
	KyvernoInCluster     KyvernoInCluster     `json:"kyverno_in_cluster,omitempty"`
}

type PolicyControlCluster struct {
	// Namespace in Policy Control Cluster to which Kcp Kubeconfig secret and Ingress TLS secret are placed and ingress resource, Kyverno deployments and service will be deployed.
	Namespace           string              `json:"namespace,omitempty"`
	IngressName         string              `json:"ingressName,omitempty"`
	IngressHost         string              `json:"ingressHost,omitempty"`
	IngressPort         int32               `json:"ingressPort,omitempty"`
	IngressTLSSecret    TLSSecret           `json:"ingressTLSSecret,omitempty"`
	KcpKubeConfigSecret KcpKubeConfigSecret `json:"kcpKubeConfigSecret,omitempty"`
}

type KcpKubeConfigSecret struct {
	Name string `json:"name,omitempty"`
	Key  string `json:"key,omitempty"`
}
type TLSSecret struct {
	Name          string `json:"name,omitempty"`
	KeyForCert    string `json:"keyForCert,omitempty"`
	KeyForPrivKey string `json:"keyForPrivKey,omitempty"`
	KeyForCacert  string `json:"keyForCacert,omitempty"`
}

type KyvernoInWorkspace struct {
	// Namespace in the target workspace where resources (e.g. cert) needed for Kyverno to start up will be placed.
	NamespaceForAPIResources string `json:"namespaceForAPIResources,omitempty"`
	KyvernoImage             string `json:"kyvernoImage,omitempty"`
}

type KyvernoInCluster struct {
	InstallNamespace string        `json:"installNamespace,omitempty"`
	OperatorGroup    OperatorGroup `json:"operatorGroup,omitempty"`
	Subscription     Subscription  `json:"subscription,omitempty"`
	KyvernoCR        KyvernoCR     `json:"kyvernoCR,omitempty"`
}

type OperatorGroup struct {
	Name string `json:"name,omitempty"`
}

type Subscription struct {
	Name         string `json:"name,omitempty"`
	OLMNamespace string `json:"olmNamespace,omitempty"`
}

type KyvernoCR struct {
	Name string `json:"name,omitempty"`
}

// PolicyControlStatus defines the observed state of PolicyControl
type PolicyControlStatus struct {
	// Represents the observations of a PolicyController's current state.
	// PolicyController.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// PolicyController.status.conditions.status are one of True, False, Unknown.
	// PolicyController.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// PolicyController.status.conditions.Message is a human readable message indicating details about the transition.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PolicyControl is the Schema for the policycontrols API
type PolicyControl struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicyControlSpec   `json:"spec,omitempty"`
	Status PolicyControlStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PolicyControlList contains a list of PolicyControl
type PolicyControlList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PolicyControl `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PolicyControl{}, &PolicyControlList{})
}
