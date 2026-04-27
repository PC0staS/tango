# Tango — Project Structure & Architecture

---

## 1. Project Layout

```
tango/
├── main.go                         # Entry point, CLI commands
├── go.mod                          # Module definition
├── go.sum
├── Makefile                        # Build, test, release commands
├── README.md
├── LICENSE
│
├── cmd/                            # CLI command implementations
│   ├── test.go                     # `tango test` command logic
│   ├── validate.go                 # `tango validate` command
│   ├── init.go                     # `tango init` template generation
│   └── root.go                     # Root command setup (cobra)
│
├── internal/                       # Private packages (not importable externally)
│   ├── config/
│   │   ├── parser.go               # YAML parsing & validation
│   │   ├── types.go                # Structs for Workflow, Step, Config
│   │   └── defaults.go             # Default values & constants
│   │
│   ├── executor/
│   │   ├── runner.go               # Main execution orchestrator
│   │   ├── dependency.go           # Topological sort, DAG building
│   │   ├── variable.go             # Variable substitution (env, capture, config)
│   │   └── state.go                # Execution state tracking
│   │
│   ├── http/
│   │   ├── client.go               # HTTP client with retries
│   │   ├── request.go              # Request building & sending
│   │   ├── response.go             # Response parsing & storage
│   │   └── retry.go                # Retry logic & backoff
│   │
│   ├── assertions/
│   │   ├── validator.go            # Main assertion validator
│   │   ├── status.go               # Status code assertions
│   │   ├── headers.go              # Header assertions
│   │   ├── jsonpath.go             # JSONPath extraction & comparison
│   │   ├── matchers.go             # Matcher implementations (equals, contains, regex, etc)
│   │   └── types.go                # Assertion result types
│   │
│   ├── output/
│   │   ├── formatter.go            # Output formatting interface
│   │   ├── text.go                 # Text/CLI output formatter
│   │   ├── json.go                 # JSON report formatter
│   │   ├── junit.go                # JUnit XML formatter
│   │   ├── colors.go               # ANSI color codes & styling
│   │   └── types.go                # Report structs
│   │
│   ├── utils/
│   │   ├── logging.go              # Debug/verbose logging
│   │   ├── env.go                  # Environment variable loading (.env parsing)
│   │   ├── file.go                 # File I/O utilities
│   │   └── templates.go            # Template rendering helpers
│   │
│   └── errors/
│       ├── errors.go               # Custom error types & handling
│       └── codes.go                # Exit codes
│
├── pkg/                            # Public packages (reusable by others)
│   ├── workflow/
│   │   ├── workflow.go             # Workflow definition & methods
│   │   ├── step.go                 # Step definition & methods
│   │   └── export.go               # Public types for external use
│   │
│   └── client/                     # Public client library (optional, for programmatic use)
│       ├── client.go               # Execute workflows programmatically
│       └── options.go              # Client options
│
├── test/                           # Test fixtures & integration tests
│   ├── fixtures/
│   │   ├── workflows/
│   │   │   ├── valid_simple.yaml
│   │   │   ├── valid_complex.yaml
│   │   │   ├── invalid_syntax.yaml
│   │   │   ├── invalid_deps.yaml
│   │   │   └── ...
│   │   ├── responses/
│   │   │   ├── success.json
│   │   │   ├── error.json
│   │   │   └── ...
│   │   └── payloads/
│   │       ├── user.json
│   │       ├── bulk_items.json
│   │       └── ...
│   │
│   ├── mocks/
│   │   ├── http_server.go          # Mock HTTP server for testing
│   │   ├── client.go               # Mock HTTP client
│   │   └── ...
│   │
│   └── integration/
│       ├── executor_test.go        # End-to-end executor tests
│       ├── parser_test.go          # YAML parsing tests
│       └── ...
│
├── examples/                       # Example workflows for users
│   ├── health_check.yaml
│   ├── pi_integration.yaml
│   ├── toolbox_auth.yaml
│   ├── esp32_monitoring.yaml
│   └── README.md
│
├── docs/                           # Documentation
│   ├── getting_started.md
│   ├── yaml_schema.md
│   ├── assertions.md
│   ├── advanced.md
│   └── architecture.md
│
└── .github/
    └── workflows/
        ├── test.yml                # CI tests
        ├── release.yml             # Release automation
        └── lint.yml                # Linting & formatting
```

---

## 2. Module Responsibilities & Flow

### 2.1 Entry Point: `main.go`

**Responsibility:** Bootstrap CLI, route commands

```go
package main

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
	"github.com/pc0stas/tango/cmd"
)

func main() {
	rootCmd := cmd.NewRootCommand()
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Dependencies:** `spf13/cobra` for CLI framework

---

### 2.2 Command Layer: `cmd/`

#### `cmd/root.go` — Command Setup
```go
// Creates root command, defines global flags
type RootOptions struct {
	EnvFile string
	Verbose bool
}

func NewRootCommand() *cobra.Command {
	opts := &RootOptions{}
	
	cmd := &cobra.Command{
		Use:   "tango",
		Short: "Distributed testing & event streaming",
	}
	
	cmd.PersistentFlags().StringVar(&opts.EnvFile, "env-file", "", "Load env vars")
	cmd.PersistentFlags().BoolVar(&opts.Verbose, "verbose", false, "Verbose output")
	
	cmd.AddCommand(
		NewTestCommand(opts),
		NewValidateCommand(opts),
		NewInitCommand(opts),
	)
	
	return cmd
}
```

#### `cmd/test.go` — Test Command
```go
// Orchestrates: parse → validate → execute → report
type TestOptions struct {
	WorkflowFile  string
	Output        string
	StopOnError   bool
	Parallel      bool
	IncludeSteps  []string
	ExcludeSteps  []string
	GlobalTimeout time.Duration
	Variables     map[string]string
	JUnitOutput   string
	ReportOutput  string
}

func NewTestCommand(rootOpts *RootOptions) *cobra.Command {
	opts := &TestOptions{}
	
	cmd := &cobra.Command{
		Use:   "test <workflow.yaml>",
		Short: "Execute a test workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(args[0], opts, rootOpts)
		},
	}
	
	// Flag setup
	cmd.Flags().StringVar(&opts.Output, "output", "text", "Output format (text, json, junit)")
	cmd.Flags().BoolVar(&opts.StopOnError, "stop-on-error", false, "Stop on first failure")
	// ... more flags
	
	return cmd
}

func runTest(workflowFile string, opts *TestOptions, rootOpts *RootOptions) error {
	// 1. Load environment
	env, err := utils.LoadEnv(rootOpts.EnvFile)
	if err != nil {
		return err
	}
	
	// 2. Parse workflow
	workflow, err := config.ParseWorkflow(workflowFile, env)
	if err != nil {
		return fmt.Errorf("failed to parse workflow: %w", err)
	}
	
	// 3. Create executor
	exec := executor.NewExecutor(workflow, executor.ExecutorOptions{
		StopOnError:   opts.StopOnError,
		Parallel:      opts.Parallel,
		GlobalTimeout: opts.GlobalTimeout,
		Verbose:       rootOpts.Verbose,
	})
	
	// 4. Execute
	result, err := exec.Run(cmd.Context())
	if err != nil {
		return err
	}
	
	// 5. Format output
	formatter := getFormatter(opts.Output)
	report := formatter.Format(result)
	
	// 6. Save reports if requested
	if opts.ReportOutput != "" {
		_ = os.WriteFile(opts.ReportOutput, report.JSONBytes(), 0644)
	}
	if opts.JUnitOutput != "" {
		_ = os.WriteFile(opts.JUnitOutput, report.JUnitBytes(), 0644)
	}
	
	// 7. Print to stdout
	fmt.Print(report.String())
	
	// 8. Exit with proper code
	if !result.Success {
		os.Exit(1)
	}
	
	return nil
}
```

#### `cmd/validate.go` — Validate Command
```go
// Checks YAML syntax without executing
func NewValidateCommand(rootOpts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <workflow.yaml>",
		Short: "Validate workflow syntax",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, _ := utils.LoadEnv(rootOpts.EnvFile)
			_, err := config.ParseWorkflow(args[0], env)
			if err != nil {
				fmt.Printf("✗ Validation failed: %v\n", err)
				return err
			}
			fmt.Println("✓ Workflow is valid")
			return nil
		},
	}
	return cmd
}
```

#### `cmd/init.go` — Init Command
```go
// Generates template workflow
func NewInitCommand(rootOpts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <workflow_name>",
		Short: "Create a template workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			template := getTemplate("basic")  // or "advanced"
			filename := args[0] + ".yaml"
			return os.WriteFile(filename, []byte(template), 0644)
		},
	}
	return cmd
}
```

---

### 2.3 Config & Parsing: `internal/config/`

#### `internal/config/types.go` — Core Structs
```go
package config

type Workflow struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Version     string                 `yaml:"version"`
	Config      WorkflowConfig         `yaml:"config"`
	Steps       []Step                 `yaml:"steps"`
}

type WorkflowConfig struct {
	TimeoutDefault    time.Duration      `yaml:"timeout_default"`
	RetryFailedSteps  bool               `yaml:"retry_failed_steps"`
	StopOnError       bool               `yaml:"stop_on_error"`
	ParallelSafe      bool               `yaml:"parallel_safe"`
	Variables         map[string]string  `yaml:"variables"`
}

type Step struct {
	Name            string              `yaml:"name"`
	Description     string              `yaml:"description"`
	Request         RequestSpec         `yaml:"request"`
	Expect          ExpectSpec          `yaml:"expect"`
	DependsOn       []string            `yaml:"depends_on"`
	Timeout         *time.Duration      `yaml:"timeout"`
	Skip            bool                `yaml:"skip"`
	SkipIfFailed    string              `yaml:"skip_if_failed"`
	RunIf           RunIfCondition      `yaml:"run_if"`
	Capture         CaptureSpec         `yaml:"capture"`
	Retry           RetrySpec           `yaml:"retry"`
	ExtractAndStore bool                `yaml:"extract_and_store"`
	LogResponse     bool                `yaml:"log_response"`
	Loop            *LoopSpec           `yaml:"loop"`
}

type RequestSpec struct {
	Method       string            `yaml:"method"`
	URL          string            `yaml:"url"`
	Headers      map[string]string `yaml:"headers"`
	Body         string            `yaml:"body"`
	BodyFile     string            `yaml:"body_file"`
	QueryParams  map[string]string `yaml:"query_params"`
}

type ExpectSpec struct {
	Status        int               `yaml:"status"`
	Headers       map[string]string `yaml:"headers"`
	Body          []AssertionSpec   `yaml:"body"`
	ResponseTime  *int              `yaml:"response_time_ms"`
}

type AssertionSpec struct {
	Path              string      `yaml:"path"`
	Equals            interface{} `yaml:"equals"`
	Type              string      `yaml:"type"`
	Contains          string      `yaml:"contains"`
	Startswith        string      `yaml:"startswith"`
	Endswith          string      `yaml:"endswith"`
	Matches           string      `yaml:"matches"`
	GreaterThan       interface{} `yaml:"greater_than"`
	LessThanOrEqual   interface{} `yaml:"less_than_or_equal"`
	Between           []interface{} `yaml:"between"`
	Length            *int        `yaml:"length"`
	MinLength         *int        `yaml:"min_length"`
	Empty             *bool       `yaml:"empty"`
	Exists            *bool       `yaml:"exists"`
	ContainsValue     interface{} `yaml:"contains_value"`
	DeepEquals        interface{} `yaml:"deep_equals"`
}

type CaptureSpec struct {
	ResponseBody string            `yaml:"response_body"`
	JSONPath     map[string]string `yaml:"json_path"`
}

type RetrySpec struct {
	Count              int           `yaml:"count"`
	Delay              time.Duration `yaml:"delay"`
	BackoffMultiplier float64       `yaml:"backoff_multiplier"`
	RetryOnStatus      []int         `yaml:"retry_on_status"`
	RetryOnError       bool          `yaml:"retry_on_error"`
}

type RunIfCondition struct {
	PreviousStep string `yaml:"previous_step"`
	Status       string `yaml:"status"`
	EnvVar       string `yaml:"env_var"`
	EnvVarExists  string `yaml:"env_var_exists"`
}

type LoopSpec struct {
	Items    []string `yaml:"items"`
	Variable string   `yaml:"variable"`
}
```

#### `internal/config/parser.go` — YAML Parsing
```go
package config

import (
	"fmt"
	"os"
	"gopkg.in/yaml.v3"
)

func ParseWorkflow(filename string, env map[string]string) (*Workflow, error) {
	// 1. Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}
	
	// 2. Unmarshal YAML
	var workflow Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	// 3. Validate structure
	if err := ValidateWorkflow(&workflow); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}
	
	// 4. Apply defaults
	ApplyDefaults(&workflow)
	
	// 5. Resolve environment variables in Config.Variables
	ResolveEnvVars(&workflow, env)
	
	// 6. Load body files
	if err := LoadBodyFiles(&workflow); err != nil {
		return nil, fmt.Errorf("failed to load body files: %w", err)
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
	
	// Check for duplicate step names
	seen := make(map[string]bool)
	for _, step := range w.Steps {
		if seen[step.Name] {
			return fmt.Errorf("duplicate step name: %s", step.Name)
		}
		seen[step.Name] = true
		
		if step.Name == "" {
			return fmt.Errorf("all steps must have a name")
		}
		if step.Request.Method == "" {
			return fmt.Errorf("step %s: must have a method", step.Name)
		}
		if step.Request.URL == "" {
			return fmt.Errorf("step %s: must have a URL", step.Name)
		}
	}
	
	// Check for invalid dependencies
	for _, step := range w.Steps {
		for _, dep := range step.DependsOn {
			if !seen[dep] {
				return fmt.Errorf("step %s depends on non-existent step: %s", step.Name, dep)
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

func ResolveEnvVars(w *Workflow, env map[string]string) {
	// Merge env into w.Config.Variables
	if w.Config.Variables == nil {
		w.Config.Variables = make(map[string]string)
	}
	for k, v := range env {
		if _, exists := w.Config.Variables[k]; !exists {
			w.Config.Variables[k] = v
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
		}
	}
	return nil
}
```

#### `internal/config/defaults.go` — Constants
```go
package config

const (
	DefaultTimeout = 10 * time.Second
	DefaultRetries = 1
	DefaultMethod  = "GET"
)

var (
	RetryableStatusCodes = []int{408, 429, 500, 502, 503, 504}
)
```

---

### 2.4 Execution Engine: `internal/executor/`

#### `internal/executor/runner.go` — Main Orchestrator
```go
package executor

import (
	"context"
	"fmt"
	"sync"
	"time"
	"github.com/pc0stas/tango/internal/config"
	"github.com/pc0stas/tango/internal/http"
	"github.com/pc0stas/tango/internal/assertions"
	"github.com/pc0stas/tango/internal/utils"
)

type ExecutorOptions struct {
	StopOnError   bool
	Parallel      bool
	GlobalTimeout time.Duration
	Verbose       bool
	Variables     map[string]string
}

type Executor struct {
	workflow *config.Workflow
	options  ExecutorOptions
	client   *http.Client
	logger   *utils.Logger
}

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
	Name             string
	Status           string // "passed", "failed", "skipped"
	RequestDuration  time.Duration
	Error            string
	ResponseStatus   int
	ResponseHeaders  map[string]string
	ResponseBody     string
	AssertionErrors  []string
	CapturedVars     map[string]interface{}
	RequestURL       string
	RequestMethod    string
}

func NewExecutor(workflow *config.Workflow, opts ExecutorOptions) *Executor {
	return &Executor{
		workflow: workflow,
		options:  opts,
		client:   http.NewClient(opts.GlobalTimeout),
		logger:   utils.NewLogger(opts.Verbose),
	}
}

func (e *Executor) Run(ctx context.Context) (*ExecutionResult, error) {
	startTime := time.Now()
	result := &ExecutionResult{
		Name:  e.workflow.Name,
		Steps: make([]StepResult, 0),
	}
	
	// 1. Build dependency graph
	dag, err := dependency.BuildDAG(e.workflow)
	if err != nil {
		return result, fmt.Errorf("dependency graph error: %w", err)
	}
	
	// 2. Initialize state
	state := NewExecutionState(e.workflow, e.options.Variables)
	
	// 3. Get execution order
	order, err := dependency.TopologicalSort(dag)
	if err != nil {
		return result, fmt.Errorf("circular dependency detected: %w", err)
	}
	
	e.logger.Debugf("Execution order: %v", order)
	
	// 4. Execute steps
	for _, stepName := range order {
		step := findStep(e.workflow, stepName)
		if step == nil {
			continue
		}
		
		// Check skip conditions
		if e.shouldSkip(step, state) {
			result.Steps = append(result.Steps, StepResult{
				Name:   step.Name,
				Status: "skipped",
			})
			result.SkippedCount++
			continue
		}
		
		// Execute step
		stepResult, err := e.executeStep(ctx, step, state)
		if err != nil {
			e.logger.Errorf("Step %s failed: %v", step.Name, err)
			stepResult.Status = "failed"
			stepResult.Error = err.Error()
			result.FailedCount++
			
			if e.options.StopOnError {
				result.Steps = append(result.Steps, stepResult)
				break
			}
		} else {
			stepResult.Status = "passed"
			result.PassedCount++
		}
		
		// Update state with captured variables
		if stepResult.CapturedVars != nil {
			state.CaptureVariables(step.Name, stepResult.CapturedVars)
		}
		
		result.Steps = append(result.Steps, stepResult)
	}
	
	result.TotalTime = time.Since(startTime)
	result.Success = result.FailedCount == 0
	
	return result, nil
}

func (e *Executor) executeStep(ctx context.Context, step *config.Step, state *ExecutionState) (StepResult, error) {
	stepResult := StepResult{
		Name:            step.Name,
		RequestMethod:   step.Request.Method,
	}
	
	// 1. Resolve URL with variables
	url, err := variable.Resolve(step.Request.URL, state)
	if err != nil {
		return stepResult, fmt.Errorf("failed to resolve URL: %w", err)
	}
	stepResult.RequestURL = url
	
	e.logger.Debugf("Executing %s %s", step.Request.Method, url)
	
	// 2. Resolve headers
	headers, err := e.resolveHeaders(step.Request.Headers, state)
	if err != nil {
		return stepResult, fmt.Errorf("failed to resolve headers: %w", err)
	}
	
	// 3. Resolve body
	body, err := variable.Resolve(step.Request.Body, state)
	if err != nil {
		return stepResult, fmt.Errorf("failed to resolve body: %w", err)
	}
	
	// 4. Build request
	req := http.BuildRequest(step.Request.Method, url, headers, body, step.Request.QueryParams)
	
	// 5. Execute with retries
	var response *http.Response
	for attempt := 1; attempt <= step.Retry.Count; attempt++ {
		response, err = e.executeWithTimeout(ctx, req, *step.Timeout)
		
		if err == nil && shouldRetry(response.StatusCode, step.Retry) {
			delay := backoffDelay(attempt, step.Retry)
			e.logger.Debugf("Retrying after %v (attempt %d/%d)", delay, attempt, step.Retry.Count)
			time.Sleep(delay)
			continue
		}
		break
	}
	
	if err != nil {
		return stepResult, fmt.Errorf("request failed: %w", err)
	}
	
	stepResult.RequestDuration = response.Duration
	stepResult.ResponseStatus = response.StatusCode
	stepResult.ResponseHeaders = response.Headers
	stepResult.ResponseBody = response.Body
	
	e.logger.Debugf("Response: %d [%v]", response.StatusCode, response.Duration)
	
	// 6. Validate expectations
	if err := assertions.Validate(step.Expect, response); err != nil {
		return stepResult, fmt.Errorf("assertion failed: %w", err)
	}
	
	// 7. Capture variables
	if step.Capture.JSONPath != nil {
		captured, err := e.captureVariables(response.Body, step.Capture.JSONPath)
		if err != nil {
			return stepResult, fmt.Errorf("failed to capture variables: %w", err)
		}
		stepResult.CapturedVars = captured
	}
	
	return stepResult, nil
}

func (e *Executor) shouldSkip(step *config.Step, state *ExecutionState) bool {
	if step.Skip {
		return true
	}
	
	if step.SkipIfFailed != "" {
		prevStatus := state.GetStepStatus(step.SkipIfFailed)
		if prevStatus == "failed" {
			return true
		}
	}
	
	// Implement RunIfCondition logic here
	
	return false
}

func (e *Executor) executeWithTimeout(ctx context.Context, req *http.Request, timeout time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	return e.client.Do(ctx, req)
}

func (e *Executor) resolveHeaders(headers map[string]string, state *ExecutionState) (map[string]string, error) {
	resolved := make(map[string]string)
	for k, v := range headers {
		val, err := variable.Resolve(v, state)
		if err != nil {
			return nil, err
		}
		resolved[k] = val
	}
	return resolved, nil
}

func (e *Executor) captureVariables(body string, jsonPaths map[string]string) (map[string]interface{}, error) {
	captured := make(map[string]interface{})
	for varName, path := range jsonPaths {
		val, err := jsonpath.Extract(body, path)
		if err != nil {
			return nil, fmt.Errorf("failed to extract %s from %s: %w", varName, path, err)
		}
		captured[varName] = val
	}
	return captured, nil
}

func findStep(w *config.Workflow, name string) *config.Step {
	for i := range w.Steps {
		if w.Steps[i].Name == name {
			return &w.Steps[i]
		}
	}
	return nil
}
```

#### `internal/executor/dependency.go` — DAG & Topological Sort
```go
package executor

import (
	"fmt"
	"github.com/pc0stas/tango/internal/config"
)

type DAG struct {
	nodes map[string]*DAGNode
}

type DAGNode struct {
	StepName   string
	Dependencies []string
}

func BuildDAG(w *config.Workflow) (*DAG, error) {
	dag := &DAG{
		nodes: make(map[string]*DAGNode),
	}
	
	for _, step := range w.Steps {
		dag.nodes[step.Name] = &DAGNode{
			StepName:     step.Name,
			Dependencies: step.DependsOn,
		}
	}
	
	return dag, nil
}

func TopologicalSort(dag *DAG) ([]string, error) {
	// Kahn's algorithm
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)
	
	// Initialize
	for name := range dag.nodes {
		inDegree[name] = 0
	}
	
	// Build adjacency list and in-degrees
	for name, node := range dag.nodes {
		for _, dep := range node.Dependencies {
			adjList[dep] = append(adjList[dep], name)
			inDegree[name]++
		}
	}
	
	// Queue of nodes with no dependencies
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
	
	if len(result) != len(dag.nodes) {
		return nil, fmt.Errorf("circular dependency detected")
	}
	
	return result, nil
}
```

#### `internal/executor/state.go` — Execution State
```go
package executor

import (
	"fmt"
	"sync"
)

type ExecutionState struct {
	mu              sync.RWMutex
	configVars      map[string]string
	capturedVars    map[string]map[string]interface{}
	stepStatus      map[string]string
}

func NewExecutionState(w *config.Workflow, globalVars map[string]string) *ExecutionState {
	vars := make(map[string]string)
	for k, v := range w.Config.Variables {
		vars[k] = v
	}
	for k, v := range globalVars {
		vars[k] = v
	}
	
	return &ExecutionState{
		configVars:   vars,
		capturedVars: make(map[string]map[string]interface{}),
		stepStatus:   make(map[string]string),
	}
}

func (s *ExecutionState) CaptureVariables(stepName string, vars map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.capturedVars[stepName] = vars
}

func (s *ExecutionState) GetCaptured(stepName, varName string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stepVars, ok := s.capturedVars[stepName]
	if !ok {
		return nil, fmt.Errorf("no captures from step %s", stepName)
	}
	
	val, ok := stepVars[varName]
	if !ok {
		return nil, fmt.Errorf("variable %s not captured from step %s", varName, stepName)
	}
	
	return val, nil
}

func (s *ExecutionState) GetEnv(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configVars[key]
}

func (s *ExecutionState) GetStepStatus(stepName string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stepStatus[stepName]
}

func (s *ExecutionState) SetStepStatus(stepName, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stepStatus[stepName] = status
}
```

#### `internal/executor/variable.go` — Variable Substitution
```go
package executor

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

var varPattern = regexp.MustCompile(`\{\{\s*(.+?)\s*\}\}`)

func (v *Variable) Resolve(text string, state *ExecutionState) (string, error) {
	if !strings.Contains(text, "{{") {
		return text, nil
	}
	
	// Use text/template for variable substitution
	tmpl, err := template.New("var").Parse(text)
	if err != nil {
		return "", err
	}
	
	// Build template data
	data := map[string]interface{}{
		"env":    state.configVars,
		"steps":  buildStepsContext(state),
		"config": buildConfigContext(state),
	}
	
	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", err
	}
	
	return result.String(), nil
}

func buildStepsContext(state *ExecutionState) map[string]map[string]interface{} {
	steps := make(map[string]map[string]interface{})
	
	for stepName, vars := range state.capturedVars {
		steps[stepName] = vars
	}
	
	return steps
}

func buildConfigContext(state *ExecutionState) map[string]string {
	return state.configVars
}
```

---

### 2.5 HTTP Client: `internal/http/`

#### `internal/http/client.go` — HTTP Client
```go
package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	timeout    time.Duration
}

type Request struct {
	Method      string
	URL         string
	Headers     map[string]string
	Body        string
	QueryParams map[string]string
}

type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       string
	Duration   time.Duration
}

func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		timeout:    timeout,
	}
}

func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()
	
	// Build URL with query params
	url := req.URL
	if len(req.QueryParams) > 0 {
		// Add query params
		// TODO: implement query param encoding
	}
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, strings.NewReader(req.Body))
	if err != nil {
		return nil, err
	}
	
	// Add headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	
	// Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()
	
	// Read response body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// Convert headers
	respHeaders := make(map[string]string)
	for k, v := range httpResp.Header {
		respHeaders[k] = v[0]
	}
	
	return &Response{
		StatusCode: httpResp.StatusCode,
		Headers:    respHeaders,
		Body:       string(bodyBytes),
		Duration:   time.Since(start),
	}, nil
}

func BuildRequest(method, url string, headers map[string]string, body string, queryParams map[string]string) *Request {
	return &Request{
		Method:      method,
		URL:         url,
		Headers:     headers,
		Body:        body,
		QueryParams: queryParams,
	}
}
```

#### `internal/http/retry.go` — Retry Logic
```go
package http

import (
	"time"
)

func ShouldRetry(statusCode int, retryableStatuses []int) bool {
	for _, status := range retryableStatuses {
		if statusCode == status {
			return true
		}
	}
	return false
}

func BackoffDelay(attempt int, initialDelay time.Duration, multiplier float64) time.Duration {
	delay := time.Duration(float64(initialDelay) * math.Pow(multiplier, float64(attempt-1)))
	return delay
}
```

---

### 2.6 Assertions: `internal/assertions/`

#### `internal/assertions/validator.go` — Main Validator
```go
package assertions

import (
	"fmt"
	"github.com/pc0stas/tango/internal/config"
	"github.com/pc0stas/tango/internal/http"
)

func Validate(expect *config.ExpectSpec, response *http.Response) error {
	// 1. Validate status
	if expect.Status != 0 && response.StatusCode != expect.Status {
		return fmt.Errorf("status code mismatch: expected %d, got %d", expect.Status, response.StatusCode)
	}
	
	// 2. Validate headers
	for k, v := range expect.Headers {
		if response.Headers[k] != v {
			return fmt.Errorf("header %s mismatch: expected %s, got %s", k, v, response.Headers[k])
		}
	}
	
	// 3. Validate body assertions
	for _, assertion := range expect.Body {
		if err := ValidateAssertion(&assertion, response.Body); err != nil {
			return err
		}
	}
	
	// 4. Validate response time
	if expect.ResponseTime != nil && response.Duration.Milliseconds() > int64(*expect.ResponseTime) {
		return fmt.Errorf("response time exceeded: %dms > %dms", response.Duration.Milliseconds(), *expect.ResponseTime)
	}
	
	return nil
}

func ValidateAssertion(assertion *config.AssertionSpec, responseBody string) error {
	// Get value from JSONPath
	value, err := jsonpath.Extract(responseBody, assertion.Path)
	if err != nil {
		return fmt.Errorf("failed to extract path %s: %w", assertion.Path, err)
	}
	
	// Apply matcher
	return applyMatcher(assertion, value)
}

func applyMatcher(assertion *config.AssertionSpec, value interface{}) error {
	if assertion.Equals != nil {
		return matchers.EqualsMatch(value, assertion.Equals)
	}
	if assertion.Contains != "" {
		return matchers.ContainsMatch(value, assertion.Contains)
	}
	if assertion.Matches != "" {
		return matchers.RegexMatch(value, assertion.Matches)
	}
	if assertion.Type != "" {
		return matchers.TypeMatch(value, assertion.Type)
	}
	if assertion.Empty != nil {
		return matchers.EmptyMatch(value, *assertion.Empty)
	}
	// ... more matchers
	
	return nil
}
```

#### `internal/assertions/jsonpath.go` — JSONPath Extraction
```go
package assertions

import (
	"github.com/tidwall/gjson"
)

func Extract(jsonStr, path string) (interface{}, error) {
	result := gjson.Get(jsonStr, path)
	if !result.Exists() {
		return nil, fmt.Errorf("path not found: %s", path)
	}
	return result.Value(), nil
}
```

#### `internal/assertions/matchers.go` — Assertion Matchers
```go
package assertions

import (
	"fmt"
	"regexp"
	"strconv"
)

func EqualsMatch(actual, expected interface{}) error {
	if actual != expected {
		return fmt.Errorf("value mismatch: expected %v, got %v", expected, actual)
	}
	return nil
}

func ContainsMatch(actual interface{}, substring string) error {
	str, ok := actual.(string)
	if !ok {
		return fmt.Errorf("value is not a string")
	}
	if !strings.Contains(str, substring) {
		return fmt.Errorf("value does not contain %q", substring)
	}
	return nil
}

func RegexMatch(actual interface{}, pattern string) error {
	str, ok := actual.(string)
	if !ok {
		return fmt.Errorf("value is not a string")
	}
	matched, err := regexp.MatchString(pattern, str)
	if err != nil {
		return fmt.Errorf("invalid regex: %w", err)
	}
	if !matched {
		return fmt.Errorf("value does not match pattern %q", pattern)
	}
	return nil
}

func TypeMatch(actual interface{}, expectedType string) error {
	var actualType string
	switch actual.(type) {
	case float64:
		actualType = "number"
	case string:
		actualType = "string"
	case bool:
		actualType = "boolean"
	case []interface{}:
		actualType = "array"
	case map[string]interface{}:
		actualType = "object"
	case nil:
		actualType = "null"
	default:
		actualType = "unknown"
	}
	
	if actualType != expectedType {
		return fmt.Errorf("type mismatch: expected %s, got %s", expectedType, actualType)
	}
	return nil
}

func EmptyMatch(actual interface{}, shouldBeEmpty bool) error {
	var isEmpty bool
	switch v := actual.(type) {
	case []interface{}:
		isEmpty = len(v) == 0
	case string:
		isEmpty = v == ""
	case map[string]interface{}:
		isEmpty = len(v) == 0
	default:
		isEmpty = v == nil
	}
	
	if isEmpty != shouldBeEmpty {
		return fmt.Errorf("empty check failed: expected empty=%v, got empty=%v", shouldBeEmpty, isEmpty)
	}
	return nil
}
```

---

### 2.7 Output Formatting: `internal/output/`

#### `internal/output/formatter.go` — Formatter Interface
```go
package output

import "github.com/pc0stas/tango/internal/executor"

type Formatter interface {
	Format(*executor.ExecutionResult) Report
}

type Report interface {
	String() string
	JSONBytes() []byte
	JUnitBytes() []byte
}
```

#### `internal/output/text.go` — Text Formatter
```go
package output

import (
	"fmt"
	"strings"
	"github.com/pc0stas/tango/internal/executor"
)

type TextFormatter struct{}

func (f *TextFormatter) Format(result *executor.ExecutionResult) Report {
	var sb strings.Builder
	
	for _, step := range result.Steps {
		icon := getStatusIcon(step.Status)
		duration := fmt.Sprintf("[%dms]", step.RequestDuration.Milliseconds())
		
		sb.WriteString(fmt.Sprintf("%s %s (%s %s) — %d %s %s\n",
			icon, step.Name, step.RequestMethod, step.RequestURL,
			step.ResponseStatus, getStatusText(step.ResponseStatus), duration))
		
		if step.Error != "" {
			sb.WriteString(fmt.Sprintf("  └─ Error: %s\n", step.Error))
		}
		if len(step.CapturedVars) > 0 {
			sb.WriteString(fmt.Sprintf("  └─ Captured: %v\n", step.CapturedVars))
		}
	}
	
	sb.WriteString(fmt.Sprintf("\nSummary: %d passed, %d failed, %d skipped (out of %d steps)\n",
		result.PassedCount, result.FailedCount, result.SkippedCount, len(result.Steps)))
	sb.WriteString(fmt.Sprintf("Total time: %v\n", result.TotalTime))
	
	return &TextReport{content: sb.String()}
}

func getStatusIcon(status string) string {
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
```

#### `internal/output/json.go` — JSON Formatter
```go
package output

import (
	"encoding/json"
	"github.com/pc0stas/tango/internal/executor"
)

type JSONReport struct {
	Name        string                 `json:"name"`
	Timestamp   string                 `json:"timestamp"`
	TotalSteps  int                    `json:"total_steps"`
	Passed      int                    `json:"passed"`
	Failed      int                    `json:"failed"`
	Skipped     int                    `json:"skipped"`
	TotalTime   int64                  `json:"total_duration_ms"`
	Steps       []executor.StepResult  `json:"steps"`
}

type JSONFormatter struct{}

func (f *JSONFormatter) Format(result *executor.ExecutionResult) Report {
	report := JSONReport{
		Name:       result.Name,
		Timestamp:  time.Now().ISO8601(),
		TotalSteps: len(result.Steps),
		Passed:     result.PassedCount,
		Failed:     result.FailedCount,
		Skipped:    result.SkippedCount,
		TotalTime:  result.TotalTime.Milliseconds(),
		Steps:      result.Steps,
	}
	
	data, _ := json.MarshalIndent(report, "", "  ")
	return &JSONReportWrapper{data: data}
}
```

---

### 2.8 Utilities: `internal/utils/`

#### `internal/utils/logging.go`
```go
package utils

import "fmt"

type Logger struct {
	verbose bool
}

func NewLogger(verbose bool) *Logger {
	return &Logger{verbose: verbose}
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.verbose {
		fmt.Printf("[DEBUG] %s\n", fmt.Sprintf(format, args...))
	}
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] %s\n", fmt.Sprintf(format, args...))
}
```

#### `internal/utils/env.go`
```go
package utils

import (
	"bufio"
	"os"
	"strings"
)

func LoadEnv(filename string) (map[string]string, error) {
	vars := make(map[string]string)
	
	if filename == "" {
		return vars, nil
	}
	
	file, err := os.Open(filename)
	if err != nil {
		return vars, err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			vars[parts[0]] = parts[1]
		}
	}
	
	return vars, scanner.Err()
}
```

---

## 3. Data Flow Diagram

```
┌─────────────────────────────────────────────────┐
│  main.go → Cobra CLI                            │
└──────────────────┬──────────────────────────────┘
                   │
         ┌─────────▼─────────┐
         │  cmd/test.go      │
         │  - Parse flags    │
         │  - Load env vars  │
         └────────┬──────────┘
                  │
         ┌────────▼──────────────────┐
         │ config/parser.go          │
         │ - Parse YAML              │
         │ - Validate workflow       │
         │ - Apply defaults          │
         │ - Load body files         │
         └────────┬──────────────────┘
                  │
         ┌────────▼──────────────────┐
         │ executor/runner.go        │
         │ - Build DAG               │
         │ - Topological sort        │
         │ - Execute steps (loop)    │
         └────────┬──────────────────┘
                  │
         ┌────────▼──────────────────────────┐
         │ For each step:                     │
         ├────────────────────────────────────┤
         │ 1. executor/variable.go            │
         │    - Resolve URL, headers, body    │
         │                                    │
         │ 2. http/client.go                  │
         │    - Build & send HTTP request     │
         │    - Retry logic                   │
         │                                    │
         │ 3. assertions/validator.go         │
         │    - Validate status, headers      │
         │    - JSONPath extraction           │
         │    - Apply matchers                │
         │                                    │
         │ 4. executor/state.go               │
         │    - Capture variables             │
         │    - Store in ExecutionState       │
         └────────┬──────────────────────────┘
                  │
         ┌────────▼──────────────────┐
         │ output/formatter.go       │
         │ - Format results          │
         │ - Text/JSON/JUnit         │
         └────────┬──────────────────┘
                  │
         ┌────────▼──────────────────┐
         │ cmd/test.go               │
         │ - Print output            │
         │ - Save reports            │
         │ - Exit code               │
         └──────────────────────────┘
```

---

## 4. Module Dependency Graph

```
main.go
  ├── cmd/
  │   ├── test.go
  │   ├── validate.go
  │   ├── init.go
  │   └── root.go
  │       ├── config/parser.go
  │       ├── executor/runner.go
  │       │   ├── executor/dependency.go
  │       │   ├── executor/state.go
  │       │   ├── executor/variable.go
  │       │   ├── http/client.go
  │       │   │   └── http/retry.go
  │       │   └── assertions/validator.go
  │       │       ├── assertions/jsonpath.go
  │       │       └── assertions/matchers.go
  │       ├── output/formatter.go
  │       │   ├── output/text.go
  │       │   ├── output/json.go
  │       │   └── output/junit.go
  │       └── utils/
  │           ├── logging.go
  │           ├── env.go
  │           └── file.go
```

---

## 5. Testing Strategy

### Unit Tests
```
test/
├── config/
│   ├── parser_test.go         # Test YAML parsing
│   └── defaults_test.go       # Test default values
├── executor/
│   ├── runner_test.go         # Test execution logic
│   ├── dependency_test.go     # Test topological sort
│   └── variable_test.go       # Test variable resolution
├── assertions/
│   ├── validator_test.go      # Test assertion validation
│   ├── matchers_test.go       # Test individual matchers
│   └── jsonpath_test.go       # Test JSONPath extraction
└── http/
    └── retry_test.go          # Test retry logic
```

### Integration Tests
```
test/
└── integration/
    ├── full_flow_test.go      # End-to-end tests
    ├── fixtures/
    │   ├── workflows/*.yaml   # Test workflows
    │   └── responses/*.json    # Mock responses
    └── mocks/
        └── http_server.go     # Mock HTTP server
```

### Mock HTTP Server (for testing)
```go
// test/mocks/http_server.go
func NewMockServer() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mock responses based on path
	})
	return httptest.NewServer(handler)
}
```

---

## 6. Build & Release

### `Makefile`
```makefile
.PHONY: build test lint run clean release

build:
	go build -o bin/tango .

test:
	go test -v ./...

lint:
	golangci-lint run

run:
	go run main.go test examples/health_check.yaml

clean:
	rm -rf bin/

release:
	# Create releases for linux, macOS, windows
	goreleaser release --clean
```

### GitHub Actions (`.github/workflows/release.yml`)
- Build on tag
- Create binaries for linux/macOS/windows
- Publish to GitHub Releases
- Publish to Homebrew, Snap (future)

---

## 7. Configuration Files

### `go.mod`
```
module github.com/pc0stas/tango

go 1.23

require (
	github.com/spf13/cobra v1.8.0
	gopkg.in/yaml.v3 v3.0.1
	github.com/tidwall/gjson v1.17.1
)
```

### `goreleaser.yaml` (for releases)
```yaml
project_name: tango

builds:
  - main: ./main.go
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64

archives:
  - format: tar.gz
    os: linux
  - format: tar.gz
    os: darwin
  - format: zip
    os: windows

release:
  github:
    owner: pc0stas
    name: tango
```

---

## 8. Getting Started (For Development)

### Clone & Setup
```bash
git clone https://github.com/pc0stas/tango
cd tango
go mod download
```

### Build
```bash
make build
./bin/tango test examples/health_check.yaml
```

### Test
```bash
make test
```

### Debug
```bash
go run main.go test examples/health_check.yaml --verbose
```

---

## 9. Phased Implementation Order

### Phase 1: Core (Week 1-2)
1. ✓ Setup project structure
2. ✓ Implement config/parser.go (YAML → structs)
3. ✓ Implement executor/runner.go (sequential execution)
4. ✓ Implement executor/dependency.go (DAG + topological sort)
5. ✓ Implement http/client.go (basic HTTP requests)
6. ✓ Implement assertions/validator.go + matchers (status, basic body)
7. ✓ Implement executor/variable.go (basic substitution)
8. ✓ Implement output/text.go (CLI output)
9. ✓ Implement cmd/test.go & main.go (CLI)

### Phase 2: Advanced (Week 3)
1. ✓ Implement http/retry.go (retry + backoff)
2. ✓ Implement executor/state.go (capture variables)
3. ✓ Advanced assertions (regex, numeric, array)
4. ✓ output/json.go (JSON report)
5. ✓ cmd/validate.go
6. ✓ cmd/init.go (template generation)

### Phase 3: Polish (Week 4)
1. ✓ output/junit.go (JUnit XML)
2. ✓ Comprehensive tests
3. ✓ Documentation
4. ✓ Examples
5. ✓ Release pipeline

---

## 10. Summary

**Key Insights:**
- **Separation of concerns:** Each module has one job
- **Data flow:** Config → Executor → Output
- **Reusability:** `internal/` is private, `pkg/` can be public
- **Testability:** Mock servers & fixtures for integration tests
- **Maintainability:** Clear dependency graph, no circular imports

**Critical Path:** Config parsing → Executor (with DAG) → HTTP client → Assertions → Output formatting

**Dependencies to minimize:** Go stdlib + `cobra`, `yaml.v3`, `gjson` only

