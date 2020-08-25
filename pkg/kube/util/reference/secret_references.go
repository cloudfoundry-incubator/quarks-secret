package reference

import (
	"context"

	"github.com/pkg/errors"
	crc "sigs.k8s.io/controller-runtime/pkg/client"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
)

// GetSecretsReferencedBy returns a list of all names for Secrets referenced by the object
// The object can be an QuarksStatefulSet or a BOSHDeployment
func GetSecretsReferencedBy(ctx context.Context, client crc.Client, object interface{}) (map[string]bool, error) {
	switch object := object.(type) {
	case qsv1a1.QuarksSecret:
		return getSecretRefFromQuarksSecret(ctx, client, object)
	default:
		return nil, errors.New("can't get secret references for unknown type; supported types are BOSHDeployment and QuarksStatefulSet")
	}
}

func getSecretRefFromQuarksSecret(ctx context.Context, client crc.Client, object qsv1a1.QuarksSecret) (map[string]bool, error) {
	result := map[string]bool{}

	result[object.Spec.SecretName] = true

	return result, nil
}
