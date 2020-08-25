package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/testing/machine"
)

var _ = Describe("QuarksUserSecret", func() {
	var (
		qsec      qsv1a1.QuarksSecret
		tearDowns []machine.TearDownFunc
		qsecName  string
	)

	AfterEach(func() {
		Expect(env.TearDownAll(tearDowns)).To(Succeed())
	})

	When("the user provides the password secret", func() {
		BeforeEach(func() {
			qsecName = "test.qsec"

			passwordSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "generated-secret",
					Namespace: env.Namespace,
				},
				StringData: map[string]string{
					"password": "securepassword",
				},
			}

			tearDown, err := env.CreateSecret(env.Namespace, *passwordSecret)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			qsec = env.DefaultQuarksSecret(qsecName)
			_, tearDown, err = env.CreateQuarksSecret(env.Namespace, qsec)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)
		})

		It("should not generate the password secret", func() {
			Eventually(func() bool {
				qsec, err := env.GetQuarksSecret(env.Namespace, qsecName)
				Expect(err).NotTo(HaveOccurred())
				if qsec.Status.Generated != nil {
					return *qsec.Status.Generated
				}
				return false
			}).Should(Equal(true))

			secret, err := env.CollectSecret(env.Namespace, "generated-secret")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(secret.Labels)).To(BeZero())
		})
	})
})
