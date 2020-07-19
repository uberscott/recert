package sslproxy

import (
	"bytes"
	"context"
	"fmt"
	"github.com/go-logr/logr"
	mightydevcov1alpha1 "github.com/uberscott/recert/go/src/operator/pkg/apis/mightydevco/v1alpha1"
	"github.com/uberscott/recert/go/src/operator/pkg/util"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_sslproxy")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new SSLProxy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSSLProxy{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("sslproxy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource SSLProxy
	err = c.Watch(&source.Kind{Type: &mightydevcov1alpha1.SSLProxy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &mightydevcov1alpha1.SSLProxy{},
	}, predicate.ResourceVersionChangedPredicate{})

	return nil
}

// blank assignment to verify that ReconcileSSLProxy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSSLProxy{}

// ReconcileSSLProxy reconciles a SSLProxy object
type ReconcileSSLProxy struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile SSLProxy
func (r *ReconcileSSLProxy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling SSLProxy")

	// Fetch the SSLProxy instance
	instance := &mightydevcov1alpha1.SSLProxy{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	result, err := r.reconcilePVC(instance, reqLogger)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileNginxConfigMap(instance, reqLogger)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileSSLSecret(instance, reqLogger)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileService(instance, reqLogger)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileCertbotService(instance, reqLogger)
	if result.Requeue || err != nil {
		return result, err
	}

	result, err = r.reconcileNginxDeployment(instance, reqLogger)

	return result, err
}

func (r *ReconcileSSLProxy) reconcilePVC(instance *mightydevcov1alpha1.SSLProxy, logger logr.Logger) (reconcile.Result, error) {

	pvc := r.newSecretPvc(instance)

	// Set SSLProxy instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pvc, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	found := &corev1.PersistentVolumeClaim{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new PVC", "PVC.Namespace", pvc.Namespace, "PVC.Name", pvc.Name)
		err = r.client.Create(context.TODO(), pvc)
		if err != nil {
			logger.Error(err, "Error when attempting to create a new PVC")
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSSLProxy) reconcileService(instance *mightydevcov1alpha1.SSLProxy, logger logr.Logger) (reconcile.Result, error) {

	service := r.newService(instance)

	// Set SSLProxy instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
		logger.Error(err, "error when setting the controller reference")
		return reconcile.Result{}, err
	}

	found := &corev1.Service{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.client.Create(context.TODO(), service)
		if err != nil {
			logger.Error(err, "Error when attempting to create a new Service")
			return reconcile.Result{}, err
		}
	} else if err != nil {
		logger.Error(err, "Error when creating service")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSSLProxy) reconcileCertbotService(instance *mightydevcov1alpha1.SSLProxy, logger logr.Logger) (reconcile.Result, error) {

	service := r.newCertbotService(instance)

	// Set SSLProxy instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
		logger.Error(err, "error when setting the controller reference")
		return reconcile.Result{}, err
	}

	found := &corev1.Service{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating Certbot Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.client.Create(context.TODO(), service)
		if err != nil {
			logger.Error(err, "Error when attempting to create a new Service")
			return reconcile.Result{}, err
		}
	} else if err != nil {
		logger.Error(err, "Error when creating service")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSSLProxy) getSSLSecret(instance *mightydevcov1alpha1.SSLProxy) (*corev1.Secret, error) {
	found := &corev1.Secret{}
	secret := r.newSSLSecret(instance)
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		return secret, nil
	}

	return found, err
}

func (r *ReconcileSSLProxy) reconcileSSLSecret(instance *mightydevcov1alpha1.SSLProxy, logger logr.Logger) (reconcile.Result, error) {

	secret := r.newSSLSecret(instance)

	// Set SSLProxy instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, secret, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	found := &corev1.Secret{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new NGINX Secret for SSL", "Secret.Namespace", secret.Namespace, "ConfigMap.Name", secret.Name)
		err = r.client.Create(context.TODO(), secret)

		if err != nil {
			logger.Error(err, "Error when attempting to create a new NGINX configMap")
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSSLProxy) reconcileNginxConfigMap(instance *mightydevcov1alpha1.SSLProxy, logger logr.Logger) (reconcile.Result, error) {

	tmpl, err := template.New("default.conf").Parse(defaultNginxConf)

	if err != nil {
		logger.Error(err, "could not parse default.conf template")
		return reconcile.Result{}, err
	}

	var content bytes.Buffer
	data := nginxConf{ReverseProxy: instance.Spec.ReverseProxy,
		CertbotService: instance.Name + "-certbot-service"}
	err = tmpl.Execute(&content, data)

	if err != nil {
		logger.Error(err, "could not execute nginx conf template")
		return reconcile.Result{}, err
	}

	configMap := r.newNginxConfigMap(instance, string(content.Bytes()))

	// Set SSLProxy instance as the owner and controller
	if err = controllerutil.SetControllerReference(instance, configMap, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	found := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new NGINX ConfigMap ", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		err = r.client.Create(context.TODO(), configMap)
		if err != nil {
			logger.Error(err, "Error when attempting to create a new NGINX configMap")
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSSLProxy) reconcileNginxDeployment(instance *mightydevcov1alpha1.SSLProxy, logger logr.Logger) (reconcile.Result, error) {

	secret, err := r.getSSLSecret(instance)

	if err != nil {
		return reconcile.Result{}, err
	}

	imagesMap, err := util.GetImagesConfigMap(r.client)

	if err != nil {
		return reconcile.Result{}, err
	}

	labels := map[string]string{
		"sslproxy": instance.Name,
	}

	annotations := map[string]string{
		"sslSecretVersion": secret.ResourceVersion,
	}

	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.Name + "-nginx-sslproxy",
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels:      labels,
				MatchExpressions: []metav1.LabelSelectorRequirement{}},
			Replicas: instance.Spec.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},

				Spec: corev1.PodSpec{

					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: imagesMap.Data["recertNginx"],
							VolumeMounts: []corev1.VolumeMount{
								{Name: "conf",
									ReadOnly:  true,
									MountPath: "/etc/recert/conf",
								},
								{Name: "ssl",
									ReadOnly:  true,
									MountPath: "/etc/recert/ssl",
								},
							},
						},
					},
					Volumes: []corev1.Volume{{
						Name:         "conf",
						VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: instance.Name + "-nginx-sslproxy"}}},
					},
						{
							Name:         "ssl",
							VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: instance.Name + "-nginx-sslproxy"}},
						},
					},
				}},
		},
	}

	// Set SSLProxy instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, deployment, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	found := &v1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new NGINX Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.client.Create(context.TODO(), deployment)
		if err != nil {
			logger.Error(err, "Error when attempting to create a new NGINX deployment")
			return reconcile.Result{}, err
		}
		// Deployment created successfully - don't requeue
	} else if err != nil {
		return reconcile.Result{}, err
	} else if found.Annotations["sslSecretVersion"] != secret.ResourceVersion {
		logger.Info("sslSecretVersion has been updated, must update deployment")
		r.client.Update(context.TODO(), deployment)
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileSSLProxy) newNginxConfigMap(cr *mightydevcov1alpha1.SSLProxy, defaultConf string) *corev1.ConfigMap {
	labels := map[string]string{
		"sslproxy": cr.Name,
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-nginx-sslproxy",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{"default.conf": defaultConf},
	}
}

func (r *ReconcileSSLProxy) newSSLSecret(cr *mightydevcov1alpha1.SSLProxy) *corev1.Secret {
	labels := map[string]string{
		"sslproxy": cr.Name,
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-nginx-sslproxy",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
	}
}

func (r *ReconcileSSLProxy) newCertbotService(cr *mightydevcov1alpha1.SSLProxy) *corev1.Service {
	labels := map[string]string{
		"certbot": cr.Name,
	}
	rtn := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-certbot-service",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{Name: "http",
					Port:       80,
					TargetPort: intstr.FromInt(80),
					Protocol:   corev1.ProtocolTCP},
			},
			Selector: map[string]string{
				"certbot": cr.Name,
			},
		},
	}

	return rtn
}

func (r *ReconcileSSLProxy) newService(cr *mightydevcov1alpha1.SSLProxy) *corev1.Service {
	labels := map[string]string{
		"sslproxy": cr.Name,
	}
	rtn := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-nginx-sslproxy",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{Name: "http",
					Port:       80,
					TargetPort: intstr.FromInt(80),
					Protocol:   corev1.ProtocolTCP},
				{Name: "https",
					Port:       443,
					TargetPort: intstr.FromInt(443),
					Protocol:   corev1.ProtocolTCP},
			},
			Selector: map[string]string{
				"sslproxy": cr.Name,
			},
		},
	}

	if strings.TrimSpace(cr.Spec.LoadBalancerIP) != "" {
		log.Info(fmt.Sprintf("LoadBalancerIp set to %v", cr.Spec.LoadBalancerIP))
		rtn.Spec.LoadBalancerIP = cr.Spec.LoadBalancerIP
	} else {
		log.Info("skipping Load Balancer which is not specified")
	}

	return rtn
}

func (r *ReconcileSSLProxy) newSecretPvc(cr *mightydevcov1alpha1.SSLProxy) *corev1.PersistentVolumeClaim {
	labels := map[string]string{
		"sslproxy": cr.Name,
	}
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-nginx-sslproxy",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: getStorageClassDefault(),
			AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			}},
	}
}

type nginxConf struct {
	CertbotService string
	ReverseProxy   string
}

var defaultNginxConf = `##############################################
# 
# NGINX TEMPLATE FROM RECERT OPERATOR
#
##############################################

server 
{
   listen 80;
   listen [::]:80;

   location /.well-known/
   {
      proxy_pass http://{{ .CertbotService }}:80;
      proxy_bind $server_addr;
      proxy_set_header X-Forwarded-Host $http_host;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-Port 80;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_redirect off;
      proxy_intercept_errors on;
      add_header Access-Control-Allow-Origin *;
      expires -1;
   }
   
   location /
   {
      proxy_pass {{ .ReverseProxy }}/;
      proxy_set_header X-Forwarded-Host $http_host;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-Port 443;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_redirect off;
      proxy_intercept_errors on;
      expires -1;
   }
}

server 
{
   listen 443 ssl http2;
   listen [::]:443 ssl http2;

   ssl_certificate     /ssl/$ssl_server_name.crt;
   ssl_certificate_key /ssl/$ssl_server_name.key;

   location /
   {
      proxy_pass {{ .ReverseProxy }}/;
      proxy_set_header X-Forwarded-Host $http_host;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-Port 443;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_redirect off;
      proxy_intercept_errors on;
      expires -1;
   }
}`

func getStorageClassDefault() *string {
	rtn, found := os.LookupEnv("STORAGE_CLASS")
	if !found {
		rtn = "standard"
	}
	if len(rtn) == 0 {
		rtn = "standard"
	}
	return &rtn
}
