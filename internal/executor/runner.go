// internal/executor/runner.go
package executor

import (
	"context"
	"fmt"
	"math"
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
	Dump     bool
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

		if state.ShouldSkip(step) {
			stepResult := config.StepResult{
				Name:          step.Name,
				RequestMethod: step.Request.Method,
				RequestURL:    step.Request.URL,
				Status:        "skipped",
			}
			result.Steps = append(result.Steps, stepResult)
			result.SkippedCount++
			state.SetStepStatus(step.Name, "skipped")
			continue
		}

		stepResult, err := e.executeStep(ctx, step, state)

		if err != nil {
			stepResult.Status = "failed"
			stepResult.Error = err.Error()
			result.FailedCount++
			state.SetStepStatus(step.Name, "failed")

			if e.workflow.Config.StopOnError {
				result.Steps = append(result.Steps, stepResult)
				break
			}
		} else {
			stepResult.Status = "passed"
			result.PassedCount++
			state.SetStepStatus(step.Name, "passed")
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

	// Dump request
	if e.Dump {
		e.printRequestDump(step.Name, req)
	}

	// Execute request with retries
	var resp *http.Response
	var reqErr error
	for attempt := 0; attempt <= step.Retry.Count; attempt++ {
		resp, reqErr = e.client.Do(ctx, req)
		if reqErr != nil {
			break
		}
		if !shouldRetry(resp.StatusCode, step.Retry.RetryOnStatus) {
			break
		}
		if attempt < step.Retry.Count {
			time.Sleep(backoffDelay(attempt+1, step.Retry))
		}
	}
	if reqErr != nil {
		return stepResult, fmt.Errorf("request failed: %w", reqErr)
	}

	stepResult.RequestDuration = resp.Duration
	stepResult.ResponseStatus = resp.StatusCode
	stepResult.ResponseBody = resp.Body

	// Dump response
	if e.Dump {
		e.printResponseDump(resp)
	}

	// Resolve templates in assertions
	expect := step.Expect
	for i := range expect.Body {
		a := &expect.Body[i]
		a.Path = e.resolveTemplates(a.Path, state)
		a.Type = e.resolveTemplates(a.Type, state)
		a.Contains = e.resolveTemplates(a.Contains, state)
		a.Startswith = e.resolveTemplates(a.Startswith, state)
		a.Endswith = e.resolveTemplates(a.Endswith, state)
		a.Matches = e.resolveTemplates(a.Matches, state)
		if s, ok := a.Equals.(string); ok {
			a.Equals = e.resolveTemplates(s, state)
		}
	}

	// Validate assertions
	if err := assertions.Validate(&expect, resp); err != nil {
		return stepResult, err
	}

	// Capture variables
	captured := make(map[string]interface{})
	if len(step.Capture.JSONPath) > 0 {
		vars, err := e.captureJSONPath(resp.Body, step.Capture.JSONPath)
		if err != nil {
			return stepResult, fmt.Errorf("capture failed: %w", err)
		}
		for k, v := range vars {
			captured[k] = v
		}
	}
	if step.Capture.ResponseBody != "" {
		captured[step.Capture.ResponseBody] = resp.Body
	}
	if len(captured) > 0 {
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

func (e *Executor) captureJSONPath(body string, jsonPaths map[string]string) (map[string]interface{}, error) {
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

func shouldRetry(status int, retryables []int) bool {
	for _, s := range retryables {
		if s == status {
			return true
		}
	}
	return false
}

func backoffDelay(attempt int, retry config.RetrySpec) time.Duration {
	base := float64(retry.Delay.Milliseconds())
	delay := base * math.Pow(retry.BackoffMultiplier, float64(attempt-1))
	return time.Duration(delay) * time.Millisecond
}

func (e *Executor) printRequestDump(stepName string, req *http.Request) {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Step: %s\n", stepName)
	fmt.Println("  → Request")
	fmt.Printf("  Method: %s\n", req.Method)
	fmt.Printf("  URL: %s\n", req.URL)
	if len(req.Headers) > 0 {
		fmt.Println("  Headers:")
		for k, v := range req.Headers {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
	if req.Body != "" {
		fmt.Println("  Body:")
		for _, line := range strings.Split(req.Body, "\n") {
			fmt.Printf("    %s\n", line)
		}
	}
}

func (e *Executor) printResponseDump(resp *http.Response) {
	fmt.Printf("  ← Response  %d  (%v)\n", resp.StatusCode, resp.Duration)
	fmt.Printf("  Body:")
	if len(resp.Body) > 500 {
		fmt.Printf(" %s...\n", resp.Body[:500])
	} else {
		fmt.Println()
		for _, line := range strings.Split(resp.Body, "\n") {
			fmt.Printf("    %s\n", line)
		}
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}