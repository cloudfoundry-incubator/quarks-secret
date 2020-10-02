package environment

// The functions in this file are only used by the extended secret component
// tests.  They were split off in preparation for standalone components.

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"code.cloudfoundry.org/quarks-secret/pkg/credsgen"
	inmemorygenerator "code.cloudfoundry.org/quarks-secret/pkg/credsgen/in_memory_generator"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-secret/pkg/kube/client/clientset/versioned"
	"code.cloudfoundry.org/quarks-utils/testing/machine"
)

// Machine produces and destroys resources for tests
type Machine struct {
	machine.Machine

	VersionedClientset *versioned.Clientset
}

// GetQuarksSecret gets a QuarksSecret custom resource
func (m *Machine) GetQuarksSecret(namespace string, name string) (*qsv1a1.QuarksSecret, error) {
	client := m.VersionedClientset.QuarkssecretV1alpha1().QuarksSecrets(namespace)
	d, err := client.Get(context.Background(), name, metav1.GetOptions{})
	return d, err
}

// CreateQuarksSecret creates a QuarksSecret custom resource and returns a function to delete it
func (m *Machine) CreateQuarksSecret(namespace string, qs qsv1a1.QuarksSecret) (*qsv1a1.QuarksSecret, machine.TearDownFunc, error) {
	client := m.VersionedClientset.QuarkssecretV1alpha1().QuarksSecrets(namespace)
	d, err := client.Create(context.Background(), &qs, metav1.CreateOptions{})
	return d, func() error {
		err := client.Delete(context.Background(), qs.GetName(), metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}, err
}

// UpdateQuarksSecret updates a QuarksSecret custom resource and returns a function to delete it
func (m *Machine) UpdateQuarksSecret(namespace string, qs qsv1a1.QuarksSecret) (*qsv1a1.QuarksSecret, machine.TearDownFunc, error) {
	client := m.VersionedClientset.QuarkssecretV1alpha1().QuarksSecrets(namespace)
	d, err := client.Update(context.Background(), &qs, metav1.UpdateOptions{})
	return d, func() error {
		err := client.Delete(context.Background(), qs.GetName(), metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}, err
}

// DeleteQuarksSecret deletes an QuarksSecret custom resource
func (m *Machine) DeleteQuarksSecret(namespace string, name string) error {
	client := m.VersionedClientset.QuarkssecretV1alpha1().QuarksSecrets(namespace)
	return client.Delete(context.Background(), name, metav1.DeleteOptions{})
}

// QuarksSecretChangedFunc returns true if something changed in the quarks secret
type QuarksSecretChangedFunc func(qsv1a1.QuarksSecret) bool

// WaitForQuarksSecretChange waits for the quarks secret to fulfill the change func
func (m *Machine) WaitForQuarksSecretChange(namespace string, name string, changed QuarksSecretChangedFunc) error {
	return wait.PollImmediate(m.PollInterval, m.PollTimeout, func() (bool, error) {
		client := m.VersionedClientset.QuarkssecretV1alpha1().QuarksSecrets(namespace)
		qs, err := client.Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, errors.Wrapf(err, "failed to query for quarks secret: %s", name)
		}

		return changed(*qs), nil
	})
}

// CreateCASecret creates a CA and stores it in a secret
func (m *Machine) CreateCASecret(log *zap.SugaredLogger, namespace string, name string) (machine.TearDownFunc, error) {
	generator := inmemorygenerator.NewInMemoryGenerator(log)
	ca, err := generator.GenerateCertificate("default-ca", credsgen.CertificateGenerationRequest{
		CommonName: "Fake CA",
		IsCA:       true,
	})
	if err != nil {
		return nil, err
	}

	casecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"ca":  ca.Certificate,
			"key": ca.PrivateKey,
		},
	}
	return m.CreateSecret(namespace, casecret)
}
