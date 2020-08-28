package reference

import (
	"github.com/pkg/errors"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
)

// GetSecretsReferencedBy returns a list of all names for Secrets referenced by the object
// The object can be an QuarksStatefulSet or a BOSHDeployment
func GetSecretsReferencedBy(object interface{}) (map[string]bool, error) {
	switch object := object.(type) {
	case qsv1a1.QuarksSecret:
		return getSecretRefFromQuarksSecret(object)
	default:
		return nil, errors.New("can't get secret references for unknown type; supported types are BOSHDeployment and QuarksStatefulSet")
	}
}

func getSecretRefFromQuarksSecret(object qsv1a1.QuarksSecret) (map[string]bool, error) {
	result := map[string]bool{}

	result[object.Spec.SecretName] = true

	return result, nil
}
