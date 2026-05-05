// internal/executor/dependency.go
package executor

import (
	"fmt"
	"sort"

	"github.com/pc0stas/tango/internal/config"
)

func TopologicalSort(steps []config.Step) ([]string, error) {
	// Build position map (original order in YAML)
	posMap := make(map[string]int)
	for i, step := range steps {
		posMap[step.Name] = i
	}

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

	// Collect initial in-degree 0 nodes
	queue := make([]string, 0)
	for _, step := range steps {
		if inDegree[step.Name] == 0 {
			queue = append(queue, step.Name)
		}
	}

	result := make([]string, 0)
	for len(queue) > 0 {
		// Sort queue by original YAML position for deterministic output
		sort.Slice(queue, func(i, j int) bool {
			return posMap[queue[i]] < posMap[queue[j]]
		})

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