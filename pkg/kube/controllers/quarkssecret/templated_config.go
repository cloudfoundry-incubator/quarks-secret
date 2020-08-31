package quarkssecret

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"code.cloudfoundry.org/quarks-secret/pkg/helm/template"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
)

// renderSecret uses the specified templating engine to draw the secret data
func (r *ReconcileQuarksSecret) renderSecret(ctx context.Context, qsec *qsv1a1.QuarksSecret) (map[string]string, error) {
	empty := map[string]string{}
	if qsec.Spec.TemplateType == "" {
		return empty, errors.New("templatedConfig needs a templateType to be specified. E.g. helm")
	}

	// Interpolate the given values, and retrieve the secret contents to fill our config with
	values := map[string]interface{}{}
	for k, v := range qsec.Spec.TemplateValues {
		fields := strings.Split(v.(string), ".")
		if len(fields) != 2 {
			return empty, errors.New("failed while reading templatedConfig values. Values must be in the `secretname.field` format. Got: " + v.(string))
		}
		secretName := fields[0]
		field := fields[1]

		existingSecret := &corev1.Secret{}
		namespacedName := types.NamespacedName{
			Namespace: qsec.Namespace,
			Name:      secretName,
		}
		err := r.client.Get(ctx, namespacedName, existingSecret)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return empty, newSecNotReadyError("secret not found")
			}
			return empty, errors.Wrap(err, "getting secret")
		}
		data, ok := existingSecret.Data[field]
		if !ok {
			return empty, errors.Errorf("Failed to get secret data key: %s", field)
		}
		values[k] = string(data)
	}

	// Call the specified rendering engine to draw our data
	switch qsec.Spec.TemplateType {
	case HelmTemplate:
		t := template.New()
		return t.ExecuteMap(qsec.Spec.Template, values), nil
	default:
		return map[string]string{}, errors.New("unsupported template type has been specified")
	}
}

func (r *ReconcileQuarksSecret) createTemplatedSecret(ctx context.Context, qsec *qsv1a1.QuarksSecret) error {
	secretData, err := r.renderSecret(ctx, qsec)
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
		StringData: secretData,
	}

	return r.createSecrets(ctx, qsec, secret)
}
