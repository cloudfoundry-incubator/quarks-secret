package integration_test

import (
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

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
			qsec, tearDown, err := env.CreateQuarksSecret(env.Namespace, qs)
			qs = *qsec
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
			qs = env.CertificateQuarksSecret(qsName, "my-ca", "ca", "key")
			secretName = qs.Spec.SecretName

			By("creating the CA and storing it in a secret")
			tearDown, err := env.CreateCASecret(env.Log, env.Namespace, "my-ca")
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

	When("quarks secret type is tls", func() {
		BeforeEach(func() {
			qs = env.TLSQuarksSecret(qsName, "my-ca", "ca", "key")
			secretName = qs.Spec.SecretName

			By("creating the CA and storing it in a secret")
			tearDown, err := env.CreateCASecret(env.Log, env.Namespace, "my-ca")
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)
		})

		It("creates the tls type", func() {
			By("checking for the generated secret")
			secret, err := env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data["tls.crt"]).To(ContainSubstring("BEGIN CERTIFICATE"))
			Expect(secret.Data["tls.key"]).To(ContainSubstring("RSA PRIVATE KEY"))
			Expect(secret.Type).To(Equal(corev1.SecretTypeTLS))

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
			qsnew := env.CertificateQuarksSecret(qsName, "mysecret", "ca", "key")
			qsnew.Spec.Request.CertificateRequest.SignerType = qsv1a1.ClusterSigner
			qsnew.Spec.Request.CertificateRequest.AlternativeNames = []string{"qux.com", "example.org"}
			_, _, err = env.CreateQuarksSecret(env.Namespace, qsnew)
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

	When("updating the qsec", func() {
		var oldSecret *corev1.Secret

		BeforeEach(func() {
			qs = env.DefaultQuarksSecret(qsName)
			secretName = qs.Spec.SecretName
		})

		JustBeforeEach(func() {
			var err error
			oldSecret, err = env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())

			By("Getting the current version")
			qs, err = env.GetQuarksSecret(env.Namespace, qs.Name)
			Expect(err).NotTo(HaveOccurred())
		})

		incResourceVersion := func(v string) string {
			n, err := strconv.Atoi(v)
			Expect(err).NotTo(HaveOccurred())
			n++
			return strconv.Itoa(n)
		}

		When("updating the spec", func() {
			JustBeforeEach(func() {
				var err error
				By("Updating the quarks secret")
				qs.Spec.SecretLabels = map[string]string{"sec": "label"}
				qs, err = env.UpdateQuarksSecret(env.Namespace, qs)
				Expect(err).NotTo(HaveOccurred())

				old := incResourceVersion(qs.GetResourceVersion())
				err = env.WaitForQuarksSecretChange(env.Namespace, qs.Name, 10*time.Second, func(qsec qsv1a1.QuarksSecret) bool {
					return old < qsec.GetResourceVersion()
				})
				Expect(err).NotTo(HaveOccurred())

			})

			It("updates the secret", func() {
				secret, err := env.CollectSecret(env.Namespace, secretName)
				Expect(err).NotTo(HaveOccurred())

				Expect(secret.Data["password"]).NotTo(Equal(oldSecret.Data["password"]))
				qs, err = env.GetQuarksSecret(env.Namespace, qs.Name)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("updating a label", func() {
			JustBeforeEach(func() {
				var err error
				qs.Labels = map[string]string{"qsec": "label"}
				qs, err = env.UpdateQuarksSecret(env.Namespace, qs)
				Expect(err).NotTo(HaveOccurred())
			})

			It("does not update the secret", func() {
				old := incResourceVersion(qs.GetResourceVersion())
				err := env.WaitForQuarksSecretChange(env.Namespace, qs.Name, 10*time.Second, func(qsec qsv1a1.QuarksSecret) bool {
					return old < qsec.GetResourceVersion()
				})
				Expect(err).To(HaveOccurred(), "check for update should fail")
			})
		})
	})
})
