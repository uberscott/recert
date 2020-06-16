package testoperator

import (
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/retry"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type Stage struct {
	Conf      *theConf
	Templates *TemplatesStruct
	Options   *k8s.KubectlOptions
}

func createStage() *Stage {

	conf := loadConf()
	stage := Stage{
		Conf:      conf,
		Templates: fromAll(conf),
		Options:   k8s.NewKubectlOptions("", "", conf.Namespace),
	}
	return &stage
}

func (stage *Stage) unstage(t *testing.T) {
	defer k8s.KubectlDeleteFromString(t, stage.Options, stage.Templates.ServiceTemplate)
	defer k8s.KubectlDeleteFromString(t, stage.Options, stage.Templates.DeploymentTemplate)
	defer k8s.KubectlDeleteFromString(t, stage.Options, stage.Templates.SslProxyTemplate)
}

/**
 * here we stage the environment which involves setting up the SSLProxy
 */
func (stage *Stage) stage(t *testing.T) {

	if stage.Conf.StageTeardown {
		stage.unstage(t)
	}

	k8s.KubectlApplyFromString(t, stage.Options, stage.Templates.ServiceTemplate)
	k8s.KubectlApplyFromString(t, stage.Options, stage.Templates.DeploymentTemplate)

	// before we create an SSLProxy the service it points to must be available
	// otherwise NGINX will crash
	k8s.WaitUntilServiceAvailable(t, stage.Options, stage.Conf.Name, 60, time.Second*5)

	k8s.KubectlApplyFromString(t, stage.Options, stage.Templates.SslProxyTemplate)

	k8s.WaitUntilNumPodsCreated(t, stage.Options, metav1.ListOptions{LabelSelector: "app.kubernetes.io/name=" + stage.Conf.Name}, 1, 60, time.Second*10)
	k8s.WaitUntilNumPodsCreated(t, stage.Options, metav1.ListOptions{LabelSelector: "sslproxy=" + stage.Conf.Name}, 1, 60, time.Second*10)

	// wait for services that the operator should create
	k8s.WaitUntilServiceAvailable(t, stage.Options, stage.Conf.Name+"-nginx-sslproxy", 30, time.Second*10)
	k8s.WaitUntilServiceAvailable(t, stage.Options, stage.Conf.Name+"-certbot-service", 30, time.Second*10)
}

func (stage *Stage) testExpectedSslProxyDefaultConf(t *testing.T) {
	pods := k8s.ListPods(t, stage.Options, metav1.ListOptions{LabelSelector: "sslproxy=" + stage.Conf.Name})
	pod := pods[0]
	k8s.WaitUntilPodAvailable(t, stage.Options, pod.Name, 4, time.Second*15)
	defaultConfig, err := k8s.RunKubectlAndGetOutputE(t, stage.Options, "exec", pod.Name, "--", "cat", "/etc/recert/conf/default.conf")
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

func (stage *Stage) testProxyWorking(t *testing.T) {
	service := k8s.GetService(t, stage.Options, stage.Conf.Name+"-nginx-sslproxy")
	ipAddress := k8s.GetServiceEndpoint(t, stage.Options, service, 80)
	http_helper.HttpGetWithRetryWithCustomValidation(t, "http://"+ipAddress, nil, 30, 5*time.Second, func(code int, body string) bool {
		return code == 200 && strings.Contains(body, "Hello World")
	})
}

func TestSslProxy(t *testing.T) {
	t.Parallel()

	stage := createStage()
	stage.stage(t)

	stage.testExpectedSslProxyDefaultConf(t)
	stage.testProxyWorking(t)
}

func TestCert(t *testing.T) {
	stage := createStage()
	stage.stage(t)

	service := k8s.GetService(t, stage.Options, stage.Conf.Name+"-nginx-sslproxy")
	ipAddress := k8s.GetServiceEndpoint(t, stage.Options, service, 80)

	if ipAddress != stage.Conf.LoadBalancerIP {
		t.Error("ip address != loadBalancerIP")
		t.Fail()
		return
	}

	defer k8s.KubectlDeleteFromString(t, stage.Options, stage.Templates.CertTemplate)
	k8s.KubectlApplyFromString(t, stage.Options, stage.Templates.CertTemplate)

	k8s.WaitUntilNumPodsCreated(t, stage.Options, metav1.ListOptions{LabelSelector: "certbot=" + stage.Conf.Name}, 1, 30, time.Second*10)

	if stage.Conf.IsFullTest {
		// now make the same call using https
		http_helper.HttpGetWithRetryWithCustomValidation(t, "https://"+stage.Conf.Domain, nil, 30, 5*time.Second, func(code int, body string) bool {

			return code == 200 && strings.Contains(body, "Hello World")
		})
	}
}

func TestSecretsSSLCreate(t *testing.T) {
	t.Parallel()

	conf := loadConf()

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

	options := k8s.NewKubectlOptions("", "", conf.Namespace)
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

/////////////////////////////////////////////
// utility methods
/////////////////////////////////////////////

func loadConf() *theConf {

	var conf theConf
	{
		filename, _ := filepath.Abs("conf.yaml")
		yamlFile, err := ioutil.ReadFile(filename)

		err2 := yaml.Unmarshal(yamlFile, &conf)

		if err != nil || err2 != nil {
			conf.IsFullTest = false
			conf.Namespace = "recert-test-namespace"
			conf.Name = "test"
			conf.StageTeardown = true
		}
	}

	// requires that `make build` be run before this test
	{
		filename, _ := filepath.Abs("../../../out/images-manifest.yaml")
		yamlFile, err := ioutil.ReadFile(filename)

		if err != nil {
			panic("cannot load \"../../../out/images-manifest.yaml\" which must be built by \"make build\" at the root of this repository before this test can be run")
		}
		err = yaml.Unmarshal(yamlFile, &conf.Images)
		if err != nil {
			panic("cannot unmarshal \"../../../out/images-manifest.yaml\" which must be built by \"make build\" at the root of this repository before this test can be run")
		}
	}

	return &conf
}

type theConf struct {
	IsFullTest     bool              `json:"isFullTest"`
	LoadBalancerIP string            `json:"loadBalancerIP"`
	Domain         string            `json:"domain"`
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	StageTeardown  bool              `json:"stageTeardown"`
	Images         map[string]string `json:"images,omitempty"`
}
