package quarkssecret

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"code.cloudfoundry.org/quarks-secret/pkg/credsgen"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"code.cloudfoundry.org/quarks-utils/pkg/pointers"
)

// NewQuarksSecretSecretMetaReconciler returns a new ReconcileQuarksSecretSecretMeta
func NewQuarksSecretSecretMetaReconciler(ctx context.Context, config *config.Config, mgr manager.Manager, generator credsgen.Generator) reconcile.Reconciler {
	return &ReconcileQuarksSecretSecretMeta{
		ctx:       ctx,
		config:    config,
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		generator: generator,
	}
}

// ReconcileQuarksSecretSecretMeta reconciles an QuarksSecret object
type ReconcileQuarksSecretSecretMeta struct {
	ctx       context.Context
	client    client.Client
	generator credsgen.Generator
	scheme    *runtime.Scheme
	config    *config.Config
}

// Reconcile applies the `SecretLabels` and `SecretAnnotations` from QuarksSecret to the generated Secret.
func (r *ReconcileQuarksSecretSecretMeta) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	quarksSecret := &qsv1a1.QuarksSecret{}

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(r.ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Infof(ctx, "Reconciling QuarksSecret %s", request.NamespacedName)
	err := r.client.Get(ctx, request.NamespacedName, quarksSecret)
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

	secret, err := GetSourceSecret(ctx, r.client, quarksSecret)
	if err != nil {
		return reconcile.Result{}, err
	}

	newSecretAnnotations := quarksSecret.Spec.SecretAnnotations
	newSecretLabels := quarksSecret.Spec.SecretLabels

	if newSecretLabels == nil {
		newSecretLabels = map[string]string{}
	}
	if newSecretAnnotations == nil {
		newSecretAnnotations = map[string]string{}
	}

	_, ok := secret.GetLabels()[qsv1a1.LabelKind]
	if ok {
		newSecretLabels[qsv1a1.LabelKind] = secret.GetLabels()[qsv1a1.LabelKind]
	}

	if !reflect.DeepEqual(newSecretLabels, secret.Labels) || !reflect.DeepEqual(newSecretAnnotations, secret.Annotations) {
		secret.SetLabels(newSecretLabels)
		secret.SetAnnotations(newSecretAnnotations)
		err = r.client.Update(ctx, secret)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "could not update secret '%s/%s'", secret.Namespace, secret.GetName())
		}

		err = r.updateStatus(ctx, quarksSecret)
		if err != nil {
			return reconcile.Result{}, err
		}
		ctxlog.Infof(ctx, "Updated secret '%s'/'%s'", secret.Name, secret.Namespace)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileQuarksSecretSecretMeta) updateStatus(ctx context.Context, qsec *qsv1a1.QuarksSecret) error {
	qsec.Status.Copied = pointers.Bool(false)
	err := r.client.Status().Update(ctx, qsec)
	if err != nil {
		return err
	}
	return nil
}
