package codegen

import (
	"fmt"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
)

// JSGenerator generates JavaScript source code from a compiled go-json program.
type JSGenerator struct{}

func (g *JSGenerator) Language() string { return "javascript" }

func (g *JSGenerator) Generate(program *lang.CompiledProgram) (string, error) {
	var b strings.Builder

	for name, fn := range program.Functions {
		g.generateFunc(&b, name, fn, 0)
		b.WriteString("\n")
	}

	g.generateSteps(&b, program.AST.Steps, 0)

	return b.String(), nil
}

func (g *JSGenerator) generateFunc(b *strings.Builder, name string, fn *lang.CompiledFunc, level int) {
	params := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		params[i] = p.Name
	}

	b.WriteString(fmt.Sprintf("%sfunction %s(%s) {\n", indent(level), name, strings.Join(params, ", ")))
	g.generateSteps(b, fn.Steps, level+1)
	b.WriteString(fmt.Sprintf("%s}\n", indent(level)))
}

func (g *JSGenerator) generateSteps(b *strings.Builder, steps []lang.Node, level int) {
	for _, step := range steps {
		comment := commentFromMeta(step.Meta())
		if comment != "" {
			b.WriteString(fmt.Sprintf("%s// %s\n", indent(level), comment))
		}
		g.generateStep(b, step, level)
	}
}

func (g *JSGenerator) generateStep(b *strings.Builder, node lang.Node, level int) {
	switch n := node.(type) {
	case *lang.LetNode:
		if n.Call != "" {
			b.WriteString(fmt.Sprintf("%sconst %s = %s(%s);\n", indent(level), n.Name, n.Call, formatJSArgs(n.CallWith)))
		} else if n.HasExpr {
			b.WriteString(fmt.Sprintf("%sconst %s = %s;\n", indent(level), n.Name, transformExpr(n.Expr, "javascript")))
		} else if n.HasValue {
			b.WriteString(fmt.Sprintf("%sconst %s = %s;\n", indent(level), n.Name, formatJSValue(n.Value)))
		}

	case *lang.SetNode:
		if n.HasExpr {
			b.WriteString(fmt.Sprintf("%s%s = %s;\n", indent(level), n.Target, transformExpr(n.Expr, "javascript")))
		} else if n.HasValue {
			b.WriteString(fmt.Sprintf("%s%s = %s;\n", indent(level), n.Target, formatJSValue(n.Value)))
		}

	case *lang.IfNode:
		b.WriteString(fmt.Sprintf("%sif (%s) {\n", indent(level), n.Condition))
		g.generateSteps(b, n.Then, level+1)
		for _, elif := range n.Elif {
			b.WriteString(fmt.Sprintf("%s} else if (%s) {\n", indent(level), elif.Condition))
			g.generateSteps(b, elif.Then, level+1)
		}
		if len(n.Else) > 0 {
			b.WriteString(fmt.Sprintf("%s} else {\n", indent(level)))
			g.generateSteps(b, n.Else, level+1)
		}
		b.WriteString(fmt.Sprintf("%s}\n", indent(level)))

	case *lang.SwitchNode:
		b.WriteString(fmt.Sprintf("%sswitch (%s) {\n", indent(level), n.Expr))
		for key, steps := range n.Cases {
			if key == "default" {
				b.WriteString(fmt.Sprintf("%sdefault:\n", indent(level+1)))
			} else {
				b.WriteString(fmt.Sprintf("%scase %q:\n", indent(level+1), key))
			}
			g.generateSteps(b, steps, level+2)
			b.WriteString(fmt.Sprintf("%sbreak;\n", indent(level+2)))
		}
		b.WriteString(fmt.Sprintf("%s}\n", indent(level)))

	case *lang.ForNode:
		if n.In != "" {
			b.WriteString(fmt.Sprintf("%sfor (const %s of %s) {\n", indent(level), n.Variable, n.In))
		} else if n.Range != nil {
			b.WriteString(fmt.Sprintf("%sfor (let %s = %v; %s < %v; %s++) {\n",
				indent(level), n.Variable, n.Range[0], n.Variable, n.Range[1], n.Variable))
		}
		g.generateSteps(b, n.Steps, level+1)
		b.WriteString(fmt.Sprintf("%s}\n", indent(level)))

	case *lang.WhileNode:
		b.WriteString(fmt.Sprintf("%swhile (%s) {\n", indent(level), n.Condition))
		g.generateSteps(b, n.Steps, level+1)
		b.WriteString(fmt.Sprintf("%s}\n", indent(level)))

	case *lang.ReturnNode:
		if n.HasExpr {
			b.WriteString(fmt.Sprintf("%sreturn %s;\n", indent(level), transformExpr(n.Expr, "javascript")))
		} else if n.HasValue {
			b.WriteString(fmt.Sprintf("%sreturn %s;\n", indent(level), formatJSValue(n.Value)))
		} else {
			b.WriteString(fmt.Sprintf("%sreturn;\n", indent(level)))
		}

	case *lang.CallNode:
		b.WriteString(fmt.Sprintf("%s%s(%s);\n", indent(level), n.Function, formatJSArgs(n.With)))

	case *lang.TryNode:
		b.WriteString(fmt.Sprintf("%stry {\n", indent(level)))
		g.generateSteps(b, n.Try, level+1)
		if n.Catch != nil {
			b.WriteString(fmt.Sprintf("%s} catch (%s) {\n", indent(level), n.Catch.As))
			g.generateSteps(b, n.Catch.Steps, level+1)
		}
		if n.Finally != nil {
			b.WriteString(fmt.Sprintf("%s} finally {\n", indent(level)))
			g.generateSteps(b, n.Finally, level+1)
		}
		b.WriteString(fmt.Sprintf("%s}\n", indent(level)))

	case *lang.ErrorNode:
		b.WriteString(fmt.Sprintf("%sthrow new Error(%s);\n", indent(level), n.Message))

	case *lang.LogNode:
		b.WriteString(fmt.Sprintf("%sconsole.log(%s);\n", indent(level), n.Message))

	case *lang.ParallelNode:
		b.WriteString(fmt.Sprintf("%sawait Promise.all([\n", indent(level)))
		for _, steps := range n.Branches {
			b.WriteString(fmt.Sprintf("%s(async () => {\n", indent(level+1)))
			g.generateSteps(b, steps, level+2)
			b.WriteString(fmt.Sprintf("%s})(),\n", indent(level+1)))
		}
		b.WriteString(fmt.Sprintf("%s]);\n", indent(level)))

	case *lang.BreakNode:
		b.WriteString(fmt.Sprintf("%sbreak;\n", indent(level)))

	case *lang.ContinueNode:
		b.WriteString(fmt.Sprintf("%scontinue;\n", indent(level)))

	case *lang.CommentNode:
		comment := commentFromMeta(node.Meta())
		if comment != "" {
			b.WriteString(fmt.Sprintf("%s// %s\n", indent(level), comment))
		}
	}
}

func formatJSArgs(with map[string]string) string {
	if len(with) == 0 {
		return ""
	}
	args := make([]string, 0, len(with))
	for _, v := range with {
		args = append(args, v)
	}
	return strings.Join(args, ", ")
}

func formatJSValue(v any) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", val)
	}
}
