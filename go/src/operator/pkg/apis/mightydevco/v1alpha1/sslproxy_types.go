package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SSLProxySpec defines the desired state of SSLProxy
type SSLProxySpec struct {
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`
	//Selector       map[string]string `json:"selector" protobuf:"bytes,2,rep,name=selector"`
	ReverseProxy string `json:"reverseProxy,omitempty"`
	Replicas     *int32 `json:"replicas,omitempty"`
}

// SSLProxyStatus defines the observed state of SSLProxy
type SSLProxyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SSLProxy is the Schema for the sslproxies API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=sslproxies,scope=Namespaced
type SSLProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SSLProxySpec   `json:"spec,omitempty"`
	Status SSLProxyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SSLProxyList contains a list of SSLProxy
type SSLProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SSLProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SSLProxy{}, &SSLProxyList{})
}
