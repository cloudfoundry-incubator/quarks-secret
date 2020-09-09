package quarkssecret_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	generatorfakes "code.cloudfoundry.org/quarks-secret/pkg/credsgen/fakes"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-secret/pkg/kube/client/clientset/versioned/scheme"
	"code.cloudfoundry.org/quarks-secret/pkg/kube/controllers"
	cfakes "code.cloudfoundry.org/quarks-secret/pkg/kube/controllers/fakes"
	qscontroller "code.cloudfoundry.org/quarks-secret/pkg/kube/controllers/quarkssecret"
	cfcfg "code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	helper "code.cloudfoundry.org/quarks-utils/testing/testhelper"
)

var _ = Describe("ReconcileCopy", func() {
	var (
		manager                        *cfakes.FakeManager
		reconciler                     reconcile.Reconciler
		request                        reconcile.Request
		ctx                            context.Context
		log                            *zap.SugaredLogger
		logs                           *observer.ObservedLogs
		config                         *cfcfg.Config
		client                         *cfakes.FakeClient
		generator                      *generatorfakes.FakeGenerator
		quarksSecret, quarksCopySecret *qsv1a1.QuarksSecret
		passwordSecret                 *corev1.Secret
		setReferenceFunc               func(owner, object metav1.Object, scheme *runtime.Scheme) error = func(owner, object metav1.Object, scheme *runtime.Scheme) error { return nil }
	)

	const (
		quarksSecretName = "test.qsec"
		defaultNamespace = "default"
		copyNamespace    = "copy"
	)

	BeforeEach(func() {
		err := controllers.AddToScheme(scheme.Scheme)
		Expect(err).ToNot(HaveOccurred())
		manager = &cfakes.FakeManager{}
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: quarksSecretName, Namespace: defaultNamespace}}
		config = &cfcfg.Config{CtxTimeOut: 10 * time.Second}
		logs, log = helper.NewTestLogger()
		ctx = ctxlog.NewParentContext(log)
		generator = &generatorfakes.FakeGenerator{}
		client = &cfakes.FakeClient{}

		quarksSecret = &qsv1a1.QuarksSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      quarksSecretName,
				Namespace: defaultNamespace,
			},
			Spec: qsv1a1.QuarksSecretSpec{
				Type:       "password",
				SecretName: "generated-secret",
				Copies: []qsv1a1.Copy{
					{
						Name:      "generated-secret-copy",
						Namespace: copyNamespace,
					},
				},
			},
		}

		passwordSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "generated-secret",
				Namespace: defaultNamespace,
			},
			StringData: map[string]string{
				"password": "securepassword",
			},
		}

		quarksCopySecret = &qsv1a1.QuarksSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      quarksSecretName,
				Namespace: copyNamespace,
				Annotations: map[string]string{
					"quarks.cloudfoundry.org/secret-copy-of": defaultNamespace + "/" + quarksSecretName,
				},
			},
			Spec: qsv1a1.QuarksSecretSpec{
				Type:       copyNamespace,
				SecretName: "generated-secret-copy",
			},
		}

		client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
			switch object := object.(type) {
			case *qsv1a1.QuarksSecret:
				if nn.Namespace == defaultNamespace {
					quarksSecret.DeepCopyInto(object)
				} else {
					return errors.NewNotFound(schema.GroupResource{}, "not found")
				}
				return nil
			case *corev1.Secret:
				if nn.Name == "generated-secret" {
					passwordSecret.DeepCopyInto(object)
				} else {
					return errors.NewNotFound(schema.GroupResource{}, "not found")
				}
			}
			return nil
		})
		client.StatusCalls(func() crc.StatusWriter { return &cfakes.FakeStatusWriter{} })
		manager.GetClientReturns(client)
	})

	JustBeforeEach(func() {
		reconciler = qscontroller.NewCopyReconciler(ctx, config, manager, generator, setReferenceFunc)
	})

	When("the source secret is not found", func() {
		It("should return an error", func() {
			client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
				switch object := object.(type) {
				case *qsv1a1.QuarksSecret:
					if nn.Namespace == defaultNamespace {
						quarksSecret.DeepCopyInto(object)
					}
					if nn.Namespace == copyNamespace {
						quarksCopySecret.DeepCopyInto(object)
					}
					return nil
				case *corev1.Secret:
					return errors.NewNotFound(schema.GroupResource{}, "not found")
				}
				return nil
			})

			result, err := reconciler.Reconcile(request)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
			Expect(reconcile.Result{}).To(Equal(result))
		})
	})

	When("validating target namespace", func() {
		It("should skip reconcile when no marker found", func() {
			result, err := reconciler.Reconcile(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(logs.FilterMessageSnippet("Skip copy creation"))
			Expect(reconcile.Result{}).To(Equal(result))
		})

		It("should skip when no copyof annotation is present", func() {
			quarksCopySecret.Annotations = map[string]string{}

			result, err := reconciler.Reconcile(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(logs.FilterMessageSnippet("Skip copy creation"))
			Expect(reconcile.Result{}).To(Equal(result))
		})
	})

	When("everything is set properly", func() {
		It("copying should be done", func() {
			client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
				switch object := object.(type) {
				case *qsv1a1.QuarksSecret:
					if nn.Namespace == defaultNamespace {
						quarksSecret.DeepCopyInto(object)
					}
					if nn.Namespace == copyNamespace {
						quarksCopySecret.DeepCopyInto(object)
					}
					return nil
				case *corev1.Secret:
					if nn.Name == "generated-secret" {
						passwordSecret.DeepCopyInto(object)
					} else {
						return errors.NewNotFound(schema.GroupResource{}, "not found")
					}
				}
				return nil
			})

			result, err := reconciler.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
			Expect(client.GetCallCount()).To(Equal(5))
			Expect(client.CreateCallCount()).To(Equal(1))
			Expect(reconcile.Result{}).To(Equal(result))
		})
	})
})
