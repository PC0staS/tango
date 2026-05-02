// internal/executor/state.go
package executor

import (
	"sync"

	"github.com/pc0stas/tango/internal/config"
)

type ExecutionState struct {
	mu           sync.RWMutex
	capturedVars map[string]map[string]interface{}
	stepStatuses map[string]string
}

func NewExecutionState() *ExecutionState {
	return &ExecutionState{
		capturedVars: make(map[string]map[string]interface{}),
		stepStatuses: make(map[string]string),
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

func (s *ExecutionState) SetStepStatus(stepName, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stepStatuses[stepName] = status
}

func (s *ExecutionState) GetStepStatus(stepName string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stepStatuses[stepName]
}

func (s *ExecutionState) ShouldSkip(step *config.Step) bool {
	if step.Skip {
		return true
	}

	if step.SkipIfFailed != "" {
		if s.GetStepStatus(step.SkipIfFailed) == "failed" {
			return true
		}
	}

	if step.RunIf != nil {
		if step.RunIf.Status != "" && step.RunIf.PreviousStep != "" {
			prevStatus := s.GetStepStatus(step.RunIf.PreviousStep)
			if prevStatus != step.RunIf.Status {
				return true
			}
		}
	}

	return false
}