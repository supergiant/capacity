package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!
// Created by "kubebuilder create resource" for you to implement the ClusterCapacity resource schema definition
// as a go struct.
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterCapacitySpec defines the desired state of ClusterCapacity
type ClusterCapacitySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "kubebuilder generate" to regenerate code after modifying this file
}

// ClusterCapacityStatus defines the observed state of ClusterCapacity
type ClusterCapacityStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "kubebuilder generate" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterCapacity
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=clustercapacities
type ClusterCapacity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterCapacitySpec   `json:"spec,omitempty"`
	Status ClusterCapacityStatus `json:"status,omitempty"`
}
