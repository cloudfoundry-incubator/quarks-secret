package mutate

import (
	"reflect"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// QuarksSecretMutateFn returns MutateFn which mutates QuarksSecret including:
// - labels, annotations
// - spec
func QuarksSecretMutateFn(qSec *qsv1a1.QuarksSecret) controllerutil.MutateFn {
	updated := qSec.DeepCopy()
	return func() error {
		qSec.Labels = updated.Labels
		qSec.Annotations = updated.Annotations
		qSec.Spec = updated.Spec

		return nil
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
