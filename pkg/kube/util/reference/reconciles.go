package reference

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"code.cloudfoundry.org/quarks-secret/pkg/kube/apis"
	res "code.cloudfoundry.org/quarks-secret/pkg/kube/util/resources"
	log "code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
)

// ReconcileType lists all the types of reconciliations we can return,
// for controllers that have types that can reference Secrets
type ReconcileType int

const (
	// ReconcileForQuarksSecret represents the QuarksSecret CRD
	ReconcileForQuarksSecret ReconcileType = iota
)

func (r ReconcileType) String() string {
	return [...]string{
		"QuarksSecret",
	}[r]
}

// GetReconcilesWithFilter returns reconciliation requests for the QuarksSecret
// that reference an object. The object can be a Secret, it accepts an admit function which is used for filtering the object
func GetReconcilesWithFilter(ctx context.Context, client crc.Client, reconcileType ReconcileType, object apis.Object, versionCheck bool, admitFn func(v interface{}) bool) ([]reconcile.Request, error) {
	objReferencedBy := func(parent interface{}) (bool, error) {
		var (
			objectReferences map[string]bool
			err              error
			name             string
		)

		switch object := object.(type) {
		case *corev1.Secret:
			objectReferences, err = GetSecretsReferencedBy(ctx, client, parent)
			name = object.Name
		default:
			return false, errors.New("can't get reconciles for unknown object type; supported types are Secret")
		}

		if err != nil {
			return false, errors.Wrap(err, "error listing references")
		}

		_, ok := objectReferences[name]
		return ok, nil
	}

	namespace := object.GetNamespace()
	result := []reconcile.Request{}

	log.Debugf(ctx, "Listing '%s' for '%s/%s'", reconcileType, namespace, object.GetName())
	switch reconcileType {
	case ReconcileForQuarksSecret:
		quarksSecrets, err := res.ListQuarksSecrets(ctx, client, namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list QuarksSecrets for Secret reconciles")
		}

		for _, quarksSecret := range quarksSecrets.Items {
			isRef, err := objReferencedBy(quarksSecret)
			if err != nil {
				return nil, err
			}

			if isRef && admitFn(quarksSecret) {
				result = append(result, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      quarksSecret.Name,
						Namespace: quarksSecret.Namespace,
					}})
			}
		}
	default:
		return nil, fmt.Errorf("unknown reconcile type %s", reconcileType.String())
	}

	return result, nil
}

// GetReconciles returns reconciliation requests for the QuarksSecret
// that reference an object. The object can be a Secret
func GetReconciles(ctx context.Context, client crc.Client, reconcileType ReconcileType, object apis.Object, versionCheck bool) ([]reconcile.Request, error) {
	return GetReconcilesWithFilter(ctx, client, reconcileType, object, versionCheck, func(v interface{}) bool { return true })
}
