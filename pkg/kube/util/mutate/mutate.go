package mutate

import (
	"reflect"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"github.com/cloudflare/cfssl/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// QuarksSecretMutateFn returns MutateFn which mutates QuarksSecret including:
// - labels, annotations
// - spec
func QuarksSecretMutateFn(qSec *qsv1a1.QuarksSecret) controllerutil.MutateFn {
	updated := qSec.DeepCopy()
	return func() error {
		changed := false
		if !reflect.DeepEqual(qSec.Labels, updated.Labels) {
			qSec.Labels = updated.Labels
			changed = true
		}

		if !reflect.DeepEqual(qSec.Annotations, updated.Annotations) {
			qSec.Annotations = updated.Annotations
			changed = true
		}

		if !reflect.DeepEqual(qSec.Spec, updated.Spec) {
			qSec.Spec = updated.Spec
			changed = true
		}

		if changed {
			return nil
		}

		return &errors.Error{Message: "Nothing updated"}
	}
}

// SecretMutateFn returns MutateFn which mutates Secret including:
// - labels, annotations
// - stringData
func SecretMutateFn(s *corev1.Secret) controllerutil.MutateFn {
	updated := s.DeepCopy()
	return func() error {
		s.Labels = updated.Labels
		s.Annotations = updated.Annotations
		for key, data := range updated.StringData {
			// Update once one of data has been changed
			oriData, ok := s.Data[key]
			if ok && reflect.DeepEqual(string(oriData), data) {
				continue
			} else {
				s.StringData = updated.StringData
				break
			}
		}
		return nil
	}
}
