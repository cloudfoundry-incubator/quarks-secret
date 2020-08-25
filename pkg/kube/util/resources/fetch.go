package resources

import (
	"context"

	"github.com/pkg/errors"
	crc "sigs.k8s.io/controller-runtime/pkg/client"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
)

// ListQuarksSecrets fetches all the quarkssecrets from the namespace
func ListQuarksSecrets(ctx context.Context, client crc.Client, namespace string) (*qsv1a1.QuarksSecretList, error) {
	result := &qsv1a1.QuarksSecretList{}
	err := client.List(ctx, result, crc.InNamespace(namespace))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list QuarksSecrets")
	}

	return result, nil
}
