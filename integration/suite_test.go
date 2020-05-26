package integration_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/rest"

	"code.cloudfoundry.org/quarks-secret/integration/environment"
	cmdHelper "code.cloudfoundry.org/quarks-utils/testing"
	utils "code.cloudfoundry.org/quarks-utils/testing/integration"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var (
	env              *environment.Environment
	namespacesToNuke []string
	kubeConfig       *rest.Config
)

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error
	kubeConfig, err = utils.KubeConfig()
	if err != nil {
		fmt.Printf("WARNING: failed to get kube config")
	}

	// Ginkgo node 1 gets to setup the CRDs
	err = environment.ApplyCRDs(kubeConfig)
	if err != nil {
		fmt.Printf("WARNING: failed to apply CRDs: %v\n", err)
	}

	return []byte{}
}, func([]byte) {
	var err error
	kubeConfig, err = utils.KubeConfig()
	if err != nil {
		fmt.Printf("WARNING: failed to get kube config: %v\n", err)
	}
})

var _ = BeforeEach(func() {
	env = environment.NewEnvironment(kubeConfig)

	err := env.SetupClientsets()
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}

	err = env.SetupNamespace()
	if err != nil {
		fmt.Printf("WARNING: failed to setup namespace %s: %v\n", env.Namespace, err)
	}
	namespacesToNuke = append(namespacesToNuke, env.Namespace)

	env.Stop, err = env.StartOperator()
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}
})

var _ = AfterEach(func() {
	env.Teardown(CurrentGinkgoTestDescription().Failed)
})

var _ = AfterSuite(func() {
	// Nuking all namespaces at the end of the run
	for _, namespace := range namespacesToNuke {
		err := cmdHelper.DeleteNamespace(namespace)
		if err != nil && !env.NamespaceDeletionInProgress(err) {
			fmt.Printf("WARNING: failed to delete namespace %s: %v\n", namespace, err)
		}
	}
})
