package quarkssecret

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	credsgen "code.cloudfoundry.org/quarks-secret/pkg/credsgen/in_memory_generator"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
)

// AddQuarksSecretSecretMeta creates a new QuarksSecrets controller to watch for the
// custom resource changes for `SecretLabels` and `SecretAnnotations`.
func AddQuarksSecretSecretMeta(ctx context.Context, config *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContextWithRecorder(ctx, "quarkssecret-secretmeta-reconciler", mgr.GetEventRecorderFor("quarkssecret-secretmeta-recorder"))
	log := ctxlog.ExtractLogger(ctx)
	r := NewQuarksSecretSecretMetaReconciler(ctx, config, mgr, credsgen.NewInMemoryGenerator(log))

	// Create a new controller
	c, err := controller.New("quarkssecret-secretmeta-controller", mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: config.MaxQuarksSecretWorkers,
	})
	if err != nil {
		return errors.Wrap(err, "Adding secret metadata controller to manager failed.")
	}

	nsPred := newNSPredicate(ctx, mgr.GetClient(), config.MonitoredID)

	// Watch for changes to QuarksSecret's `SecretLabels` and `SecretAnnotations` fields.
	p := predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return false },
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			n := e.ObjectNew.(*qsv1a1.QuarksSecret)
			o := e.ObjectOld.(*qsv1a1.QuarksSecret)

			if !reflect.DeepEqual(o.Spec.SecretLabels, n.Spec.SecretLabels) || !reflect.DeepEqual(o.Spec.SecretAnnotations, n.Spec.SecretAnnotations) {
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
		return errors.Wrapf(err, "Watching quarks secrets failed in quarkssecret-secretmeta controller.")
	}

	return nil
}
