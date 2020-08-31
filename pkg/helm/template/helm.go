package template

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

// Template is a rendering engine based on helm, which can render templates in a map
type Template struct{}

// New returns a new Helm engine
func New() Template {
	return Template{}
}

// ExecuteMap renders the templates with the passed in values
func (h Template) ExecuteMap(templates map[string]string, values map[string]interface{}) map[string]string {
	rendered := make(map[string]string, len(templates))
	vals := map[string]interface{}{"Values": values}
	for k, template := range templates {
		rendered[k] = h.render(template, vals)
	}
	return rendered
}

// render renders the template interpolating the supplied values
func (h Template) render(template string, values map[string]interface{}) string {
	out := h.renderFiles([]*chart.File{
		{Name: "templates", Data: []byte(template)},
	}, map[string]interface{}{}, values)

	return out["templates"]
}

// renderFiles uses helms chartutil to render the passed in chart.Files
func (h Template) renderFiles(files []*chart.File, values map[string]interface{}, defaults map[string]interface{}) map[string]string {
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
