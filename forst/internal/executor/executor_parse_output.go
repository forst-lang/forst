package executor

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseExecutionOutput parses the output of a function execution.
func (e *FunctionExecutor) parseExecutionOutput(output string) (*ExecutionResult, error) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t := strings.TrimSpace(output)
		if err2 := json.Unmarshal([]byte(t), &parsed); err2 != nil {
			return &ExecutionResult{
				Success: true,
				Output:  output,
			}, nil
		}
	}

	// Executor-generated main prints {"success":false,"error":"..."} on Err (RFC §18.1).
	if succ, ok := parsed["success"].(bool); ok && !succ {
		msg := ""
		if s, ok := parsed["error"].(string); ok {
			msg = s
		}
		return &ExecutionResult{
			Success: false,
			Error:   msg,
		}, nil
	}

	resultValue, hasResult := parsed["result"]
	if !hasResult {
		resultData, _ := json.Marshal(parsed)
		return &ExecutionResult{
			Success: true,
			Output:  output,
			Result:  resultData,
		}, nil
	}

	return executionResultFromValue(resultValue), nil
}

func executionResultFromValue(resultValue any) *ExecutionResult {
	switch typedValue := resultValue.(type) {
	case string:
		return &ExecutionResult{
			Success: true,
			Output:  typedValue,
			Result:  []byte(fmt.Sprintf("%q", typedValue)),
		}
	case float64:
		value := fmt.Sprintf("%v", typedValue)
		return &ExecutionResult{
			Success: true,
			Output:  value,
			Result:  []byte(value),
		}
	case int:
		value := fmt.Sprintf("%d", typedValue)
		return &ExecutionResult{
			Success: true,
			Output:  value,
			Result:  []byte(value),
		}
	case bool:
		value := fmt.Sprintf("%t", typedValue)
		return &ExecutionResult{
			Success: true,
			Output:  value,
			Result:  []byte(value),
		}
	default:
		resultData, _ := json.Marshal(resultValue)
		return &ExecutionResult{
			Success: true,
			Output:  string(resultData),
			Result:  resultData,
		}
	}
}
