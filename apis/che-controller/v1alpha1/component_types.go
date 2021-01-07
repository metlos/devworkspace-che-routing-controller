//
// Copyright (c) 2019-2020 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RoutingType string

const (
	SingleHost RoutingType = "singlehost"
	MultiHost  RoutingType = "multihost"
)

// CheManagerSpec holds the configuration of the Che controller.
// +k8s:openapi-gen=true
type CheManagerSpec struct {
	// The hostname to use for creating the workspace endpoints
	// This is used as a full hostname in the singlehost mode. In the multihost mode, the individual
	// endpoints are exposed on subdomains of the specified host.
	Host string `json:"host,omitempty"`

	// Routing defines how the Che Router exposes the workspaces and components within
	Routing RoutingType `json:"routing,omitempty"`

	// GatewayImage is the docker image to use for the Che gateway.  This is only used in
	// the singlehost mode. If not defined in the CR, it is taken from
	// the `RELATED_IMAGE_gateway` environment variable of the che operator
	// deployment/pod. If not defined there it defaults to a hardcoded value.
	GatewayImage string `json:"gatewayImage,omitempty"`

	// GatewayConfigureImage is the docker image to use for the sidecar of the Che gateway that is
	// used to configure it. This is only used in the singlehost mode. If not defined in the CR,
	// it is taken from the `RELATED_IMAGE_gateway_configurer` environment variable of the che
	// operator deployment/pod. If not defined there it defaults to a hardcoded value.
	GatewayConfigurerImage string `json:"gatewayConfigurerImage,omitempty"`
}

type GatewayPhase string

const (
	GatewayPhaseInitializing = "Initializing"
	GatewayPhaseEstablished  = "Established"
	GatewayPhaseInactive     = "Inactive"
)

// +k8s:openapi-gen=true
type CheManagerStatus struct {
	GatewayPhase GatewayPhase `json:"gatewayPhase,omitempty"`
}

// CheManager is the configuration of the CheManager layer of Devworkspace.
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=chemanagers,scope=Namespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CheManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CheManagerSpec   `json:"spec,omitempty"`
	Status CheManagerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CheManagerList is the list type for Che
type CheManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CheManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CheManager{}, &CheManagerList{})
}
