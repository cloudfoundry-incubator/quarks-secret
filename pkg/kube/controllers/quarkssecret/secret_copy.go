package quarkssecret

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-secret/pkg/kube/util/mutate"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
)

// Skip copy creation when
// * secret and qsecret (copy type) doesn't exist
// * (q)secret exists, but it's not marked as generated (label) and a copy (annotation)
func (r *ReconcileQuarksSecret) skipCopy(ctx context.Context, qsec *qsv1a1.QuarksSecret, sourceQsec *qsv1a1.QuarksSecret) (bool, bool, error) {
	copyOf := sourceQsec.GetNamespacedName()
	if qsec.Status.Generated != nil && *qsec.Status.Generated {
		ctxlog.Debugf(ctx, "Existing secret %s/%s has already been generated",
			qsec.Namespace,
			qsec.Spec.SecretName,
		)
		return true, false, nil
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

	// We want to create the secret if a qsec is present, or update a secret if the secret is already there (with the correct annotation)
	existingQSec := &qsv1a1.QuarksSecret{}
	err := r.client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: qsec.GetNamespace()}, existingQSec)
	if err != nil {
		if apierrors.IsNotFound(err) {
			notFoundQsec = true
		} else {
			return false, false, errors.Wrapf(err, "could not get qsecret")
		}
	}

	err = r.client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: qsec.GetNamespace()}, existingSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			notFoundSec = true
		} else {
			return false, false, errors.Wrapf(err, "could not get secret")
		}
	}

	// If both are absent, we will skip the copy
	if notFoundSec && notFoundQsec {
		ctxlog.WithEvent(sourceQsec, "SkipCreation").Infof(ctx, "No Valid QSecret or Secret found")
		return true, false, nil
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
			return true, false, nil
		}

		return !validateCopySecret(ctx, sourceQsec, secretLabels, secretAnnotations, copyOf), true, nil
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

		return !validateCopySecret(ctx, sourceQsec, secretLabels, secretAnnotations, copyOf), false, nil
	}

	return true, false, nil
}

func validateCopySecret(ctx context.Context, qsec *qsv1a1.QuarksSecret, secretLabels, secretAnnotations map[string]string, copyOf string) bool {

	valid := true

	// check if the secret is marked as generated
	if secretLabels[qsv1a1.LabelKind] != qsv1a1.GeneratedSecretKind {
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

// createCopySecret applies common properties(labels and ownerReferences) to the secret and creates it
func (r *ReconcileQuarksSecret) createCopySecret(ctx context.Context, qsec *qsv1a1.QuarksSecret, secret *corev1.Secret, copyOf string) error {
	ctxlog.Debugf(ctx, "Creating secret '%s/%s', owned by quarks secret '%s'", secret.Namespace, secret.Name, qsec.GetNamespacedName())

	secretLabels := secret.GetLabels()
	if secretLabels == nil {
		secretLabels = map[string]string{}
	}

	secretAnnotations := secret.GetAnnotations()
	if secretAnnotations == nil {
		secretAnnotations = map[string]string{}
	}

	secretLabels[qsv1a1.LabelKind] = qsv1a1.GeneratedSecretKind
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
	uncachedSecret.Object["data"] = secret.Data
	err := r.client.Update(ctx, uncachedSecret)

	if err != nil {
		return errors.Wrapf(err, "could not update secret '%s/%s'", secret.Namespace, secret.GetName())
	}

	return nil
}
