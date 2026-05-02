package config

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

func ParseWorkflow(filename string) (*Workflow, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file %s: %w", filename, err)
	}

	var workflow Workflow
	err = yaml.Unmarshal(data, &workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	if err := validateWorkflow(&workflow); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	applyDefaults(&workflow)

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

	seen := make(map[string]bool)
	for _, step := range w.Steps {
		if step.Name == "" {
			return fmt.Errorf("step name is required")
		}
		if seen[step.Name] {
			return fmt.Errorf("duplicate step name: %s", step.Name)
		}
		seen[step.Name] = true

		if step.Request.Method == "" {
			return fmt.Errorf("step %s: request method is required", step.Name)
		}
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

		if step.RunIf != nil && step.RunIf.PreviousStep != "" {
			addDepIfMissing(&step.DependsOn, step.RunIf.PreviousStep)
		}
		if step.SkipIfFailed != "" {
			addDepIfMissing(&step.DependsOn, step.SkipIfFailed)
		}
		for _, dep := range findStepRefs(step.Request.URL) {
			addDepIfMissing(&step.DependsOn, dep)
		}
		for _, dep := range findStepRefs(step.Request.Body) {
			addDepIfMissing(&step.DependsOn, dep)
		}
		for _, hdrVal := range step.Request.Headers {
			for _, dep := range findStepRefs(hdrVal) {
				addDepIfMissing(&step.DependsOn, dep)
			}
		}
	}
}

func addDepIfMissing(deps *[]string, target string) {
	for _, d := range *deps {
		if d == target {
			return
		}
	}
	*deps = append(*deps, target)
}

func findStepRefs(s string) []string {
	var refs []string
	re := regexp.MustCompile(`\{\{\s*steps\.([^.]+)\.`)
	matches := re.FindAllStringSubmatch(s, -1)
	for _, m := range matches {
		refs = append(refs, m[1])
	}
	return refs
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
