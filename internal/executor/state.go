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