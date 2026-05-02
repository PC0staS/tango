// internal/executor/runner.go
package executor

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pc0stas/tango/internal/assertions"
	"github.com/pc0stas/tango/internal/config"
	"github.com/pc0stas/tango/internal/http"
	"github.com/tidwall/gjson"
)

var tmplRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

type Executor struct {
	workflow *config.Workflow
	client   *http.Client
}

type ExecutionResult struct {
	Name         string
	Steps        []config.StepResult
	Success      bool
	TotalTime    time.Duration
	PassedCount  int
	FailedCount  int
	SkippedCount int
}

func NewExecutor(workflow *config.Workflow) *Executor {
	return &Executor{
		workflow: workflow,
		client:   http.NewClient(workflow.Config.TimeoutDefault),
	}
}

func (e *Executor) Run(ctx context.Context) (*ExecutionResult, error) {
	startTime := time.Now()

	result := &ExecutionResult{
		Name:  e.workflow.Name,
		Steps: make([]config.StepResult, 0),
	}

	// 1. Topological sort
	order, err := TopologicalSort(e.workflow.Steps)
	if err != nil {
		return result, fmt.Errorf("dependency error: %w", err)
	}

	// 2. Execution state
	state := NewExecutionState()

	// 3. Map step names to step configs
	stepMap := make(map[string]*config.Step)
	for i := range e.workflow.Steps {
		stepMap[e.workflow.Steps[i].Name] = &e.workflow.Steps[i]
	}

	// 4. Execute steps in order
	for _, stepName := range order {
		step := stepMap[stepName]

		stepResult, err := e.executeStep(ctx, step, state)

		if err != nil {
			stepResult.Status = "failed"
			stepResult.Error = err.Error()
			result.FailedCount++

			if e.workflow.Config.StopOnError {
				result.Steps = append(result.Steps, stepResult)
				break
			}
		} else {
			stepResult.Status = "passed"
			result.PassedCount++
		}

		result.Steps = append(result.Steps, stepResult)
	}

	result.TotalTime = time.Since(startTime)
	result.Success = result.FailedCount == 0

	return result, nil
}

func (e *Executor) executeStep(ctx context.Context, step *config.Step, state *ExecutionState) (config.StepResult, error) {
	url := e.resolveTemplates(step.Request.URL, state)
	body := e.resolveTemplates(step.Request.Body, state)
	headers := make(map[string]string, len(step.Request.Headers))
	for k, v := range step.Request.Headers {
		headers[k] = e.resolveTemplates(v, state)
	}

	stepResult := config.StepResult{
		Name:          step.Name,
		RequestMethod: step.Request.Method,
		RequestURL:    url,
	}

	req := &http.Request{
		Method:  step.Request.Method,
		URL:     url,
		Headers: headers,
		Body:    body,
	}

	// Timeout
	ctx, cancel := context.WithTimeout(ctx, *step.Timeout)
	defer cancel()

	// Execute request
	resp, err := e.client.Do(ctx, req)
	if err != nil {
		return stepResult, fmt.Errorf("request failed: %w", err)
	}

	stepResult.RequestDuration = resp.Duration
	stepResult.ResponseStatus = resp.StatusCode
	stepResult.ResponseBody = resp.Body

	// Validate assertions
	if err := assertions.Validate(&step.Expect, resp); err != nil {
		return stepResult, err
	}

	// Capture variables
	if len(step.Capture.JSONPath) > 0 {
		captured, err := e.captureVariables(resp.Body, step.Capture.JSONPath)
		if err != nil {
			return stepResult, fmt.Errorf("capture failed: %w", err)
		}
		stepResult.CapturedVars = captured
		state.Capture(step.Name, captured)
	}

	return stepResult, nil
}

func (e *Executor) resolveTemplates(s string, state *ExecutionState) string {
	return tmplRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := strings.TrimSpace(match[2 : len(match)-2])

		if strings.HasPrefix(inner, ".") {
			name := strings.TrimPrefix(inner, ".")
			if val, ok := e.workflow.Config.Variables[name]; ok {
				return val
			}
			return match
		}

		if strings.HasPrefix(inner, "config.") {
			name := strings.TrimPrefix(inner, "config.")
			if val, ok := e.workflow.Config.Variables[name]; ok {
				return val
			}
			return match
		}

		if strings.HasPrefix(inner, "steps.") {
			parts := strings.SplitN(inner, ".", 3)
			if len(parts) == 3 {
				if val := state.GetCaptured(parts[1], parts[2]); val != nil {
					return fmt.Sprintf("%v", val)
				}
			}
			return match
		}

		if strings.HasPrefix(inner, "env.") {
			name := strings.TrimPrefix(inner, "env.")
			if val := os.Getenv(name); val != "" {
				return val
			}
			return match
		}

		return match
	})
}

func (e *Executor) captureVariables(body string, jsonPaths map[string]string) (map[string]interface{}, error) {
	captured := make(map[string]interface{})
	for varName, path := range jsonPaths {
		result := gjson.Get(body, path)
		if !result.Exists() {
			return nil, fmt.Errorf("path not found: %s", path)
		}
		captured[varName] = result.Value()
	}
	return captured, nil
}