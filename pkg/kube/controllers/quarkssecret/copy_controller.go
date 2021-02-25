package quarkssecret

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	credsgen "code.cloudfoundry.org/quarks-secret/pkg/credsgen/in_memory_generator"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"code.cloudfoundry.org/quarks-utils/pkg/skip"
)

// AddCopy creates a new QuarksSecrets controller to watch for the
// user defined secrets.
func AddCopy(ctx context.Context, config *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContextWithRecorder(ctx, "copy-reconciler", mgr.GetEventRecorderFor("copy-recorder"))
	log := ctxlog.ExtractLogger(ctx)
	r := NewCopyReconciler(ctx, config, mgr, credsgen.NewInMemoryGenerator(log), controllerutil.SetControllerReference)

	c, err := controller.New("copy-controller", mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: config.MaxQuarksSecretWorkers,
	})
	if err != nil {
		return errors.Wrap(err, "Adding copy controller to manager failed.")
	}

	nsPred := newNSPredicate(ctx, mgr.GetClient(), config.MonitoredID)

	// Watch for changes to the copied status of QuarksSecrets
	p := predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return false },
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			n := e.ObjectNew.(*qsv1a1.QuarksSecret)

			if n.Status.Copied != nil {
				ctxlog.Debugf(ctx, "Skipping QuarksSecret '%s', if copy status '%v' is true", n.Name, *n.Status.Copied)
				return !(*n.Status.Copied)
			}

			return true
		},
	}
	err = c.Watch(&source.Kind{Type: &qsv1a1.QuarksSecret{}}, &handler.EnqueueRequestForObject{}, nsPred, p)
	if err != nil {
		return errors.Wrapf(err, "Watching quarks secrets failed in copy controller.")
	}

	// Watch for changes to user created secrets
	p = predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return false },
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			n := e.ObjectNew.(*corev1.Secret)
			o := e.ObjectOld.(*corev1.Secret)

			shouldProcessReconcile := isUserCreatedSecret(n)
			if reflect.DeepEqual(n.Data, o.Data) && reflect.DeepEqual(n.Labels, o.Labels) &&
				reflect.DeepEqual(n.Annotations, o.Annotations) {
				return false
			}

			return shouldProcessReconcile
		},
	}
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(
		func(a crc.Object) []reconcile.Request {
			secret := a.(*corev1.Secret)

			if skip.Reconciles(ctx, mgr.GetClient(), secret) {
				return []reconcile.Request{}
			}

			reconciles, err := listQuarksSecretsReconciles(ctx, mgr.GetClient(), secret, secret.Namespace)
			if err != nil {
				ctxlog.Errorf(ctx, "Failed to calculate reconciles for secret '%s/%s': %v", secret.Namespace, secret.Name, err)
			}
			if len(reconciles) > 0 {
				return reconciles
			}

			return reconciles
		}), nsPred, p)
	if err != nil {
		return errors.Wrapf(err, "Watching user defined secrets failed in copy controller.")
	}

	return nil
}

func isUserCreatedSecret(secret *corev1.Secret) bool {
	secretLabels := secret.GetLabels()
	value, ok := secretLabels[qsv1a1.LabelKind]
	if !ok {
		return true
	}
	if value != qsv1a1.GeneratedSecretKind {
		return true
	}
	return !ok
}

// listQuarksSecretsReconciles lists all Quarks Secrets associated with the a particular secret.
func listQuarksSecretsReconciles(ctx context.Context, client crc.Client, secret *corev1.Secret, namespace string) ([]reconcile.Request, error) {
	quarksSecretList := &qsv1a1.QuarksSecretList{}
	err := client.List(ctx, quarksSecretList, crc.InNamespace(namespace))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list QuarksSecrets")
	}

	result := []reconcile.Request{}
	for _, quarksSecret := range quarksSecretList.Items {
		if quarksSecret.Spec.SecretName == secret.Name {
			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      quarksSecret.Name,
					Namespace: quarksSecret.Namespace,
				}}
			result = append(result, request)
			ctxlog.NewMappingEvent(secret).Debug(ctx, request, "QuarksSecret", secret.Name, qsv1a1.KubeSecretReference)
		}
	}
	return result, nil
}
