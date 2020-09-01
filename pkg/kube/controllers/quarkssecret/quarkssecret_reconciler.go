package quarkssecret

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"code.cloudfoundry.org/quarks-secret/pkg/credsgen"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"code.cloudfoundry.org/quarks-utils/pkg/meltdown"
	"code.cloudfoundry.org/quarks-utils/pkg/pointers"
)

// HelmTemplate is the constant used to identify the helm based templating
const HelmTemplate = "helm"

type setReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error

// NewQuarksSecretReconciler returns a new ReconcileQuarksSecret
func NewQuarksSecretReconciler(ctx context.Context, config *config.Config, mgr manager.Manager, generator credsgen.Generator, srf setReferenceFunc) reconcile.Reconciler {
	return &ReconcileQuarksSecret{
		ctx:          ctx,
		config:       config,
		client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		generator:    generator,
		setReference: srf,
	}
}

// ReconcileQuarksSecret reconciles an QuarksSecret object
type ReconcileQuarksSecret struct {
	ctx          context.Context
	client       client.Client
	generator    credsgen.Generator
	scheme       *runtime.Scheme
	setReference setReferenceFunc
	config       *config.Config
}

type caNotReadyError struct {
	message string
}

func newCaNotReadyError(message string) *caNotReadyError {
	return &caNotReadyError{message: message}
}

// Error returns the error message
func (e *caNotReadyError) Error() string {
	return e.message
}

func isCaNotReady(o interface{}) bool {
	err := o.(error)
	err = errors.Cause(err)
	_, ok := err.(*caNotReadyError)
	return ok
}

type secNotReadyError struct {
	message string
}

func newSecNotReadyError(message string) *secNotReadyError {
	return &secNotReadyError{message: message}
}

// Error returns the error message
func (e *secNotReadyError) Error() string {
	return e.message
}

func isSecNotReady(o interface{}) bool {
	err := o.(error)
	err = errors.Cause(err)
	_, ok := err.(*secNotReadyError)
	return ok
}

// Reconcile reads that state of the cluster for a QuarksSecret object and makes changes based on the state read
// and what is in the QuarksSecret.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileQuarksSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	qsec := &qsv1a1.QuarksSecret{}

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(r.ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Infof(ctx, "Reconciling QuarksSecret %s", request.NamespacedName)
	err := r.client.Get(ctx, request.NamespacedName, qsec)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			ctxlog.Info(ctx, "Skip reconcile: quarks secret not found")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		ctxlog.Info(ctx, "Error reading the object")
		return reconcile.Result{}, errors.Wrap(err, "Error reading quarksSecret")
	}
	if meltdown.NewWindow(r.config.MeltdownDuration, qsec.Status.LastReconcile).Contains(time.Now()) {
		ctxlog.WithEvent(qsec, "Meltdown").Debugf(ctx, "Resource '%s' is in meltdown, requeue reconcile after %s", qsec.GetNamespacedName(), r.config.MeltdownRequeueAfter)
		return reconcile.Result{RequeueAfter: r.config.MeltdownRequeueAfter}, nil
	}

	// Create secret
	switch qsec.Spec.Type {
	case qsv1a1.Password:
		ctxlog.Info(ctx, "Generating password")
		err = r.createPasswordSecret(ctx, qsec)
		if err != nil {
			ctxlog.Infof(ctx, "Error generating password secret: %s", err.Error())
			return reconcile.Result{}, errors.Wrap(err, "generating password secret failed.")
		}
	case qsv1a1.RSAKey:
		ctxlog.Info(ctx, "Generating RSA Key")
		err = r.createRSASecret(ctx, qsec)
		if err != nil {
			ctxlog.Infof(ctx, "Error generating RSA key secret: %s", err.Error())
			return reconcile.Result{}, errors.Wrap(err, "generating RSA key secret failed.")
		}
	case qsv1a1.SSHKey:
		ctxlog.Info(ctx, "Generating SSH Key")
		err = r.createSSHSecret(ctx, qsec)
		if err != nil {
			ctxlog.Infof(ctx, "Error generating SSH key secret: %s", err.Error())
			return reconcile.Result{}, errors.Wrap(err, "generating SSH key secret failed.")
		}
	case qsv1a1.Certificate:
		ctxlog.Info(ctx, "Generating certificate")
		err = r.createCertificateSecret(ctx, qsec)
		if err != nil {
			if isCaNotReady(err) {
				ctxlog.Info(ctx, fmt.Sprintf("CA for secret '%s' is not ready yet: %s", request.NamespacedName, err))
				return reconcile.Result{RequeueAfter: time.Second * 5}, nil
			}
			ctxlog.Info(ctx, "Error generating certificate secret: "+err.Error())
			return reconcile.Result{}, errors.Wrap(err, "generating certificate secret.")
		}
	case qsv1a1.BasicAuth:
		err = r.createBasicAuthSecret(ctx, qsec)
		if err != nil {
			return reconcile.Result{}, errors.Wrap(err, "generating basic-auth secret")
		}
	case qsv1a1.TemplatedConfig:
		if err := r.createTemplatedConfigSecret(ctx, qsec); err != nil {
			if isSecNotReady(err) {
				ctxlog.Info(ctx, fmt.Sprintf("Secrets '%s' is not ready yet: %s", request.NamespacedName, err))
				return reconcile.Result{RequeueAfter: time.Second * 5}, nil
			}
			ctxlog.Info(ctx, "Error generating templatedConfig secret: "+err.Error())
			return reconcile.Result{}, errors.Wrap(err, "generating templatedConfig secret.")
		}
	case qsv1a1.SecretCopy:
		// noop
		return reconcile.Result{}, nil
	case qsv1a1.DockerConfigJSON:
		ctxlog.Info(ctx, "Generating dockerConfigJson")
		err = r.createDockerConfigJSON(ctx, qsec)
		if err != nil {
			if isSecNotReady(err) {
				ctxlog.Info(ctx, fmt.Sprintf("Secrets '%s' is not ready yet: %s", request.NamespacedName, err))
				return reconcile.Result{RequeueAfter: time.Second * 5}, nil
			}
			ctxlog.Info(ctx, "Error generating dockerConfigJson secret: "+err.Error())
			return reconcile.Result{}, errors.Wrap(err, "generating dockerConfigJson secret.")
		}
	default:
		err = ctxlog.WithEvent(qsec, "InvalidTypeError").Errorf(ctx, "Invalid type: %s", qsec.Spec.Type)
		return reconcile.Result{}, err
	}
	r.updateStatus(ctx, qsec)
	return reconcile.Result{}, nil
}

func (r *ReconcileQuarksSecret) updateStatus(ctx context.Context, qsec *qsv1a1.QuarksSecret) {
	qsec.Status.Generated = pointers.Bool(true)

	now := metav1.Now()
	qsec.Status.LastReconcile = &now
	err := r.client.Status().Update(ctx, qsec)
	if err != nil {
		ctxlog.Errorf(ctx, "could not create or update QuarksSecret status '%s': %v", qsec.GetNamespacedName(), err)
	}
}

// Skip creation when
// * secret is already generated according to qsecs status field
// * secret exists, but was not generated (user created secret)
func (r *ReconcileQuarksSecret) skipCreation(ctx context.Context, qsec *qsv1a1.QuarksSecret) (bool, error) {
	if qsec.Status.Generated != nil && *qsec.Status.Generated {
		ctxlog.Debugf(ctx, "Existing secret %s/%s has already been generated",
			qsec.Namespace,
			qsec.Spec.SecretName,
		)
		return true, nil
	}

	secretName := qsec.Spec.SecretName

	existingSecret := &corev1.Secret{}

	err := r.client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: qsec.GetNamespace()}, existingSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		return false, errors.Wrapf(err, "could not get secret")
	}

	secretLabels := existingSecret.GetLabels()
	if secretLabels == nil {
		secretLabels = map[string]string{}
	}

	// skip if the secret was not created by the operator
	if secretLabels[qsv1a1.LabelKind] != qsv1a1.GeneratedSecretKind {
		ctxlog.Debugf(ctx, "Existing secret %s/%s doesn't have a label %s=%s",
			existingSecret.GetNamespace(),
			existingSecret.GetName(),
			qsv1a1.LabelKind,
			qsv1a1.GeneratedSecretKind,
		)
		return true, nil
	}

	return false, nil
}

func (r *ReconcileQuarksSecret) createSecrets(ctx context.Context, qsec *qsv1a1.QuarksSecret, secret *corev1.Secret) error {
	// Create the main secret
	// Check if allowed to generate secret, could be already done or
	// created manually by a user
	skipCreation, err := r.skipCreation(ctx, qsec)
	if err != nil {
		ctxlog.Errorf(ctx, "Error reading the secret: %v", err.Error())
	}
	if skipCreation {
		ctxlog.WithEvent(qsec, "SkipCreation").Infof(ctx, "Skip creation: Secret '%s/%s' already exists and it's not generated", qsec.Namespace, qsec.Spec.SecretName)
	} else {
		if err := r.createSecret(ctx, qsec, secret); err != nil {
			return err
		}
	}

	// See if we have to make any copies
	for _, copy := range qsec.Spec.Copies {
		copiedQSec := qsec.DeepCopy()
		copiedSecret := &corev1.Secret{}

		copiedQSec.Spec.SecretName = copy.Name
		copiedQSec.Namespace = copy.Namespace
		copiedSecret.Name = copy.Name
		copiedSecret.Namespace = copy.Namespace
		copiedSecret.Data = secret.Data
		copiedSecret.Annotations = secret.Annotations
		copiedSecret.Labels = secret.Labels

		// We look and see if the secret is of the generated type
		// And also check if we're allowed to create the resource in the target namespace
		// We require an existing secret or qsecret with a label already be present in the target namespace
		skip, needsCreation, err := r.skipCopy(ctx, copiedQSec, qsec)
		if err != nil {
			ctxlog.Errorf(ctx, "Error reading the secret: %v", err.Error())
		}
		if skip {
			ctxlog.WithEvent(qsec, "SkipCreation").Infof(ctx, "Skip copy creation: Secret/QSecret '%s' must exist and have the appropriate labels and annotations to receive a copy", copy.String())
		} else if needsCreation {
			if err := r.createCopySecret(ctx, copiedQSec, copiedSecret, qsec.GetNamespacedName()); err != nil {
				return err
			}
		} else {
			if err := r.updateCopySecret(ctx, copiedSecret); err != nil {
				return err
			}
		}
	}

	return nil
}
