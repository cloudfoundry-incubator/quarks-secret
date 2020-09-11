package quarkssecret

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	qsv1a1 "code.cloudfoundry.org/quarks-secret/pkg/kube/apis/quarkssecret/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/pkg/pointers"
)

var _ = Describe("Status.Generated", func() {
	Context("generated and not generated", func() {
		type test struct {
			o  qsv1a1.QuarksSecretStatus
			g  bool
			ng bool
		}
		It("should always return false for nil", func() {
			tests := []test{
				{o: qsv1a1.QuarksSecretStatus{Generated: nil}, g: false, ng: false},
				{o: qsv1a1.QuarksSecretStatus{Generated: pointers.Bool(false)}, g: false, ng: true},
				{o: qsv1a1.QuarksSecretStatus{Generated: pointers.Bool(true)}, g: true, ng: false},
			}

			for _, t := range tests {
				Expect(t.o.NotGenerated()).To(Equal(t.ng), fmt.Sprintf(
					"expected NotGenerated to return %v for %v/%v\n",
					t.ng,
					t.o.Generated,
					t.o.Generated != nil && *t.o.Generated,
				))
				Expect(t.o.IsGenerated()).To(Equal(t.g), fmt.Sprintf(
					"expected IsGenerated to return %v for %v/%v\n",
					t.g,
					t.o.Generated,
					t.o.Generated != nil && *t.o.Generated))
			}
		})
	})

	Context("reconcileForGenerated", func() {
		type test struct {
			o qsv1a1.QuarksSecretStatus
			n qsv1a1.QuarksSecretStatus
			r bool
		}

		newTest := func(o, n, r bool) test {
			return test{
				o: qsv1a1.QuarksSecretStatus{Generated: pointers.Bool(o)},
				n: qsv1a1.QuarksSecretStatus{Generated: pointers.Bool(n)},
				r: r}
		}
		newTestP := func(o, n *bool, r bool) test {
			return test{
				o: qsv1a1.QuarksSecretStatus{Generated: o},
				n: qsv1a1.QuarksSecretStatus{Generated: n},
				r: r}
		}

		It("should honor results from the table in documentation", func() {
			tests := []test{
				newTest(true, true, false),
				newTest(false, true, false),
				newTestP(nil, pointers.Bool(true), false),
				newTest(true, false, true),
				newTest(false, false, true),
				newTestP(nil, pointers.Bool(false), true),
				newTestP(pointers.Bool(true), nil, false),
				newTestP(pointers.Bool(false), nil, true),
				newTestP(nil, nil, true),
			}

			for _, t := range tests {
				Expect(reconcileForGenerated(t.o, t.n)).To(Equal(t.r),
					fmt.Sprintf("for %v/%v|%v/%v expected result of %v\n",
						t.o.Generated, t.o.Generated != nil && *t.o.Generated,
						t.n.Generated, t.n.Generated != nil && *t.n.Generated,
						reconcileForGenerated(t.o, t.n),
					),
				)
			}
		})
	})
})
