// internal/executor/dependency.go
package executor

import (
	"fmt"

	"github.com/pc0stas/tango/internal/config"
)

func TopologicalSort(steps []config.Step) ([]string, error) {
	// Build graph and in-degree map
	stepMap := make(map[string]bool)
	inDegree := make(map[string]int)

	for _, step := range steps {
		stepMap[step.Name] = true
		inDegree[step.Name] = 0
	}

	// Build adjacency list and calculate in-degrees
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