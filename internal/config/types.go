package config

import "time"

type Workflow struct {
	Name string `yaml:"name"`
	Description string `yaml:"description"`
	Config WorkflowConfig `yaml:"config"`	
	Steps []Step `yaml:"steps"`
}

type WorkflowConfig struct {
	TimeoutDefault time.Duration `yaml:"timeout_default"`
	StopOnError bool `yaml:"stop_on_error"`
	Variables map[string]string `yaml:"variables"`
}

type Step struct {
	Name string `yaml:"name"`
	Description string `yaml:"description"`
	Request RequestSpec `yaml:"request"`
	Expect ExpectSpec `yaml:"expect"`
	DependsOn []string `yaml:"depends_on"`
	Timeout *time.Duration `yaml:"timeout"`
	Capture CaptureSpec `yaml:"capture"`
	Retry RetrySpec `yaml:"retry"`
}

type RequestSpec struct {
	Method string `yaml:"method"`
	URL string `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
	Body string `yaml:"body"`
	BodyFile string `yaml:"body_file"`
	QueryParams map[string]string `yaml:"query_params"`
}

type ExpectSpec struct {
	Status int `yaml:"status"`
	Headers map[string]string `yaml:"headers"`
	Body []AssertionSpec `yaml:"body"`
	ResponseTime *int `yaml:"response_time_ms"`
}

type CaptureSpec struct {
	JSONPath map[string]string `yaml:"json_path"`
}

type RetrySpec struct {
	Count int `yaml:"count"`
	Delay time.Duration `yaml:"delay"`
	BackoffMultiplier float64 `yaml:"backoff_multiplier"`
	RetryOnStatus []int `yaml:"retry_on_status"`
}

type ExecutionResult struct {
	Name string
	Steps []StepResult
	Success bool
	TotalTime time.Duration
	PassedCount int
	FailedCount int
	SkippedCount int
}

type StepResult struct {
	Name string
	Status string // "passed", "failed", "skipped"
	RequestMethod string
	RequestURL string
	RequestDuration time.Duration
	ResponseStatus int
	ResponseBody string
	Error string
	CapturedVars map[string]interface{}
}

type AssertionSpec struct {
	Path            string `yaml:"path"`
	Equals          any    `yaml:"equals,omitempty"`
	Type            string `yaml:"type,omitempty"`
	Contains        string `yaml:"contains,omitempty"`
	Startswith      string `yaml:"startswith,omitempty"`
	Endswith        string `yaml:"endswith,omitempty"`
	Matches         string `yaml:"matches,omitempty"`
	GreaterThan     any    `yaml:"greater_than,omitempty"`
	LessThanOrEqual any    `yaml:"less_than_or_equal,omitempty"`
	Between         []any  `yaml:"between,omitempty"`
	Length          int    `yaml:"length,omitempty"`
	MinLength       int    `yaml:"min_length,omitempty"`
	Empty           *bool  `yaml:"empty,omitempty"`
	Exists          *bool  `yaml:"exists,omitempty"`
	ContainsValue   any    `yaml:"contains_value,omitempty"`
	DeepEquals      any    `yaml:"deep_equals,omitempty"`
}