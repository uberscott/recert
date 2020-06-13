package testoperator

import (
	"github.com/gruntwork-io/terratest/modules/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
	"time"
)

func TestSslProxy(t *testing.T) {
	t.Parallel()

	// Path to the Kubernetes resource config we will test.
	sslProxy :=
		`apiVersion: williamsheavyindustries.com/v1alpha1
kind: SSLProxy
metadata:
  name: example
spec:
  selector:
    app: blah
  reverseProxy: http://blah:80
  replicas: 1`

	// Setup the kubectl config and context.
	options := k8s.NewKubectlOptions("", "", "recert")

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

	// ensure SSL was created

}
