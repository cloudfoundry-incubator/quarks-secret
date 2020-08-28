package quarkssecret

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	"code.cloudfoundry.org/quarks-secret/pkg/kube/util/reference"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"code.cloudfoundry.org/quarks-utils/pkg/pointers"
	"code.cloudfoundry.org/quarks-utils/pkg/skip"
)

// AddQuarksSecret creates a new QuarksSecrets controller to watch for the
// custom resource and reconcile it into k8s secrets.
func AddQuarksSecret(ctx context.Context, config *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContextWithRecorder(ctx, "quarks-secret-reconciler", mgr.GetEventRecorderFor("quarks-secret-recorder"))
	log := ctxlog.ExtractLogger(ctx)
	r := NewQuarksSecretReconciler(ctx, config, mgr, credsgen.NewInMemoryGenerator(log), controllerutil.SetControllerReference)

	// Create a new controller
	c, err := controller.New("quarks-secret-controller", mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: config.MaxQuarksSecretWorkers,
	})
	if err != nil {
		return errors.Wrap(err, "Adding quarks secret controller to manager failed.")
	}

	nsPred := newNSPredicate(ctx, mgr.GetClient(), config.MonitoredID)

	// Watch for changes to QuarksSecrets
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			o := e.Object.(*qsv1a1.QuarksSecret)
			secrets, err := listSecrets(ctx, mgr.GetClient(), o)
			if err != nil {
				ctxlog.Errorf(ctx, "Failed to list secrets owned by QuarksSecret '%s': %s in quarksSecret controller", o.GetNamespacedName(), err)
			}
			if len(secrets) == 0 {
				ctxlog.NewPredicateEvent(e.Object).Debug(
					ctx, e.Meta, "qsv1a1.QuarksSecret",
					fmt.Sprintf("Create predicate passed for '%s/%s'", e.Meta.GetNamespace(), e.Meta.GetName()),
				)
				return true
			}
			return false
		},
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			n := e.ObjectNew.(*qsv1a1.QuarksSecret)
			o := e.ObjectOld.(*qsv1a1.QuarksSecret)

			if reflect.DeepEqual(n.Spec, o.Spec) && reflect.DeepEqual(n.Labels, o.Labels) &&
				reflect.DeepEqual(n.Annotations, o.Annotations) && reflect.DeepEqual(n.Status, o.Status) {
				return false
			}

			// When should we reconcile?
			// | old   | new   | reconcile? |
			// | ----- | ----- | ---------- |
			// | true  | true  | false      |
			// | false | true  | false      |
			// | nil   | true  | false      |
			// | true  | false | true       |
			// | false | false | true       |
			// | nil   | false | true       |
			// | true  | nil   | false      |
			// | false | nil   | true       |
			// | nil   | nil   | true       |
			if !n.Status.NotGenerated() || (n.Status.Generated == nil && !o.Status.IsGenerated()) {
				ctxlog.NewPredicateEvent(e.ObjectNew).Debug(
					ctx, e.MetaNew, "qsv1a1.QuarksSecret",
					fmt.Sprintf("Update predicate passed for '%s/%s'.", e.MetaNew.GetNamespace(), e.MetaNew.GetName()),
				)
				return true
			}
			return false
		},
	}
	err = c.Watch(&source.Kind{Type: &qsv1a1.QuarksSecret{}}, &handler.EnqueueRequestForObject{}, nsPred, p)
	if err != nil {
		return errors.Wrapf(err, "Watching quarks secrets failed in quarksSecret controller.")
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
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			secret := a.Object.(*corev1.Secret)

			if skip.Reconciles(ctx, mgr.GetClient(), secret) {
				return []reconcile.Request{}
			}

			reconciles, err := reference.GetReconciles(ctx, mgr.GetClient(), reference.ReconcileForQuarksSecret, secret, false)
			if err != nil {
				ctxlog.Errorf(ctx, "Failed to calculate reconciles for secret '%s/%s': %v", secret.Namespace, secret.Name, err)
			}

			for _, reconciliation := range reconciles {
				ctxlog.NewMappingEvent(a.Object).Debug(ctx, reconciliation, "QuarksSecret", a.Meta.GetName(), qsv1a1.KubeSecretReference)

				qsec := qsv1a1.QuarksSecret{}
				err := mgr.GetClient().Get(ctx, types.NamespacedName{Name: reconciliation.Name, Namespace: reconciliation.Namespace}, &qsec)
				if err != nil {
					ctxlog.Errorf(ctx, "could not get QuarksSecret '%s': %v", qsec.GetNamespacedName(), err)
				}

				if qsec.Status.Generated == nil {
					qsec.Status = qsv1a1.QuarksSecretStatus{}
				}
				qsec.Status.Generated = pointers.Bool(false)
				qsec.Status.LastReconcile = nil
				err = mgr.GetClient().Status().Update(ctx, &qsec)
				if err != nil {
					ctxlog.Errorf(ctx, "could not update QuarksSecret status '%s': %v", qsec.GetNamespacedName(), err)
				}
			}

			return reconciles
		}),
	}, nsPred, p)
	if err != nil {
		return errors.Wrapf(err, "Watching secrets failed in quarks secret controller.")
	}

	return nil
}

// listSecrets gets all Secrets owned by the QuarksSecret
func listSecrets(ctx context.Context, client crc.Client, qsec *qsv1a1.QuarksSecret) ([]corev1.Secret, error) {
	ctxlog.Debug(ctx, "Listing Secrets owned by QuarksSecret '", qsec.GetNamespacedName(), "'")

	secretLabels := map[string]string{qsv1a1.LabelKind: qsv1a1.GeneratedSecretKind}
	result := []corev1.Secret{}

	allSecrets := &corev1.SecretList{}
	err := client.List(ctx, allSecrets,
		crc.InNamespace(qsec.Namespace),
		crc.MatchingLabels(secretLabels),
	)
	if err != nil {
		return nil, err
	}

	for _, s := range allSecrets.Items {
		secret := s
		if metav1.IsControlledBy(&secret, qsec) {
			result = append(result, secret)
			ctxlog.Debug(ctx, "Found Secret '", secret.Name, "' owned by QuarksSecret '", qsec.GetNamespacedName(), "'")
		}
	}

	if len(result) == 0 {
		ctxlog.Debug(ctx, "Did not find any Secret owned by QuarksSecret '", qsec.GetNamespacedName(), "'")
	}

	return result, nil
}

func isUserCreatedSecret(secret *corev1.Secret) bool {
	secretLabels := secret.GetLabels()
	_, ok := secretLabels[qsv1a1.LabelKind]
	return !ok
}
