package codegen

import (
	"fmt"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
)

// PythonGenerator generates Python source code from a compiled go-json program.
type PythonGenerator struct{}

func (g *PythonGenerator) Language() string { return "python" }

func (g *PythonGenerator) Generate(program *lang.CompiledProgram) (string, error) {
	var b strings.Builder

	for name, fn := range program.Functions {
		g.generateFunc(&b, name, fn, 0)
		b.WriteString("\n\n")
	}

	b.WriteString("if __name__ == \"__main__\":\n")
	if len(program.AST.Steps) == 0 {
		b.WriteString("    pass\n")
	} else {
		g.generateSteps(&b, program.AST.Steps, 1)
	}

	return b.String(), nil
}

func (g *PythonGenerator) generateFunc(b *strings.Builder, name string, fn *lang.CompiledFunc, level int) {
	params := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		pyType := pythonTypeMap(p.Type)
		if pyType != "" {
			params[i] = fmt.Sprintf("%s: %s", p.Name, pyType)
		} else {
			params[i] = p.Name
		}
	}

	retType := pythonTypeMap(fn.Returns)
	if retType != "" {
		b.WriteString(fmt.Sprintf("%sdef %s(%s) -> %s:\n", pyIndent(level), name, strings.Join(params, ", "), retType))
	} else {
		b.WriteString(fmt.Sprintf("%sdef %s(%s):\n", pyIndent(level), name, strings.Join(params, ", ")))
	}
	g.generateSteps(b, fn.Steps, level+1)
}

func (g *PythonGenerator) generateSteps(b *strings.Builder, steps []lang.Node, level int) {
	if len(steps) == 0 {
		b.WriteString(fmt.Sprintf("%spass\n", pyIndent(level)))
		return
	}
	for _, step := range steps {
		comment := commentFromMeta(step.Meta())
		if comment != "" {
			b.WriteString(fmt.Sprintf("%s# %s\n", pyIndent(level), comment))
		}
		g.generateStep(b, step, level)
	}
}

func (g *PythonGenerator) generateStep(b *strings.Builder, node lang.Node, level int) {
	switch n := node.(type) {
	case *lang.LetNode:
		if n.Call != "" {
			b.WriteString(fmt.Sprintf("%s%s = %s(%s)\n", pyIndent(level), n.Name, n.Call, formatPyArgs(n.CallWith)))
		} else if n.HasExpr {
			b.WriteString(fmt.Sprintf("%s%s = %s\n", pyIndent(level), n.Name, transformExpr(n.Expr, "python")))
		} else if n.HasValue {
			b.WriteString(fmt.Sprintf("%s%s = %s\n", pyIndent(level), n.Name, formatPyValue(n.Value)))
		}

	case *lang.SetNode:
		if n.HasExpr {
			b.WriteString(fmt.Sprintf("%s%s = %s\n", pyIndent(level), n.Target, transformExpr(n.Expr, "python")))
		} else if n.HasValue {
			b.WriteString(fmt.Sprintf("%s%s = %s\n", pyIndent(level), n.Target, formatPyValue(n.Value)))
		}

	case *lang.IfNode:
		b.WriteString(fmt.Sprintf("%sif %s:\n", pyIndent(level), transformExpr(n.Condition, "python")))
		g.generateSteps(b, n.Then, level+1)
		for _, elif := range n.Elif {
			b.WriteString(fmt.Sprintf("%selif %s:\n", pyIndent(level), transformExpr(elif.Condition, "python")))
			g.generateSteps(b, elif.Then, level+1)
		}
		if len(n.Else) > 0 {
			b.WriteString(fmt.Sprintf("%selse:\n", pyIndent(level)))
			g.generateSteps(b, n.Else, level+1)
		}

	case *lang.ForNode:
		if n.In != "" {
			if n.Index != "" {
				b.WriteString(fmt.Sprintf("%sfor %s, %s in enumerate(%s):\n", pyIndent(level), n.Index, n.Variable, n.In))
			} else {
				b.WriteString(fmt.Sprintf("%sfor %s in %s:\n", pyIndent(level), n.Variable, n.In))
			}
		} else if n.Range != nil {
			if len(n.Range) == 3 {
				b.WriteString(fmt.Sprintf("%sfor %s in range(%v, %v, %v):\n", pyIndent(level), n.Variable, n.Range[0], n.Range[1], n.Range[2]))
			} else {
				b.WriteString(fmt.Sprintf("%sfor %s in range(%v, %v):\n", pyIndent(level), n.Variable, n.Range[0], n.Range[1]))
			}
		}
		g.generateSteps(b, n.Steps, level+1)

	case *lang.WhileNode:
		b.WriteString(fmt.Sprintf("%swhile %s:\n", pyIndent(level), transformExpr(n.Condition, "python")))
		g.generateSteps(b, n.Steps, level+1)

	case *lang.ReturnNode:
		if n.HasExpr {
			b.WriteString(fmt.Sprintf("%sreturn %s\n", pyIndent(level), transformExpr(n.Expr, "python")))
		} else if n.HasValue {
			b.WriteString(fmt.Sprintf("%sreturn %s\n", pyIndent(level), formatPyValue(n.Value)))
		} else {
			b.WriteString(fmt.Sprintf("%sreturn\n", pyIndent(level)))
		}

	case *lang.CallNode:
		b.WriteString(fmt.Sprintf("%s%s(%s)\n", pyIndent(level), n.Function, formatPyArgs(n.With)))

	case *lang.TryNode:
		b.WriteString(fmt.Sprintf("%stry:\n", pyIndent(level)))
		g.generateSteps(b, n.Try, level+1)
		if n.Catch != nil {
			b.WriteString(fmt.Sprintf("%sexcept Exception as %s:\n", pyIndent(level), n.Catch.As))
			g.generateSteps(b, n.Catch.Steps, level+1)
		}
		if n.Finally != nil {
			b.WriteString(fmt.Sprintf("%sfinally:\n", pyIndent(level)))
			g.generateSteps(b, n.Finally, level+1)
		}

	case *lang.ErrorNode:
		b.WriteString(fmt.Sprintf("%sraise Exception(%s)\n", pyIndent(level), n.Message))

	case *lang.LogNode:
		b.WriteString(fmt.Sprintf("%sprint(%s)\n", pyIndent(level), n.Message))

	case *lang.ParallelNode:
		b.WriteString(fmt.Sprintf("%s# TODO: parallel execution — use asyncio.gather\n", pyIndent(level)))
		for _, steps := range n.Branches {
			g.generateSteps(b, steps, level)
		}

	case *lang.BreakNode:
		b.WriteString(fmt.Sprintf("%sbreak\n", pyIndent(level)))

	case *lang.ContinueNode:
		b.WriteString(fmt.Sprintf("%scontinue\n", pyIndent(level)))

	case *lang.CommentNode:
		comment := commentFromMeta(node.Meta())
		if comment != "" {
			b.WriteString(fmt.Sprintf("%s# %s\n", pyIndent(level), comment))
		}
	}
}

func pyIndent(level int) string {
	return strings.Repeat("    ", level)
}

func formatPyArgs(with map[string]string) string {
	if len(with) == 0 {
		return ""
	}
	args := make([]string, 0, len(with))
	for k, v := range with {
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(args, ", ")
}

func formatPyValue(v any) string {
	if v == nil {
		return "None"
	}
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case bool:
		if val {
			return "True"
		}
		return "False"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func pythonTypeMap(t string) string {
	switch t {
	case "string":
		return "str"
	case "int":
		return "int"
	case "float":
		return "float"
	case "bool":
		return "bool"
	case "any":
		return "Any"
	case "map":
		return "dict"
	case "":
		return ""
	default:
		if strings.HasPrefix(t, "[]") {
			inner := pythonTypeMap(t[2:])
			return fmt.Sprintf("list[%s]", inner)
		}
		return t
	}
}
