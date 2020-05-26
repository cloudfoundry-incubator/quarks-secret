package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-secret/pkg/kube/controllers/quarkssecret"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
)

// Theses funcs construct controllers and add them to the controller-runtime
// manager. The manager will set fields on the controllers and start them, when
// itself is started.
var addToManagerFuncs = []func(context.Context, *config.Config, manager.Manager) error{
	quarkssecret.AddCertificateSigningRequest,
	quarkssecret.AddQuarksSecret,
	quarkssecret.AddSecretRotation,
}

var addToSchemes = runtime.SchemeBuilder{
	qsv1a1.AddToScheme,
}

// AddToManager adds all Controllers to the Manager
func AddToManager(ctx context.Context, config *config.Config, m manager.Manager) error {
	for _, f := range addToManagerFuncs {
		if err := f(ctx, config, m); err != nil {
			return err
		}
	}
	return nil
}

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return addToSchemes.AddToScheme(s)
}
