package quarkssecret_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"code.cloudfoundry.org/quarks-secret/pkg/credsgen"
	generatorfakes "code.cloudfoundry.org/quarks-secret/pkg/credsgen/fakes"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-secret/pkg/kube/client/clientset/versioned/scheme"
	"code.cloudfoundry.org/quarks-secret/pkg/kube/controllers"
	cfakes "code.cloudfoundry.org/quarks-secret/pkg/kube/controllers/fakes"
	qscontroller "code.cloudfoundry.org/quarks-secret/pkg/kube/controllers/quarkssecret"
	cfcfg "code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"code.cloudfoundry.org/quarks-utils/pkg/pointers"
	helper "code.cloudfoundry.org/quarks-utils/testing/testhelper"
)

var _ = Describe("ReconcileQuarksSecret", func() {
	var (
		manager          *cfakes.FakeManager
		reconciler       reconcile.Reconciler
		request          reconcile.Request
		ctx              context.Context
		log              *zap.SugaredLogger
		config           *cfcfg.Config
		client           *cfakes.FakeClient
		generator        *generatorfakes.FakeGenerator
		qSecret          *qsv1a1.QuarksSecret
		setReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error = func(owner, object metav1.Object, scheme *runtime.Scheme) error { return nil }
	)

	BeforeEach(func() {
		err := controllers.AddToScheme(scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())
		manager = &cfakes.FakeManager{}
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}
		config = &cfcfg.Config{CtxTimeOut: 10 * time.Second}
		_, log = helper.NewTestLogger()
		ctx = ctxlog.NewParentContext(log)
		qSecret = &qsv1a1.QuarksSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: qsv1a1.QuarksSecretSpec{
				Type:       "password",
				SecretName: "generated-secret",
			},
		}
		generator = &generatorfakes.FakeGenerator{}
		client = &cfakes.FakeClient{}
		client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
			switch object := object.(type) {
			case *qsv1a1.QuarksSecret:
				qSecret.DeepCopyInto(object)
			case *corev1.Secret:
				return errors.NewNotFound(schema.GroupResource{}, "not found")
			}
			return nil
		})
		client.StatusCalls(func() crc.StatusWriter { return &cfakes.FakeStatusWriter{} })
		manager.GetClientReturns(client)
	})

	JustBeforeEach(func() {
		reconciler = qscontroller.NewQuarksSecretReconciler(ctx, config, manager, generator, setReferenceFunc)
	})

	Context("if the resource can not be resolved", func() {
		It("skips if the resource was not found", func() {
			client.GetReturns(errors.NewNotFound(schema.GroupResource{}, "not found is requeued"))

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(reconcile.Result{}).To(Equal(result))
		})
	})

	Context("if the resource is invalid", func() {
		It("returns an error", func() {
			qSecret.Spec.Type = "foo"

			result, err := reconciler.Reconcile(request)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid type"))
			Expect(reconcile.Result{}).To(Equal(result))
		})
	})

	Context("when generating passwords", func() {
		BeforeEach(func() {
			generator.GeneratePasswordReturns("securepassword")
		})

		It("skips reconciling if the secret exists", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "generated-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"password": []byte("foo"),
				},
			}

			client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
				switch object := object.(type) {
				case *qsv1a1.QuarksSecret:
					qSecret.DeepCopyInto(object)
				case *corev1.Secret:
					if nn.Name == "generated-secret" {
						secret.DeepCopyInto(object)
					}
				}
				return nil
			})

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.CreateCallCount()).To(Equal(0))
			Expect(reconcile.Result{}).To(Equal(result))
		})

		It("generates passwords", func() {
			client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
				secret := object.(*corev1.Secret)
				Expect(secret.StringData["password"]).To(Equal("securepassword"))
				Expect(secret.GetName()).To(Equal("generated-secret"))
				Expect(secret.GetLabels()).To(HaveKeyWithValue(qsv1a1.LabelKind, qsv1a1.GeneratedSecretKind))
				return nil
			})

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.CreateCallCount()).To(Equal(1))
			Expect(reconcile.Result{}).To(Equal(result))
		})
	})

	Context("when generating RSA keys", func() {
		BeforeEach(func() {
			qSecret.Spec.Type = "rsa"

			generator.GenerateRSAKeyReturns(credsgen.RSAKey{PrivateKey: []byte("private"), PublicKey: []byte("public")}, nil)
		})

		It("generates RSA keys", func() {
			client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
				secret := object.(*corev1.Secret)
				Expect(secret.StringData["private_key"]).To(Equal("private"))
				Expect(secret.StringData["public_key"]).To(Equal("public"))
				Expect(secret.GetName()).To(Equal("generated-secret"))
				Expect(secret.GetLabels()).To(HaveKeyWithValue(qsv1a1.LabelKind, qsv1a1.GeneratedSecretKind))
				return nil
			})

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.CreateCallCount()).To(Equal(1))
			Expect(reconcile.Result{}).To(Equal(result))
		})
	})

	Context("when generating SSH keys", func() {
		BeforeEach(func() {
			qSecret.Spec.Type = "ssh"

			generator.GenerateSSHKeyReturns(credsgen.SSHKey{
				PrivateKey:  []byte("private"),
				PublicKey:   []byte("public"),
				Fingerprint: "fingerprint",
			}, nil)
		})

		It("generates SSH keys", func() {
			client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
				secret := object.(*corev1.Secret)
				Expect(secret.StringData["private_key"]).To(Equal("private"))
				Expect(secret.StringData["public_key"]).To(Equal("public"))
				Expect(secret.StringData["public_key_fingerprint"]).To(Equal("fingerprint"))
				Expect(secret.GetName()).To(Equal("generated-secret"))
				Expect(secret.GetLabels()).To(HaveKeyWithValue(qsv1a1.LabelKind, qsv1a1.GeneratedSecretKind))
				return nil
			})

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.CreateCallCount()).To(Equal(1))
			Expect(reconcile.Result{}).To(Equal(result))
		})
	})

	Context("when generating certificates", func() {
		BeforeEach(func() {
			qSecret.Spec.Type = "certificate"
			qSecret.Spec.Request.CertificateRequest.IsCA = false
			qSecret.Spec.Request.CertificateRequest.CARef = qsv1a1.SecretReference{Name: "mysecret", Key: "ca"}
			qSecret.Spec.Request.CertificateRequest.CAKeyRef = qsv1a1.SecretReference{Name: "mysecret", Key: "key"}
			qSecret.Spec.Request.CertificateRequest.CommonName = "foo.com"
			qSecret.Spec.Request.CertificateRequest.AlternativeNames = []string{"bar.com", "baz.com"}
		})

		Context("if the CA is not ready", func() {
			It("requeues generation", func() {
				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.CreateCallCount()).To(Equal(0))
				Expect(reconcile.Result{RequeueAfter: time.Second * 5}).To(Equal(result))
			})
		})

		Context("if the CA is ready", func() {
			BeforeEach(func() {
				ca := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mysecret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"ca":  []byte("theca"),
						"key": []byte("the_private_key"),
					},
				}

				client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
					switch object := object.(type) {
					case *qsv1a1.QuarksSecret:
						qSecret.DeepCopyInto(object)
					case *corev1.Secret:
						if nn.Name == "mysecret" {
							ca.DeepCopyInto(object)
						} else {
							return errors.NewNotFound(schema.GroupResource{}, "not found is requeued")
						}
					}
					return nil
				})
			})

			Context("and the generated secret is not a ca", func() {
				It("triggers generation of a secret", func() {
					generator.GenerateCertificateCalls(func(name string, request credsgen.CertificateGenerationRequest) (credsgen.Certificate, error) {
						Expect(request.CA.Certificate).To(Equal([]byte("theca")))
						Expect(request.CA.PrivateKey).To(Equal([]byte("the_private_key")))

						return credsgen.Certificate{Certificate: []byte("the_cert"), PrivateKey: []byte("private_key"), IsCA: false}, nil
					})
					client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
						secret := object.(*corev1.Secret)
						Expect(secret.StringData["certificate"]).To(Equal("the_cert"))
						Expect(secret.StringData["private_key"]).To(Equal("private_key"))
						Expect(secret.StringData["ca"]).To(Equal("theca"))
						Expect(secret.GetLabels()).To(HaveKeyWithValue(qsv1a1.LabelKind, qsv1a1.GeneratedSecretKind))
						return nil
					})

					result, err := reconciler.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(client.CreateCallCount()).To(Equal(1))
					Expect(reconcile.Result{}).To(Equal(result))
				})

				It("considers generation parameters", func() {
					generator.GenerateCertificateCalls(func(name string, request credsgen.CertificateGenerationRequest) (credsgen.Certificate, error) {
						Expect(request.IsCA).To(BeFalse())
						Expect(request.CommonName).To(Equal("foo.com"))
						Expect(request.AlternativeNames).To(Equal([]string{"bar.com", "baz.com"}))
						return credsgen.Certificate{Certificate: []byte("the_cert"), PrivateKey: []byte("private_key"), IsCA: false}, nil
					})
					client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
						secret := object.(*corev1.Secret)
						Expect(secret.StringData["certificate"]).To(Equal("the_cert"))
						Expect(secret.StringData["private_key"]).To(Equal("private_key"))
						Expect(secret.StringData["ca"]).To(Equal("theca"))
						Expect(secret.GetLabels()).To(HaveKeyWithValue(qsv1a1.LabelKind, qsv1a1.GeneratedSecretKind))
						return nil
					})

					result, err := reconciler.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(client.CreateCallCount()).To(Equal(1))
					Expect(reconcile.Result{}).To(Equal(result))
				})
			})

			Context("and the generated cert is a ca", func() {
				BeforeEach(func() {
					qSecret.Spec.Request.CertificateRequest.IsCA = true
					qSecret.Spec.Request.CertificateRequest.CommonName = ""
					qSecret.Spec.Request.CertificateRequest.AlternativeNames = []string{}
				})

				It("considers the configured CA for signing", func() {
					generator.GenerateCertificateCalls(func(name string, request credsgen.CertificateGenerationRequest) (credsgen.Certificate, error) {
						Expect(request.IsCA).To(BeTrue())
						Expect(request.CA.IsCA).To(BeTrue())
						Expect(len(request.CA.PrivateKey) > 0).To(BeTrue())
						return credsgen.Certificate{Certificate: []byte("the_cert"), PrivateKey: []byte("private_key"), IsCA: true}, nil
					})
					client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
						secret := object.(*corev1.Secret)
						Expect(secret.StringData["certificate"]).To(Equal("the_cert"))
						Expect(secret.StringData["private_key"]).To(Equal("private_key"))
						Expect(secret.StringData["ca"]).To(Equal("theca"))
						Expect(secret.GetLabels()).To(HaveKeyWithValue(qsv1a1.LabelKind, qsv1a1.GeneratedSecretKind))
						return nil
					})

					result, err := reconciler.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(reconcile.Result{}).To(Equal(result))
				})
			})
		})
	})

	Context("when creating copies", func() {
		var copiedSecret *corev1.Secret
		BeforeEach(func() {
			copiedSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "generated-secret-copy",
					Namespace: "notdefault",
					Labels: map[string]string{
						"quarks.cloudfoundry.org/secret-kind": "generated",
					},
					Annotations: map[string]string{
						"quarks.cloudfoundry.org/secret-copy-of": "default/foo",
					},
				},
			}

			qSecret = &qsv1a1.QuarksSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: qsv1a1.QuarksSecretSpec{
					Type:       "password",
					SecretName: "generated-secret",
					Copies: []qsv1a1.Copy{
						{
							Name:      "generated-secret-copy",
							Namespace: "notdefault",
						},
					},
				},
			}

			client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
				switch object := object.(type) {
				case *qsv1a1.QuarksSecret:
					qSecret.DeepCopyInto(object)
				case *corev1.Secret:
					if nn.String() == "default/generated-secret" {
						return errors.NewNotFound(schema.GroupResource{}, "not found")
					}
				case *unstructured.Unstructured:
					if nn.String() == "notdefault/generated-secret-copy" {
						object.SetName(copiedSecret.Name)
						object.SetNamespace(copiedSecret.Namespace)
						object.SetLabels(copiedSecret.Labels)
						object.SetAnnotations(copiedSecret.Annotations)
						object.Object["data"] = copiedSecret.Data
					}
				}
				return nil
			})
		})

		It("it succeeds if everything is setup correctly", func() {
			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.GetCallCount()).To(Equal(4))
			Expect(client.CreateCallCount()).To(Equal(1))
			Expect(client.UpdateCallCount()).To(Equal(1))
			Expect(reconcile.Result{}).To(Equal(result))
		})

		It("it doesn't copy if the copy secret is missing", func() {
			client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
				switch object := object.(type) {
				case *qsv1a1.QuarksSecret:
					qSecret.DeepCopyInto(object)
				case *corev1.Secret:
					if nn.String() == "default/generated-secret" {
						return errors.NewNotFound(schema.GroupResource{}, "not found")
					}
					if nn.String() == "notdefault/generated-secret-copy" {
						return nil
					}
				}
				return nil
			})

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.GetCallCount()).To(Equal(4))
			Expect(client.CreateCallCount()).To(Equal(1))
			Expect(client.UpdateCallCount()).To(Equal(0))
			Expect(reconcile.Result{}).To(Equal(result))
		})

		It("it doesn't copy if the copy secret is missing a label", func() {
			copiedSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "generated-secret-copy",
					Namespace: "notdefault",
					Labels: map[string]string{
						"quarks.cloudfoundry.org/secret-kind": "generated",
					},
					Annotations: map[string]string{
						"quarks.cloudfoundry.org/secret-copy-of": "default/bar",
					},
				},
			}

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.GetCallCount()).To(Equal(4))
			Expect(client.CreateCallCount()).To(Equal(1))
			Expect(client.UpdateCallCount()).To(Equal(0))
			Expect(reconcile.Result{}).To(Equal(result))
		})
	})

	Context("when creating basic-auth", func() {
		BeforeEach(func() {
			qSecret.Spec.Type = "basic-auth"
		})

		It("creates a secret with k8s type basic-auth", func() {
			client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
				secret := object.(*corev1.Secret)
				Expect(secret.Type).To(Equal(corev1.SecretTypeBasicAuth))
				return nil
			})

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.CreateCallCount()).To(Equal(1))
			Expect(reconcile.Result{}).To(Equal(result))
		})

		It("creates a secret in the correct namespace", func() {
			client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
				secret := object.(*corev1.Secret)
				Expect(secret.Namespace).To(Equal(qSecret.Namespace))
				return nil
			})

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.CreateCallCount()).To(Equal(1))
			Expect(reconcile.Result{}).To(Equal(result))
		})

		It("creates a secret with the correct name", func() {
			client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
				secret := object.(*corev1.Secret)
				Expect(secret.Name).To(Equal(qSecret.Spec.SecretName))
				return nil
			})

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.CreateCallCount()).To(Equal(1))
			Expect(reconcile.Result{}).To(Equal(result))
		})

		When("create basic auth secret fails", func() {
			It("returns an error message", func() {
				client.CreateReturns(fmt.Errorf("something went terribly wrong"))

				_, err := reconciler.Reconcile(request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("generating basic-auth secret: could not create or update secret 'default/generated-secret': something went terribly wrong"))
			})

			It("does not requeue", func() {
				client.CreateReturns(fmt.Errorf("something went terribly wrong"))

				result, _ := reconciler.Reconcile(request)
				Expect(result.Requeue).To(BeFalse())
			})
		})

		When("username is not provided", func() {
			It("generates a username and password", func() {
				generator.GeneratePasswordReturnsOnCall(0, "some-secret-user")
				generator.GeneratePasswordReturnsOnCall(1, "some-secret-password")

				client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
					secret := object.(*corev1.Secret)
					Expect(secret.Type).To(Equal(corev1.SecretTypeBasicAuth))
					Expect(secret.StringData["username"]).To(Equal("some-secret-user"))
					Expect(secret.StringData["password"]).To(Equal("some-secret-password"))
					return nil
				})

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(generator.GeneratePasswordCallCount()).To(Equal(2))
				Expect(client.CreateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})

		When("username is provided", func() {
			It("generates a password, but not a username", func() {
				qSecret.Spec.Request.BasicAuthRequest.Username = "some-passed-in-username"
				generator.GeneratePasswordReturns("some-secret-password")

				client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
					secret := object.(*corev1.Secret)
					Expect(secret.Type).To(Equal(corev1.SecretTypeBasicAuth))
					Expect(secret.StringData["username"]).To(Equal("some-passed-in-username"))
					Expect(secret.StringData["password"]).To(Equal("some-secret-password"))
					return nil
				})

				result, err := reconciler.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(generator.GeneratePasswordCallCount()).To(Equal(1))
				Expect(client.CreateCallCount()).To(Equal(1))
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})
	})

	Context("when secret is set manually", func() {
		var (
			password string
			secret   *corev1.Secret
		)

		BeforeEach(func() {
			qSecret.Spec.Type = "password"
			qSecret.Spec.SecretName = "mysecret"

			password = "new-generated-password"
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysecret",
					Namespace: "default",
				},
				StringData: map[string]string{
					"password": "securepassword",
				},
			}

			client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
				switch object := object.(type) {
				case *qsv1a1.QuarksSecret:
					qSecret.DeepCopyInto(object)
				case *corev1.Secret:
					if nn.Name == "mysecret" {
						secret.DeepCopyInto(object)
					} else {
						return errors.NewNotFound(schema.GroupResource{}, "not found is requeued")
					}
				}
				return nil
			})

			generator.GeneratePasswordReturns(password)
		})

		It("Skips generation of a secret when existing secret has not `generated` label", func() {
			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.CreateCallCount()).To(Equal(0))
			Expect(client.UpdateCallCount()).To(Equal(0))
			Expect(reconcile.Result{}).To(Equal(result))
		})

		It("Skips generation of a secret when quarksSecret's `generated` status is true", func() {
			secret.Labels = map[string]string{
				qsv1a1.LabelKind: qsv1a1.GeneratedSecretKind,
			}
			qSecret.Status.Generated = pointers.Bool(true)

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.CreateCallCount()).To(Equal(0))
			Expect(client.UpdateCallCount()).To(Equal(0))
			Expect(reconcile.Result{}).To(Equal(result))
		})

		It("Regenerates a secret when the existing secret has a `generated` label", func() {
			secret.Labels = map[string]string{
				qsv1a1.LabelKind: qsv1a1.GeneratedSecretKind,
			}

			client.UpdateCalls(func(context context.Context, object runtime.Object, _ ...crc.UpdateOption) error {
				switch object := object.(type) {
				case *qsv1a1.QuarksSecret:
					Expect(object.Status.Generated).To(Equal(true))
				case *corev1.Secret:
					Expect(object.StringData["password"]).To(Equal(password))
					Expect(object.GetName()).To(Equal("mysecret"))
					Expect(object.GetLabels()).To(HaveKeyWithValue(qsv1a1.LabelKind, qsv1a1.GeneratedSecretKind))
				}
				return nil
			})

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.CreateCallCount()).To(Equal(0))
			Expect(client.UpdateCallCount()).To(Equal(1))
			Expect(reconcile.Result{}).To(Equal(result))

		})
	})
})
