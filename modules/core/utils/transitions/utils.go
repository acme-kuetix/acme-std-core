package transitions

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kuetix/engine/engine/domain"
	"github.com/kuetix/engine/engine/domain/interfaces"
	"github.com/kuetix/engine/engine/workflow"
)

// PROMOTION-CANDIDATE: stable since Wave 3, no acme-* deps, used in 16 packages.
// Provides FlowStepResult helpers (Fail/FailErr), type coercion (ToFloatVal/
// ToIntVal/AsStringVal/AsBoolVal/ToStringSliceVal), DocsFromListVal, RoundMoney.
// Extends std-core's common transitions.
// Consider promoting to std-core after kuetix review.

// utilsTransitions exposes generic helper methods (type coercion,
// parsing, comparison) as WSL-callable transitions so acme-* domain
// packages can share one implementation instead of duplicating fail,
// toFloat, toInt, asString, asBool, etc. in every package.
type utilsTransitions struct {
	workflow.BaseServiceTransition
}

// NewUtilsTransitions returns the WSL-callable core/utils transition.
func NewUtilsTransitions() interfaces.ServiceTransitions {
	return &utilsTransitions{}
}

// ToInt coerces a JSON scalar (float64, int, int64, string) to int.
// Returns {value: int}. Strings are parsed via strconv.Atoi; nil yields 0.
// WSL: core/utils.ToInt(v: $json.count)
func (t *utilsTransitions) ToInt(v interface{}) (r domain.FlowStepResult) {
	r.Success = true
	r.StatusCode = 200
	r.Response = map[string]interface{}{"value": toInt(v)}
	return
}

// ParseTaxTemplateIds parses a comma-separated string, []interface{},
// or []string into a clean []string of template IDs. Returns {ids: []string}.
// WSL: core/utils.ParseTaxTemplateIds(v: $json.taxTemplateIds)
func (t *utilsTransitions) ParseTaxTemplateIds(v interface{}) (r domain.FlowStepResult) {
	r.Success = true
	r.StatusCode = 200
	ids := parseTaxTemplateIds(v)
	out := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		out = append(out, id)
	}
	r.Response = map[string]interface{}{"ids": out}
	return
}

// SetValue stores a value in the workflow session context under the given key.
// The value is accessible by downstream states and sub-workflows via $<key>.
// Returns {value: <the value>} so callers can chain if needed.
// WSL: core/utils.SetValue(key: "subtotal", value: $SubtotalResult.result)
func (t *utilsTransitions) SetValue(key string, value interface{}) (r domain.FlowStepResult) {
	t.BaseServiceTransition.SetValue(key, value)
	r.Success = true
	r.StatusCode = 200
	r.Response = map[string]interface{}{"value": value}
	return
}

// GetValue retrieves a value from the workflow session context by key.
// Returns {value: <the value>, found: bool}. If the key is not set,
// value is nil and found is false.
// WSL: core/utils.GetValue(key: "subtotal")
func (t *utilsTransitions) GetValue(key string) (r domain.FlowStepResult) {
	v := t.BaseServiceTransition.GetValue(key)
	r.Success = true
	r.StatusCode = 200
	r.Response = map[string]interface{}{"value": v, "found": v != nil}
	return
}

// ─── internal helpers (ported from acme-* packages) ──────────────

// toFloat coerces interface{} (float64 from JSON, int, string, etc.) to float64.
func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		if parsed, err := strconv.ParseFloat(n, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// toInt coerces interface{} (float64 from JSON, int, string, etc.) to int.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	case nil:
		return 0
	default:
		return 0
	}
}

// asString coerces an interface{} to string. Non-string values yield "".
// Numbers, bools, and other types are stringified via fmt.Sprintf("%v").
func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}
	return fmt.Sprintf("%v", v)
}

// asBool coerces an interface{} to bool. Non-bool values yield false.
func asBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// parseTaxTemplateIds accepts comma-separated string OR []interface{} OR
// []string and returns a clean []string of template IDs.
func parseTaxTemplateIds(v interface{}) []string {
	var tplIds []string
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s != "" {
			for _, raw := range strings.Split(s, ",") {
				id := strings.TrimSpace(raw)
				if id != "" {
					tplIds = append(tplIds, id)
				}
			}
		}
	case []interface{}:
		for _, raw := range t {
			id := strings.TrimSpace(fmt.Sprintf("%v", raw))
			if id != "" {
				tplIds = append(tplIds, id)
			}
		}
	case []string:
		for _, raw := range t {
			id := strings.TrimSpace(raw)
			if id != "" {
				tplIds = append(tplIds, id)
			}
		}
	}
	return tplIds
}
