package testoperator

import (
	"bytes"
	"fmt"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"html/template"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"
	"testing"
	"time"
)

func getLoadBalancerIP() (string, error) {
	rtn, found := os.LookupEnv("LOAD_BALANCER_IP")

	if !found {
		return "", fmt.Errorf("LOAD_BALANCER_IP must be set in order to perform FULL_TEST")
	}
	return rtn, nil
}

func getDomain() (string, error) {
	rtn, found := os.LookupEnv("DOMAIN")

	if !found {
		return "", fmt.Errorf("DOMAIN must be set in order to perform FULL_TEST")
	}
	return rtn, nil
}

func isFullTest() bool {
	rtn, found := os.LookupEnv("FULL_TEST")
	if !found {
		return false
	}
	return strings.TrimSpace(rtn) == "true"
}

func getNameSpace() string {

	rtn, found := os.LookupEnv("NAMESPACE")
	if !found {
		rtn = "default"
	}
	return rtn
}

func TestSslProxy(t *testing.T) {
	t.Parallel()

	// Path to the Kubernetes resource config we will test.
	sslProxyTemplate := `apiVersion: mightydevco.com/v1alpha1
kind: SSLProxy
metadata:
  name: example
spec:
  {{ if .isFullTest }}
  loadBalancerIP: {{ .LoadBalancerIP }}
  {{ end }} 
  reverseProxy: http://example:80
  replicas: 1`

	certTemplate := `apiVersion: mightydevco.com/v1alpha1
kind: Cert
metadata:
  name: example
spec:
  domain: {{ .Domain }}
  email: scott@mightydevco.com
  sslProxy: example`

	conf := yamlConf{
		IsFullTest: isFullTest(),
	}

	if isFullTest() {
		conf.LoadBalancerIP, _ = getLoadBalancerIP()
		conf.Domain, _ = getDomain()
	}

	var sslProxy string
	{
		tmpl, _ := template.New("conf").Parse(sslProxyTemplate)
		var w bytes.Buffer
		tmpl.Execute(&w, conf)
		sslProxy = string(w.Bytes())
		print(sslProxy)
	}

	// Setup the kubectl config and context.
	options := k8s.NewKubectlOptions("", "", getNameSpace())

	// At the end of the test, run "kubectl delete" to clean up any resources that were created.
	defer k8s.KubectlDeleteFromString(t, options, sslProxy)

	// Run `kubectl apply` to deploy. Fail the test if there are any errors.
	k8s.KubectlApplyFromString(t, options, sslProxy)

	// Verify the service is available and get the URL for it.
	err := k8s.WaitUntilNumPodsCreatedE(t, options, metav1.ListOptions{LabelSelector: "sslproxy=example"}, 1, 60, time.Second*10)

	if err != nil {
		t.Error(err)
		t.Fail()
	}

	pods := k8s.ListPods(t, options, metav1.ListOptions{LabelSelector: "sslproxy=example"})
	for i := 0; i < len(pods); i++ {
		pod := pods[i]

		err := k8s.WaitUntilPodAvailableE(t, options, pod.Name, 1, time.Second*15)

		if err != nil {
			t.Error(err)
		}

		defaultConfig, err := k8s.RunKubectlAndGetOutputE(t, options, "exec", pod.Name, "--", "cat", "/etc/recert/conf/default.conf")

		if err != nil {
			t.Error(err)
			t.Error("expected defaultConfig to be located at /etc/recert/conf/default.conf")
			t.Fail()
		}

		if !strings.Contains(defaultConfig, "NGINX TEMPLATE FROM RECERT OPERATOR") {
			t.Error("expected header text in default.conf file not found: 'NGINX TEMPLATE FROM RECERT OPERATOR'")
			t.Fail()
		}
	}

	k8s.WaitUntilServiceAvailable(t, options, "example-nginx-sslproxy", 30, time.Second*10)

	k8s.WaitUntilServiceAvailable(t, options, "example-certbot-service", 30, time.Second*10)

	if isFullTest() {

		service := k8s.GetService(t, options, "example-nginx-sslproxy")
		ipAddress := k8s.GetServiceEndpoint(t, options, service, 80)

		if ipAddress != conf.LoadBalancerIP {
			t.Error("ip address != loadBalancerIP")
			t.Fail()
		}

		var cert string
		{
			tmpl, _ := template.New("conf").Parse(certTemplate)
			var w bytes.Buffer
			tmpl.Execute(&w, conf)
			cert = string(w.Bytes())
			print(cert)
		}
		defer k8s.KubectlDeleteFromString(t, options, cert)
		k8s.KubectlApplyFromString(t, options, cert)

		err = k8s.WaitUntilNumPodsCreatedE(t, options, metav1.ListOptions{LabelSelector: "certbot=example"}, 1, 30, time.Second*10)

		if err != nil {
			t.Error("error when waiting for certbot pod to be created")
			t.Fail()
		}
		err = k8s.WaitUntilNumPodsCreatedE(t, options, metav1.ListOptions{LabelSelector: "certbot=example"}, 0, 30, time.Second*10)

		if err != nil {
			t.Error("error when waiting for certbot pod to destroyed ")
			t.Fail()
		}

	}

}

type yamlConf struct {
	IsFullTest     bool
	LoadBalancerIP string
	Domain         string
}
