package rendering

// Engine is an interface representing
// a generic rendering engine
type Engine interface {
	Render(content string, values map[string]interface{}) string
	RenderMap(contentMap map[string]string, values map[string]interface{}) map[string]string
}
