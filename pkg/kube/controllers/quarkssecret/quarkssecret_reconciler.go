package quarkssecret

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	certv1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"code.cloudfoundry.org/quarks-secret/pkg/credsgen"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-secret/pkg/kube/util/mutate"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"code.cloudfoundry.org/quarks-utils/pkg/meltdown"
	"code.cloudfoundry.org/quarks-utils/pkg/names"
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
	case qsv1a1.Certificate, qsv1a1.TLS:
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
func (r *ReconcileQuarksSecret) skipCreation(ctx context.Context, qsec *qsv1a1.QuarksSecret) (bool, *corev1.Secret, error) {
	if qsec.Status.Generated != nil && *qsec.Status.Generated {
		ctxlog.Debugf(ctx, "Existing secret %s/%s has already been generated",
			qsec.Namespace,
			qsec.Spec.SecretName,
		)
		return true, nil, nil
	}

	secretName := qsec.Spec.SecretName

	existingSecret := &corev1.Secret{}

	err := r.client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: qsec.GetNamespace()}, existingSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil, nil
		}
		return false, nil, errors.Wrapf(err, "could not get secret")
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
		return true, existingSecret, nil
	}

	return false, nil, nil
}

// Skip copy creation when
// * secret and qsecret (copy type) doesn't exist
// * (q)secret exists, but it's not marked as generated (label) and a copy (annotation)
func (r *ReconcileQuarksSecret) skipCopy(ctx context.Context, qsec *qsv1a1.QuarksSecret, sourceQsec *qsv1a1.QuarksSecret, userCreatedSecret bool) (bool, bool, *unstructured.Unstructured, error) {
	copyOf := sourceQsec.GetNamespacedName()
	if qsec.Status.Generated != nil && *qsec.Status.Generated {
		ctxlog.Debugf(ctx, "Existing secret %s/%s has already been generated",
			qsec.Namespace,
			qsec.Spec.SecretName,
		)
		return true, false, nil, nil
	}

	secretName := qsec.Spec.SecretName
	notFoundQsec := false
	notFoundSec := false

	// We use an unstructured object so we don't hit the _namespaced_ cache
	// since our object could live in another namespace
	existingSecret := &unstructured.Unstructured{}
	existingSecret.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Kind:    "Secret",
		Version: "v1",
	})

	ctxlog.Infof(ctx, "Qsec name and namespace", qsec.Name, qsec.GetNamespace())
	ctxlog.Infof(ctx, "Sec name and namespace", secretName, qsec.GetNamespace())

	// We want to create the secret if a qsec is present, or update a secret if the secret is already there (with the correct annotation)
	existingQSec := &qsv1a1.QuarksSecret{}
	err := r.client.Get(ctx, types.NamespacedName{Name: qsec.Name, Namespace: qsec.GetNamespace()}, existingQSec)
	if err != nil {
		if apierrors.IsNotFound(err) {
			notFoundQsec = true
		} else {
			return false, false, nil, errors.Wrapf(err, "could not get qsecret")
		}
	}

	err = r.client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: qsec.GetNamespace()}, existingSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			notFoundSec = true
		} else {
			return false, false, nil, errors.Wrapf(err, "could not get secret")
		}
	}

	// If both are absent, we will skip the copy
	if notFoundSec && notFoundQsec {
		ctxlog.WithEvent(sourceQsec, "SkipCreation").Infof(ctx, "No Valid QSecret or Secret found")
		return true, false, nil, nil
	}

	needsCreation := true
	// If both of them are false, give preference to qsec
	if !notFoundSec && !notFoundQsec {
		ctxlog.WithEvent(sourceQsec, "SkipCreation").Infof(ctx, "Both QSecret & Secret found. Giving preference to QSecret")
		notFoundSec = true
		needsCreation = false
	}

	if notFoundSec {
		// Validate QSecret
		ctxlog.WithEvent(sourceQsec, "SkipCreation").Infof(ctx, "Valid QSecret found")

		secretLabels := existingQSec.GetLabels()
		if secretLabels == nil {
			secretLabels = map[string]string{}
		}

		secretAnnotations := existingQSec.GetAnnotations()
		if secretAnnotations == nil {
			secretAnnotations = map[string]string{}
		}

		if existingQSec.Spec.Type != qsv1a1.SecretCopy {
			ctxlog.WithEvent(sourceQsec, "SkipCreation").Infof(ctx, "Invalid type for QSecret. It must be 'copy' type.")
			return true, false, nil, nil
		}

		return !validateCopySecret(ctx, sourceQsec, secretLabels, secretAnnotations, copyOf, userCreatedSecret), needsCreation, nil, nil
	} else if notFoundQsec {
		// Validate Secret
		secretLabels := existingSecret.GetLabels()
		if secretLabels == nil {
			secretLabels = map[string]string{}
		}

		secretAnnotations := existingSecret.GetAnnotations()
		if secretAnnotations == nil {
			secretAnnotations = map[string]string{}
		}

		return !validateCopySecret(ctx, sourceQsec, secretLabels, secretAnnotations, copyOf, userCreatedSecret), false, existingSecret, nil
	}

	return true, false, nil, nil
}

func validateCopySecret(ctx context.Context, qsec *qsv1a1.QuarksSecret, secretLabels, secretAnnotations map[string]string, copyOf string, userCreatedSecret bool) bool {
	valid := true
	// check if the secret is marked as generated
	if secretLabels[qsv1a1.LabelKind] != qsv1a1.GeneratedSecretKind && !userCreatedSecret {
		ctxlog.WithEvent(qsec, "SkipCopyCreation").Infof(ctx, "Secret doesn't have generated label")
		valid = false
	}
	// skip if this is a copy, and we're missing a copy-of label
	if secretAnnotations[qsv1a1.AnnotationCopyOf] != copyOf {
		ctxlog.WithEvent(qsec, "SkipCopyCreation").Infof(ctx, "Secret doesn't have the corresponding annotation %s vs %s", secretAnnotations[qsv1a1.AnnotationCopyOf], copyOf)
		valid = false
	}

	return valid
}

func (r *ReconcileQuarksSecret) createSecrets(ctx context.Context, qsec *qsv1a1.QuarksSecret, secret *corev1.Secret) error {
	// Create the main secret
	// Check if allowed to generate secret, could be already done or
	// created manually by a user

	userCreatedSecret := false
	skipCreation, existingSecret, err := r.skipCreation(ctx, qsec)
	if err != nil {
		ctxlog.Errorf(ctx, "Error reading the secret: %v", err.Error())
	}

	if skipCreation {
		ctxlog.WithEvent(qsec, "SkipCreation").Infof(ctx, "Skip creation: Secret '%s/%s' already exists and it's not generated", qsec.Namespace, qsec.Spec.SecretName)
		secret = existingSecret

		// Add user created annotation
		if secret != nil {
			if secret.Annotations == nil {
				secret.Annotations = map[string]string{}
			}
			secret.Annotations[qsv1a1.AnnotationUserCreatedSecret] = "true"
			userCreatedSecret = true
		}
	} else {
		if err := r.createSecret(ctx, qsec, secret); err != nil {
			return err
		}
	}

	for _, copy := range qsec.Spec.Copies {
		copiedQSec := qsec.DeepCopy()
		copiedQSec.Spec.SecretName = copy.Name
		copiedQSec.Namespace = copy.Namespace
		copiedSecret := &corev1.Secret{}

		// We look and see if the secret is of the generated type
		// And also check if we're allowed to create the resource in the target namespace
		// We require an existing secret or qsecret with a label already be present in the target namespace
		skip, needsCreation, existingSecretCopyNamespace, err := r.skipCopy(ctx, copiedQSec, qsec, userCreatedSecret)
		if err != nil {
			ctxlog.Errorf(ctx, "Error reading the secret: %v", err.Error())
		}

		if existingSecretCopyNamespace != nil {
			if secret.Annotations == nil {
				secret.Annotations = map[string]string{}
			}
			if secret.Labels == nil {
				secret.Labels = map[string]string{}
			}

			for k, v := range existingSecretCopyNamespace.GetAnnotations() {
				secret.Annotations[k] = v
			}

			for k, v := range existingSecretCopyNamespace.GetLabels() {
				secret.Labels[k] = v
			}
		}

		copiedSecret.Name = copy.Name
		copiedSecret.Namespace = copy.Namespace
		copiedSecret.Data = secret.Data
		copiedSecret.Annotations = secret.Annotations
		copiedSecret.Labels = secret.Labels

		if skip {
			ctxlog.WithEvent(qsec, "SkipCreation").Infof(ctx, "Skip copy creation: Secret/QSecret '%s' must exist and have the appropriate labels and annotations to receive a copy", copy.String())
		} else if needsCreation {
			if err := r.createCopySecret(ctx, copiedQSec, copiedSecret, qsec.GetNamespacedName(), userCreatedSecret); err != nil {
				return err
			}
			ctxlog.WithEvent(qsec, "SkipCreation").Infof(ctx, "Copy secret '%s' has been created in namespace '%s'", copy.Name, copy.Namespace)
		} else {
			if err := r.updateCopySecret(ctx, copiedSecret); err != nil {
				return err
			}
			ctxlog.WithEvent(qsec, "SkipCreation").Infof(ctx, "Copy secret '%s' has been updated in namespace '%s'", copy.Name, copy.Namespace)
		}
	}

	return nil
}

// createCopySecret applies common properties(labels and ownerReferences) to the secret and creates it
func (r *ReconcileQuarksSecret) createCopySecret(ctx context.Context, qsec *qsv1a1.QuarksSecret, secret *corev1.Secret, copyOf string, userCreatedSecret bool) error {
	ctxlog.Debugf(ctx, "Creating secret '%s/%s', owned by quarks secret '%s'", secret.Namespace, secret.Name, qsec.GetNamespacedName())

	secretLabels := secret.GetLabels()
	if secretLabels == nil {
		secretLabels = map[string]string{}
	}

	secretAnnotations := secret.GetAnnotations()
	if secretAnnotations == nil {
		secretAnnotations = map[string]string{}
	}

	if !userCreatedSecret {
		secretLabels[qsv1a1.LabelKind] = qsv1a1.GeneratedSecretKind
	}
	secretAnnotations[qsv1a1.AnnotationCopyOf] = copyOf

	secret.SetLabels(secretLabels)
	secret.SetAnnotations(secretAnnotations)
	if err := r.setReference(qsec, secret, r.scheme); err != nil {
		return errors.Wrapf(err, "error setting owner for secret '%s' to QuarksSecret '%s'", secret.GetName(), qsec.GetNamespacedName())
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.client, secret, mutate.SecretMutateFn(secret))
	if err != nil {
		return errors.Wrapf(err, "could not create or update secret '%s/%s'", secret.Namespace, secret.GetName())
	}

	if op != "unchanged" {
		ctxlog.Debugf(ctx, "Secret '%s' has been %s", secret.Name, op)
	}

	return nil
}

// createSecret applies common properties(labels and ownerReferences) to the secret and creates it
func (r *ReconcileQuarksSecret) createSecret(ctx context.Context, qsec *qsv1a1.QuarksSecret, secret *corev1.Secret) error {
	ctxlog.Debugf(ctx, "Creating secret '%s/%s', owned by quarks secret '%s'", secret.Namespace, secret.Name, qsec.GetNamespacedName())

	secretLabels := secret.GetLabels()
	if secretLabels == nil {
		secretLabels = map[string]string{}
	}

	secretLabels[qsv1a1.LabelKind] = qsv1a1.GeneratedSecretKind

	secret.SetLabels(secretLabels)

	if err := r.setReference(qsec, secret, r.scheme); err != nil {
		return errors.Wrapf(err, "error setting owner for secret '%s' to QuarksSecret '%s'", secret.GetName(), qsec.GetNamespacedName())
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.client, secret, mutate.SecretMutateFn(secret))
	if err != nil {
		return errors.Wrapf(err, "could not create or update secret '%s/%s'", secret.Namespace, secret.GetName())
	}

	if op != "unchanged" {
		ctxlog.Debugf(ctx, "Secret '%s' has been %s", secret.Name, op)
	}

	return nil
}

// updateCopySecret updates a copied destination Secret
func (r *ReconcileQuarksSecret) updateCopySecret(ctx context.Context, qsec *qsv1a1.QuarksSecret, secret *corev1.Secret) error {
	// If this is a copy (lives in a different namespace), we only do an update,
	// since we're not allowed to create, and we don't set a reference, because
	// cross namespace references are not supported
	uncachedSecret := &unstructured.Unstructured{}
	uncachedSecret.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Kind:    "Secret",
		Version: "v1",
	})
	uncachedSecret.SetName(secret.Name)
	uncachedSecret.SetNamespace(secret.Namespace)
	uncachedSecret.SetLabels(secret.Labels)
	uncachedSecret.SetAnnotations(secret.Annotations)
	uncachedSecret.Object["data"] = secret.Data
	err := r.client.Update(ctx, uncachedSecret)

	if err != nil {
		return errors.Wrapf(err, "could not update secret '%s/%s'", secret.Namespace, secret.GetName())
	}

	return nil
}

// generateCertificateGenerationRequest generates CertificateGenerationRequest for certificate
func (r *ReconcileQuarksSecret) generateCertificateGenerationRequest(ctx context.Context, namespace string, certificateRequest qsv1a1.CertificateRequest) (credsgen.CertificateGenerationRequest, error) {
	var request credsgen.CertificateGenerationRequest
	switch certificateRequest.SignerType {
	case qsv1a1.ClusterSigner:
		// Generate cluster-signed CA certificate
		request = credsgen.CertificateGenerationRequest{
			CommonName:       certificateRequest.CommonName,
			AlternativeNames: certificateRequest.AlternativeNames,
		}
	case qsv1a1.LocalSigner:
		// Generate local-issued CA certificate
		request = credsgen.CertificateGenerationRequest{
			IsCA:             certificateRequest.IsCA,
			CommonName:       certificateRequest.CommonName,
			AlternativeNames: certificateRequest.AlternativeNames,
		}

		if len(certificateRequest.CARef.Name) > 0 {
			// Get CA certificate
			caSecret := &corev1.Secret{}
			caNamespacedName := types.NamespacedName{
				Namespace: namespace,
				Name:      certificateRequest.CARef.Name,
			}
			err := r.client.Get(ctx, caNamespacedName, caSecret)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return request, newCaNotReadyError("CA secret not found")
				}
				return request, errors.Wrap(err, "getting CA secret")
			}
			ca := caSecret.Data[certificateRequest.CARef.Key]

			// Get CA key
			if certificateRequest.CAKeyRef.Name != certificateRequest.CARef.Name {
				caSecret = &corev1.Secret{}
				caNamespacedName = types.NamespacedName{
					Namespace: namespace,
					Name:      certificateRequest.CAKeyRef.Name,
				}
				err = r.client.Get(ctx, caNamespacedName, caSecret)
				if err != nil {
					if apierrors.IsNotFound(err) {
						return request, newCaNotReadyError("CA key secret not found")
					}
					return request, errors.Wrap(err, "getting CA Key secret")
				}
			}
			key := caSecret.Data[certificateRequest.CAKeyRef.Key]
			request.CA = credsgen.Certificate{
				IsCA:        true,
				PrivateKey:  key,
				Certificate: ca,
			}
		}
	default:
		return request, fmt.Errorf("unrecognized signer type: %s", certificateRequest.SignerType)
	}

	return request, nil
}

// createCertificateSigningRequest creates CertificateSigningRequest Object
func (r *ReconcileQuarksSecret) createCertificateSigningRequest(ctx context.Context, qsec *qsv1a1.QuarksSecret, csr []byte) error {
	csrName := names.CSRName(qsec.Namespace, qsec.Name)
	ctxlog.Debugf(ctx, "Creating certificatesigningrequest '%s'", csrName)

	annotations := qsec.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[qsv1a1.AnnotationCertSecretName] = qsec.Spec.SecretName
	annotations[qsv1a1.AnnotationQSecNamespace] = qsec.Namespace
	annotations[qsv1a1.AnnotationQSecName] = qsec.Name
	annotations[qsv1a1.AnnotationMonitoredID] = r.config.MonitoredID

	csrObj := &certv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:        csrName,
			Labels:      qsec.Labels,
			Annotations: annotations,
		},
		Spec: certv1.CertificateSigningRequestSpec{
			Request: csr,
			Usages:  qsec.Spec.Request.CertificateRequest.Usages,
		},
	}

	oldCsrObj := &certv1.CertificateSigningRequest{}

	// CSR spec is immutable after the request is created
	err := r.client.Get(ctx, types.NamespacedName{Name: csrObj.Name}, oldCsrObj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = r.client.Create(ctx, csrObj)
			if err != nil {
				return errors.Wrapf(err, "could not create certificatesigningrequest '%s'", csrObj.Name)
			}
			return nil
		}
		return errors.Wrapf(err, "could not get certificatesigningrequest '%s'", csrObj.Name)
	}

	ctxlog.Infof(ctx, "Ignoring immutable CSR '%s'", csrObj.Name)
	return nil
}
