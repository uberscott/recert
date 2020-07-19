package util

import (
	"context"
	"fmt"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
)

// GetImagesConfigMap return the images config map which is a map of docker images
// that the operator uses to do its various jobs
func GetImagesConfigMap(client client.Client) (*corev1.ConfigMap, error) {
	rtn := &corev1.ConfigMap{}

	// if the operator namespace is not accessable this program should have already failed
	operatorNamespace, _ := k8sutil.GetOperatorNamespace()

	// if it's empty, then get from the ENVIRONMENT variable (probably running locally)
	if operatorNamespace == "" {
		operatorNamespace, _ = os.LookupEnv("NAMESPACE")
	}

	err := client.Get(context.TODO(), types.NamespacedName{Name: "recert-images-configmap", Namespace: operatorNamespace}, rtn)

	return rtn, err
}

var serviceAccount = "SERVICE_ACCOUNT"

// GetServiceAccount return the ServiceAccountName that the operator is using
func GetServiceAccount() (string, error) {
	ns, found := os.LookupEnv(serviceAccount)
	if !found {
		return "", fmt.Errorf("%s must be set", serviceAccount)
	}
	return ns, nil
}

// GetName return the NAME that the operator is using
func GetName() string {
	name, found := os.LookupEnv("NAME")
	if !found {
		return "recert-operator"
	}
	return name
}

// GetOperatorNamespace returns the namespace of the operator either from k8sutil OR
// return from environment variables if operator is running in DLV
func GetOperatorNamespace() string {
	rtn, err := k8sutil.GetOperatorNamespace()
	if err != nil || rtn == "" {
		name, found := os.LookupEnv("NAMESPACE")
		if !found {
			return "default"
		}
		return name
	}
	return rtn
}

// GetOperatorOptionsConfigMap a map of options that can modify the behavior of the
// operator even while it is running
func GetOperatorOptionsConfigMap(client client.Client) *corev1.ConfigMap {
	rtn := &corev1.ConfigMap{}

	// if the operator namespace is not accessable this program should have already failed
	operatorNamespace := GetOperatorNamespace()

	err := client.Get(context.TODO(), types.NamespacedName{Name: GetName(), Namespace: operatorNamespace}, rtn)

	if err != nil {
		return nil
	}

	return rtn
}

// GetCertCreateMode returns the create mode for the agent
func GetCertCreateMode(client client.Client) string {
	rtn := "create"

	operatorOptionsConfigMap := GetOperatorOptionsConfigMap(client)

	if operatorOptionsConfigMap != nil {
		if operatorOptionsConfigMap.Data["CERT_CREATE_MODE"] != "" {
			rtn = operatorOptionsConfigMap.Data["CERT_CREATE_MODE"]
		}
	}

	return rtn
}

// GetCertFailureBackoffSeconds returns the number of seconds to backoff
// after a failure
func GetCertFailureBackoffSeconds(client client.Client) time.Duration {
	var rtn = time.Duration(time.Second * 60)

	operatorOptionsConfigMap := GetOperatorOptionsConfigMap(client)

	if operatorOptionsConfigMap != nil {
		if operatorOptionsConfigMap.Data["CERT_BACKOFF_SECONDS"] != "" {
			r, err := strconv.ParseInt(operatorOptionsConfigMap.Data["CERT_BACKOFF_SECONDS"], 10, 64)
			println("Error parsing CERT_BACKOFF_SECONDS: ")
			println(err)
			if err == nil {
				return rtn
			}
			rtn = (time.Duration(int64(time.Second) * r))
		}
	}

	return rtn
}

// GetRenewInterval returns the number of seconds between renewals
func GetRenewInterval(client client.Client) time.Duration {

	// every 30 days
	var rtn = time.Duration(time.Hour * 24 * 30)

	operatorOptionsConfigMap := GetOperatorOptionsConfigMap(client)

	if operatorOptionsConfigMap != nil {
		if operatorOptionsConfigMap.Data["CERT_RENEW_INTERVAL"] != "" {
			r, err := strconv.ParseInt(operatorOptionsConfigMap.Data["CERT_RENEW_INTERVAL"], 10, 64)
			println("Error parsing CERT_RENEW_INTERVAL: ")
			println(err)
			if err == nil {
				return rtn
			}
			rtn = (time.Duration(int64(time.Second) * r))
		}
	}

	return rtn
}

// GetUpdateRequeueDelay returns the number of seconds to reque after a delay
func GetUpdateRequeueDelay(client client.Client) time.Duration {

	// every 24 hours days
	var rtn = time.Duration(time.Second * 24 * 60 * 60)

	operatorOptionsConfigMap := GetOperatorOptionsConfigMap(client)

	if operatorOptionsConfigMap != nil {
		if operatorOptionsConfigMap.Data["CERT_UPDATE_REQUEUE_DELAY"] != "" {
			r, err := strconv.ParseInt(operatorOptionsConfigMap.Data["CERT_UPDATE_REQUEUE_DELAY"], 10, 64)
			println("Error parsing CERT_UPDATE_REQUEUE_DELAY:")
			println(err)
			if err == nil {
				return rtn
			}
			rtn = (time.Duration(int64(time.Second) * r))
		}
	}

	return rtn
}
