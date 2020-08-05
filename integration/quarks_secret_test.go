package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"code.cloudfoundry.org/quarks-secret/pkg/credsgen"
	inmemorygenerator "code.cloudfoundry.org/quarks-secret/pkg/credsgen/in_memory_generator"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/testing/machine"
)

var _ = Describe("QuarksSecret", func() {
	var (
		secretName string
		qs         qsv1a1.QuarksSecret
		qsName     = "qsec-test"
		tearDowns  []machine.TearDownFunc
	)

	JustBeforeEach(func() {
		By("Creating the quarks secret", func() {
			qs, tearDown, err := env.CreateQuarksSecret(env.Namespace, qs)
			Expect(err).NotTo(HaveOccurred())
			Expect(qs).NotTo(Equal(nil))
			tearDowns = append(tearDowns, tearDown)
		})

	})

	AfterEach(func() {
		Expect(env.TearDownAll(tearDowns)).To(Succeed())
	})

	deleteQuarksSecret := func() {
		By("waiting for the generated secret to disappear, when deleting the quarks secret")
		err := env.DeleteQuarksSecret(env.Namespace, qsName)
		Expect(err).NotTo(HaveOccurred())
		err = env.WaitForSecretDeletion(env.Namespace, secretName)
		Expect(err).NotTo(HaveOccurred(), "dependent secret not deleted")
	}

	When("type is password", func() {
		BeforeEach(func() {
			qs = env.DefaultQuarksSecret(qsName)
			secretName = qs.Spec.SecretName
		})

		It("generates a secret with a password and deletes it when being deleted", func() {
			// check for generated secret
			secret, err := env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data["password"]).To(MatchRegexp("^\\w{64}$"))

			deleteQuarksSecret()
		})
	})

	When("type is rsa", func() {
		BeforeEach(func() {
			qs = env.RSAQuarksSecret(qsName)
			secretName = qs.Spec.SecretName
		})

		It("generates a secret with an rsa key and deletes it when being deleted", func() {
			// check for generated secret
			secret, err := env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data["private_key"]).To(ContainSubstring("RSA PRIVATE KEY"))
			Expect(secret.Data["public_key"]).To(ContainSubstring("PUBLIC KEY"))

			deleteQuarksSecret()
		})
	})

	When("type is ssh", func() {
		BeforeEach(func() {
			qs = env.SSHQuarksSecret(qsName)
			secretName = qs.Spec.SecretName
		})

		It("generates a secret with an ssh key and deletes it when being deleted", func() {
			// check for generated secret
			secret, err := env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data["private_key"]).To(ContainSubstring("RSA PRIVATE KEY"))
			Expect(secret.Data["public_key"]).To(ContainSubstring("ssh-rsa "))
			Expect(secret.Data["public_key_fingerprint"]).To(MatchRegexp("([0-9a-f]{2}:){15}[0-9a-f]{2}"))

			deleteQuarksSecret()
		})
	})

	When("quarks secret is a certificate", func() {
		BeforeEach(func() {
			qs = env.CertificateQuarksSecret(qsName, "mysecret", "ca", "key")
			secretName = qs.Spec.SecretName

			By("creating the CA and storing it in a secret")
			generator := inmemorygenerator.NewInMemoryGenerator(env.Log)
			ca, err := generator.GenerateCertificate("default-ca", credsgen.CertificateGenerationRequest{
				CommonName: "Fake CA",
				IsCA:       true,
			})
			Expect(err).ToNot(HaveOccurred())

			casecret := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysecret",
					Namespace: env.Namespace,
				},
				Data: map[string][]byte{
					"ca":  ca.Certificate,
					"key": ca.PrivateKey,
				},
			}
			tearDown, err := env.CreateSecret(env.Namespace, casecret)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)
		})

		It("creates the certificate", func() {
			By("checking for the generated secret")
			secret, err := env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data["certificate"]).To(ContainSubstring("BEGIN CERTIFICATE"))
			Expect(secret.Data["private_key"]).To(ContainSubstring("RSA PRIVATE KEY"))

			deleteQuarksSecret()
		})
	})

	When("certificate signer type is cluster", func() {
		BeforeEach(func() {
			qs = env.CertificateQuarksSecret(qsName, "mysecret", "ca", "key")
			qs.Spec.Request.CertificateRequest.SignerType = qsv1a1.ClusterSigner
			secretName = qs.Spec.SecretName
		})

		It("does not use the old csr", func() {
			By("waiting for the generated secret")
			oldSecret, err := env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())

			deleteQuarksSecret()

			By("creating an new cluster ca qsec")
			qs.Spec.Request.CertificateRequest.AlternativeNames = []string{"qux.com", "example.org"}
			_, _, err = env.CreateQuarksSecret(env.Namespace, qs)
			Expect(err).NotTo(HaveOccurred())

			By("checking for a working generated secret")
			secret, err := env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(oldSecret.Data["ca"]).To(Equal(secret.Data["ca"]))
			Expect(oldSecret.Data["private_key"]).NotTo(Equal(secret.Data["private_key"]))
			Expect(oldSecret.Data["certificate"]).NotTo(Equal(secret.Data["certificate"]))
		})
	})

	When("type is basic-auth", func() {
		BeforeEach(func() {
			qs = env.BasicAuthQuarksSecret(qsName)
			secretName = qs.Spec.SecretName
		})

		It("generates a basic auth secret with a username and password and deletes it when being deleted", func() {
			// check for generated secret
			secret, err := env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data["password"]).To(MatchRegexp("^\\w{64}$"))
			Expect(string(secret.Data["username"])).To(Equal("some-passed-in-username"))

			deleteQuarksSecret()
		})
	})

})
