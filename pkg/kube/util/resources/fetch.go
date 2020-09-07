package resources

import (
	"context"

	crc "sigs.k8s.io/controller-runtime/pkg/client"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
)

// ListQuarksSecrets fetches all the quarkssecrets from the namespace
func ListQuarksSecrets(ctx context.Context, client crc.Client, namespace string) (*qsv1a1.QuarksSecretList, error) {

}
