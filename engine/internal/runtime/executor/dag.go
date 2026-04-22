package executor

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
)

func (e *Executor) ExecuteDAG(ctx context.Context, process *parser.ProcessDefinition, input map[string]any, userID string) (*Context, error) {
	execCtx := &Context{
		Input:     input,
		Variables: make(map[string]any),
		UserID:    userID,
	}

	graph := buildGraph(process)

	if cycle := detectCycle(graph); cycle != "" {
		return nil, fmt.Errorf("cycle detected in DAG: %s", cycle)
	}

	if err := e.executeGraph(ctx, execCtx, process, graph); err != nil {
		return nil, err
	}

	return execCtx, nil
}

type dagGraph struct {
	inDegree  map[string]int
	outEdges  map[string][]dagEdge
	inEdges   map[string][]dagEdge
	allNodes  []string
}

type dagEdge struct {
	to        string
	from      string
	condition string
}

func buildGraph(proc *parser.ProcessDefinition) *dagGraph {
	g := &dagGraph{
		inDegree: make(map[string]int),
		outEdges: make(map[string][]dagEdge),
		inEdges:  make(map[string][]dagEdge),
	}

	for id := range proc.Nodes {
		g.inDegree[id] = 0
		g.allNodes = append(g.allNodes, id)
	}

	for _, edge := range proc.Edges {
		e := dagEdge{to: edge.To, from: edge.From, condition: edge.Condition}
		g.outEdges[edge.From] = append(g.outEdges[edge.From], e)
		g.inEdges[edge.To] = append(g.inEdges[edge.To], dagEdge{to: edge.To, from: edge.From, condition: edge.Condition})
		g.inDegree[edge.To]++
	}

	return g
}

func detectCycle(g *dagGraph) string {
	visited := make(map[string]int)
	for _, n := range g.allNodes {
		visited[n] = 0
	}

	var dfs func(node string) string
	dfs = func(node string) string {
		visited[node] = 1
		for _, edge := range g.outEdges[node] {
			if visited[edge.to] == 1 {
				return fmt.Sprintf("%s -> %s", node, edge.to)
			}
			if visited[edge.to] == 0 {
				if cycle := dfs(edge.to); cycle != "" {
					return cycle
				}
			}
		}
		visited[node] = 2
		return ""
	}

	for _, n := range g.allNodes {
		if visited[n] == 0 {
			if cycle := dfs(n); cycle != "" {
				return cycle
			}
		}
	}
	return ""
}

func (e *Executor) executeGraph(ctx context.Context, execCtx *Context, proc *parser.ProcessDefinition, g *dagGraph) error {
	completed := make(map[string]bool)
	nodeErrors := make(map[string]error)
	var mu sync.Mutex

	inDegree := make(map[string]int)
	for k, v := range g.inDegree {
		inDegree[k] = v
	}

	for {
		var ready []string
		mu.Lock()
		for _, id := range g.allNodes {
			if !completed[id] && inDegree[id] == 0 {
				ready = append(ready, id)
			}
		}
		mu.Unlock()

		if len(ready) == 0 {
			break
		}

		if len(ready) == 1 {
			id := ready[0]
			if err := e.executeNode(ctx, execCtx, proc, id); err != nil {
				return fmt.Errorf("node %q failed: %w", id, err)
			}
			mu.Lock()
			completed[id] = true
			for _, edge := range g.outEdges[id] {
				if edge.condition != "" && !e.evaluateEdgeCondition(edge.condition, execCtx) {
					continue
				}
				inDegree[edge.to]--
			}
			mu.Unlock()
		} else {
			var wg sync.WaitGroup
			for _, id := range ready {
				wg.Add(1)
				go func(nodeID string) {
					defer wg.Done()
					if err := e.executeNode(ctx, execCtx, proc, nodeID); err != nil {
						mu.Lock()
						nodeErrors[nodeID] = err
						mu.Unlock()
						return
					}
					mu.Lock()
					completed[nodeID] = true
					for _, edge := range g.outEdges[nodeID] {
						if edge.condition != "" && !e.evaluateEdgeCondition(edge.condition, execCtx) {
							continue
						}
						inDegree[edge.to]--
					}
					mu.Unlock()
				}(id)
			}
			wg.Wait()

			if len(nodeErrors) > 0 {
				for id, err := range nodeErrors {
					return fmt.Errorf("node %q failed: %w", id, err)
				}
			}
		}
	}

	for _, id := range g.allNodes {
		if !completed[id] {
			allIncomingSkipped := true
			for _, inEdge := range g.inEdges[id] {
				if completed[inEdge.from] {
					if inEdge.condition == "" || e.evaluateEdgeCondition(inEdge.condition, execCtx) {
						allIncomingSkipped = false
						break
					}
				}
			}
			if !allIncomingSkipped {
				return fmt.Errorf("node %q was not executed (possible deadlock)", id)
			}
		}
	}

	return nil
}

func (e *Executor) executeNode(ctx context.Context, execCtx *Context, proc *parser.ProcessDefinition, nodeID string) error {
	node := proc.Nodes[nodeID]
	handler, ok := e.handlers[node.Type]
	if !ok {
		return fmt.Errorf("no handler for step type %q", node.Type)
	}
	return handler.Execute(ctx, execCtx, node)
}

func (e *Executor) evaluateEdgeCondition(condition string, execCtx *Context) bool {
	condition = e.interpolateCondition(condition, execCtx)

	if strings.Contains(condition, " > ") {
		return true
	}
	if strings.Contains(condition, " == ") {
		parts := strings.SplitN(condition, " == ", 2)
		return strings.TrimSpace(parts[0]) == strings.TrimSpace(parts[1])
	}

	val := e.resolveConditionVar(condition, execCtx)
	if b, ok := val.(bool); ok {
		return b
	}
	return val != nil && val != "" && val != 0
}

func (e *Executor) interpolateCondition(s string, execCtx *Context) string {
	result := s
	for key, val := range execCtx.Input {
		result = strings.ReplaceAll(result, "{{input."+key+"}}", fmt.Sprintf("%v", val))
	}
	for key, val := range execCtx.Variables {
		result = strings.ReplaceAll(result, "{{"+key+"}}", fmt.Sprintf("%v", val))
	}
	return result
}

func (e *Executor) resolveConditionVar(name string, execCtx *Context) any {
	name = strings.TrimPrefix(name, "{{")
	name = strings.TrimSuffix(name, "}}")
	name = strings.TrimSpace(name)

	if strings.HasPrefix(name, "input.") {
		key := strings.TrimPrefix(name, "input.")
		return execCtx.Input[key]
	}
	if val, ok := execCtx.Variables[name]; ok {
		return val
	}
	return nil
}
