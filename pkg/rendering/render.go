package rendering

type RenderingEngine interface {
	Render(content string, values map[string]interface{}) string
	RenderMap(contentMap map[string]string, values map[string]interface{}) map[string]string
}
