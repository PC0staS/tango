# Tango — Guía Paso a Paso

**Objetivo:** Ir paso a paso, sin perder el hilo. Cada paso es tangible, testeable, y rápido.

---

## FASE 1: Setup Básico (30 min)

### Paso 1.1: Crear proyecto Go

```bash
mkdir ~/projects/tango
cd ~/projects/tango
git init
go mod init github.com/pc0stas/tango
```

### Paso 1.2: Estructura mínima

```bash
mkdir -p cmd internal/{config,executor,http,assertions,output,utils}
touch main.go go.mod go.sum Makefile README.md
```

### Paso 1.3: Dependencias iniciales

```bash
go get github.com/spf13/cobra
go get gopkg.in/yaml.v3
go get github.com/tidwall/gjson
```

### Paso 1.4: Makefile básico

```makefile
# Makefile
build:
	go build -o bin/tango .

run:
	go run main.go test examples/health_check.yaml

clean:
	rm -rf bin/

test:
	go test -v ./...

.PHONY: build run clean test
```

**Checkpoint:** Estructura creada, dependencias listadas.

---

## FASE 2: Tipos de Datos (1 hora)

### Paso 2.1: Crear `internal/config/types.go`

Este archivo define TODAS las estructuras. Es el "mapa" del proyecto.

```go
// internal/config/types.go
package config

import "time"

type Workflow struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Version     string         `yaml:"version"`
	Config      WorkflowConfig `yaml:"config"`
	Steps       []Step         `yaml:"steps"`
}

type WorkflowConfig struct {
	TimeoutDefault time.Duration      `yaml:"timeout_default"`
	StopOnError    bool               `yaml:"stop_on_error"`
	Variables      map[string]string  `yaml:"variables"`
}

type Step struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Request     RequestSpec `yaml:"request"`
	Expect      ExpectSpec  `yaml:"expect"`
	DependsOn   []string    `yaml:"depends_on"`
	Timeout     *time.Duration `yaml:"timeout"`
	Capture     CaptureSpec `yaml:"capture"`
	Retry       RetrySpec   `yaml:"retry"`
}

type RequestSpec struct {
	Method      string            `yaml:"method"`
	URL         string            `yaml:"url"`
	Headers     map[string]string `yaml:"headers"`
	Body        string            `yaml:"body"`
	BodyFile    string            `yaml:"body_file"`
	QueryParams map[string]string `yaml:"query_params"`
}

type ExpectSpec struct {
	Status       int              `yaml:"status"`
	Headers      map[string]string `yaml:"headers"`
	Body         []AssertionSpec  `yaml:"body"`
	ResponseTime *int             `yaml:"response_time_ms"`
}

type AssertionSpec struct {
	Path        string      `yaml:"path"`
	Equals      interface{} `yaml:"equals"`
	Type        string      `yaml:"type"`
	Contains    string      `yaml:"contains"`
	Matches     string      `yaml:"matches"`
	Exists      *bool       `yaml:"exists"`
	Empty       *bool       `yaml:"empty"`
}

type CaptureSpec struct {
	JSONPath map[string]string `yaml:"json_path"`
}

type RetrySpec struct {
	Count              int           `yaml:"count"`
	Delay              time.Duration `yaml:"delay"`
	BackoffMultiplier  float64       `yaml:"backoff_multiplier"`
	RetryOnStatus      []int         `yaml:"retry_on_status"`
}

// Execution results
type ExecutionResult struct {
	Name         string
	Steps        []StepResult
	Success      bool
	TotalTime    time.Duration
	PassedCount  int
	FailedCount  int
	SkippedCount int
}

type StepResult struct {
	Name            string
	Status          string // "passed", "failed", "skipped"
	RequestMethod   string
	RequestURL      string
	RequestDuration time.Duration
	ResponseStatus  int
	ResponseBody    string
	Error           string
	CapturedVars    map[string]interface{}
}
```

**Checkpoint:** Todos los tipos definidos. Este archivo es la "fuente de verdad" para el resto del proyecto.

---

## FASE 3: Parser YAML (1.5 horas)

### Paso 3.1: Crear `internal/config/parser.go`

```go
// internal/config/parser.go
package config

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v3"
)

func ParseWorkflow(filename string) (*Workflow, error) {
	// 1. Leer archivo
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 2. Parsear YAML
	var workflow Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// 3. Validar
	if err := ValidateWorkflow(&workflow); err != nil {
		return nil, err
	}

	// 4. Aplicar defaults
	ApplyDefaults(&workflow)

	// 5. Cargar body files
	if err := LoadBodyFiles(&workflow); err != nil {
		return nil, err
	}

	return &workflow, nil
}

func ValidateWorkflow(w *Workflow) error {
	if w.Name == "" {
		return fmt.Errorf("workflow must have a name")
	}
	if len(w.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	// Nombres únicos
	seen := make(map[string]bool)
	for _, step := range w.Steps {
		if step.Name == "" {
			return fmt.Errorf("all steps must have a name")
		}
		if seen[step.Name] {
			return fmt.Errorf("duplicate step name: %s", step.Name)
		}
		seen[step.Name] = true

		// Request validation
		if step.Request.Method == "" {
			return fmt.Errorf("step %s: method is required", step.Name)
		}
		if step.Request.URL == "" {
			return fmt.Errorf("step %s: URL is required", step.Name)
		}
	}

	// Validar dependencias
	for _, step := range w.Steps {
		for _, dep := range step.DependsOn {
			if !seen[dep] {
				return fmt.Errorf("step %s: depends on non-existent step %s", step.Name, dep)
			}
		}
	}

	return nil
}

func ApplyDefaults(w *Workflow) {
	if w.Version == "" {
		w.Version = "1.0"
	}

	if w.Config.TimeoutDefault == 0 {
		w.Config.TimeoutDefault = 10 * time.Second
	}

	for i := range w.Steps {
		step := &w.Steps[i]

		if step.Request.Method == "" {
			step.Request.Method = "GET"
		}

		if step.Timeout == nil {
			t := w.Config.TimeoutDefault
			step.Timeout = &t
		}

		if step.Retry.Count == 0 {
			step.Retry.Count = 1
		}
		if step.Retry.BackoffMultiplier == 0 {
			step.Retry.BackoffMultiplier = 2.0
		}
	}
}

func LoadBodyFiles(w *Workflow) error {
	for i := range w.Steps {
		step := &w.Steps[i]
		if step.Request.BodyFile != "" {
			data, err := os.ReadFile(step.Request.BodyFile)
			if err != nil {
				return fmt.Errorf("step %s: failed to load body file: %w", step.Name, err)
			}
			step.Request.Body = string(data)
			step.Request.BodyFile = "" // Clear it after loading
		}
	}
	return nil
}
```

### Paso 3.2: Crear ejemplo de workflow

```bash
mkdir -p examples
```

```yaml
# examples/health_check.yaml
name: "health-check"
description: "Test basic endpoint"
version: "1.0"

config:
  timeout_default: 5s
  stop_on_error: false

steps:
  - name: "ping"
    description: "Check if server is alive"
    request:
      method: "GET"
      url: "http://httpbin.org/status/200"
    expect:
      status: 200
```

### Paso 3.3: Crear test para parser

```go
// internal/config/parser_test.go
package config

import (
	"testing"
)

func TestParseValidWorkflow(t *testing.T) {
	w, err := ParseWorkflow("../../examples/health_check.yaml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if w.Name != "health-check" {
		t.Errorf("expected name 'health-check', got %s", w.Name)
	}

	if len(w.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(w.Steps))
	}

	if w.Steps[0].Name != "ping" {
		t.Errorf("expected step name 'ping', got %s", w.Steps[0].Name)
	}
}

func TestValidateWorkflow_MissingName(t *testing.T) {
	w := &Workflow{
		Steps: []Step{{Name: "test"}},
	}
	err := ValidateWorkflow(w)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
```

### Paso 3.4: Testear

```bash
go test ./internal/config -v
```

**Checkpoint:** Parser funciona, lee YAML, valida, aplica defaults.

---

## FASE 4: HTTP Client (1 hora)

### Paso 4.1: Crear `internal/http/client.go`

```go
// internal/http/client.go
package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Request struct {
	Method string
	URL    string
	Headers map[string]string
	Body   string
}

type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       string
	Duration   time.Duration
}

type Client struct {
	httpClient *http.Client
	timeout    time.Duration
}

func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		timeout:    timeout,
	}
}

func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	// Crear HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, strings.NewReader(req.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Agregar headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Ejecutar
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Leer body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Convertir headers
	respHeaders := make(map[string]string)
	for k, v := range httpResp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	return &Response{
		StatusCode: httpResp.StatusCode,
		Headers:    respHeaders,
		Body:       string(bodyBytes),
		Duration:   time.Since(start),
	}, nil
}
```

### Paso 4.2: Test HTTP client

```go
// internal/http/client_test.go
package http

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Do(t *testing.T) {
	// Crear mock server
	server := httptest.NewServer(nil)
	defer server.Close()

	client := NewClient(5 * time.Second)
	req := &Request{
		Method: "GET",
		URL:    server.URL,
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
```

### Paso 4.3: Testear

```bash
go test ./internal/http -v
```

**Checkpoint:** HTTP client funciona, envía requests, captura responses.

---

## FASE 5: Assertions (1.5 horas)

### Paso 5.1: Crear `internal/assertions/validator.go`

```go
// internal/assertions/validator.go
package assertions

import (
	"fmt"
	"regexp"
	"strings"
	"github.com/pc0stas/tango/internal/config"
	httppkg "github.com/pc0stas/tango/internal/http"
	"github.com/tidwall/gjson"
)

func Validate(expect *config.ExpectSpec, response *httppkg.Response) error {
	// 1. Status code
	if response.StatusCode != expect.Status {
		return fmt.Errorf("status mismatch: expected %d, got %d", expect.Status, response.StatusCode)
	}

	// 2. Body assertions
	for i, assertion := range expect.Body {
		if err := ValidateAssertion(&assertion, response.Body); err != nil {
			return fmt.Errorf("assertion %d failed: %w", i+1, err)
		}
	}

	// 3. Response time
	if expect.ResponseTime != nil {
		if response.Duration.Milliseconds() > int64(*expect.ResponseTime) {
			return fmt.Errorf("response time exceeded: %dms > %dms",
				response.Duration.Milliseconds(), *expect.ResponseTime)
		}
	}

	return nil
}

func ValidateAssertion(assertion *config.AssertionSpec, body string) error {
	// Extraer valor con JSONPath
	result := gjson.Get(body, assertion.Path)
	if !result.Exists() {
		if assertion.Exists != nil && !*assertion.Exists {
			return nil // Era esperado que no existiera
		}
		return fmt.Errorf("path not found: %s", assertion.Path)
	}

	value := result.Value()

	// Aplicar matchers
	if assertion.Equals != nil {
		if value != assertion.Equals {
			return fmt.Errorf("value mismatch: expected %v, got %v", assertion.Equals, value)
		}
		return nil
	}

	if assertion.Contains != "" {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("value is not a string")
		}
		if !strings.Contains(str, assertion.Contains) {
			return fmt.Errorf("value does not contain %q", assertion.Contains)
		}
		return nil
	}

	if assertion.Matches != "" {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("value is not a string")
		}
		matched, err := regexp.MatchString(assertion.Matches, str)
		if err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
		if !matched {
			return fmt.Errorf("value does not match pattern %q", assertion.Matches)
		}
		return nil
	}

	if assertion.Type != "" {
		actualType := getType(value)
		if actualType != assertion.Type {
			return fmt.Errorf("type mismatch: expected %s, got %s", assertion.Type, actualType)
		}
		return nil
	}

	if assertion.Empty != nil {
		isEmpty := checkEmpty(value)
		if isEmpty != *assertion.Empty {
			return fmt.Errorf("empty check failed: expected %v, got %v", *assertion.Empty, isEmpty)
		}
		return nil
	}

	return nil
}

func getType(v interface{}) string {
	switch v.(type) {
	case float64:
		return "number"
	case string:
		return "string"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func checkEmpty(v interface{}) bool {
	switch val := v.(type) {
	case string:
		return val == ""
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	case nil:
		return true
	default:
		return false
	}
}
```

### Paso 5.2: Test assertions

```go
// internal/assertions/validator_test.go
package assertions

import (
	"testing"
	"github.com/pc0stas/tango/internal/config"
	httppkg "github.com/pc0stas/tango/internal/http"
)

func TestValidate_Status(t *testing.T) {
	expect := &config.ExpectSpec{
		Status: 200,
		Body:   []config.AssertionSpec{},
	}
	resp := &httppkg.Response{
		StatusCode: 200,
		Body:       "{}",
	}

	err := Validate(expect, resp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidate_BodyAssertion(t *testing.T) {
	expect := &config.ExpectSpec{
		Status: 200,
		Body: []config.AssertionSpec{
			{
				Path:   "$.status",
				Equals: "ok",
			},
		},
	}
	resp := &httppkg.Response{
		StatusCode: 200,
		Body:       `{"status":"ok"}`,
	}

	err := Validate(expect, resp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
```

### Paso 5.3: Testear

```bash
go test ./internal/assertions -v
```

**Checkpoint:** Assertions funciona, valida status y body con JSONPath.

---

## FASE 6: Executor - El Motor (2 horas)

### Paso 6.1: Crear `internal/executor/dependency.go` (Topological Sort)

```go
// internal/executor/dependency.go
package executor

import (
	"fmt"
	"github.com/pc0stas/tango/internal/config"
)

func TopologicalSort(steps []config.Step) ([]string, error) {
	// Mapear step name -> index
	stepMap := make(map[string]bool)
	inDegree := make(map[string]int)

	for _, step := range steps {
		stepMap[step.Name] = true
		inDegree[step.Name] = 0
	}

	// Construir grafo de dependencias
	adjList := make(map[string][]string)
	for _, step := range steps {
		for _, dep := range step.DependsOn {
			if !stepMap[dep] {
				return nil, fmt.Errorf("step %s depends on non-existent step %s", step.Name, dep)
			}
			adjList[dep] = append(adjList[dep], step.Name)
			inDegree[step.Name]++
		}
	}

	// Kahn's algorithm
	queue := make([]string, 0)
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	result := make([]string, 0)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, neighbor := range adjList[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(result) != len(steps) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}
```

### Paso 6.2: Crear `internal/executor/state.go` (Tracking de ejecución)

```go
// internal/executor/state.go
package executor

import (
	"sync"
)

type ExecutionState struct {
	mu           sync.RWMutex
	capturedVars map[string]map[string]interface{}
}

func NewExecutionState() *ExecutionState {
	return &ExecutionState{
		capturedVars: make(map[string]map[string]interface{}),
	}
}

func (s *ExecutionState) Capture(stepName string, vars map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.capturedVars[stepName] = vars
}

func (s *ExecutionState) GetCaptured(stepName, varName string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if stepVars, ok := s.capturedVars[stepName]; ok {
		if val, ok := stepVars[varName]; ok {
			return val
		}
	}
	return nil
}
```

### Paso 6.3: Crear `internal/executor/runner.go` (Main executor)

```go
// internal/executor/runner.go
package executor

import (
	"context"
	"fmt"
	"time"
	"github.com/pc0stas/tango/internal/config"
	"github.com/pc0stas/tango/internal/http"
	"github.com/pc0stas/tango/internal/assertions"
)

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

	// 3. Mapeo de steps por nombre
	stepMap := make(map[string]*config.Step)
	for i := range e.workflow.Steps {
		stepMap[e.workflow.Steps[i].Name] = &e.workflow.Steps[i]
	}

	// 4. Ejecutar steps
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
	stepResult := config.StepResult{
		Name:          step.Name,
		RequestMethod: step.Request.Method,
		RequestURL:    step.Request.URL,
	}

	// Crear HTTP request
	req := &http.Request{
		Method:  step.Request.Method,
		URL:     step.Request.URL,
		Headers: step.Request.Headers,
		Body:    step.Request.Body,
	}

	// Timeout
	ctx, cancel := context.WithTimeout(ctx, *step.Timeout)
	defer cancel()

	// Ejecutar
	resp, err := e.client.Do(ctx, req)
	if err != nil {
		return stepResult, fmt.Errorf("request failed: %w", err)
	}

	stepResult.RequestDuration = resp.Duration
	stepResult.ResponseStatus = resp.StatusCode
	stepResult.ResponseBody = resp.Body

	// Validar
	if err := assertions.Validate(&step.Expect, resp); err != nil {
		return stepResult, err
	}

	// Capturar variables
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
```

(Nota: Necesitas `import "github.com/tidwall/gjson"` en runner.go)

### Paso 6.4: Test executor

```go
// internal/executor/runner_test.go
package executor

import (
	"context"
	"testing"
	"time"
	"github.com/pc0stas/tango/internal/config"
)

func TestExecutor_Simplestep(t *testing.T) {
	workflow := &config.Workflow{
		Name: "test",
		Config: config.WorkflowConfig{
			TimeoutDefault: 5 * time.Second,
		},
		Steps: []config.Step{
			{
				Name: "ping",
				Request: config.RequestSpec{
					Method: "GET",
					URL:    "http://httpbin.org/status/200",
				},
				Expect: config.ExpectSpec{
					Status: 200,
				},
				Timeout: ptrDuration(5 * time.Second),
			},
		},
	}

	exec := NewExecutor(workflow)
	result, err := exec.Run(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success")
	}

	if result.PassedCount != 1 {
		t.Errorf("expected 1 passed, got %d", result.PassedCount)
	}
}

func ptrDuration(d time.Duration) *time.Duration {
	return &d
}
```

### Paso 6.5: Testear

```bash
go test ./internal/executor -v
```

**Checkpoint:** Executor funciona, ejecuta steps secuencialmente, valida y captura variables.

---

## FASE 7: Output Formatting (1 hora)

### Paso 7.1: Crear `internal/output/text.go`

```go
// internal/output/text.go
package output

import (
	"fmt"
	"strings"
	"github.com/pc0stas/tango/internal/executor"
)

func FormatText(result *executor.ExecutionResult) string {
	var sb strings.Builder

	for _, step := range result.Steps {
		icon := getIcon(step.Status)
		ms := step.RequestDuration.Milliseconds()
		status := getHTTPStatus(step.ResponseStatus)

		line := fmt.Sprintf("%s %s (%s %s) — %d %s [%dms]",
			icon, step.Name, step.RequestMethod, step.RequestURL,
			step.ResponseStatus, status, ms)

		sb.WriteString(line + "\n")

		if step.Error != "" {
			sb.WriteString(fmt.Sprintf("  └─ Error: %s\n", step.Error))
		}
		if len(step.CapturedVars) > 0 {
			sb.WriteString(fmt.Sprintf("  └─ Captured: %v\n", step.CapturedVars))
		}
	}

	sb.WriteString("\n" + strings.Repeat("-", 50) + "\n")
	sb.WriteString(fmt.Sprintf("Summary: %d passed, %d failed, %d skipped\n",
		result.PassedCount, result.FailedCount, result.SkippedCount))
	sb.WriteString(fmt.Sprintf("Total time: %v\n", result.TotalTime))

	return sb.String()
}

func getIcon(status string) string {
	switch status {
	case "passed":
		return "✓"
	case "failed":
		return "✗"
	case "skipped":
		return "⊘"
	default:
		return "?"
	}
}

func getHTTPStatus(code int) string {
	statuses := map[int]string{
		200: "OK",
		201: "Created",
		204: "No Content",
		400: "Bad Request",
		404: "Not Found",
		500: "Internal Server Error",
	}
	if s, ok := statuses[code]; ok {
		return s
	}
	return "Unknown"
}
```

**Checkpoint:** Output formatting funciona, imprime resultados legibles.

---

## FASE 8: CLI Principal (1 hora)

### Paso 8.1: Crear `main.go`

```go
// main.go
package main

import (
	"context"
	"fmt"
	"os"
	"github.com/spf13/cobra"
	"github.com/pc0stas/tango/internal/config"
	"github.com/pc0stas/tango/internal/executor"
	"github.com/pc0stas/tango/internal/output"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "tango",
		Short: "Distributed testing CLI",
	}

	testCmd := &cobra.Command{
		Use:   "test <workflow.yaml>",
		Short: "Execute a test workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(args[0])
		},
	}

	rootCmd.AddCommand(testCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runTest(workflowFile string) error {
	// 1. Parse
	workflow, err := config.ParseWorkflow(workflowFile)
	if err != nil {
		return fmt.Errorf("parse failed: %w", err)
	}

	// 2. Execute
	exec := executor.NewExecutor(workflow)
	result, err := exec.Run(context.Background())
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// 3. Output
	fmt.Print(output.FormatText(result))

	// 4. Exit code
	if !result.Success {
		os.Exit(1)
	}

	return nil
}
```

### Paso 8.2: Build & Test

```bash
make build
./bin/tango test examples/health_check.yaml
```

**Checkpoint:** CLI funciona, puedes ejecutar un workflow completo.

---

## FASE 9: Mejoras Rápidas (1 hora)

### Paso 9.1: Crear `cmd/validate.go`

```go
// cmd/validate.go (o agregar a main.go)
validateCmd := &cobra.Command{
	Use:   "validate <workflow.yaml>",
	Short: "Validate workflow syntax",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := config.ParseWorkflow(args[0])
		if err != nil {
			fmt.Printf("✗ Validation failed: %v\n", err)
			return err
		}
		fmt.Println("✓ Workflow is valid")
		return nil
	},
}

rootCmd.AddCommand(validateCmd)
```

### Paso 9.2: Agregar retry logic

En `internal/http/client.go`, envolver `Do()` con reintentos:

```go
func (e *Executor) executeStep(...) {
	// ... setup ...

	var resp *http.Response
	for attempt := 1; attempt <= step.Retry.Count; attempt++ {
		resp, err = e.client.Do(ctx, req)

		if err == nil && ShouldRetry(resp.StatusCode, step.Retry.RetryOnStatus) {
			time.Sleep(backoffDelay(attempt, step.Retry))
			continue
		}
		break
	}
}

func ShouldRetry(status int, retryables []int) bool {
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
```

### Paso 9.3: Agregar más assertions

En `internal/assertions/validator.go`, agregar:

```go
// Greater than
if assertion.GreaterThan != nil {
	num, ok := value.(float64)
	if !ok {
		return fmt.Errorf("value is not a number")
	}
	threshold, ok := assertion.GreaterThan.(float64)
	if !ok {
		return fmt.Errorf("threshold is not a number")
	}
	if num <= threshold {
		return fmt.Errorf("value %v not greater than %v", num, threshold)
	}
	return nil
}
```

**Checkpoint:** Funcionalidades básicas completadas.

---

## FASE 10: Primeros Ejemplos Reales (30 min)

### Paso 10.1: Crear ejemplo Pi

```yaml
# examples/pi_integration.yaml
name: "pi-services-check"
description: "Check all Pi homelab services"
version: "1.0"

config:
  timeout_default: 5s
  stop_on_error: false
  variables:
    pi_host: "192.168.1.102"

steps:
  - name: "check-pi-hole"
    request:
      method: "GET"
      url: "http://{{ .pi_host }}:80/admin"
    expect:
      status: 200

  - name: "check-nextcloud"
    request:
      method: "GET"
      url: "http://{{ .pi_host }}/nextcloud"
    expect:
      status: 200

  - name: "check-grafana"
    request:
      method: "GET"
      url: "http://{{ .pi_host }}:3000/api/health"
    expect:
      status: 200
      body:
        - path: "$.database"
          equals: "ok"
```

### Paso 10.2: Testear

```bash
./bin/tango test examples/pi_integration.yaml
```

---

## Checklist de Progreso

- [ ] FASE 1: Setup básico
- [ ] FASE 2: Tipos de datos
- [ ] FASE 3: Parser YAML
- [ ] FASE 4: HTTP client
- [ ] FASE 5: Assertions
- [ ] FASE 6: Executor
- [ ] FASE 7: Output formatting
- [ ] FASE 8: CLI principal
- [ ] FASE 9: Mejoras rápidas
- [ ] FASE 10: Ejemplos reales

---

## Próximos Pasos (No críticos para MVP)

1. Variable substitution con `{{ }}` (usar `text/template`)
2. JSON output format
3. JUnit XML output
4. GitHub Actions integration
5. Loop support
6. Workflow composition (imports)

---

## Tips para No Perderse

- **Después de cada FASE:** Testea con `make test` y `make run`
- **Mantén main.go simple:** Agrega features al executor, no al CLI
- **Los tipos son la guía:** Si algo falla, checa `internal/config/types.go`
- **Cada módulo es independiente:** Puedes testear `config` sin `executor`

---
