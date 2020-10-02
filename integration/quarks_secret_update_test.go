package integration_test

import (
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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

	Context("QuarksSecret Updates", func() {
		BeforeEach(func() {
			qs = env.DefaultQuarksSecretWithLabels(qsName)
			secretName = qs.Spec.SecretName
		})

		It("does not rotate/regenerate secret, when quarks secret's `SecretLabels` key is changed", func() {
			By("Check for the generated secret")
			secret, err := env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data["password"]).To(MatchRegexp("^\\w{64}$"))
			Expect(secret.Labels).To(Equal(map[string]string{
				"LabelKey":                            "LabelValue",
				"quarks.cloudfoundry.org/secret-kind": "generated",
			}))
			oldSecretData := secret.Data["password"]

			By("Add labels to `SecretLabels` spec of quarkssecret")
			qs, err := env.GetQuarksSecret(env.Namespace, qsName)
			Expect(err).NotTo(HaveOccurred())
			qs.Spec.SecretLabels["LabelKeyv2"] = "LabelValuev2"
			qs, _, err = env.UpdateQuarksSecret(env.Namespace, *qs)
			Expect(err).NotTo(HaveOccurred())
			Expect(qs).NotTo(Equal(nil))

			By("Check for the updated secret")
			secret, err = env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				secret, err = env.CollectSecret(env.Namespace, secretName)
				Expect(err).NotTo(HaveOccurred())
				return reflect.DeepEqual(secret.Labels, map[string]string{
					"LabelKey":                            "LabelValue",
					"LabelKeyv2":                          "LabelValuev2",
					"quarks.cloudfoundry.org/secret-kind": "generated",
				})
			}, 10*time.Second).Should(Equal(true))
			secret, err = env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data["password"]).To(MatchRegexp("^\\w{64}$"))
			Expect(oldSecretData).To(Equal(secret.Data["password"]))

			By("Remove labels to `SecretLabels` spec of quarkssecret")
			qs, err = env.GetQuarksSecret(env.Namespace, qsName)
			Expect(err).NotTo(HaveOccurred())
			qs.Spec.SecretLabels = nil
			qs, _, err = env.UpdateQuarksSecret(env.Namespace, *qs)
			Expect(err).NotTo(HaveOccurred())
			Expect(qs).NotTo(Equal(nil))

			By("Check for the updated secret")
			Eventually(func() bool {
				secret, err = env.CollectSecret(env.Namespace, secretName)
				Expect(err).NotTo(HaveOccurred())
				return reflect.DeepEqual(secret.Labels, map[string]string{
					"quarks.cloudfoundry.org/secret-kind": "generated",
				})
			}, 10*time.Second).Should(Equal(true))
			secret, err = env.CollectSecret(env.Namespace, secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data["password"]).To(MatchRegexp("^\\w{64}$"))
			Expect(oldSecretData).To(Equal(secret.Data["password"]))
		})
	})
})
