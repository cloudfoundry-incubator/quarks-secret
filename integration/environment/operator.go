package environment

import (
	"os"

	"github.com/pkg/errors"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" //from https://github.com/kubernetes/client-go/issues/345
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"code.cloudfoundry.org/quarks-secret/pkg/kube/operator"
	sharedcfg "code.cloudfoundry.org/quarks-utils/pkg/config"
)

// StartOperator starts the quarks secret operator
func (e *Environment) StartOperator() (chan struct{}, error) {
	mgr, err := e.setupOperator()
	if err != nil {
		return nil, err
	}
	stop := make(chan struct{})
	go func() {
		err := mgr.Start(stop)
		if err != nil {
			panic(err)
		}
	}()
	return stop, err
}

func (e *Environment) setupOperator() (manager.Manager, error) {
	ctx := e.SetupLoggerContext("quarks-secret-tests")

	dockerImageOrg, found := os.LookupEnv("DOCKER_IMAGE_ORG")
	if !found {
		dockerImageOrg = "cfcontainerization"
	}

	dockerImageRepo, found := os.LookupEnv("DOCKER_IMAGE_REPOSITORY")
	if !found {
		dockerImageRepo = "quarks-secret"
	}

	dockerImageTag, found := os.LookupEnv("DOCKER_IMAGE_TAG")
	if !found {
		return nil, errors.Errorf("required environment variable DOCKER_IMAGE_TAG not set")
	}

	err := sharedcfg.SetupOperatorDockerImage(dockerImageOrg, dockerImageRepo, dockerImageTag)
	if err != nil {
		return nil, err
	}

	mgr, err := operator.NewManager(ctx, e.Config, e.KubeConfig, manager.Options{
		MetricsBindAddress: "0",
		LeaderElection:     false,
		Host:               "0.0.0.0",
	})

	return mgr, err
}
