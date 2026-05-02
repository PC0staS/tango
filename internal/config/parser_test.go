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
	err := validateWorkflow(w)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}