package executor

import (
	"encoding/json"
	"fmt"

	"forst/internal/forsterr"
)

// parseExecutionOutput parses the output of a function execution.
func (e *FunctionExecutor) parseExecutionOutput(output string) (*ExecutionResult, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		return &ExecutionResult{
			Success: true,
			Output:  output,
		}, nil
	}

	if successRaw, ok := raw["success"]; ok {
		var success bool
		if err := json.Unmarshal(successRaw, &success); err == nil && !success {
			var envelope struct {
				Success    bool                `json:"success"`
				Output     string              `json:"output"`
				Error      string              `json:"error"`
				ErrorValue *forsterr.WireError `json:"errorValue"`
				Result     json.RawMessage     `json:"result"`
			}
			if err := json.Unmarshal([]byte(output), &envelope); err != nil {
				return nil, err
			}
			return &ExecutionResult{
				Success:    false,
				Output:     envelope.Output,
				Error:      envelope.Error,
				ErrorValue: envelope.ErrorValue,
				Result:     envelope.Result,
			}, nil
		}
	}

	resultValueRaw, hasResult := raw["result"]
	if !hasResult {
		resultData, _ := json.Marshal(raw)
		return &ExecutionResult{
			Success: true,
			Output:  output,
			Result:  resultData,
		}, nil
	}

	return executionResultFromRawResult(resultValueRaw), nil
}

func executionResultFromRawResult(raw json.RawMessage) *ExecutionResult {
	var resultValue any
	if err := json.Unmarshal(raw, &resultValue); err != nil {
		return &ExecutionResult{
			Success: true,
			Output:  string(raw),
			Result:  raw,
		}
	}
	return executionResultFromValue(resultValue)
}

func executionResultFromValue(resultValue any) *ExecutionResult {
	switch typedValue := resultValue.(type) {
	case string:
		return &ExecutionResult{
			Success: true,
			Output:  typedValue,
			Result:  fmt.Appendf(nil, "%q", typedValue),
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
