package testoperator

import (
	"fmt"
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

func TestSslProxy(t *testing.T) {
	t.Parallel()

	stage := createStage()
	stage.logConf()
	stage.stage(t)

	stage.testExpectedSslProxyDefaultConf(t)
	stage.testProxyWorking(t)
	if stage.Conf.IsFullTest {

		stage.testExpectedIP(t)
	}
}

func TestCert(t *testing.T) {

	stage := createStage()

	if !stage.Conf.IsFullTest {
		t.SkipNow()
		return
	}

	stage.stage(t)
	stage.setRunOptions(t, &RunOptions{CertRunMode: "dryrun"})

	stage.testExpectedIP(t)

	defer k8s.KubectlDeleteFromString(t, stage.Options, stage.Templates.CertTemplate)
	k8s.KubectlApplyFromString(t, stage.Options, stage.Templates.CertTemplate)

	// wait for Cert to go into Updated mode

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
// STAGE
/////////////////////////////////////////////

type Stage struct {
	Conf       *TheConf
	Templates  *TemplatesStruct
	Options    *k8s.KubectlOptions
	RunOptions *RunOptions
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

func (stage *Stage) logConf() {
	fmt.Printf("Conf.Name: %v\n", stage.Conf.Name)
	fmt.Printf("Conf.IsFullTest: %v\n", stage.Conf.IsFullTest)
	fmt.Printf("Conf.LoadBalancerIP: %v\n", stage.Conf.LoadBalancerIP)
	fmt.Printf("Conf.StageTeardown: %v\n\n", stage.Conf.StageTeardown)
}

func (stage *Stage) unStage(t *testing.T) {
	defer k8s.KubectlDeleteFromString(t, stage.Options, stage.Templates.ServiceTemplate)
	defer k8s.KubectlDeleteFromString(t, stage.Options, stage.Templates.DeploymentTemplate)
	defer k8s.KubectlDeleteFromString(t, stage.Options, stage.Templates.SslProxyTemplate)
}

/**
 * here we stage the environment which involves setting up the SSLProxy
 */
func (stage *Stage) stage(t *testing.T) {

	fmt.Printf("***> stage.Conf.StageTeardown %v <***\n", stage.Conf.StageTeardown)
	if stage.Conf.StageTeardown {
		stage.unStage(t)
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

func (stage *Stage) setRunOptions(t *testing.T, runOptions *RunOptions) {
	stage.RunOptions = runOptions
	k8s.KubectlApplyFromString(t, stage.Options, from(OperatorOptionsConfigMap, runOptions))
}

/////////////////////////////////////////////
// stage tests
/////////////////////////////////////////////

func (stage *Stage) testExpectedIP(t *testing.T) {

	service := k8s.GetService(t, stage.Options, stage.Conf.Name+"-nginx-sslproxy")
	ipAddress := k8s.GetServiceEndpoint(t, stage.Options, service, 80)

	if ipAddress != stage.Conf.LoadBalancerIP+":80" {
		t.Errorf("ip address != loadBalancerIP expected: %v found: %v", stage.Conf.LoadBalancerIP, ipAddress)
		t.Fail()
		return
	}
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

/////////////////////////////////////////////
// data structures
/////////////////////////////////////////////

// TheConf holds all the test configuration information which can be saved in a file conf.yaml
// The information in TheConf holds some information that can't be hard coded in the test framework
// like for instance the static ip Address and domain that will be tested
type TheConf struct {
	IsFullTest     bool              `yaml:"isFullTest,omitempty"`
	LoadBalancerIP string            `yaml:"loadBalancerIP,omitempty"`
	Domain         string            `yaml:"domain,omitempty"`
	Name           string            `yaml:"name"`
	Namespace      string            `yaml:"namespace"`
	StageTeardown  bool              `yaml:"stageTeardown,omitempty"`
	Images         map[string]string `yaml:"images,omitempty"`
}

// RunOptions options that modify how the operator will behave for an individual test
// typically expressed through the operator options ConfigMap
type RunOptions struct {
	CertRunMode string
}

/////////////////////////////////////////////
// utility methods
/////////////////////////////////////////////

func loadConf() *TheConf {

	var conf TheConf
	{
		filename, _ := filepath.Abs("conf.yaml")
		yamlFile, err := ioutil.ReadFile(filename)

		err2 := yaml.Unmarshal(yamlFile, &conf)

		if err != nil || err2 != nil {
			println("using default conf.")
			conf.IsFullTest = false
			conf.Namespace = "recert-test-namespace"
			conf.Name = "test"
			conf.StageTeardown = true
		} else {
			println("loading conf from conf.yaml")
		}

	}

	// requires that `make build` be run before this test
	{
		filename, _ := filepath.Abs("../../../out/images-manifest.yaml")
		yamlFile, err := ioutil.ReadFile(filename)

		if err != nil {
			panic("cannot load \"../../../out/images-manifest.yaml\" which must be built by \"make build\" at the root of this repository before this test can be run")
		}
		err = yaml.UnmarshalStrict(yamlFile, &conf.Images)
		if err != nil {
			panic("cannot unmarshal \"../../../out/images-manifest.yaml\" which must be built by \"make build\" at the root of this repository before this test can be run")
		}
	}

	return &conf
}
