package v1alpha1

import (
	"fmt"

	certv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apis "code.cloudfoundry.org/quarks-secret/pkg/kube/apis"
)

// This file is safe to edit
// It's used as input for the Kube code generator
// Run "make generate" after modifying this file

// ReferenceType lists all the types of Reference we can supports
type ReferenceType = string

// Valid values for ref types
const (
	// SecretReference represents Secret reference
	KubeSecretReference ReferenceType = "secret"
)

// SecretType defines the type of the generated secret
type SecretType = string

// Valid values for secret types
const (
	Password         SecretType = "password"
	Certificate      SecretType = "certificate"
	TLS              SecretType = "tls"
	SSHKey           SecretType = "ssh"
	RSAKey           SecretType = "rsa"
	BasicAuth        SecretType = "basic-auth"
	DockerConfigJSON SecretType = "dockerconfigjson"
	SecretCopy       SecretType = "copy"
	TemplatedConfig  SecretType = "templatedconfig"
)

// SignerType defines the type of the certificate signer
type SignerType = string

// Valid values for signer types
const (
	// LocalSigner defines the local as certificate signer
	LocalSigner SignerType = "local"
	// ClusterSigner defines the cluster as certificate signer
	ClusterSigner SignerType = "cluster"
)

var (
	// LabelKind is the label key for secret kind
	LabelKind = fmt.Sprintf("%s/secret-kind", apis.GroupName)
	// LabelNamespace key for label on a namespace to indicate that cf-operator is monitoring it.
	// Can be used as an ID, to keep operators in a cluster from intefering with each other.
	LabelNamespace = fmt.Sprintf("%s/monitored", apis.GroupName)
	// AnnotationCopyOf is a label key for secrets that are copies of generated secrets
	AnnotationCopyOf = fmt.Sprintf("%s/secret-copy-of", apis.GroupName)
	// AnnotationCertSecretName is the annotation key for certificate secret name
	AnnotationCertSecretName = fmt.Sprintf("%s/cert-secret-name", apis.GroupName)
	// AnnotationQSecName is the annotation key for the name of the owning quarks secret
	AnnotationQSecName = fmt.Sprintf("%s/quarks-secret-name", apis.GroupName)
	// AnnotationQSecNamespace is the annotation key for quarks secret namespace
	// since CSR are not namespaced
	AnnotationQSecNamespace = fmt.Sprintf("%s/quarks-secret-namespace", apis.GroupName)
	// AnnotationMonitoredID is used to link a CSR to a operator, so we don't have to
	// infer that via the namespace
	AnnotationMonitoredID = fmt.Sprintf("%s/monitored-id", apis.GroupName)
	// LabelSecretRotationTrigger is set on a config map to trigger secret
	// rotation. If set, then creating the config map will trigger secret
	// rotation.
	LabelSecretRotationTrigger = fmt.Sprintf("%s/secret-rotation", apis.GroupName)
	// RotateQSecretListName is the name of the config map entry, which
	// contains a JSON array of quarks secret names to rotate
	RotateQSecretListName = "secrets"
)

const (
	// GeneratedSecretKind is the kind of generated secret
	GeneratedSecretKind = "generated"
)

// SecretReference specifies a reference to another secret
type SecretReference struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// ServiceReference specifies a reference to a service
type ServiceReference struct {
	Name string
}

// CertificateRequest specifies the details for the certificate generation
type CertificateRequest struct {
	CommonName                  string             `json:"commonName"`
	AlternativeNames            []string           `json:"alternativeNames"`
	IsCA                        bool               `json:"isCA"`
	CARef                       SecretReference    `json:"CARef"`
	CAKeyRef                    SecretReference    `json:"CAKeyRef"`
	SignerType                  SignerType         `json:"signerType,omitempty"`
	Usages                      []certv1.KeyUsage  `json:"usages"`
	ServiceRef                  []ServiceReference `json:"serviceRef"`
	ActivateEKSWorkaroundForSAN bool               `json:"activateEKSWorkaroundForSAN,omitempty"`
}

// BasicAuthRequest specifies the details for generating a basic-auth secret
type BasicAuthRequest struct {
	Username string `json:"username"`
}

// ImageCredentialsRequest specifies the details for the image credentials
type ImageCredentialsRequest struct {
	Username SecretReference `json:"username"`
	Password SecretReference `json:"password"`
	Registry string          `json:"registry"`
	Email    string          `json:"email"`
}

// TemplatedConfigRequest defines the type of the template engine, a map of templates, one
// per key and the variables for the templates.
type TemplatedConfigRequest struct {
	Type      string                     `json:"type,omitempty"`
	Templates map[string]string          `json:"templates,omitempty"`
	Values    map[string]SecretReference `json:"values,omitempty"`
}

// Request specifies details for the secret generation
type Request struct {
	BasicAuthRequest        BasicAuthRequest        `json:"basic-auth"`
	CertificateRequest      CertificateRequest      `json:"certificate"`
	ImageCredentialsRequest ImageCredentialsRequest `json:"imageCredentials"`
	TemplatedConfigRequest  TemplatedConfigRequest  `json:"templatedConfig,omitempty"`
}

// Copy defines the destination of a copied generated secret
// We can't use types.NamespacedName because it doesn't marshal properly
type Copy struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func (c *Copy) String() string {
	return fmt.Sprintf("%s/%s", c.Namespace, c.Name)
}

// QuarksSecretSpec defines the desired state of QuarksSecret
type QuarksSecretSpec struct {
	Type              SecretType        `json:"type"`
	Request           Request           `json:"request"`
	SecretName        string            `json:"secretName"`
	Copies            []Copy            `json:"copies,omitempty"`
	SecretLabels      map[string]string `json:"secretLabels,omitempty"`
	SecretAnnotations map[string]string `json:"secretAnnotations,omitempty"`
}

// QuarksSecretStatus defines the observed state of QuarksSecret
type QuarksSecretStatus struct {
	// Timestamp for the last reconcile
	LastReconcile *metav1.Time `json:"lastReconcile"`
	// Indicates if the secret has already been generated
	Generated *bool `json:"generated"`
	// Indicates if the copy secrets have been updated
	Copied *bool `json:"copied"`
}

// IsCopied returns true if the copied field is a true value
func (qs QuarksSecretStatus) IsCopied() bool {
	return qs.Copied != nil && *qs.Copied

}

// NotCopied returns true if the copied field is a false value
func (qs QuarksSecretStatus) NotCopied() bool {
	return qs.Copied != nil && *qs.Copied
}

// IsGenerated returns true if the Generated field is a true value
func (qs QuarksSecretStatus) IsGenerated() bool {
	if qs.Generated == nil {
		return false
	}
	return *qs.Generated
}

// NotGenerated returns true if the Generated field is set to false, but not nil
func (qs QuarksSecretStatus) NotGenerated() bool {
	if qs.Generated == nil {
		return false
	}
	return !*qs.Generated
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// QuarksSecret is the Schema for the QuarksSecrets API
// +k8s:openapi-gen=true
type QuarksSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec         QuarksSecretSpec   `json:"spec,omitempty"`
	Status       QuarksSecretStatus `json:"status,omitempty"`
	SecretLabels map[string]string  `json:"secretLabels,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// QuarksSecretList contains a list of QuarksSecret
type QuarksSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QuarksSecret `json:"items"`
}

// GetNamespacedName returns the resource name with its namespace
func (qs *QuarksSecret) GetNamespacedName() string {
	return fmt.Sprintf("%s/%s", qs.Namespace, qs.Name)
}

// IsMonitoredNamespace returns true if the namespace has all the necessary
// labels and should be included in controller watches.
func IsMonitoredNamespace(n *corev1.Namespace, id string) bool {
	if value, ok := n.Labels[LabelNamespace]; ok && value == id {
		return true
	}
	return false
}
