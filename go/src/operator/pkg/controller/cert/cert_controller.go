package cert

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	mightydevco "github.com/uberscott/recert/go/src/operator/pkg/apis/mightydevco/v1alpha1"
	"github.com/uberscott/recert/go/src/operator/pkg/util"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_cert")

// Add creates a new Cert Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCert{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("cert-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Cert
	err = c.Watch(&source.Kind{Type: &mightydevco.Cert{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCert implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCert{}

// ReconcileCert reconciles a Cert object
type ReconcileCert struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Cert object and makes changes based on the state read
// and what is in the Cert.Spec
func (r *ReconcileCert) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Cert")

	instance := &mightydevco.Cert{}

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

	if instance.Status.State == "" {
		return r.reconcileNone(instance, reqLogger)
	} else if instance.Status.State == mightydevco.Pending {
		return r.reconcilePending(instance, reqLogger)
	} else if instance.Status.State == mightydevco.Creating {
		return r.reconcileCreating(instance, reqLogger)
	} else if instance.Status.State == mightydevco.FailureBackoff {
		return r.reconcileFailureBackoff(instance, reqLogger)
	} else if instance.Status.State == mightydevco.Updated {
		return r.reconcileFailureBackoff(instance, reqLogger)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileCert) reconcileNone(instance *mightydevco.Cert, reqLogger logr.Logger) (reconcile.Result, error) {
	if err := r.changeCertState(instance, mightydevco.Pending, reqLogger); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{Requeue: true}, nil
}

func (r *ReconcileCert) reconcilePending(instance *mightydevco.Cert, reqLogger logr.Logger) (reconcile.Result, error) {

	job := r.createRecertAgentPod(instance)

	if err := controllerutil.SetControllerReference(instance, job, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	exists, err := r.existsByName(&v1.Job{}, agentName(instance), instance)

	if err != nil {
		return reconcile.Result{}, err
	} else if !exists {
		reqLogger.Info(fmt.Sprintf("Creating %v job.", job.Name))
		err = r.client.Create(context.TODO(), job)
		for i := 0; i < 6; i++ {
			_, err = r.findJob(instance)
			if err == nil {
				break
			}
			reqLogger.Info("waiting for job to be ready...")
			time.Sleep(5 * time.Second)
		}

		if err != nil {
			return reconcile.Result{}, err
		}

		if err := r.changeCertState(instance, mightydevco.Creating, reqLogger); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 30}, nil
	} else {
		reqLogger.Info("Pending job exists, waiting another 5 minutes")
		// if the job already exists then another cert is being refreshed
		// requeue and check 5 minutes later
		return reconcile.Result{Requeue: true, RequeueAfter: util.GetCertFailureBackoffSeconds(r.client)}, nil
	}
}

func (r *ReconcileCert) reconcileCreating(instance *mightydevco.Cert, reqLogger logr.Logger) (reconcile.Result, error) {

	job, err := r.findJob(instance)

	// if we can't find the job, then something went really wrong FailureBackoff
	if err != nil {
		r.changeCertState(instance, mightydevco.FailureBackoff, reqLogger)
		reqLogger.Error(err, "Job could not be found.")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * util.GetCertFailureBackoffSeconds(r.client)}, err
	}

	if job.Status.Failed > 0 {
		r.changeCertState(instance, mightydevco.FailureBackoff, reqLogger)
		reqLogger.Info("Job failed.")
		r.client.Delete(context.TODO(), job)
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * util.GetCertFailureBackoffSeconds(r.client)}, err
	}

	if job.Status.Succeeded <= 0 {
		reqLogger.Info("Job is still running.")
		return reconcile.Result{Requeue: true, RequeueAfter: 15 * time.Second}, err
	}

	// delete the old job
	r.client.Delete(context.TODO(), job)

	// delete leftover pods
	var pod corev1.Pod
	pod.Namespace = instance.Namespace
	err = r.client.DeleteAllOf(context.TODO(), &pod, &client.DeleteAllOfOptions{
		ListOptions: client.ListOptions{Namespace: instance.Namespace, LabelSelector: labels.SelectorFromSet(r.createAgentPodLabels(instance))},
	})

	if err != nil {
		reqLogger.Error(err, "could not delete job pods")
	}

	// if we have gotten to this point the job must have succeeded
	newSecret, err := r.findNewSecret(instance)
	secret, err := r.findSecret(instance)

	if err != nil {
		reqLogger.Error(err, "could not find the new newSecret")
		r.changeCertState(instance, mightydevco.FailureBackoff, reqLogger)
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * util.GetCertFailureBackoffSeconds(r.client)}, err
	}

	secret.Data = newSecret.Data

	err = r.client.Delete(context.TODO(), newSecret)

	if err != nil {
		reqLogger.Error(err, "cannot delete new secret")
	}

	r.client.Delete(context.TODO(), newSecret)
	err = r.client.Update(context.TODO(), secret)

	if err != nil {
		reqLogger.Error(err, "cannot update secret")
		r.changeCertState(instance, mightydevco.FailureBackoff, reqLogger)
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * util.GetCertFailureBackoffSeconds(r.client)}, err
	}

	err = r.changeCertState(instance, mightydevco.Updated, reqLogger)

	deployment, err := r.findSslProxyDeployment(instance)

	if err == nil {
		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = make(map[string]string)
		}
		deployment.Spec.Template.Annotations["updated"] = time.Now().String()
		err = r.client.Update(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "could not update sslProxy deployment")
		}
	} else {
		reqLogger.Error(err, "cannot find sslProxy deployment")
	}

	// requeue once per day
	return reconcile.Result{Requeue: true, RequeueAfter: 24 * 60 * 60}, nil
}

func (r *ReconcileCert) reconcileFailureBackoff(instance *mightydevco.Cert, reqLogger logr.Logger) (reconcile.Result, error) {

	reqLogger.Info("processing FailureBackoff")
	backoffSeconds := util.GetCertFailureBackoffSeconds(r.client)
	elapsed := time.Now().Unix() - instance.Status.LastStateChange

	reqLogger.Info(fmt.Sprintf("ELAPSED: %v BACKOFFSECONDS: %v", elapsed, int(backoffSeconds.Seconds())))

	if elapsed < int64(backoffSeconds.Seconds()) {
		reqLogger.Info("backoff threshold not yet reached, requeue...")
		return reconcile.Result{Requeue: true, RequeueAfter: 24 * 60 * 60}, nil
	}

	err := r.changeCertState(instance, mightydevco.Pending, reqLogger)
	return reconcile.Result{Requeue: true}, err
}

func (r *ReconcileCert) reconcileUpdated(instance *mightydevco.Cert, reqLogger logr.Logger) (reconcile.Result, error) {

	reqLogger.Info("processing Updated")
	renewIntervalSeconds := util.GetRenewInterval(r.client)
	elapsed := time.Now().Unix() - instance.Status.LastStateChange

	reqLogger.Info(fmt.Sprintf("ELAPSED: %v BACKOFFSECONDS: %v", elapsed, int(renewIntervalSeconds.Seconds())))

	if elapsed < int64(renewIntervalSeconds.Seconds()) {
		reqLogger.Info("update threshold not yet reached, requeue...")
		return reconcile.Result{Requeue: true, RequeueAfter: renewIntervalSeconds}, nil
	}

	err := r.changeCertState(instance, mightydevco.Pending, reqLogger)
	return reconcile.Result{Requeue: true, RequeueAfter: util.GetUpdateRequeueDelay(r.client)}, err
}

/////////////////////////////////////
// NAMES
/////////////////////////////////////

func agentName(instance *mightydevco.Cert) string {
	return util.AgentName(instance)
}

func secretName(instance *mightydevco.Cert) string {
	return util.SecretNameFromCert(instance)
}

func newSecretName(instance *mightydevco.Cert) string {
	return util.NewSecretNameFromCert(instance)
}

func sslDeploymentName(instance *mightydevco.Cert) string {
	return util.SslProxyDeploymentNameFromCert(instance)
}

/////////////////////////////////////
// SUB RESOURCE CREATION
/////////////////////////////////////

func (r *ReconcileCert) createAgentPodLabels(cr *mightydevco.Cert) map[string]string {
	return map[string]string{
		"certbot": cr.Spec.SslProxy,
	}
}

func (r *ReconcileCert) createRecertAgentPod(cr *mightydevco.Cert) *v1.Job {

	imagesMap, _ := util.GetImagesConfigMap(r.client)
	serviceAccountName, _ := util.GetServiceAccount()

	labels := r.createAgentPodLabels(cr)

	var backoffLimit int32 = 0
	var activeDeadlineSeconds int64 = 60 * 5

	return &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentName(cr),
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: v1.JobSpec{
			BackoffLimit:          &backoffLimit,
			ActiveDeadlineSeconds: &activeDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   agentName(cr),
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccountName,
					Containers: []corev1.Container{
						{
							Name:  "certbot",
							Image: imagesMap.Data["recertCertbot"],
							Command: []string{"/opt/mightydevco/launcher.sh",
								util.GetCertCreateMode(r.client),
								cr.Spec.Domain,
								cr.Spec.Email,
								cr.Name + "-nginx-sslproxy"},

							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "pvc",
									MountPath: "/etc/letsencrypt",
									ReadOnly:  false,
								},
								{
									Name:      "ssl",
									MountPath: "/ssl",
									ReadOnly:  true,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name:         "pvc",
							VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: cr.Spec.SslProxy + "-nginx-sslproxy"}},
						},
						{
							Name:         "ssl",
							VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: cr.Spec.SslProxy + "-nginx-sslproxy"}},
						},
					},
				},
			},
		},
	}
}

/////////////////////////////////////
//  UTILITY
/////////////////////////////////////

func (r *ReconcileCert) changeCertState(instance *mightydevco.Cert, state string, reqLogger logr.Logger) error {
	prevState := instance.Status.State

	// first lets get the latest instance
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}, instance)

	instance.Status.State = state

	if state == mightydevco.Updated {
		instance.Status.LastUpdated = time.Now().String()
	}

	instance.Status.LastStateChange = time.Now().Unix()

	err = r.client.Status().Update(context.TODO(), instance)

	if err != nil {
		reqLogger.Error(err, "error when attempting to change Cert status")
		return err
	}

	if prevState == "" {
		prevState = "None"
	}

	reqLogger.Info(fmt.Sprintf("Status going from %v to %v", prevState, state), "PrevStatus", prevState, "NewStatus", state)
	return nil
}

func (r *ReconcileCert) exists(obj runtime.Object, instance *mightydevco.Cert) (bool, error) {
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, obj)
	if err != nil && errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}

}

func (r *ReconcileCert) existsByName(obj runtime.Object, name string, instance *mightydevco.Cert) (bool, error) {
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.Namespace}, obj)
	if err != nil && errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}

}

func (r *ReconcileCert) findJob(instance *mightydevco.Cert) (*v1.Job, error) {
	var found v1.Job
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: agentName(instance), Namespace: instance.Namespace}, &found)
	return &found, err
}

func (r *ReconcileCert) findNewSecret(instance *mightydevco.Cert) (*corev1.Secret, error) {
	return r.findSecretByName(instance, newSecretName(instance))
}

func (r *ReconcileCert) findSecret(instance *mightydevco.Cert) (*corev1.Secret, error) {
	return r.findSecretByName(instance, secretName(instance))
}

func (r *ReconcileCert) findSecretByName(instance *mightydevco.Cert, name string) (*corev1.Secret, error) {
	var found corev1.Secret
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.Namespace}, &found)
	return &found, err
}

func (r *ReconcileCert) findSslProxyDeployment(instance *mightydevco.Cert) (*v12.Deployment, error) {
	var found v12.Deployment
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: sslDeploymentName(instance), Namespace: instance.Namespace}, &found)
	return &found, err
}
