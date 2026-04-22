package template

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Engine struct {
	templates map[string]*template.Template
	partials  map[string]string
	funcMap   template.FuncMap
	mu        sync.RWMutex
}

func NewEngine() *Engine {
	return &Engine{
		templates: make(map[string]*template.Template),
		partials:  make(map[string]string),
		funcMap:   defaultFuncMap(),
	}
}

func defaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04")
		},
		"formatCurrency": func(amount float64) string {
			return fmt.Sprintf("$%.2f", amount)
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"dict": func(values ...any) map[string]any {
			if len(values)%2 != 0 {
				return nil
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				m[key] = values[i+1]
			}
			return m
		},
		"eq": func(a, b any) bool {
			return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
		},
		"neq": func(a, b any) bool {
			return fmt.Sprintf("%v", a) != fmt.Sprintf("%v", b)
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"seq": func(start, end int) []int {
			var result []int
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
			return result
		},
		"join":      strings.Join,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"default": func(def, val any) any {
			if val == nil || val == "" || val == 0 || val == false {
				return def
			}
			return val
		},
	}
}

func (e *Engine) RegisterHelper(name string, fn any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.funcMap[name] = fn
}

func (e *Engine) LoadDirectory(dir string) error {
	return e.LoadDirectoryWithPrefix(dir, "")
}

func (e *Engine) LoadDirectoryWithPrefix(dir string, prefix string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	type fileEntry struct {
		relPath string
		content string
	}

	var partialFiles []fileEntry
	var templateFiles []fileEntry

	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		relPath, _ := filepath.Rel(dir, path)
		relPath = filepath.ToSlash(relPath)
		if prefix != "" {
			relPath = prefix + "/" + relPath
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		entry := fileEntry{relPath: relPath, content: string(content)}
		if strings.Contains(relPath, "partials/") {
			partialFiles = append(partialFiles, entry)
		} else {
			templateFiles = append(templateFiles, entry)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, pf := range partialFiles {
		e.partials[pf.relPath] = pf.content
	}

	for _, tf := range templateFiles {
		tmpl := e.buildTemplateWithPartials(tf.relPath, tf.content)
		if tmpl == nil {
			return fmt.Errorf("failed to parse template %s", tf.relPath)
		}
		e.templates[tf.relPath] = tmpl
	}

	return nil
}

func (e *Engine) buildTemplateWithPartials(name string, content string) *template.Template {
	tmpl := template.New(name).Funcs(e.funcMap)

	for pName, pContent := range e.partials {
		var err error
		tmpl, err = tmpl.New(pName).Parse(pContent)
		if err != nil {
			continue
		}
	}

	var err error
	tmpl, err = tmpl.New(name).Parse(content)
	if err != nil {
		return nil
	}

	return tmpl
}

func (e *Engine) rebuildTemplatesLocked() {
	rebuilt := make(map[string]*template.Template)
	for name := range e.templates {
		if strings.Contains(name, "partials/") {
			continue
		}
		oldTmpl := e.templates[name]
		if oldTmpl == nil {
			continue
		}

		src := templateSource(oldTmpl, name)
		if src == "" {
			rebuilt[name] = oldTmpl
			continue
		}

		newTmpl := e.buildTemplateWithPartials(name, src)
		if newTmpl != nil {
			rebuilt[name] = newTmpl
		} else {
			rebuilt[name] = oldTmpl
		}
	}
	e.templates = rebuilt
}

func templateSource(t *template.Template, name string) string {
	if t == nil {
		return ""
	}
	found := t.Lookup(name)
	if found == nil {
		return ""
	}
	return found.Tree.Root.String()
}

func (e *Engine) LoadString(name string, content string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if strings.Contains(name, "partials/") {
		e.partials[name] = content
		e.rebuildTemplatesLocked()
		return nil
	}

	tmpl := e.buildTemplateWithPartials(name, content)
	if tmpl == nil {
		return fmt.Errorf("failed to parse template %s", name)
	}

	e.templates[name] = tmpl
	return nil
}

func (e *Engine) Render(name string, data any) (string, error) {
	e.mu.RLock()
	tmpl, ok := e.templates[name]
	e.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("template %q not found", name)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", name, err)
	}
	return buf.String(), nil
}

func (e *Engine) RenderWithLayout(layoutName string, contentName string, data map[string]any) (string, error) {
	contentHTML, err := e.Render(contentName, data)
	if err != nil {
		return "", fmt.Errorf("failed to render content %s: %w", contentName, err)
	}

	layoutData := make(map[string]any)
	for k, v := range data {
		layoutData[k] = v
	}
	layoutData["Content"] = template.HTML(contentHTML)

	return e.Render(layoutName, layoutData)
}

func (e *Engine) RenderString(content string, data any) (string, error) {
	e.mu.RLock()
	funcMap := e.funcMap
	partials := make(map[string]string, len(e.partials))
	for k, v := range e.partials {
		partials[k] = v
	}
	e.mu.RUnlock()

	tmpl := template.New("inline").Funcs(funcMap)
	for pName, pContent := range partials {
		tmpl, _ = tmpl.New(pName).Parse(pContent)
	}
	tmpl, err := tmpl.New("inline").Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse inline template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "inline", data); err != nil {
		return "", fmt.Errorf("failed to render inline template: %w", err)
	}
	return buf.String(), nil
}

func (e *Engine) Has(name string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, ok := e.templates[name]
	return ok
}

func (e *Engine) HasPartial(name string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, ok := e.partials[name]
	return ok
}

func (e *Engine) ListTemplates() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var names []string
	for name := range e.templates {
		names = append(names, name)
	}
	return names
}
