package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CertSpec defines the desired state of Cert
type CertSpec struct {
	Domain   string `json:"domain"`
	Email    string `json:"email"`
	SslProxy string `json:"sslProxy"`
}

// CertStatus defines the observed state of Cert
type CertStatus struct {
	State           string `json:"state,omitempty"`
	LastUpdated     string `json:"lastUpdated,omitempty"`
	LastStateChange int64  `json:"lastStateChange,omitempty"`
}

// CertState is the state of the Cert
type CertState string

const (

	// Pending when we are waiting  for a chance to do something
	Pending = "Pending"

	// FailureBackoff after a failure there is a backoff period
	FailureBackoff = "FailureBackoff"

	// Creating when the Cert is first being created by certbot
	Creating = "Creating"

	// Updated when the Cert has been updated by certbot
	Updated = "Updated"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cert is the Schema for the certs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=certs,scope=Namespaced
type Cert struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertSpec   `json:"spec,omitempty"`
	Status CertStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CertList contains a list of Cert
type CertList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cert `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cert{}, &CertList{})
}
