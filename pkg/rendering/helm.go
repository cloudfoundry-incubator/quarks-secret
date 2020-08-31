package rendering

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

// HelmEngine is the helm engine constant to identify the engine type
const HelmEngine = "helm"

// HelmRenderingEngine is a rendering engine based on helm
type HelmRenderingEngine struct{}

// NewHelmRenderingEngine returns a new Helm engine
func NewHelmRenderingEngine() HelmRenderingEngine {
	return HelmRenderingEngine{}
}

// RenderMap renders the values passed from a map[string]string
func (h HelmRenderingEngine) RenderMap(contentMap map[string]string, values map[string]interface{}) map[string]string {
	for k, v := range contentMap {
		contentMap[k] = h.render(v, values)
	}
	return contentMap
}

// render renders the template interpolating the supplied values
func (h HelmRenderingEngine) render(content string, values map[string]interface{}) string {
	out := h.renderFiles([]*chart.File{
		{Name: "templates", Data: []byte(content)},
	}, map[string]interface{}{}, values)

	return out["templates"]
}

// renderFiles render files as they where charts with helm
func (h HelmRenderingEngine) renderFiles(files []*chart.File, values map[string]interface{}, defaults map[string]interface{}) map[string]string {
	c := &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    "",
			Version: "",
		},
		Templates: files,
		Values:    values,
	}

	v, err := chartutil.CoalesceValues(c, defaults)
	if err != nil {
		fmt.Println(err)
	}
	out, err := engine.Render(c, v)
	if err != nil {
		fmt.Println(err)
	}

	return out
}
