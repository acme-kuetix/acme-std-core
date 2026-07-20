package transitions

import (
	"fmt"
	"math"

	"github.com/kuetix/engine/engine/domain"
)

// Exported wrappers around the unexported helpers (toFloat, toInt, etc.)
// so acme-* packages can import them and stop duplicating the logic.
// Each local helper in an acme-* package becomes a 1-line delegation
// to these, eliminating the duplicated implementation while preserving
// the local func signature (so callers don't need to change).

// Fail builds an error FlowStepResult with the given status, code, and message.
// Package-level func usable by acme-* packages that need a fail helper
// without instantiating utilsTransitions.
func Fail(statusCode int, code, message string) domain.FlowStepResult {
	return domain.FlowStepResult{
		Success:    false,
		StatusCode: statusCode,
		Error:      fmt.Errorf("%s", message),
		Response:   map[string]interface{}{"code": code, "message": message},
	}
}

// ToFloatVal coerces a JSON scalar (float64, float32, int, int64, string)
// to float64. Strings are parsed via strconv.ParseFloat; unparseable
// values yield 0.
func ToFloatVal(v interface{}) float64 { return toFloat(v) }

// AsStringVal coerces an interface{} to string. Non-string values yield "".
func AsStringVal(v interface{}) string { return asString(v) }

// AsBoolVal coerces an interface{} to bool. Non-bool values yield false.
func AsBoolVal(v interface{}) bool { return asBool(v) }

// DocsFromListVal extracts the docs slice from a persistence/store.List
// response (or any map with a "docs" field). Handles both
// []map[string]interface{} (native memory_store type) and []interface{}
// (JSON round-trip shape). Returns nil if the response shape is unexpected.
func DocsFromListVal(resp interface{}) []map[string]interface{} {
	if resp == nil {
		return nil
	}
	// Allow callers to pass a FlowStepResult directly.
	if fsr, ok := resp.(domain.FlowStepResult); ok {
		if !fsr.Success {
			return nil
		}
		resp = fsr.Response
	}
	m, ok := resp.(map[string]interface{})
	if !ok {
		return nil
	}
	docsVal, ok := m["docs"]
	if !ok {
		return nil
	}
	if docs, ok := docsVal.([]map[string]interface{}); ok {
		return docs
	}
	if arr, ok := docsVal.([]interface{}); ok {
		out := make([]map[string]interface{}, 0, len(arr))
		for _, d := range arr {
			if doc, ok := d.(map[string]interface{}); ok {
				out = append(out, doc)
			}
		}
		return out
	}
	return nil
}

// FailErr builds an error FlowStepResult with the given status, code, and
// message, attaching the underlying error for diagnostics.
func FailErr(statusCode int, code, message string, underlying error) domain.FlowStepResult {
	msg := message
	if underlying != nil {
		msg = fmt.Sprintf("%s: %v", message, underlying)
	}
	return domain.FlowStepResult{
		Success:    false,
		StatusCode: statusCode,
		Error:      fmt.Errorf("%s", msg),
		Response:   map[string]interface{}{"code": code, "message": message},
	}
}

// RoundMoney rounds v to 2 decimal places (half away from zero).
// Package-level func usable by acme-* packages that need a roundMoney
// helper without instantiating utilsTransitions.
func RoundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
