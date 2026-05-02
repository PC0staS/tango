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
		401: "Unauthorized",
		403: "Forbidden",
		404: "Not Found",
		500: "Internal Server Error",
	}
	if s, ok := statuses[code]; ok {
		return s
	}
	return "Unknown"
}