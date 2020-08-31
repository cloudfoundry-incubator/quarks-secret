package template_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "code.cloudfoundry.org/quarks-secret/pkg/helm/template"
)

var _ = Describe("Template", func() {
	var (
		engine Template
	)
	defaults := map[string]interface{}{
		"outer": "spouter",
		"inner": "inn",
		"global": map[string]interface{}{
			"callme": "Ishmael",
		},
	}

	BeforeEach(func() {
		engine = New()
	})

	act := func(templates map[string]string) map[string]string {
		return engine.ExecuteMap(templates, defaults)
	}

	Describe("RenderMap", func() {
		It("renders helm template functions and values", func() {
			out := act(map[string]string{
				"test2": "{{.Values.global.callme | lower }}",
			})
			Expect(out).To(HaveLen(1))
			Expect(out).To(HaveKeyWithValue("test2", "ishmael"))

			out = act(map[string]string{
				"test": "{{toJson .Values}}",
			})
			Expect(out).To(HaveKeyWithValue("test", `{"global":{"callme":"Ishmael"},"inner":"inn","outer":"spouter"}`))
		})

		It("renders multiple templates", func() {
			out := act(map[string]string{
				"test1": "{{.Values.outer | title }} {{.Values.inner | title}}",
				"test3": "{{.noValue}}",
				"test4": "{{toJson .Values}}",
			})
			Expect(out).To(HaveLen(3))
			Expect(out).To(HaveKeyWithValue("test1", "Spouter Inn"))
			Expect(out).To(HaveKeyWithValue("test3", ""))
			Expect(out).To(HaveKeyWithValue("test4", `{"global":{"callme":"Ishmael"},"inner":"inn","outer":"spouter"}`))
		})

		It("renders an empty string", func() {
			out := act(map[string]string{
				"test": "{{.noValue}}",
			})
			Expect(out).To(HaveKeyWithValue("test", ""))
		})
	})
})
