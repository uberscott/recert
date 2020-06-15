package testoperator

import (
	"bytes"
	"fmt"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/retry"
	"gopkg.in/yaml.v2"
	"html/template"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
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

func getName() string {

	rtn, found := os.LookupEnv("NAME")
	if !found {
		rtn = "test"
	}
	return rtn
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
  name: {{ .Name }}
spec:
  {{- if .IsFullTest }}
  loadBalancerIP: {{ .LoadBalancerIP }}
  {{- end }} 
  reverseProxy: http://{{ .Name }}:80
  replicas: 1`

	certTemplate := `apiVersion: mightydevco.com/v1alpha1
kind: Cert
metadata:
  name: {{ .Name }}
spec:
  domain: {{ .Domain }}
  email: scott@mightydevco.com
  sslProxy: {{ .Name }}`

	serviceTemplate := `apiVersion: v1
kind: Service
metadata:
  name: {{ .Name }}
  labels:
    app.kubernetes.io/name: {{ .Name }}
    app.kubernetes.io/instance: {{ .Name }}
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
    name: http
  selector:
    app.kubernetes.io/name: {{ .Name }}
    app.kubernetes.io/instance: {{ .Name }}
`

	deploymentTemplate := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
  labels:
    app.kubernetes.io/name: {{ .Name }}
    app.kubernetes.io/instance: {{ .Name }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ .Name }}
      app.kubernetes.io/instance: {{ .Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ .Name }}
        app.kubernetes.io/instance: {{ .Name }}
    spec:
      containers:
        - name: nginx
          image: nginxdemos/hello:latest

          imagePullPolicy: Always
          ports:
            - name: http
              containerPort: 80
              protocol: TCP`

	conf := theConf{
		IsFullTest: isFullTest(),
		Name:       getName(),
	}

	if isFullTest() {
		conf.LoadBalancerIP, _ = getLoadBalancerIP()
		conf.Domain, _ = getDomain()
	}

	var service string
	{
		tmpl, _ := template.New("service").Parse(serviceTemplate)
		var w bytes.Buffer
		tmpl.Execute(&w, conf)
		service = string(w.Bytes())
	}

	var deployment string
	{
		tmpl, _ := template.New("deployment").Parse(deploymentTemplate)
		var w bytes.Buffer
		tmpl.Execute(&w, conf)
		deployment = string(w.Bytes())
	}

	var sslProxy string
	{
		tmpl, _ := template.New("sslProxy").Parse(sslProxyTemplate)
		var w bytes.Buffer
		tmpl.Execute(&w, conf)
		sslProxy = string(w.Bytes())
	}

	options := k8s.NewKubectlOptions("", "", getNameSpace())

	// First the service must be setup or Nginx will crash
	defer k8s.KubectlDeleteFromString(t, options, service)
	defer k8s.KubectlDeleteFromString(t, options, deployment)
	k8s.KubectlApplyFromString(t, options, service)
	k8s.KubectlApplyFromString(t, options, deployment)
	k8s.WaitUntilServiceAvailable(t, options, conf.Name, 60, time.Second*5)
	k8s.WaitUntilNumPodsCreated(t, options, metav1.ListOptions{LabelSelector: "app.kubernetes.io/name=" + conf.Name}, 1, 60, time.Second*10)

	// At the end of the test, run "kubectl delete" to clean up any resources that were created.
	defer k8s.KubectlDeleteFromString(t, options, sslProxy)
	k8s.KubectlApplyFromString(t, options, sslProxy)
	k8s.WaitUntilNumPodsCreated(t, options, metav1.ListOptions{LabelSelector: "sslproxy=" + conf.Name}, 1, 60, time.Second*10)

	pods := k8s.ListPods(t, options, metav1.ListOptions{LabelSelector: "sslproxy=" + conf.Name})
	for i := 0; i < len(pods); i++ {
		pod := pods[i]

		k8s.WaitUntilPodAvailable(t, options, pod.Name, 4, time.Second*15)

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

	k8s.WaitUntilServiceAvailable(t, options, conf.Name+"-nginx-sslproxy", 30, time.Second*10)

	k8s.WaitUntilServiceAvailable(t, options, conf.Name+"-certbot-service", 30, time.Second*10)

	http_helper.HttpGetWithRetryWithCustomValidation(t, "http://"+conf.Domain, nil, 30, 5*time.Second, func(code int, body string) bool {

		return code == 200 && strings.Contains(body, "Hello World")
	})

	if isFullTest() {

		service := k8s.GetService(t, options, conf.Name+"-nginx-sslproxy")
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
		}
		defer k8s.KubectlDeleteFromString(t, options, cert)
		k8s.KubectlApplyFromString(t, options, cert)

		k8s.WaitUntilNumPodsCreated(t, options, metav1.ListOptions{LabelSelector: "certbot=" + conf.Name}, 1, 30, time.Second*10)

		// now make the same call using https
		http_helper.HttpGetWithRetryWithCustomValidation(t, "https://"+conf.Domain, nil, 30, 5*time.Second, func(code int, body string) bool {

			return code == 200 && strings.Contains(body, "Hello World")
		})

	}

}

func from(templateSrc string, data interface{}) string {
	tmpl, _ := template.New("conf").Parse(templateSrc)
	var w bytes.Buffer
	tmpl.Execute(&w, data)
	return string(w.Bytes())
}

func TestSecretsSSLCreate(t *testing.T) {
	t.Parallel()

	deploymentTemplate := `apiVersion: v1
kind: Pod
metadata:
  name: {{ .Name }}-secrets-test
spec:
  serviceAccountName: operator
  containers:
    - name: certbot
      image: {{ .Images.recertCertbot }}
      command: 
      - /opt/mightydevco/launcher.sh
      - mock
      - certbot2.mightydevco.com
      - scott@mightydevco.com
      - {{ .Name }}-secrets-test

      imagePullPolicy: Always
      ports:
      - name: http
        containerPort: 80
        protocol: TCP`

	options := k8s.NewKubectlOptions("", "", getNameSpace())
	conf := loadConf()
	{
		deployment := from(deploymentTemplate, conf)
		defer k8s.KubectlDeleteFromString(t, options, deployment)
		k8s.KubectlApplyFromString(t, options, deployment)
	}

	podName := conf.Name + "-secrets-test"

	// wait for it to be 1 then wait for it to be 0
	k8s.WaitUntilPodAvailable(t, options, podName, 1, time.Second*10)

	retry.DoWithRetry(t, "get new secret", 10, time.Second*5, func() (string, error) {
		_, err := k8s.GetSecretE(t, options, conf.Name+"-secrets-test-new")
		if err == nil {
			return "OK", nil
		}
		return "", err

	})

	newSecret := k8s.GetSecret(t, options, conf.Name+"-secrets-test-new")

	if string(newSecret.Data["certbot2.mightydevco.com.crt"]) == "" {
		t.Error("missing certbot2.mightydevco.com.crt which should have been created")
		t.Fail()
	}

	if string(newSecret.Data["certbot2.mightydevco.com.key"]) == "" {
		t.Error("missing certbot2.mightydevco.com.key which should have been created")
		t.Fail()
	}

}

func loadConf() theConf {
	conf := theConf{
		IsFullTest: isFullTest(),
		Name:       getName(),
	}

	if isFullTest() {
		conf.LoadBalancerIP, _ = getLoadBalancerIP()
		conf.Domain, _ = getDomain()
	}

	// requires that `make build` be run before this test
	filename, _ := filepath.Abs("../../../out/images-manifest.yaml")
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		panic("cannot load \"../../../out/images-manifest.yaml\" which must be built by \"make build\" at the root of this repository before this test can be run")
	}
	err = yaml.Unmarshal(yamlFile, &conf.Images)
	if err != nil {
		panic("cannot unmarshal \"../../../out/images-manifest.yaml\" which must be built by \"make build\" at the root of this repository before this test can be run")
	}

	return conf
}

type theConf struct {
	IsFullTest     bool
	LoadBalancerIP string
	Domain         string
	Name           string
	Images         map[string]string
}
