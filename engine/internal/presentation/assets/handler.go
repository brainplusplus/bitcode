package assets

import (
	"net/http"
	"os"
	"path/filepath"
)

type Handler struct {
	componentDir string
	staticDir    string
}

func NewHandler(componentDir string, staticDir string) *Handler {
	return &Handler{
		componentDir: componentDir,
		staticDir:    staticDir,
	}
}

func (h *Handler) ServeComponents() http.Handler {
	if h.componentDir != "" {
		if _, err := os.Stat(h.componentDir); err == nil {
			return http.StripPrefix("/assets/components/", http.FileServer(http.Dir(h.componentDir)))
		}
	}
	return http.NotFoundHandler()
}

func (h *Handler) ServeStatic() http.Handler {
	if h.staticDir != "" {
		if _, err := os.Stat(h.staticDir); err == nil {
			return http.StripPrefix("/static/", http.FileServer(http.Dir(h.staticDir)))
		}
	}
	return http.NotFoundHandler()
}

func FindComponentDir() string {
	candidates := []string{
		"packages/components/dist/bc-components",
		"../packages/components/dist/bc-components",
		"static/components",
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}
	return ""
}

func ComponentScriptTag() string {
	return `<script type="module" src="/assets/components/bc-components.esm.js"></script>
<script nomodule src="/assets/components/bc-components.js"></script>`
}
