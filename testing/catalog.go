package testing

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
)

// Catalog provides several instances for tests
type Catalog struct{}

// DefaultQuarksSecret for use in tests
func (c *Catalog) DefaultQuarksSecret(name string) qsv1a1.QuarksSecret {
	return qsv1a1.QuarksSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qsv1a1.QuarksSecretSpec{
			Type:       "password",
			SecretName: "generated-secret",
		},
	}
}

// CACertificateQuarksSecret for use in tests, creates a CA certificate
func (c *Catalog) CACertificateQuarksSecret(name string, secretref string, cacertref string, keyref string) qsv1a1.QuarksSecret {
	return qsv1a1.QuarksSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qsv1a1.QuarksSecretSpec{
			SecretName: "generated-cert-secret",
			Type:       "certificate",
			Request: qsv1a1.Request{
				CertificateRequest: qsv1a1.CertificateRequest{
					IsCA:             true,
					CommonName:       "example.com",
					CARef:            qsv1a1.SecretReference{Name: secretref, Key: cacertref},
					CAKeyRef:         qsv1a1.SecretReference{Name: secretref, Key: keyref},
					AlternativeNames: []string{"qux.com"},
				},
			},
		},
	}
}

// CertificateQuarksSecret for use in tests, creates a certificate
func (c *Catalog) CertificateQuarksSecret(name string, secretref string, cacertref string, keyref string) qsv1a1.QuarksSecret {
	return qsv1a1.QuarksSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qsv1a1.QuarksSecretSpec{
			SecretName: "generated-cert-secret",
			Type:       "certificate",
			Request: qsv1a1.Request{
				CertificateRequest: qsv1a1.CertificateRequest{
					CommonName:       "example.com",
					CARef:            qsv1a1.SecretReference{Name: secretref, Key: cacertref},
					CAKeyRef:         qsv1a1.SecretReference{Name: secretref, Key: keyref},
					AlternativeNames: []string{"qux.com"},
				},
			},
		},
	}
}

// TLSQuarksSecret for use in tests, creates a tls type secret
func (c *Catalog) TLSQuarksSecret(name string, secretref string, cacertref string, keyref string) qsv1a1.QuarksSecret {
	return qsv1a1.QuarksSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qsv1a1.QuarksSecretSpec{
			SecretName: "generated-cert-secret",
			Type:       "tls",
			Request: qsv1a1.Request{
				CertificateRequest: qsv1a1.CertificateRequest{
					CommonName:       "example.com",
					CARef:            qsv1a1.SecretReference{Name: secretref, Key: cacertref},
					CAKeyRef:         qsv1a1.SecretReference{Name: secretref, Key: keyref},
					AlternativeNames: []string{"qux.com"},
				},
			},
		},
	}
}

// SSHQuarksSecret returns a 'ssh' type quarks secret for testing
func (c *Catalog) SSHQuarksSecret(name string) qsv1a1.QuarksSecret {
	return qsv1a1.QuarksSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qsv1a1.QuarksSecretSpec{
			Type:       "ssh",
			SecretName: "generated-ssh-secret",
		},
	}
}

// RSAQuarksSecret returns a 'rsa' type quarks secret for testing
func (c *Catalog) RSAQuarksSecret(name string) qsv1a1.QuarksSecret {
	return qsv1a1.QuarksSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qsv1a1.QuarksSecretSpec{
			Type:       "rsa",
			SecretName: "generated-rsa-secret",
		},
	}
}

// BasicAuthQuarksSecret returns a 'basic auth' type quarks secret for testing
func (c *Catalog) BasicAuthQuarksSecret(name string) qsv1a1.QuarksSecret {
	return qsv1a1.QuarksSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qsv1a1.QuarksSecretSpec{
			Type:       "basic-auth",
			SecretName: "generated-basic-auth-secret",
			Request: qsv1a1.Request{
				BasicAuthRequest: qsv1a1.BasicAuthRequest{Username: "some-passed-in-username"},
			},
		},
	}
}

// RotationConfig is a config map, which triggers secret rotation
func (c *Catalog) RotationConfig(name string) corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rotation-config1",
			Labels: map[string]string{
				qsv1a1.LabelSecretRotationTrigger: "yes",
			},
		},
		Data: map[string]string{
			qsv1a1.RotateQSecretListName: fmt.Sprintf(`["%s"]`, name),
		},
	}
}
