/*
Copyright 2024.

Licensed under the Apache License;Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing;software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND;either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentSpec defines the desired state of Agent
type AgentSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=14
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Symbol string `json:"callsign,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=COSMIC;VOID;GALACTIC;QUANTUM;DOMINION;ASTRO;CORSAIRS;OBSIDIAN;AEGIS;UNITED;SOLITARY;COBALT;OMEGA;ECHO;LORDS;CULT;ANCIENTS;SHADOW;ETHEREAL
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Faction string `json:"faction,omitempty"`
}

// AgentStatus defines the observed state of Agent
type AgentStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status
	AccountID string `json:"accountId,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Headquarters string `json:"headquarters,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Credits int32 `json:"credits,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=status
	StartingFaction string `json:"startingFaction,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=status
	ShipCount int32 `json:"shipCount,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Agent is the Schema for the agents API
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentSpec   `json:"spec,omitempty"`
	Status AgentStatus `json:"status,omitempty"`
}

func (a *Agent) AccessTokenSecretName() string {
	return a.Name + "-access-token"
}

//+kubebuilder:object:root=true

// AgentList contains a list of Agent
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Agent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Agent{}, &AgentList{})
}
