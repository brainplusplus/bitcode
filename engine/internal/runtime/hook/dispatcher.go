package hook

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type ProcessLoader interface {
	LoadProcess(name string) (*parser.ProcessDefinition, error)
}

type ScriptRunner interface {
	Run(ctx context.Context, script string, params map[string]any) (any, error)
}

type Dispatcher struct {
	processLoader ProcessLoader
	scriptRunner  ScriptRunner
	onChangeDepth int
}

func NewDispatcher(loader ProcessLoader, runner ScriptRunner) *Dispatcher {
	return &Dispatcher{
		processLoader: loader,
		scriptRunner:  runner,
		onChangeDepth: 5,
	}
}

func (d *Dispatcher) SetOnChangeMaxDepth(depth int) {
	d.onChangeDepth = depth
}

func (d *Dispatcher) Dispatch(ctx context.Context, eventName string, handlers []parser.EventHandler, eventCtx *EventContext) error {
	if len(handlers) == 0 {
		return nil
	}

	sorted := make([]parser.EventHandler, len(handlers))
	copy(sorted, handlers)
	sort.SliceStable(sorted, func(i, j int) bool {
		pi := sorted[i].Priority
		pj := sorted[j].Priority
		if pi == 0 {
			pi = 50
		}
		if pj == 0 {
			pj = 50
		}
		return pi < pj
	})

	for _, handler := range sorted {
		if err := d.executeHandler(ctx, eventName, handler, eventCtx); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) DispatchAsync(ctx context.Context, eventName string, handlers []parser.EventHandler, eventCtx *EventContext) {
	if len(handlers) == 0 {
		return
	}

	immutableCtx := eventCtx.ImmutableCopy()

	for _, handler := range handlers {
		h := handler
		go func() {
			if err := d.executeHandlerWithRetry(ctx, eventName, h, immutableCtx); err != nil {
				onError := h.GetOnError(eventName)
				if onError == "log" {
					log.Printf("[HOOK] async %s handler error: %v", eventName, err)
				}
			}
		}()
	}
}

func (d *Dispatcher) DispatchSplit(ctx context.Context, eventName string, handlers []parser.EventHandler, eventCtx *EventContext) error {
	var syncHandlers []parser.EventHandler
	var asyncHandlers []parser.EventHandler

	for _, h := range handlers {
		if h.IsSync(eventName) || h.GetOnError(eventName) == "fail" {
			syncHandlers = append(syncHandlers, h)
		} else {
			asyncHandlers = append(asyncHandlers, h)
		}
	}

	if err := d.Dispatch(ctx, eventName, syncHandlers, eventCtx); err != nil {
		return err
	}

	if len(asyncHandlers) > 0 {
		d.DispatchAsync(ctx, eventName, asyncHandlers, eventCtx)
	}

	return nil
}

func (d *Dispatcher) DispatchOnChange(ctx context.Context, eventCtx *EventContext, events *parser.EventsDefinition, depth int) error {
	if events == nil || len(events.OnChange) == 0 || eventCtx.Changes == nil {
		return nil
	}
	if depth > d.onChangeDepth {
		log.Printf("[HOOK] on_change cascade depth exceeded (%d) for model %s", depth, eventCtx.Model)
		return nil
	}
	if depth > 3 {
		log.Printf("[HOOK] on_change cascade depth %d for model %s (warning: deep cascade)", depth, eventCtx.Model)
	}

	for fieldName := range eventCtx.Changes {
		handlers, ok := events.OnChange[fieldName]
		if !ok || len(handlers) == 0 {
			continue
		}

		dataBefore := copyMap(eventCtx.Data)

		onChangeCtx := &EventContext{
			Model:     eventCtx.Model,
			Module:    eventCtx.Module,
			Event:     "on_change",
			Operation: eventCtx.Operation,
			Data:      eventCtx.Data,
			OldData:   eventCtx.OldData,
			Changes:   map[string]any{fieldName: eventCtx.Changes[fieldName]},
			UserID:    eventCtx.UserID,
			TenantID:  eventCtx.TenantID,
			Session:   eventCtx.Session,
		}

		if err := d.Dispatch(ctx, "on_change", handlers, onChangeCtx); err != nil {
			return err
		}

		newChanges := make(map[string]any)
		for k, v := range eventCtx.Data {
			if oldV, ok := dataBefore[k]; ok {
				if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", oldV) && k != fieldName {
					newChanges[k] = v
				}
			}
		}

		if len(newChanges) > 0 {
			cascadeCtx := &EventContext{
				Model:     eventCtx.Model,
				Module:    eventCtx.Module,
				Event:     "on_change",
				Operation: eventCtx.Operation,
				Data:      eventCtx.Data,
				OldData:   dataBefore,
				Changes:   newChanges,
				UserID:    eventCtx.UserID,
				TenantID:  eventCtx.TenantID,
				Session:   eventCtx.Session,
			}
			if err := d.DispatchOnChange(ctx, cascadeCtx, events, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *Dispatcher) executeHandler(ctx context.Context, eventName string, handler parser.EventHandler, eventCtx *EventContext) error {
	if handler.Condition != "" {
		condData := d.buildConditionData(eventCtx)
		if !evaluateHandlerCondition(handler.Condition, condData) {
			return nil
		}
	}

	timeoutStr := handler.GetTimeout(eventName)
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		timeout = 30 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, execErr := d.runHandler(execCtx, handler, eventCtx)

	if execErr != nil {
		onError := handler.GetOnError(eventName)
		switch onError {
		case "fail":
			return execErr
		case "log":
			log.Printf("[HOOK] %s handler error (logged): %v", eventName, execErr)
			return nil
		case "ignore":
			return nil
		default:
			return execErr
		}
	}

	if result != nil && isBeforeEvent(eventName) {
		if modifications, ok := result.(map[string]any); ok {
			for k, v := range modifications {
				if len(k) > 0 && k[0] != '_' {
					eventCtx.Data[k] = v
				}
			}
		}
	}

	return nil
}

func (d *Dispatcher) executeHandlerWithRetry(ctx context.Context, eventName string, handler parser.EventHandler, eventCtx *EventContext) error {
	err := d.executeHandler(ctx, eventName, handler, eventCtx)
	if err == nil || handler.Retry == nil || handler.Retry.Max <= 0 {
		return err
	}

	delay := 5 * time.Second
	if handler.Retry.Delay != "" {
		if parsed, parseErr := time.ParseDuration(handler.Retry.Delay); parseErr == nil {
			delay = parsed
		}
	}

	for attempt := 1; attempt <= handler.Retry.Max; attempt++ {
		time.Sleep(delay)
		err = d.executeHandler(ctx, eventName, handler, eventCtx)
		if err == nil {
			return nil
		}
		if handler.Retry.Backoff == "exponential" {
			delay *= 2
		}
		log.Printf("[HOOK] retry %d/%d for %s handler: %v", attempt, handler.Retry.Max, eventName, err)
	}
	return err
}

func (d *Dispatcher) runHandler(ctx context.Context, handler parser.EventHandler, eventCtx *EventContext) (any, error) {
	if handler.Process != "" {
		return d.runProcess(ctx, handler.Process, eventCtx)
	}
	if handler.Script != nil {
		return d.runScript(ctx, handler.Script, eventCtx)
	}
	return nil, fmt.Errorf("event handler must specify either process or script")
}

func (d *Dispatcher) runProcess(ctx context.Context, processName string, eventCtx *EventContext) (any, error) {
	if d.processLoader == nil {
		return nil, fmt.Errorf("no process loader configured")
	}

	proc, err := d.processLoader.LoadProcess(processName)
	if err != nil {
		return nil, fmt.Errorf("process %q not found: %w", processName, err)
	}

	input := eventCtx.ToProcessInput()

	execResult, execErr := executeProcess(ctx, proc, input, eventCtx.UserID)
	if execErr != nil {
		return nil, execErr
	}

	return execResult, nil
}

type processExecutorFunc func(ctx context.Context, proc *parser.ProcessDefinition, input map[string]any, userID string) (any, error)

var executeProcess processExecutorFunc

func SetProcessExecutor(fn processExecutorFunc) {
	executeProcess = fn
}

func (d *Dispatcher) runScript(ctx context.Context, script *parser.ScriptRef, eventCtx *EventContext) (any, error) {
	if d.scriptRunner == nil {
		return nil, fmt.Errorf("no script runner configured")
	}

	params := map[string]any{
		"model":     eventCtx.Model,
		"module":    eventCtx.Module,
		"event":     eventCtx.Event,
		"operation": eventCtx.Operation,
		"data":      eventCtx.Data,
		"old_data":  eventCtx.OldData,
		"changes":   eventCtx.Changes,
		"user_id":   eventCtx.UserID,
		"tenant_id": eventCtx.TenantID,
		"session":   eventCtx.Session,
	}

	return d.scriptRunner.Run(ctx, script.File, params)
}

func (d *Dispatcher) buildConditionData(eventCtx *EventContext) map[string]any {
	data := copyMap(eventCtx.Data)
	if eventCtx.OldData != nil {
		data["__old"] = eventCtx.OldData
	}
	if eventCtx.Session != nil {
		data["__session"] = eventCtx.Session
	}
	data["is_create"] = eventCtx.Operation == "create"
	data["is_update"] = eventCtx.Operation == "update"
	data["is_delete"] = eventCtx.Operation == "delete"
	data["is_bulk"] = eventCtx.IsBulk
	return data
}

func evaluateHandlerCondition(condition string, data map[string]any) bool {
	return evaluateSimpleExpr(condition, data)
}

func isBeforeEvent(name string) bool {
	return parser.IsBeforeEvent(name)
}
