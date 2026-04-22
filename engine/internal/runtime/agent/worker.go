package agent

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/domain/event"
)

type ScriptRunner interface {
	Run(ctx context.Context, script string, params map[string]any) (any, error)
}

type Worker struct {
	bus          *event.Bus
	scriptRunner ScriptRunner
	agents       []*parser.AgentDefinition
	mu           sync.Mutex
}

func NewWorker(bus *event.Bus, scriptRunner ScriptRunner) *Worker {
	return &Worker{
		bus:          bus,
		scriptRunner: scriptRunner,
	}
}

func (w *Worker) RegisterAgent(agentDef *parser.AgentDefinition) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.agents = append(w.agents, agentDef)

	for _, trigger := range agentDef.Triggers {
		t := trigger
		retryMax := agentDef.Retry.Max
		w.bus.Subscribe(t.Event, func(ctx context.Context, eventName string, data map[string]any) error {
			return w.executeWithRetry(ctx, t.Script, data, retryMax)
		})
	}
}

func (w *Worker) executeWithRetry(ctx context.Context, script string, data map[string]any, maxRetries int) error {
	if maxRetries <= 0 {
		maxRetries = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		_, err := w.scriptRunner.Run(ctx, script, data)
		if err == nil {
			return nil
		}
		lastErr = err
		log.Printf("[agent] attempt %d/%d failed for %s: %v", attempt, maxRetries, script, err)
	}
	return fmt.Errorf("all %d attempts failed for %s: %w", maxRetries, script, lastErr)
}
