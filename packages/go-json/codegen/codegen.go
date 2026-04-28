package codegen

import (
	"fmt"
	"strings"

	"github.com/bitcode-framework/go-json/lang"
)

// CodeGenerator generates source code from a compiled go-json program.
type CodeGenerator interface {
	Generate(program *lang.CompiledProgram) (string, error)
	Language() string
}

func indent(level int) string {
	return strings.Repeat("\t", level)
}

func indentSpaces(level int, spaces int) string {
	return strings.Repeat(strings.Repeat(" ", spaces), level)
}

func commentFromMeta(meta *lang.NodeMeta) string {
	if meta.Comment != "" {
		return meta.Comment
	}
	if len(meta.Comments) > 0 {
		return strings.Join(meta.Comments, "\n")
	}
	return ""
}

func stepsToNodes(steps []lang.Node) []lang.Node {
	return steps
}

// NodeType constants matching the design doc §4.3.
const (
	NodeProgram    = "program"
	NodeLet        = "let"
	NodeSet        = "set"
	NodeIf         = "if"
	NodeSwitch     = "switch"
	NodeForIn      = "for"
	NodeWhile      = "while"
	NodeBreak      = "break"
	NodeContinue   = "continue"
	NodeReturn     = "return"
	NodeCall       = "call"
	NodeTry        = "try"
	NodeError      = "error"
	NodeLog        = "log"
	NodeParallel   = "parallel"
	NodeComment    = "comment"
)

func transformExpr(expr string, lang string) string {
	if lang == "go" {
		return expr
	}
	if lang == "javascript" {
		expr = strings.ReplaceAll(expr, "&&", "&&")
		expr = strings.ReplaceAll(expr, "||", "||")
		return expr
	}
	if lang == "python" {
		expr = strings.ReplaceAll(expr, "&&", " and ")
		expr = strings.ReplaceAll(expr, "||", " or ")
		expr = strings.ReplaceAll(expr, "!", "not ")
		expr = strings.ReplaceAll(expr, "true", "True")
		expr = strings.ReplaceAll(expr, "false", "False")
		expr = strings.ReplaceAll(expr, "nil", "None")
		return expr
	}
	return expr
}

func formatValue(v any) string {
	if v == nil {
		return "nil"
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
