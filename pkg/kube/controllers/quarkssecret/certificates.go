package quarkssecret

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	certv1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"code.cloudfoundry.org/quarks-secret/pkg/credsgen"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"code.cloudfoundry.org/quarks-utils/pkg/names"
)

func (r *ReconcileQuarksSecret) createCertificateSecret(ctx context.Context, qsec *qsv1a1.QuarksSecret) error {
	serviceIPForEKSWorkaround := ""

	for _, serviceRef := range qsec.Spec.Request.CertificateRequest.ServiceRef {
		service := &corev1.Service{}

		err := r.client.Get(ctx, types.NamespacedName{Namespace: qsec.Namespace, Name: serviceRef.Name}, service)

		if err != nil {
			return errors.Wrapf(err, "Failed to get service reference '%s' for QuarksSecret '%s'", serviceRef.Name, qsec.GetNamespacedName())
		}

		if serviceIPForEKSWorkaround == "" {
			serviceIPForEKSWorkaround = service.Spec.ClusterIP
		}

		qsec.Spec.Request.CertificateRequest.AlternativeNames = append(append(
			qsec.Spec.Request.CertificateRequest.AlternativeNames,
			service.Name,
			service.Name+"."+service.Namespace,
			"*."+service.Name,
			"*."+service.Name+"."+service.Namespace,
			service.Spec.ClusterIP,
			service.Spec.LoadBalancerIP,
			service.Spec.ExternalName,
		), service.Spec.ExternalIPs...)
	}

	if len(qsec.Spec.Request.CertificateRequest.SignerType) == 0 {
		qsec.Spec.Request.CertificateRequest.SignerType = qsv1a1.LocalSigner
	}

	generationRequest, err := r.generateCertificateGenerationRequest(ctx, qsec.Namespace, qsec.Spec.Request.CertificateRequest)
	if err != nil {
		return errors.Wrap(err, "generating certificate generation request")
	}

	switch qsec.Spec.Request.CertificateRequest.SignerType {
	case qsv1a1.ClusterSigner:
		if qsec.Spec.Request.CertificateRequest.ActivateEKSWorkaroundForSAN {
			if serviceIPForEKSWorkaround == "" {
				return errors.Errorf("can't activate EKS workaround for QuarksSecret '%s'; couldn't find a ClusterIP for any service reference", qsec.GetNamespacedName())
			}

			ctxlog.Infof(ctx, "Activating EKS workaround for QuarksSecret '%s'. Using IP '%s' as a common name. See 'https://github.com/awslabs/amazon-eks-ami/issues/341' for more details.", qsec.GetNamespacedName(), serviceIPForEKSWorkaround)

			generationRequest.CommonName = serviceIPForEKSWorkaround
		}

		ctxlog.Info(ctx, "Generating certificate signing request and its key")
		csr, key, err := r.generator.GenerateCertificateSigningRequest(generationRequest)
		if err != nil {
			return err
		}

		// private key Secret which will be merged to certificate Secret later
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        names.CsrPrivateKeySecretName(names.CSRName(qsec.Namespace, qsec.Name)),
				Namespace:   qsec.GetNamespace(),
				Labels:      qsec.Spec.SecretLabels,
				Annotations: qsec.Spec.SecretAnnotations,
			},
			StringData: map[string]string{
				"private_key": string(key),
				"is_ca":       strconv.FormatBool(qsec.Spec.Request.CertificateRequest.IsCA),
			},
		}

		err = r.createSecrets(ctx, qsec, secret)
		if err != nil {
			return err
		}

		return r.createCertificateSigningRequest(ctx, qsec, csr)
	case qsv1a1.LocalSigner:
		// Generate certificate
		cert, err := r.generator.GenerateCertificate(qsec.GetName(), generationRequest)
		if err != nil {
			return err
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        qsec.Spec.SecretName,
				Namespace:   qsec.GetNamespace(),
				Labels:      qsec.Spec.SecretLabels,
				Annotations: qsec.Spec.SecretAnnotations,
			},
			StringData: map[string]string{
				"certificate": string(cert.Certificate),
				"private_key": string(cert.PrivateKey),
				"is_ca":       strconv.FormatBool(qsec.Spec.Request.CertificateRequest.IsCA),
			},
		}

		if len(generationRequest.CA.Certificate) > 0 {
			secret.StringData["ca"] = string(generationRequest.CA.Certificate)
		}

		return r.createSecrets(ctx, qsec, secret)
	default:
		return fmt.Errorf("unrecognized signer type: %s", qsec.Spec.Request.CertificateRequest.SignerType)
	}
}

// generateCertificateGenerationRequest generates CertificateGenerationRequest for certificate
func (r *ReconcileQuarksSecret) generateCertificateGenerationRequest(ctx context.Context, namespace string, certificateRequest qsv1a1.CertificateRequest) (credsgen.CertificateGenerationRequest, error) {
	var request credsgen.CertificateGenerationRequest
	switch certificateRequest.SignerType {
	case qsv1a1.ClusterSigner:
		// Generate cluster-signed CA certificate
		request = credsgen.CertificateGenerationRequest{
			CommonName:       certificateRequest.CommonName,
			AlternativeNames: certificateRequest.AlternativeNames,
		}
	case qsv1a1.LocalSigner:
		// Generate local-issued CA certificate
		request = credsgen.CertificateGenerationRequest{
			IsCA:             certificateRequest.IsCA,
			CommonName:       certificateRequest.CommonName,
			AlternativeNames: certificateRequest.AlternativeNames,
		}

		if len(certificateRequest.CARef.Name) > 0 {
			// Get CA certificate
			caSecret := &corev1.Secret{}
			caNamespacedName := types.NamespacedName{
				Namespace: namespace,
				Name:      certificateRequest.CARef.Name,
			}
			err := r.client.Get(ctx, caNamespacedName, caSecret)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return request, newCaNotReadyError("CA secret not found")
				}
				return request, errors.Wrap(err, "getting CA secret")
			}
			ca := caSecret.Data[certificateRequest.CARef.Key]

			// Get CA key
			if certificateRequest.CAKeyRef.Name != certificateRequest.CARef.Name {
				caSecret = &corev1.Secret{}
				caNamespacedName = types.NamespacedName{
					Namespace: namespace,
					Name:      certificateRequest.CAKeyRef.Name,
				}
				err = r.client.Get(ctx, caNamespacedName, caSecret)
				if err != nil {
					if apierrors.IsNotFound(err) {
						return request, newCaNotReadyError("CA key secret not found")
					}
					return request, errors.Wrap(err, "getting CA Key secret")
				}
			}
			key := caSecret.Data[certificateRequest.CAKeyRef.Key]
			request.CA = credsgen.Certificate{
				IsCA:        true,
				PrivateKey:  key,
				Certificate: ca,
			}
		}
	default:
		return request, fmt.Errorf("unrecognized signer type: %s", certificateRequest.SignerType)
	}

	return request, nil
}

// createCertificateSigningRequest creates CertificateSigningRequest Object
func (r *ReconcileQuarksSecret) createCertificateSigningRequest(ctx context.Context, qsec *qsv1a1.QuarksSecret, csr []byte) error {
	csrName := names.CSRName(qsec.Namespace, qsec.Name)
	ctxlog.Debugf(ctx, "Creating certificatesigningrequest '%s'", csrName)

	annotations := qsec.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[qsv1a1.AnnotationCertSecretName] = qsec.Spec.SecretName
	annotations[qsv1a1.AnnotationQSecNamespace] = qsec.Namespace
	annotations[qsv1a1.AnnotationQSecName] = qsec.Name
	annotations[qsv1a1.AnnotationMonitoredID] = r.config.MonitoredID

	csrObj := &certv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:        csrName,
			Labels:      qsec.Labels,
			Annotations: annotations,
		},
		Spec: certv1.CertificateSigningRequestSpec{
			Request: csr,
			Usages:  qsec.Spec.Request.CertificateRequest.Usages,
		},
	}

	oldCsrObj := &certv1.CertificateSigningRequest{}

	// CSR spec is immutable after the request is created
	err := r.client.Get(ctx, types.NamespacedName{Name: csrObj.Name}, oldCsrObj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = r.client.Create(ctx, csrObj)
			if err != nil {
				return errors.Wrapf(err, "could not create certificatesigningrequest '%s'", csrObj.Name)
			}
			return nil
		}
		return errors.Wrapf(err, "could not get certificatesigningrequest '%s'", csrObj.Name)
	}

	ctxlog.Infof(ctx, "Ignoring immutable CSR '%s'", csrObj.Name)
	return nil
}
