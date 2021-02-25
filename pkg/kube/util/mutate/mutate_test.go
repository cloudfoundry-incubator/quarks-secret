package mutate_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	cfakes "code.cloudfoundry.org/quarks-secret/pkg/kube/controllers/fakes"
	"code.cloudfoundry.org/quarks-secret/pkg/kube/util/mutate"
)

var _ = Describe("Mutate", func() {
	var (
		ctx    context.Context
		client *cfakes.FakeClient
	)

	BeforeEach(func() {
		client = &cfakes.FakeClient{}
	})

	Describe("QuarksSecretMutateFn", func() {
		var (
			qSec *qsv1a1.QuarksSecret
		)

		BeforeEach(func() {
			qSec = &qsv1a1.QuarksSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: qsv1a1.QuarksSecretSpec{
					Type:       qsv1a1.Password,
					SecretName: "dummy-secret",
				},
			}
		})

		Context("when the quarksSecret is not found", func() {
			It("creates the quarksSecret", func() {
				client.GetCalls(func(context context.Context, nn types.NamespacedName, object crc.Object) error {
					return apierrors.NewNotFound(schema.GroupResource{}, nn.Name)
				})

				ops, err := controllerutil.CreateOrUpdate(ctx, client, qSec, mutate.QuarksSecretMutateFn(qSec))
				Expect(err).ToNot(HaveOccurred())
				Expect(ops).To(Equal(controllerutil.OperationResultCreated))
			})
		})

		Context("when the quarksSecret is found", func() {
			It("updates the quarksSecret when spec is changed", func() {
				client.GetCalls(func(context context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *qsv1a1.QuarksSecret:
						existing := &qsv1a1.QuarksSecret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "foo",
								Namespace: "default",
							},
							Spec: qsv1a1.QuarksSecretSpec{
								Type:       qsv1a1.Password,
								SecretName: "initial-secret",
							},
						}
						existing.DeepCopyInto(object)

						return nil
					}

					return apierrors.NewNotFound(schema.GroupResource{}, nn.Name)
				})
				ops, err := controllerutil.CreateOrUpdate(ctx, client, qSec, mutate.QuarksSecretMutateFn(qSec))
				Expect(err).ToNot(HaveOccurred())
				Expect(ops).To(Equal(controllerutil.OperationResultUpdated))
			})

			It("does not update the quarksSecret when nothing is changed", func() {
				client.GetCalls(func(context context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *qsv1a1.QuarksSecret:
						qSec.DeepCopyInto(object)

						return nil
					}

					return apierrors.NewNotFound(schema.GroupResource{}, nn.Name)
				})
				ops, err := controllerutil.CreateOrUpdate(ctx, client, qSec, mutate.QuarksSecretMutateFn(qSec))
				Expect(err).ToNot(HaveOccurred())
				Expect(ops).To(Equal(controllerutil.OperationResultNone))
			})
		})
	})

	Describe("SecretMutateFn", func() {
		var (
			sec *corev1.Secret
		)

		BeforeEach(func() {
			sec = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				StringData: map[string]string{
					"dummy": "foo-value",
				},
			}
		})

		Context("when the secret is not found", func() {
			It("creates the secret", func() {
				client.GetCalls(func(context context.Context, nn types.NamespacedName, object crc.Object) error {
					return apierrors.NewNotFound(schema.GroupResource{}, nn.Name)
				})

				ops, err := controllerutil.CreateOrUpdate(ctx, client, sec, mutate.SecretMutateFn(sec))
				Expect(err).ToNot(HaveOccurred())
				Expect(ops).To(Equal(controllerutil.OperationResultCreated))
			})
		})

		Context("when the secret is found", func() {
			It("updates the secret when secret data is changed", func() {
				client.GetCalls(func(context context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *corev1.Secret:
						existing := &corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "foo",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"dummy": []byte("initial-value"),
							},
						}
						existing.DeepCopyInto(object)

						return nil
					}

					return apierrors.NewNotFound(schema.GroupResource{}, nn.Name)
				})
				ops, err := controllerutil.CreateOrUpdate(ctx, client, sec, mutate.SecretMutateFn(sec))
				Expect(err).ToNot(HaveOccurred())
				Expect(ops).To(Equal(controllerutil.OperationResultUpdated))
			})

			It("does not update the secret when secret data is not changed", func() {
				client.GetCalls(func(context context.Context, nn types.NamespacedName, object crc.Object) error {
					switch object := object.(type) {
					case *corev1.Secret:
						existing := &corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "foo",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"dummy": []byte("foo-value"),
							},
						}
						existing.DeepCopyInto(object)

						return nil
					}

					return apierrors.NewNotFound(schema.GroupResource{}, nn.Name)
				})
				ops, err := controllerutil.CreateOrUpdate(ctx, client, sec, mutate.SecretMutateFn(sec))
				Expect(err).ToNot(HaveOccurred())
				Expect(ops).To(Equal(controllerutil.OperationResultNone))
			})
		})
	})
})
