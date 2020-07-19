package util

import mightydevco "github.com/uberscott/recert/go/src/operator/pkg/apis/mightydevco/v1alpha1"

// AgentName the name of the Agent
func AgentName(instance *mightydevco.Cert) string {
	return instance.Name + "-agent"
}

// SecretNameFromCert the name of the SLL secret
func SecretNameFromCert(instance *mightydevco.Cert) string {
	return instance.Spec.SslProxy + "-nginx-sslproxy"
}

// NewSecretNameFromCert this is the temporary name of the ssl secret before it gets
// copied back into Secret
func NewSecretNameFromCert(instance *mightydevco.Cert) string {
	return instance.Spec.SslProxy + "-nginx-sslproxy-new"
}

// SslProxyDeploymentNameFromCert is the name of the SSL Nginx Proxy Deployment
func SslProxyDeploymentNameFromCert(instance *mightydevco.Cert) string {
	return instance.Spec.SslProxy + "-nginx-sslproxy"
}
