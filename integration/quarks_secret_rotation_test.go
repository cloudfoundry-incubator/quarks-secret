package integration_test

import (
	"bytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/testing/machine"
)

var _ = Describe("QuarksSecretRotation", func() {
	var (
		qsec      qsv1a1.QuarksSecret
		oldSecret *corev1.Secret
		tearDowns []machine.TearDownFunc
	)

	const (
		qsecName = "test.qsec"
	)

	notGenerated := func(status qsv1a1.QuarksSecretStatus) bool {
		return status.Generated == nil || (status.Generated != nil && !*status.Generated)
	}

	JustBeforeEach(func() {
		By("Creating the quarks secret", func() {
			_, tearDown, err := env.CreateQuarksSecret(env.Namespace, qsec)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			oldSecret, err = env.CollectSecret(env.Namespace, qsec.Spec.SecretName)
			Expect(err).NotTo(HaveOccurred())
		})

		By("Rotating the secret", func() {
			rotationConfig := env.RotationConfig(qsecName)
			tearDown, err := env.CreateConfigMap(env.Namespace, rotationConfig)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			err = env.WaitForConfigMap(env.Namespace, "rotation-config1")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	AfterEach(func() {
		Expect(env.TearDownAll(tearDowns)).To(Succeed())
	})

	When("rotating a password", func() {
		var oldPassword []byte

		BeforeEach(func() {
			qsec = env.DefaultQuarksSecret(qsecName)
		})

		It("modifies quarks secret and a a new password is generated", func() {
			err := env.WaitForQuarksSecretChange(env.Namespace, qsecName, func(qs qsv1a1.QuarksSecret) bool {
				return notGenerated(qs.Status)
			})
			Expect(err).NotTo(HaveOccurred())

			oldPassword = oldSecret.Data["password"]
			err = env.WaitForSecretChange(env.Namespace, qsec.Spec.SecretName, func(s corev1.Secret) bool {
				return !bytes.Equal(oldPassword, s.Data["password"])
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	When("rotating a certificate", func() {
		BeforeEach(func() {
			qsec = env.CertificateQuarksSecret(qsecName, "mysecret", "ca", "key")
			qsec.Spec.Request.CertificateRequest.SignerType = qsv1a1.ClusterSigner
		})

		It("modifies quarks secret and updates certificate and key", func() {
			err := env.WaitForQuarksSecretChange(env.Namespace, qsecName, func(qs qsv1a1.QuarksSecret) bool {
				return notGenerated(qs.Status)
			})
			Expect(err).NotTo(HaveOccurred())

			err = env.WaitForSecretChange(env.Namespace, qsec.Spec.SecretName, func(s corev1.Secret) bool {
				return !bytes.Equal(oldSecret.Data["certificate"], s.Data["certificate"]) &&
					!bytes.Equal(oldSecret.Data["private_key"], s.Data["private_key"])
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	When("rotating a ssh secret", func() {
		BeforeEach(func() {
			qsec = env.SSHQuarksSecret(qsecName)
		})

		It("modifies quarks secret and the secret is updated", func() {
			err := env.WaitForQuarksSecretChange(env.Namespace, qsecName, func(qs qsv1a1.QuarksSecret) bool {
				return notGenerated(qs.Status)
			})
			Expect(err).NotTo(HaveOccurred())

			err = env.WaitForSecretChange(env.Namespace, qsec.Spec.SecretName, func(s corev1.Secret) bool {
				return !bytes.Equal(oldSecret.Data["private_key"], s.Data["private_key"]) &&
					!bytes.Equal(oldSecret.Data["public_key_fingerprint"], s.Data["public_key_fingerprint"])
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	When("rotating a rsa secret", func() {
		BeforeEach(func() {
			qsec = env.RSAQuarksSecret(qsecName)
		})

		It("modifies quarks secret and the secret is updated", func() {
			err := env.WaitForQuarksSecretChange(env.Namespace, qsecName, func(qs qsv1a1.QuarksSecret) bool {
				return notGenerated(qs.Status)
			})
			Expect(err).NotTo(HaveOccurred())

			err = env.WaitForSecretChange(env.Namespace, qsec.Spec.SecretName, func(s corev1.Secret) bool {
				return !bytes.Equal(oldSecret.Data["private_key"], s.Data["private_key"]) &&
					!bytes.Equal(oldSecret.Data["public_key"], s.Data["public_key"])
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	When("rotating a basic-auth secret", func() {
		BeforeEach(func() {
			qsec = env.BasicAuthQuarksSecret(qsecName)
		})

		It("modifies quarks secret and the secret is updated", func() {
			err := env.WaitForQuarksSecretChange(env.Namespace, qsecName, func(qs qsv1a1.QuarksSecret) bool {
				return notGenerated(qs.Status)
			})
			Expect(err).NotTo(HaveOccurred())

			err = env.WaitForSecretChange(env.Namespace, qsec.Spec.SecretName, func(s corev1.Secret) bool {
				return !bytes.Equal(oldSecret.Data["password"], s.Data["password"]) &&
					bytes.Equal(oldSecret.Data["username"], s.Data["username"])
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
