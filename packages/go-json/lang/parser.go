package lang

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Parse parses JSONC/JSON input into a Program AST.
func Parse(input []byte) (*Program, error) {
	cleaned := StripComments(input)

	var raw map[string]any
	if err := json.Unmarshal(cleaned, &raw); err != nil {
		return nil, CompileError("JSON_PARSE", "invalid JSON: "+err.Error(), -1)
	}

	return parseProgram(raw, cleaned)
}

// ParseFile reads a file and parses it.
func ParseFile(path string) (*Program, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, CompileError("FILE_READ", "cannot read file: "+err.Error(), -1)
	}
	return Parse(data)
}

func parseProgram(raw map[string]any, cleanedJSON []byte) (*Program, error) {
	prog := &Program{
		Functions: make(map[string]*FuncDef),
	}

	if name, ok := raw["name"].(string); ok {
		prog.Name = name
	}
	if ver, ok := raw["go_json"].(string); ok {
		prog.GoJSON = ver
	}

	parseComment(&prog.NodeMeta, raw)

	// Parse input schema.
	if inputRaw, ok := raw["input"].(map[string]any); ok {
		for name, typ := range inputRaw {
			typStr, _ := typ.(string)
			prog.Input = append(prog.Input, InputField{Name: name, Type: TypeFromJSON(typStr)})
		}
	}

	// Parse imports.
	if importsRaw, ok := raw["imports"].([]any); ok {
		for _, imp := range importsRaw {
			if s, ok := imp.(string); ok {
				prog.Imports = append(prog.Imports, s)
			}
		}
	}

	// Parse limits.
	if limitsRaw, ok := raw["limits"].(map[string]any); ok {
		prog.Limits = parseLimits(limitsRaw)
	}

	// Parse functions — need ordered params from raw JSON.
	if funcsRaw, ok := raw["functions"].(map[string]any); ok {
		for name, fRaw := range funcsRaw {
			fMap, ok := fRaw.(map[string]any)
			if !ok {
				return nil, CompileError("INVALID_FUNC", fmt.Sprintf("function '%s' must be an object", name), -1)
			}
			fd, err := parseFuncDef(name, fMap, cleanedJSON)
			if err != nil {
				return nil, err
			}
			prog.Functions[name] = fd
		}
	}

	// Parse steps.
	if stepsRaw, ok := raw["steps"].([]any); ok {
		steps, err := parseSteps(stepsRaw)
		if err != nil {
			return nil, err
		}
		prog.Steps = steps
	}

	return prog, nil
}

func parseSteps(rawSteps []any) ([]Node, error) {
	if len(rawSteps) == 0 {
		return nil, nil
	}

	nodes := make([]Node, 0, len(rawSteps))
	for i, raw := range rawSteps {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, CompileError("INVALID_STEP", fmt.Sprintf("step %d must be an object", i), i)
		}

		node, err := parseStep(m, i)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func parseStep(m map[string]any, index int) (Node, error) {
	// Check for standalone comment first.
	if isCommentOnly(m) {
		node := &CommentNode{}
		node.StepIndex = index
		parseComment(&node.NodeMeta, m)
		return node, nil
	}

	// Detect step type by key presence (priority order).
	if _, ok := m["let"]; ok {
		// Check for call shorthand: {"let": "x", "call": "fn", ...}
		if _, hasCall := m["call"]; hasCall {
			return parseLetCallNode(m, index)
		}
		return parseLetNode(m, index)
	}
	if _, ok := m["set"]; ok {
		return parseSetNode(m, index)
	}
	if _, ok := m["if"]; ok {
		return parseIfNode(m, index)
	}
	if _, ok := m["switch"]; ok {
		return parseSwitchNode(m, index)
	}
	if _, ok := m["for"]; ok {
		return parseForNode(m, index)
	}
	if _, ok := m["while"]; ok {
		return parseWhileNode(m, index)
	}
	if _, ok := m["return"]; ok {
		return parseReturnNode(m, index)
	}
	if _, ok := m["call"]; ok {
		return parseCallNode(m, index)
	}
	if _, ok := m["try"]; ok {
		return parseTryNode(m, index)
	}
	if _, ok := m["error"]; ok {
		return parseErrorNode(m, index)
	}
	if _, ok := m["log"]; ok {
		return parseLogNode(m, index)
	}
	if _, ok := m["break"]; ok {
		return parseBreakNode(m, index)
	}
	if _, ok := m["continue"]; ok {
		return parseContinueNode(m, index)
	}

	// Unknown step type.
	keys := make([]string, 0, len(m))
	for k := range m {
		if k != "_c" {
			keys = append(keys, k)
		}
	}
	return nil, CompileError("UNKNOWN_STEP",
		fmt.Sprintf("unknown step type at step %d (keys: %s)", index, strings.Join(keys, ", ")), index)
}

// --- Individual step parsers ---

func parseLetNode(m map[string]any, index int) (*LetNode, error) {
	node := &LetNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	name, ok := m["let"].(string)
	if !ok {
		return nil, CompileError("INVALID_LET", "let name must be a string", index)
	}
	node.Name = name

	if typ, ok := m["type"].(string); ok {
		node.Type = TypeFromJSON(typ)
	}

	modes := 0
	if _, ok := m["value"]; ok {
		node.Value = m["value"]
		node.HasValue = true
		modes++
	}
	if expr, ok := m["expr"].(string); ok {
		node.Expr = expr
		node.HasExpr = true
		modes++
	}
	if withRaw, ok := m["with"].(map[string]any); ok {
		node.With = toStringMap(withRaw)
		node.HasWith = true
		modes++
	}

	if modes == 0 {
		return nil, CompileError("MISSING_VALUE", fmt.Sprintf("let '%s' requires one of: value, expr, with", name), index)
	}
	if modes > 1 {
		return nil, CompileError("MULTIPLE_VALUES", fmt.Sprintf("let '%s' has multiple value modes (use exactly one of: value, expr, with)", name), index)
	}

	return node, nil
}

func parseLetCallNode(m map[string]any, index int) (*LetNode, error) {
	node := &LetNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	name, ok := m["let"].(string)
	if !ok {
		return nil, CompileError("INVALID_LET", "let name must be a string", index)
	}
	node.Name = name

	if typ, ok := m["type"].(string); ok {
		node.Type = TypeFromJSON(typ)
	}

	callName, ok := m["call"].(string)
	if !ok {
		return nil, CompileError("INVALID_CALL", "call function name must be a string", index)
	}
	node.Call = callName

	if withRaw, ok := m["with"].(map[string]any); ok {
		node.CallWith = toStringMap(withRaw)
	}

	return node, nil
}

func parseSetNode(m map[string]any, index int) (*SetNode, error) {
	node := &SetNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	target, ok := m["set"].(string)
	if !ok {
		return nil, CompileError("INVALID_SET", "set target must be a string", index)
	}
	node.Target = target

	modes := 0
	if _, ok := m["value"]; ok {
		node.Value = m["value"]
		node.HasValue = true
		modes++
	}
	if expr, ok := m["expr"].(string); ok {
		node.Expr = expr
		node.HasExpr = true
		modes++
	}
	if withRaw, ok := m["with"].(map[string]any); ok {
		node.With = toStringMap(withRaw)
		node.HasWith = true
		modes++
	}

	if modes == 0 {
		return nil, CompileError("MISSING_VALUE", fmt.Sprintf("set '%s' requires one of: value, expr, with", target), index)
	}
	if modes > 1 {
		return nil, CompileError("MULTIPLE_VALUES", fmt.Sprintf("set '%s' has multiple value modes", target), index)
	}

	return node, nil
}

func parseIfNode(m map[string]any, index int) (*IfNode, error) {
	node := &IfNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	cond, ok := m["if"].(string)
	if !ok {
		return nil, CompileError("INVALID_IF", "if condition must be a string expression", index)
	}
	node.Condition = cond

	// Parse then (required).
	thenRaw, ok := m["then"].([]any)
	if !ok {
		return nil, CompileError("MISSING_THEN", "if requires 'then' steps array", index)
	}
	thenSteps, err := parseSteps(thenRaw)
	if err != nil {
		return nil, err
	}
	node.Then = thenSteps

	// Parse elif (optional).
	if elifRaw, ok := m["elif"].([]any); ok {
		for i, eRaw := range elifRaw {
			eMap, ok := eRaw.(map[string]any)
			if !ok {
				return nil, CompileError("INVALID_ELIF", fmt.Sprintf("elif[%d] must be an object", i), index)
			}
			cond, ok := eMap["condition"].(string)
			if !ok {
				return nil, CompileError("INVALID_ELIF", fmt.Sprintf("elif[%d] requires 'condition' string", i), index)
			}
			thenRaw, ok := eMap["then"].([]any)
			if !ok {
				return nil, CompileError("INVALID_ELIF", fmt.Sprintf("elif[%d] requires 'then' steps array", i), index)
			}
			thenSteps, err := parseSteps(thenRaw)
			if err != nil {
				return nil, err
			}
			node.Elif = append(node.Elif, ElifBlock{Condition: cond, Then: thenSteps})
		}
	}

	// Parse else (optional).
	if elseRaw, ok := m["else"].([]any); ok {
		elseSteps, err := parseSteps(elseRaw)
		if err != nil {
			return nil, err
		}
		node.Else = elseSteps
	}

	return node, nil
}

func parseSwitchNode(m map[string]any, index int) (*SwitchNode, error) {
	node := &SwitchNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	exprStr, ok := m["switch"].(string)
	if !ok {
		return nil, CompileError("INVALID_SWITCH", "switch expression must be a string", index)
	}
	node.Expr = exprStr

	casesRaw, ok := m["cases"].(map[string]any)
	if !ok {
		return nil, CompileError("MISSING_CASES", "switch requires 'cases' object", index)
	}

	node.Cases = make(map[string][]Node)
	for key, stepsRaw := range casesRaw {
		stepsArr, ok := stepsRaw.([]any)
		if !ok {
			return nil, CompileError("INVALID_CASE", fmt.Sprintf("case '%s' must be a steps array", key), index)
		}
		steps, err := parseSteps(stepsArr)
		if err != nil {
			return nil, err
		}
		node.Cases[key] = steps
	}

	return node, nil
}

func parseForNode(m map[string]any, index int) (*ForNode, error) {
	node := &ForNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	varName, ok := m["for"].(string)
	if !ok {
		return nil, CompileError("INVALID_FOR", "for variable must be a string", index)
	}
	node.Variable = varName

	if inExpr, ok := m["in"].(string); ok {
		node.In = inExpr
	}
	if rangeRaw, ok := m["range"].([]any); ok {
		node.Range = rangeRaw
	}

	if node.In == "" && node.Range == nil {
		return nil, CompileError("MISSING_ITERABLE", "for requires 'in' expression or 'range' array", index)
	}

	if idxName, ok := m["index"].(string); ok {
		node.Index = idxName
	}

	stepsRaw, ok := m["steps"].([]any)
	if !ok {
		return nil, CompileError("MISSING_STEPS", "for requires 'steps' array", index)
	}
	steps, err := parseSteps(stepsRaw)
	if err != nil {
		return nil, err
	}
	node.Steps = steps

	return node, nil
}

func parseWhileNode(m map[string]any, index int) (*WhileNode, error) {
	node := &WhileNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	cond, ok := m["while"].(string)
	if !ok {
		return nil, CompileError("INVALID_WHILE", "while condition must be a string expression", index)
	}
	node.Condition = cond

	stepsRaw, ok := m["steps"].([]any)
	if !ok {
		return nil, CompileError("MISSING_STEPS", "while requires 'steps' array", index)
	}
	steps, err := parseSteps(stepsRaw)
	if err != nil {
		return nil, err
	}
	node.Steps = steps

	return node, nil
}

func parseReturnNode(m map[string]any, index int) (*ReturnNode, error) {
	node := &ReturnNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	retVal := m["return"]

	switch v := retVal.(type) {
	case string:
		// Expression mode (most common): {"return": "expr"}
		node.Expr = v
		node.HasExpr = true
	case map[string]any:
		// Object mode: {"return": {"value": ...}} or {"return": {"expr": ...}} or {"return": {"with": ...}}
		if val, ok := v["value"]; ok {
			node.Value = val
			node.HasValue = true
		} else if expr, ok := v["expr"].(string); ok {
			node.Expr = expr
			node.HasExpr = true
		} else if withRaw, ok := v["with"].(map[string]any); ok {
			node.With = toStringMap(withRaw)
			node.HasWith = true
		} else {
			// Literal map return: {"return": {"status": "ok"}}
			node.Value = v
			node.HasValue = true
		}
	default:
		// Literal return: {"return": 42}, {"return": null}, {"return": true}
		node.Value = retVal
		node.HasValue = true
	}

	return node, nil
}

func parseCallNode(m map[string]any, index int) (*CallNode, error) {
	node := &CallNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	funcName, ok := m["call"].(string)
	if !ok {
		return nil, CompileError("INVALID_CALL", "call function name must be a string", index)
	}
	node.Function = funcName

	if withRaw, ok := m["with"].(map[string]any); ok {
		node.With = toStringMap(withRaw)
	}

	return node, nil
}

func parseTryNode(m map[string]any, index int) (*TryNode, error) {
	node := &TryNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	tryRaw, ok := m["try"].([]any)
	if !ok {
		return nil, CompileError("INVALID_TRY", "try must be a steps array", index)
	}
	trySteps, err := parseSteps(tryRaw)
	if err != nil {
		return nil, err
	}
	node.Try = trySteps

	if catchRaw, ok := m["catch"].(map[string]any); ok {
		cb := &CatchBlock{}
		if as, ok := catchRaw["as"].(string); ok {
			cb.As = as
		} else {
			cb.As = "err"
		}
		if stepsRaw, ok := catchRaw["steps"].([]any); ok {
			steps, err := parseSteps(stepsRaw)
			if err != nil {
				return nil, err
			}
			cb.Steps = steps
		}
		node.Catch = cb
	}

	if finallyRaw, ok := m["finally"].([]any); ok {
		steps, err := parseSteps(finallyRaw)
		if err != nil {
			return nil, err
		}
		node.Finally = steps
	}

	return node, nil
}

func parseErrorNode(m map[string]any, index int) (*ErrorNode, error) {
	node := &ErrorNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	errVal := m["error"]

	switch v := errVal.(type) {
	case string:
		node.Message = v
		node.IsStructured = false
	case map[string]any:
		node.IsStructured = true
		if code, ok := v["code"].(string); ok {
			node.Code = code
		}
		if msg, ok := v["message"].(string); ok {
			node.Message = msg
		}
		if details, ok := v["details"].(string); ok {
			node.Details = details
		}
	default:
		return nil, CompileError("INVALID_ERROR", "error must be a string expression or structured object", index)
	}

	return node, nil
}

func parseLogNode(m map[string]any, index int) (*LogNode, error) {
	node := &LogNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	logVal := m["log"]

	switch v := logVal.(type) {
	case string:
		node.Message = v
		node.IsStructured = false
	case map[string]any:
		node.IsStructured = true
		if msg, ok := v["message"].(string); ok {
			node.Message = msg
		}
		if level, ok := v["level"].(string); ok {
			node.Level = level
		}
		if dataRaw, ok := v["data"].(map[string]any); ok {
			node.Data = toStringMap(dataRaw)
		}
	default:
		return nil, CompileError("INVALID_LOG", "log must be a string expression or structured object", index)
	}

	return node, nil
}

func parseBreakNode(m map[string]any, index int) (*BreakNode, error) {
	node := &BreakNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)
	return node, nil
}

func parseContinueNode(m map[string]any, index int) (*ContinueNode, error) {
	node := &ContinueNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)
	return node, nil
}

// --- Function parsing ---

func parseFuncDef(name string, raw map[string]any, cleanedJSON []byte) (*FuncDef, error) {
	fd := &FuncDef{Name: name}

	if ret, ok := raw["returns"].(string); ok {
		fd.Returns = TypeFromJSON(ret)
	}

	// Parse params with preserved order.
	if paramsRaw, ok := raw["params"].(map[string]any); ok {
		orderedKeys := extractOrderedKeys(cleanedJSON, name, "params")
		if orderedKeys == nil {
			// Fallback: use map iteration order (non-deterministic but functional).
			for pName, pType := range paramsRaw {
				typStr, _ := pType.(string)
				fd.Params = append(fd.Params, FuncParam{
					Name: pName,
					Type: TypeFromJSON(typStr),
				})
			}
		} else {
			for _, pName := range orderedKeys {
				pType, ok := paramsRaw[pName]
				if !ok {
					continue
				}
				typStr, _ := pType.(string)
				fd.Params = append(fd.Params, FuncParam{
					Name: pName,
					Type: TypeFromJSON(typStr),
				})
			}
		}
	}

	if stepsRaw, ok := raw["steps"].([]any); ok {
		steps, err := parseSteps(stepsRaw)
		if err != nil {
			return nil, err
		}
		fd.Steps = steps
	}

	return fd, nil
}

// extractOrderedKeys parses the raw JSON to find the key order of a function's params.
// This is necessary because Go's map[string]any doesn't preserve insertion order.
func extractOrderedKeys(jsonData []byte, funcName, field string) []string {
	// Strategy: use json.Decoder to tokenize and find the params object
	// within the specific function definition.
	dec := json.NewDecoder(bytes.NewReader(jsonData))

	// Find "functions" → funcName → field
	if !seekKey(dec, "functions") {
		return nil
	}
	if !seekKey(dec, funcName) {
		return nil
	}
	if !seekKey(dec, field) {
		return nil
	}

	// Now read the keys of this object in order.
	t, err := dec.Token()
	if err != nil {
		return nil
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return nil
	}

	var keys []string
	depth := 0
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			break
		}

		if depth == 0 {
			if key, ok := t.(string); ok {
				keys = append(keys, key)
				// Skip the value.
				skipValue(dec)
			}
		} else {
			if delim, ok := t.(json.Delim); ok {
				switch delim {
				case '{', '[':
					depth++
				case '}', ']':
					depth--
				}
			}
		}
	}

	return keys
}

// seekKey advances the decoder past nested structures until it finds the given key.
func seekKey(dec *json.Decoder, key string) bool {
	depth := 0
	for {
		t, err := dec.Token()
		if err != nil {
			return false
		}

		switch v := t.(type) {
		case json.Delim:
			switch v {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth < 0 {
					return false
				}
			}
		case string:
			if depth == 1 && v == key {
				return true
			}
		}
	}
}

func skipValue(dec *json.Decoder) {
	t, err := dec.Token()
	if err != nil {
		return
	}
	if delim, ok := t.(json.Delim); ok {
		if delim == '{' || delim == '[' {
			depth := 1
			for depth > 0 {
				t, err := dec.Token()
				if err != nil {
					return
				}
				if d, ok := t.(json.Delim); ok {
					switch d {
					case '{', '[':
						depth++
					case '}', ']':
						depth--
					}
				}
			}
		}
	}
}

// --- Helpers ---

func parseComment(meta *NodeMeta, m map[string]any) {
	c, ok := m["_c"]
	if !ok {
		return
	}
	switch v := c.(type) {
	case string:
		meta.Comment = v
	case []any:
		for _, line := range v {
			if s, ok := line.(string); ok {
				meta.Comments = append(meta.Comments, s)
			}
		}
	}
}

func isCommentOnly(m map[string]any) bool {
	for k := range m {
		if k != "_c" {
			return false
		}
	}
	_, hasComment := m["_c"]
	return hasComment
}

func parseLimits(raw map[string]any) *LimitsDef {
	ld := &LimitsDef{}
	if v, ok := toInt(raw["max_depth"]); ok {
		ld.MaxDepth = &v
	}
	if v, ok := toInt(raw["max_steps"]); ok {
		ld.MaxSteps = &v
	}
	if v, ok := toInt(raw["max_loop_iterations"]); ok {
		ld.MaxLoopIterations = &v
	}
	if v, ok := toInt(raw["max_nodes"]); ok {
		ld.MaxNodes = &v
	}
	if v, ok := toInt(raw["max_variables"]); ok {
		ld.MaxVariables = &v
	}
	if v, ok := toInt(raw["max_variable_size"]); ok {
		ld.MaxVariableSize = &v
	}
	if v, ok := toInt(raw["max_output_size"]); ok {
		ld.MaxOutputSize = &v
	}
	if v, ok := raw["timeout"].(string); ok {
		ld.Timeout = &v
	}
	return ld
}

func toStringMap(m map[string]any) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k] = s
		} else {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}
