package integration_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/testing/machine"
)

var _ = Describe("QuarksUserSecret", func() {
	var (
		qsec          qsv1a1.QuarksSecret
		tearDowns     []machine.TearDownFunc
		qsecName      string
		copyNamespace string
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

			By("Creating user password secret")
			tearDown, err := env.CreateSecret(env.Namespace, *passwordSecret)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("Creating quarkssecret")
			qsec = env.DefaultQuarksSecret(qsecName)
			_, tearDown, err = env.CreateQuarksSecret(env.Namespace, qsec)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)
		})

		It("should not generate the password secret", func() {
			By("Checking the quarkssecret status")
			Eventually(func() bool {
				qsec, err := env.GetQuarksSecret(env.Namespace, qsecName)
				Expect(err).NotTo(HaveOccurred())
				if qsec.Status.Generated != nil {
					return *qsec.Status.Generated
				}
				return false
			}).Should(Equal(true))

			By("Checking if it is the user created secret")
			secret, err := env.CollectSecret(env.Namespace, "generated-secret")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(secret.Labels)).To(BeZero())
			Expect(string(secret.Data["password"])).To(Equal("securepassword"))
		})
	})

	When("the user wants copies of the user password secret", func() {
		BeforeEach(func() {
			qsecName = "test.qsec"
			copyNamespace = fmt.Sprintf("%s-%s", env.Namespace, "copy")

			passwordSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "generated-secret",
					Namespace: env.Namespace,
				},
				StringData: map[string]string{
					"password": "securepassword",
				},
			}

			passwordCopySecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "generated-secret-copy",
					Namespace: copyNamespace,
					Labels: map[string]string{
						"quarks.cloudfoundry.org/secret-kind": "generated",
					},
					Annotations: map[string]string{
						"quarks.cloudfoundry.org/secret-copy-of": env.Namespace + "/" + qsecName,
					},
				},
			}

			By("Creating copy namespace")
			tearDown, err := env.CreateNamespace(copyNamespace)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("Creating user password secret")
			tearDown, err = env.CreateSecret(env.Namespace, *passwordSecret)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("Creating copy empty password secret in copy namespace")
			tearDown, err = env.CreateSecret(copyNamespace, *passwordCopySecret)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("Creating quarkssecret with copies")
			qsec = env.DefaultQuarksSecretWithCopy(qsecName, copyNamespace)
			_, tearDown, err = env.CreateQuarksSecret(env.Namespace, qsec)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)
		})

		It("should copy into other namespaces", func() {
			By("Checking the quarkssecret status")
			Eventually(func() bool {
				qsec, err := env.GetQuarksSecret(env.Namespace, qsecName)
				Expect(err).NotTo(HaveOccurred())
				if qsec.Status.Generated != nil {
					return *qsec.Status.Generated
				}
				return false
			}).Should(Equal(true))

			By("Checking the copied secret data")
			secret, err := env.CollectSecret(copyNamespace, "generated-secret-copy")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(secret.Labels)).To(BeZero())
			Expect(string(secret.Data["password"])).To(Equal("securepassword"))
		})
	})

	/*When("the user wants updates the user password secret", func() {
		BeforeEach(func() {
			qsecName = "test.qsec"
			copyNamespace = fmt.Sprintf("%s-%s", env.Namespace, "copy")

			passwordSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "generated-secret",
					Namespace: env.Namespace,
				},
				StringData: map[string]string{
					"password": "securepassword",
				},
			}

			passwordCopySecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "generated-secret-copy",
					Namespace: copyNamespace,
					Labels: map[string]string{
						"quarks.cloudfoundry.org/secret-kind": "generated",
					},
					Annotations: map[string]string{
						"quarks.cloudfoundry.org/secret-copy-of": env.Namespace + "/" + qsecName,
					},
				},
			}

			By("Creating copy namespace")
			tearDown, err := env.CreateNamespace(copyNamespace)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("Creating user password secret")
			tearDown, err = env.CreateSecret(env.Namespace, *passwordSecret)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("Creating copy empty password secret in copy namespace")
			tearDown, err = env.CreateSecret(copyNamespace, *passwordCopySecret)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("Creating quarkssecret with copies")
			qsec = env.DefaultQuarksSecretWithCopy(qsecName, copyNamespace)
			_, tearDown, err = env.CreateQuarksSecret(env.Namespace, qsec)
			Expect(err).NotTo(HaveOccurred())
			tearDowns = append(tearDowns, tearDown)

			By("Updating the user password secret")
			passwordSecret.Data["password"] = []byte("supersecurepassword")
			passwordSecret, tearDown, err = env.UpdateSecret(env.Namespace, *passwordSecret)
		})

		It("should update the copies in other namespaces", func() {
			By("Checking the quarkssecret status")
			Eventually(func() bool {
				qsec, err := env.GetQuarksSecret(env.Namespace, qsecName)
				Expect(err).NotTo(HaveOccurred())
				if qsec.Status.Generated != nil {
					return *qsec.Status.Generated
				}
				return false
			}).Should(Equal(true))

			By("Checking the copied secret data")
			secret, err := env.CollectSecret(copyNamespace, "generated-secret-copy")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(secret.Labels)).To(BeZero())
			Expect(string(secret.Data["password"])).To(Equal("securepassword"))
		})
	})*/
})
