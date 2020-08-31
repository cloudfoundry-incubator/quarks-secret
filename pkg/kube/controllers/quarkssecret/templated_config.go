package quarkssecret

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"code.cloudfoundry.org/quarks-secret/pkg/helm/template"
	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
)

// TemplateEngine renders TemplatedConfigs, which are stored in secret.data
type TemplateEngine interface {
	// ExecuteMap renders the templates in templates with variables from values
	ExecuteMap(templates map[string]string, values map[string]interface{}) map[string]string
}

// renderSecret uses the specified templating engine to render the templates in Spec.Templates with the data from the referenced secrets.
func (r *ReconcileQuarksSecret) renderSecret(ctx context.Context, namespace string, request qsv1a1.TemplatedConfigRequest) (map[string]string, error) {
	empty := map[string]string{}

	// Interpolate the given values, and retrieve the secret contents to fill our config with
	values := map[string]interface{}{}
	for name, ref := range request.Values {
		secretName := ref.Name
		field := ref.Key

		existingSecret := &corev1.Secret{}
		namespacedName := types.NamespacedName{
			Namespace: namespace,
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
		values[name] = string(data)
	}

	// Call the specified rendering engine to draw our data
	var engine TemplateEngine
	switch request.Type {
	case HelmTemplate:
		engine = template.New()
	default:
		return empty, errors.New("unsupported template type has been specified")
	}
	return engine.ExecuteMap(request.Templates, values), nil
}

func (r *ReconcileQuarksSecret) createTemplatedConfigSecret(ctx context.Context, qsec *qsv1a1.QuarksSecret) error {
	secretData, err := r.renderSecret(ctx, qsec.Namespace, qsec.Spec.Request.TemplatedConfigRequest)
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
