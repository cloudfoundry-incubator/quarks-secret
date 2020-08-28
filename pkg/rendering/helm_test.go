package rendering_test

import (
	. "code.cloudfoundry.org/quarks-secret/pkg/rendering"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/chart"
)

var _ = Describe("Rendering", func() {
	var (
		engine RenderingEngine
	)
	values := map[string]interface{}{"outer": "DEFAULT", "inner": "DEFAULT"}
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
	Describe("Using helm engine", func() {

		It("render templates correctly", func() {

			helm := engine.(HelmRenderingEngine)
			files := []*chart.File{
				{Name: "templates/test1", Data: []byte("{{.Values.outer | title }} {{.Values.inner | title}}")},
				{Name: "templates/test2", Data: []byte("{{.Values.global.callme | lower }}")},
				{Name: "templates/test3", Data: []byte("{{.noValue}}")},
				{Name: "templates/test4", Data: []byte("{{toJson .Values}}")},
			}

			out := helm.RenderFiles(files, values, defaults)

			Expect(out).To(Equal(map[string]string{
				"templates/test1": "Spouter Inn",
				"templates/test2": "ishmael",
				"templates/test3": "",
				"templates/test4": `{"global":{"callme":"Ishmael"},"inner":"inn","outer":"spouter"}`,
			}))
		})

	})

	Describe("Render", func() {
		It("render templates correctly", func() {
			out := engine.Render("{{.Values.outer | title }} {{.Values.inner | title}}", defaults)
			Expect(out).To(Equal("Spouter Inn"))
		})
		It("render templates correctly", func() {
			out := engine.Render("{{.Values.global.callme | lower }}", defaults)
			Expect(out).To(Equal("ishmael"))
		})
		It("render templates correctly", func() {
			out := engine.Render("{{.noValue}}", defaults)
			Expect(out).To(Equal(""))
		})
		It("render templates correctly", func() {
			out := engine.Render("{{toJson .Values}}", defaults)
			Expect(out).To(Equal(`{"global":{"callme":"Ishmael"},"inner":"inn","outer":"spouter"}`))
		})
	})

	Describe("RenderMap", func() {
		It("render templates correctly", func() {
			out := engine.RenderMap(map[string]string{
				"test2": "{{.Values.global.callme | lower }}"}, defaults)
			Expect(out).To(Equal(map[string]string{"test2": "ishmael"}))
		})

	})

})
