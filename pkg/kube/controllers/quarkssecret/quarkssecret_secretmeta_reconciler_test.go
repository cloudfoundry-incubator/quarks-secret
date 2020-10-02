package quarkssecret_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"

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

var _ = Describe("ReconcileQuarksSecretSecretMeta", func() {
	var (
		manager    *cfakes.FakeManager
		reconciler reconcile.Reconciler
		request    reconcile.Request
		ctx        context.Context
		log        *zap.SugaredLogger
		config     *cfcfg.Config
		client     *cfakes.FakeClient
		generator  *generatorfakes.FakeGenerator
		qSecret    *qsv1a1.QuarksSecret
		secret     *corev1.Secret
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
				SecretLabels: map[string]string{
					"LabelKey": "LabelValue",
				},
			},
		}

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "generated-secret",
				Namespace: "default",
			},
			StringData: map[string]string{
				"password": "password",
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
		reconciler = qscontroller.NewQuarksSecretSecretMetaReconciler(ctx, config, manager, generator)
	})

	Context("if the source secret is not found ", func() {
		It("should return an error", func() {
			_, err := reconciler.Reconcile(request)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not fetch source secret"))
		})
	})

	Context("if the source secret is found", func() {
		It("should have the correct labels", func() {
			client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
				switch object := object.(type) {
				case *qsv1a1.QuarksSecret:
					qSecret.DeepCopyInto(object)
				case *corev1.Secret:
					secret.DeepCopyInto(object)
				}
				return nil
			})

			client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
				secret := object.(*corev1.Secret)
				Expect(secret.GetLabels()).To(Equal(map[string]string{
					"LabelKey": "LabelValue",
				}))
				return nil
			})

			_, err := reconciler.Reconcile(request)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
