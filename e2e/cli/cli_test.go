package cli_test

import (
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("CLI", func() {
	act := func(arg ...string) (session *gexec.Session, err error) {
		cmd := exec.Command(cliPath, arg...)
		session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		return
	}

	BeforeEach(func() {
		os.Setenv("DOCKER_IMAGE_TAG", "v0.0.0")
	})

	Describe("help", func() {
		It("should show the help for server", func() {
			session, err := act("help")
			Expect(err).ToNot(HaveOccurred())
			Eventually(session.Out).Should(Say(`Usage:`))
		})

		It("shows all available commands", func() {
			session, err := act("help")
			Expect(err).ToNot(HaveOccurred())
			Eventually(session.Out).Should(Say(`Available Commands:
  help        Help about any command
  version     Print the version number

`))
		})
	})

	Describe("default", func() {

		It("should start the server", func() {
			session, err := act()
			Expect(err).ToNot(HaveOccurred())
			Eventually(session.Err).Should(Say(`Starting quarks-secret \d+\.\d+\.\d+, monitoring namespaces labeled with`))
			Eventually(session.Err).ShouldNot(Say(`Applying CRDs...`))
		})

		Context("when specifying monitored id for namespaces to monitor", func() {
			Context("via environment variables", func() {
				BeforeEach(func() {
					os.Setenv("MONITORED_ID", "env-test")
				})

				AfterEach(func() {
					os.Setenv("MONITORED_ID", "")
				})

				It("should start for that id", func() {
					session, err := act()
					Expect(err).ToNot(HaveOccurred())
					Eventually(session.Err).Should(Say(`Starting quarks-secret \d+\.\d+\.\d+, monitoring namespaces labeled with 'env-test'`))
				})
			})

			Context("via using switches", func() {
				It("should start for namespace", func() {
					session, err := act("--monitored-id", "switch-test")
					Expect(err).ToNot(HaveOccurred())
					Eventually(session.Err).Should(Say(`Starting quarks-secret \d+\.\d+\.\d+, monitoring namespaces labeled with 'switch-test'`))
				})
			})
		})

		Context("when enabling apply-crd", func() {
			Context("via environment variables", func() {
				BeforeEach(func() {
					os.Setenv("APPLY_CRD", "true")
				})

				AfterEach(func() {
					os.Setenv("APPLY_CRD", "")
				})

				It("should apply CRDs", func() {
					session, err := act()
					Expect(err).ToNot(HaveOccurred())
					Eventually(session.Err).Should(Say(`Applying CRDs...`))
				})
			})

			Context("via using switches", func() {
				It("should apply CRDs", func() {
					session, err := act("--apply-crd")
					Expect(err).ToNot(HaveOccurred())
					Eventually(session.Err).Should(Say(`Applying CRDs...`))
				})
			})
		})
	})

	Describe("version", func() {
		It("should show a semantic version number", func() {
			session, err := act("version")
			Expect(err).ToNot(HaveOccurred())
			Eventually(session.Out).Should(Say(`Quarks-Secret Version: \d+.\d+.\d+`))
		})
	})
})
