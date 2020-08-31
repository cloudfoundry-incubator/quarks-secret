package rendering_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "code.cloudfoundry.org/quarks-secret/pkg/rendering"
)

var _ = Describe("Template", func() {
	var (
		engine HelmRenderingEngine
	)
	defaults := map[string]interface{}{
		"Values": map[string]interface{}{
			"outer": "spouter",
			"inner": "inn",
			"global": map[string]interface{}{
				"callme": "Ishmael",
			},
		},
	}

	BeforeEach(func() {
		engine = NewHelmRenderingEngine()
	})

	Describe("RenderMap", func() {
		It("renders helm template functions and values", func() {
			out := engine.RenderMap(map[string]string{
				"test2": "{{.Values.global.callme | lower }}",
			}, defaults)
			Expect(out).To(HaveLen(1))
			Expect(out).To(HaveKeyWithValue("test2", "ishmael"))

			out = engine.RenderMap(map[string]string{
				"test": "{{toJson .Values}}",
			}, defaults)
			Expect(out).To(HaveKeyWithValue("test", `{"global":{"callme":"Ishmael"},"inner":"inn","outer":"spouter"}`))
		})

		It("renders multiple templates", func() {
			out := engine.RenderMap(map[string]string{
				"test1": "{{.Values.outer | title }} {{.Values.inner | title}}",
				"test3": "{{.noValue}}",
				"test4": "{{toJson .Values}}",
			}, defaults)
			Expect(out).To(HaveLen(3))
			Expect(out).To(HaveKeyWithValue("test1", "Spouter Inn"))
			Expect(out).To(HaveKeyWithValue("test3", ""))
			Expect(out).To(HaveKeyWithValue("test4", `{"global":{"callme":"Ishmael"},"inner":"inn","outer":"spouter"}`))
		})

		It("renders an empty string", func() {
			out := engine.RenderMap(map[string]string{
				"test": "{{.noValue}}",
			}, defaults)
			Expect(out).To(HaveKeyWithValue("test", ""))
		})
	})
})
