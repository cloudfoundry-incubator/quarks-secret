package integration_test

import (
	"reflect"
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

	When("updating the qsec", func() {
		var oldSecret *corev1.Secret

		BeforeEach(func() {
			qs = env.DefaultQuarksSecretWithLabels(qsName)
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

		When("updating `SecretLabels` key", func() {
			JustBeforeEach(func() {
				var err error
				qs.Spec.SecretLabels["LabelKeyv2"] = "LabelValuev2"
				qs, err = env.UpdateQuarksSecret(env.Namespace, qs)
				Expect(err).NotTo(HaveOccurred())
			})

			It("does not rotate/regenerate secret, when quarks secret's `SecretLabels` key is changed", func() {
				By("Check for the updated secret")
				secret, err := env.CollectSecret(env.Namespace, secretName)
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
				Expect(oldSecret.Data["password"]).To(Equal(secret.Data["password"]))

				By("Remove labels to `SecretLabels` spec of quarkssecret")
				qs, err = env.GetQuarksSecret(env.Namespace, qsName)
				Expect(err).NotTo(HaveOccurred())
				qs.Spec.SecretLabels = nil
				qs, err = env.UpdateQuarksSecret(env.Namespace, qs)
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
				Expect(oldSecret.Data["password"]).To(Equal(secret.Data["password"]))
			})
		})
	})
})
