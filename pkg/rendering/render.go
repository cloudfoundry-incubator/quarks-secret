package rendering

type RenderingEngine interface {
	Render(content string, values map[string]interface{}) string
}
