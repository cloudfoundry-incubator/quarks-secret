package environment

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	gomegaConfig "github.com/onsi/ginkgo/config"
	"github.com/spf13/afero"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-secret/pkg/kube/client/clientset/versioned"
	"code.cloudfoundry.org/quarks-secret/pkg/kube/operator"
	"code.cloudfoundry.org/quarks-secret/testing"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/names"
	utils "code.cloudfoundry.org/quarks-utils/testing/integration"
	"code.cloudfoundry.org/quarks-utils/testing/machine"
)

// Environment test env with helpers to create structs and k8s resources
type Environment struct {
	*utils.Environment
	Machine
	testing.Catalog
}

var (
	namespaceCounter int32
)

// NewEnvironment returns a new test environment
func NewEnvironment(kubeConfig *rest.Config) *Environment {
	atomic.AddInt32(&namespaceCounter, 1)
	namespaceID := gomegaConfig.GinkgoConfig.ParallelNode*100 + int(namespaceCounter)
	ns, _ := names.JobName(utils.GetNamespaceName(namespaceID))

	shared := &config.Config{
		CtxTimeOut:           10 * time.Second,
		MeltdownDuration:     1 * time.Second,
		MeltdownRequeueAfter: 500 * time.Millisecond,
		Fs:                   afero.NewOsFs(),
		MonitoredID:          ns,
	}
	return &Environment{
		Environment: &utils.Environment{
			ID:         namespaceID,
			Namespace:  ns,
			KubeConfig: kubeConfig,
			Config:     shared,
		},
		Machine: Machine{
			Machine: machine.NewMachine(),
		},
	}
}

// SetupClientsets initializes kube clientsets
func (e *Environment) SetupClientsets() error {
	var err error
	e.Clientset, err = kubernetes.NewForConfig(e.KubeConfig)
	if err != nil {
		return err
	}

	e.VersionedClientset, err = versioned.NewForConfig(e.KubeConfig)
	if err != nil {
		return err
	}

	return nil
}

// NamespaceDeletionInProgress returns true if the error indicates deletion will happen
// eventually
func (e *Environment) NamespaceDeletionInProgress(err error) bool {
	return strings.Contains(err.Error(), "namespace will automatically be purged")
}

// SetupNamespace creates the namespace and the clientsets and prepares the teardowm
func (e *Environment) SetupNamespace() error {
	return utils.SetupNamespace(e.Environment, e.Machine.Machine,
		map[string]string{
			qsv1a1.LabelNamespace: e.Namespace,
		})
}

// ApplyCRDs applies the CRDs to the cluster
func ApplyCRDs(kubeConfig *rest.Config) error {
	err := operator.ApplyCRDs(context.Background(), kubeConfig)
	if err != nil {
		return err
	}
	return nil
}
