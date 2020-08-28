package rendering

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

const HelmEngine = "helm"

type HelmRenderingEngine struct{}

func NewHelmRenderingEngine() RenderingEngine {
	return HelmRenderingEngine{}
}

func (h HelmRenderingEngine) Render(content string, values map[string]interface{}) string {
	out := h.RenderFiles([]*chart.File{
		{Name: "templates", Data: []byte(content)},
	}, map[string]interface{}{}, values)

	return out["templates"]
}

// RenderMap renders the values passed from a map[string]string
func (h HelmRenderingEngine) RenderMap(contentMap map[string]string, values map[string]interface{}) map[string]string {
	for k, v := range contentMap {
		contentMap[k] = h.Render(v, values)
	}
	return contentMap
}

func (h HelmRenderingEngine) RenderFiles(files []*chart.File, values map[string]interface{}, defaults map[string]interface{}) map[string]string {
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
