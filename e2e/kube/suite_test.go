package kube_test

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/quarks-utils/testing/e2ehelper"
)

const examplesDir = "../../docs/examples/"

var (
	nsIndex           int
	teardowns         []e2ehelper.TearDownFunc
	namespace         string
	operatorNamespace string
)

func FailAndCollectDebugInfo(description string, callerSkip ...int) {
	Fail(description, callerSkip...)
}

func TestE2EKube(t *testing.T) {
	nsIndex = 0

	RegisterFailHandler(FailAndCollectDebugInfo)
	RunSpecs(t, "E2E Kube Suite")
}

var _ = BeforeEach(func() {
	var err error
	var teardown e2ehelper.TearDownFunc

	dir, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())

	chartPath := fmt.Sprintf("%s%s", dir, "/../../deploy/helm/quarks-secret")

	namespace, operatorNamespace, teardown, err = e2ehelper.CreateNamespace()
	Expect(err).ToNot(HaveOccurred())
	teardowns = append(teardowns, teardown)

	teardown, err = e2ehelper.CreateMonitoredNamespace(namespace, operatorNamespace)
	Expect(err).ToNot(HaveOccurred())
	teardowns = append(teardowns, teardown)

	teardown, err = e2ehelper.InstallChart(chartPath, operatorNamespace,
		"--set", fmt.Sprintf("global.monitoredID=%s", operatorNamespace),
	)
	Expect(err).ToNot(HaveOccurred())
	// prepend helm clean up
	teardowns = append([]e2ehelper.TearDownFunc{teardown}, teardowns...)
})

var _ = AfterEach(func() {
	err := e2ehelper.TearDownAll(teardowns)
	if err != nil {
		fmt.Printf("Failures while cleaning up test environment:\n %v", err)
	}
	teardowns = []e2ehelper.TearDownFunc{}
})

var _ = AfterSuite(func() {
	err := e2ehelper.TearDownAll(teardowns)
	if err != nil {
		fmt.Printf("Failures while cleaning up test environment:\n %v", err)
	}
})
