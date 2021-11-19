package quarkssecret

import (
	"context"

	"github.com/pkg/errors"
	certv1 "k8s.io/api/certificates/v1"
	certv1client "k8s.io/client-go/kubernetes/typed/certificates/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
)

// AddCertificateSigningRequest creates a new CertificateSigningRequest controller to watch for new and changed
// certificate signing request. Reconciliation will approve them and create a secret.
func AddCertificateSigningRequest(ctx context.Context, config *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContextWithRecorder(ctx, "csr-reconciler", mgr.GetEventRecorderFor("csr-recorder"))
	certClient, err := certv1client.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, "Could not get kube client")
	}
	r := NewCertificateSigningRequestReconciler(ctx, config, mgr, certClient, controllerutil.SetControllerReference)

	// Create a new controller
	c, err := controller.New("certificate-signing-request-controller", mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: config.MaxQuarksSecretWorkers,
	})
	if err != nil {
		return errors.Wrap(err, "Adding certificate signing request controller to manager failed.")
	}

	// Watch for changes to CertificateSigningRequests
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			o := e.Object.(*certv1.CertificateSigningRequest)

			return ownedByQuarksSecret(config.MonitoredID, o.Annotations)
		},
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			o := e.ObjectNew.(*certv1.CertificateSigningRequest)

			return ownedByQuarksSecret(config.MonitoredID, o.Annotations)
		},
	}
	err = c.Watch(&source.Kind{Type: &certv1.CertificateSigningRequest{}}, &handler.EnqueueRequestForObject{}, p)
	if err != nil {
		return errors.Wrapf(err, "Watching certificate signing requests failed in certificate signing request controller.")
	}

	return nil
}

// ownedByQuarksSecret checks if the CSR is owned by a qsec and that the qsec is a namespace monitored by this operator
func ownedByQuarksSecret(monitoredID string, annotations map[string]string) bool {
	if _, ok := annotations[qsv1a1.AnnotationCertSecretName]; ok {
		if _, ok := annotations[qsv1a1.AnnotationQSecNamespace]; ok {
			if id, ok := annotations[qsv1a1.AnnotationMonitoredID]; ok {
				return monitoredID == id
			}
		}
	}
	return false
}
