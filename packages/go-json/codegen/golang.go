package codegen

import (
	"fmt"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
)

// GoGenerator generates Go source code from a compiled go-json program.
type GoGenerator struct {
	PackageName string
}

func (g *GoGenerator) Language() string { return "go" }

func (g *GoGenerator) Generate(program *lang.CompiledProgram) (string, error) {
	var b strings.Builder
	pkg := g.PackageName
	if pkg == "" {
		pkg = "main"
	}

	b.WriteString(fmt.Sprintf("package %s\n\n", pkg))
	b.WriteString("import \"fmt\"\n\n")

	for name, fn := range program.Functions {
		g.generateFunc(&b, name, fn, 0)
		b.WriteString("\n")
	}

	b.WriteString("func main() {\n")
	g.generateSteps(&b, program.AST.Steps, 1)
	b.WriteString("}\n")

	return b.String(), nil
}

func (g *GoGenerator) generateFunc(b *strings.Builder, name string, fn *lang.CompiledFunc, level int) {
	params := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		goType := goTypeMap(p.Type)
		params[i] = fmt.Sprintf("%s %s", p.Name, goType)
	}

	retType := goTypeMap(fn.Returns)
	if retType == "" {
		retType = "any"
	}

	b.WriteString(fmt.Sprintf("%sfunc %s(%s) %s {\n", indent(level), name, strings.Join(params, ", "), retType))
	g.generateSteps(b, fn.Steps, level+1)
	b.WriteString(fmt.Sprintf("%s}\n", indent(level)))
}

func (g *GoGenerator) generateSteps(b *strings.Builder, steps []lang.Node, level int) {
	for _, step := range steps {
		comment := commentFromMeta(step.Meta())
		if comment != "" {
			b.WriteString(fmt.Sprintf("%s// %s\n", indent(level), comment))
		}
		g.generateStep(b, step, level)
	}
}

func (g *GoGenerator) generateStep(b *strings.Builder, node lang.Node, level int) {
	switch n := node.(type) {
	case *lang.LetNode:
		if n.Call != "" {
			b.WriteString(fmt.Sprintf("%s%s := %s(%s)\n", indent(level), n.Name, n.Call, formatGoArgs(n.CallWith)))
		} else if n.HasExpr {
			b.WriteString(fmt.Sprintf("%s%s := %s\n", indent(level), n.Name, n.Expr))
		} else if n.HasValue {
			b.WriteString(fmt.Sprintf("%s%s := %s\n", indent(level), n.Name, formatValue(n.Value)))
		}

	case *lang.SetNode:
		if n.HasExpr {
			b.WriteString(fmt.Sprintf("%s%s = %s\n", indent(level), n.Target, n.Expr))
		} else if n.HasValue {
			b.WriteString(fmt.Sprintf("%s%s = %s\n", indent(level), n.Target, formatValue(n.Value)))
		}

	case *lang.IfNode:
		b.WriteString(fmt.Sprintf("%sif %s {\n", indent(level), n.Condition))
		g.generateSteps(b, n.Then, level+1)
		for _, elif := range n.Elif {
			b.WriteString(fmt.Sprintf("%s} else if %s {\n", indent(level), elif.Condition))
			g.generateSteps(b, elif.Then, level+1)
		}
		if len(n.Else) > 0 {
			b.WriteString(fmt.Sprintf("%s} else {\n", indent(level)))
			g.generateSteps(b, n.Else, level+1)
		}
		b.WriteString(fmt.Sprintf("%s}\n", indent(level)))

	case *lang.SwitchNode:
		b.WriteString(fmt.Sprintf("%sswitch %s {\n", indent(level), n.Expr))
		for key, steps := range n.Cases {
			if key == "default" {
				b.WriteString(fmt.Sprintf("%sdefault:\n", indent(level)))
			} else {
				b.WriteString(fmt.Sprintf("%scase %q:\n", indent(level), key))
			}
			g.generateSteps(b, steps, level+1)
		}
		b.WriteString(fmt.Sprintf("%s}\n", indent(level)))

	case *lang.ForNode:
		if n.In != "" {
			idx := "_"
			if n.Index != "" {
				idx = n.Index
			}
			b.WriteString(fmt.Sprintf("%sfor %s, %s := range %s {\n", indent(level), idx, n.Variable, n.In))
		} else if n.Range != nil {
			b.WriteString(fmt.Sprintf("%sfor %s := %v; %s < %v; %s++ {\n",
				indent(level), n.Variable, n.Range[0], n.Variable, n.Range[1], n.Variable))
		}
		g.generateSteps(b, n.Steps, level+1)
		b.WriteString(fmt.Sprintf("%s}\n", indent(level)))

	case *lang.WhileNode:
		b.WriteString(fmt.Sprintf("%sfor %s {\n", indent(level), n.Condition))
		g.generateSteps(b, n.Steps, level+1)
		b.WriteString(fmt.Sprintf("%s}\n", indent(level)))

	case *lang.ReturnNode:
		if n.HasExpr {
			b.WriteString(fmt.Sprintf("%sreturn %s\n", indent(level), n.Expr))
		} else if n.HasValue {
			b.WriteString(fmt.Sprintf("%sreturn %s\n", indent(level), formatValue(n.Value)))
		} else {
			b.WriteString(fmt.Sprintf("%sreturn\n", indent(level)))
		}

	case *lang.CallNode:
		b.WriteString(fmt.Sprintf("%s%s(%s)\n", indent(level), n.Function, formatGoArgs(n.With)))

	case *lang.TryNode:
		b.WriteString(fmt.Sprintf("%sfunc() {\n", indent(level)))
		b.WriteString(fmt.Sprintf("%sdefer func() {\n", indent(level+1)))
		b.WriteString(fmt.Sprintf("%sif r := recover(); r != nil {\n", indent(level+2)))
		if n.Catch != nil {
			b.WriteString(fmt.Sprintf("%s%s := r\n", indent(level+3), n.Catch.As))
			b.WriteString(fmt.Sprintf("%s_ = %s\n", indent(level+3), n.Catch.As))
			g.generateSteps(b, n.Catch.Steps, level+3)
		}
		b.WriteString(fmt.Sprintf("%s}\n", indent(level+2)))
		b.WriteString(fmt.Sprintf("%s}()\n", indent(level+1)))
		g.generateSteps(b, n.Try, level+1)
		b.WriteString(fmt.Sprintf("%s}()\n", indent(level)))

	case *lang.ErrorNode:
		if n.IsStructured {
			b.WriteString(fmt.Sprintf("%spanic(fmt.Errorf(\"%s: %%s\", %s))\n", indent(level), n.Code, n.Message))
		} else {
			b.WriteString(fmt.Sprintf("%spanic(fmt.Errorf(\"%%s\", %s))\n", indent(level), n.Message))
		}

	case *lang.LogNode:
		if n.IsStructured {
			b.WriteString(fmt.Sprintf("%sfmt.Printf(\"[%%s] %%s\\n\", %s, %s)\n", indent(level), n.Level, n.Message))
		} else {
			b.WriteString(fmt.Sprintf("%sfmt.Println(%s)\n", indent(level), n.Message))
		}

	case *lang.ParallelNode:
		b.WriteString(fmt.Sprintf("%s// TODO: parallel execution — use sync.WaitGroup\n", indent(level)))
		for branchName, steps := range n.Branches {
			b.WriteString(fmt.Sprintf("%s// branch: %s\n", indent(level), branchName))
			g.generateSteps(b, steps, level)
		}

	case *lang.BreakNode:
		b.WriteString(fmt.Sprintf("%sbreak\n", indent(level)))

	case *lang.ContinueNode:
		b.WriteString(fmt.Sprintf("%scontinue\n", indent(level)))

	case *lang.CommentNode:
		comment := commentFromMeta(node.Meta())
		if comment != "" {
			b.WriteString(fmt.Sprintf("%s// %s\n", indent(level), comment))
		}
	}
}

func formatGoArgs(with map[string]string) string {
	if len(with) == 0 {
		return ""
	}
	args := make([]string, 0, len(with))
	for _, v := range with {
		args = append(args, v)
	}
	return strings.Join(args, ", ")
}

func goTypeMap(t string) string {
	switch t {
	case "string":
		return "string"
	case "int":
		return "int64"
	case "float":
		return "float64"
	case "bool":
		return "bool"
	case "any", "":
		return "any"
	case "map":
		return "map[string]any"
	default:
		if strings.HasPrefix(t, "[]") {
			return "[]" + goTypeMap(t[2:])
		}
		return t
	}
}
