package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func ParseWorkflow(filename string ) (*Workflow, error) {
	// 1. Read file content
	data , err := os.ReadFile(filename)
	if err != nil {
		return nil,fmt.Errorf("failed to read workflow file %s: %w", filename, err)
	}
	// 2. Unmarshal YAML
	var workflow Workflow
	err = yaml.Unmarshal(data, &workflow)
	if err != nil {
		return nil,fmt.Errorf("failed to parse workflow YAML: %w", err)
	}
	// 3. Validate workflow structure
	if err := validateWorkflow(&workflow); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}
	// 4. Apply default values
	applyDefaults(&workflow)
	// 5. Load body files
	err = loadBodyFiles(&workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to load body files: %w", err)
	}

	return &workflow, nil
}

func validateWorkflow(w *Workflow) error {
	if w.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	if len(w.Steps) == 0 {
		return fmt.Errorf("workflow must contain at least one step")
	}
	// Unique step names
	seen := make(map[string]bool)
	for _ , step := range w.Steps {
		if step.Name == "" {
			return fmt.Errorf("step name is required")
		}
		if seen[step.Name] {
			return fmt.Errorf("duplicate step name: %s", step.Name)
		}
		seen[step.Name] = true

		// Validate request method
		if step.Request.Method == "" {
			return fmt.Errorf("step %s: request method is required", step.Name)
		}
		// Validate request URL
		if step.Request.URL == "" {
			return fmt.Errorf("step %s: request URL is required", step.Name)
		}
	}
	for _, step := range w.Steps {
		for _, dep := range step.DependsOn {
			if !seen[dep] {
				return fmt.Errorf("step %s depends on undefined step %s", step.Name, dep)
			}
		}
	}
	return nil
}
func applyDefaults(w *Workflow) {
	if w.Config.TimeoutDefault == 0 {
		w.Config.TimeoutDefault = 10 * time.Second
	}
	for i := range w.Steps {
		step := &w.Steps[i] 
		if step.Request.Method == "" {
			step.Request.Method = "GET"
		}
		if step.Timeout == nil {
			step.Timeout = &w.Config.TimeoutDefault
		}
		if step.Retry.Count == 0 {
			step.Retry.Count = 0
		}
		if step.Retry.Delay == 0 {
			step.Retry.Delay = 1 * time.Second
		}
		if step.Retry.BackoffMultiplier == 0 {
			step.Retry.BackoffMultiplier = 2.0
		}
	}
}
func loadBodyFiles(w *Workflow) error {
	for i := range w.Steps {
		step := &w.Steps[i]
		if step.Request.BodyFile != "" {
			data, err := os.ReadFile(step.Request.BodyFile)
			if err != nil {
				return fmt.Errorf("failed to read body file for step %s: %w", step.Name, err)
			}
			step.Request.Body = string(data)
			step.Request.BodyFile = ""
		}
	}
	return nil
}