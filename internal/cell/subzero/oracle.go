package subzero

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/steveyegge/gastown/internal/cell/parser"
)

// EvalOracle evaluates an oracle block against a cell's output.
// Returns nil if all checks pass, error describing the failure otherwise.
func EvalOracle(oracle *parser.OracleBlock, output string) error {
	if oracle == nil {
		return nil
	}

	// Track parsed JSON for use across statements
	var parsed interface{}

	for _, stmt := range oracle.Statements {
		switch stmt.Kind {
		case "json_parse":
			if err := json.Unmarshal([]byte(output), &parsed); err != nil {
				return fmt.Errorf("json_parse failed: %w", err)
			}

		case "keys_present":
			obj, ok := parsed.(map[string]interface{})
			if !ok {
				return fmt.Errorf("keys_present: output is not a JSON object")
			}
			// Args format: [target, '["key1", "key2"]']
			// Skip first arg (target name like "output"), parse keys from second arg
			keysArg := ""
			if len(stmt.Args) >= 2 {
				keysArg = stmt.Args[1]
			} else if len(stmt.Args) == 1 {
				keysArg = stmt.Args[0]
			}
			keys := parseKeysList(keysArg)
			for _, key := range keys {
				if _, exists := obj[key]; !exists {
					return fmt.Errorf("keys_present: missing key %q", key)
				}
			}

		case "assert":
			if err := evalAssert(stmt.Expr, output, parsed); err != nil {
				return fmt.Errorf("assert failed: %w", err)
			}

		case "reject":
			// "reject if <expr>" — if expr is true, reject the output
			if stmt.Expr != "" {
				if matches, _ := evalCondition(stmt.Expr, output, parsed); matches {
					return fmt.Errorf("rejected: %s", stmt.Expr)
				}
			}

		case "accept":
			// "accept if <expr>" — noop for now, validation is about rejection
			continue

		case "score", "score_if":
			// Scoring is advisory, not blocking — skip in Sub-Zero v0
			continue

		case "for":
			// for loops in oracles — skip for now
			continue

		case "if":
			// conditional oracle blocks — skip for now
			continue

		default:
			// Unknown oracle statement — warn but don't fail
			continue
		}
	}

	return nil
}

// parseKeysList extracts key names from a keys_present arg like `["message", "status"]`.
func parseKeysList(arg string) []string {
	arg = strings.TrimSpace(arg)
	// Try JSON array parse first
	var keys []string
	if err := json.Unmarshal([]byte(arg), &keys); err == nil {
		return keys
	}
	// Fallback: strip brackets and split by comma
	arg = strings.Trim(arg, "[]")
	parts := strings.Split(arg, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"'`)
		if p != "" {
			keys = append(keys, p)
		}
	}
	return keys
}

// evalAssert evaluates simple assert expressions.
// Supports: "X in [a, b, c]", "len(X) > N", "X contains Y", simple comparisons.
func evalAssert(expr string, output string, parsed interface{}) error {
	expr = strings.TrimSpace(expr)

	// "output.X in [Y, Z]"
	if strings.Contains(expr, " in [") {
		return evalInExpr(expr, parsed)
	}

	// "len(output.X) > 0"
	if strings.HasPrefix(expr, "len(") {
		return evalLenExpr(expr, parsed)
	}

	// "output.X contains Y"
	if strings.Contains(expr, " contains ") {
		return evalContainsExpr(expr, output, parsed)
	}

	// Fallback: can't evaluate complex expressions in Sub-Zero v0
	return nil
}

func evalInExpr(expr string, parsed interface{}) error {
	parts := strings.SplitN(expr, " in ", 2)
	if len(parts) != 2 {
		return nil
	}

	fieldPath := strings.TrimSpace(parts[0])
	val := resolveField(fieldPath, parsed)
	if val == "" {
		return fmt.Errorf("%s is empty", fieldPath)
	}

	// Parse the list: [a, b, c] or ["a", "b", "c"]
	listStr := strings.TrimSpace(parts[1])
	listStr = strings.Trim(listStr, "[]")
	items := strings.Split(listStr, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		item = strings.Trim(item, `"'`)
		if val == item {
			return nil
		}
	}

	return fmt.Errorf("%s=%q not in %s", fieldPath, val, parts[1])
}

func evalLenExpr(expr string, parsed interface{}) error {
	// len(output.X) > 0
	// Extract field path from len(...)
	start := strings.Index(expr, "(")
	end := strings.Index(expr, ")")
	if start < 0 || end < 0 {
		return nil
	}
	fieldPath := expr[start+1 : end]
	val := resolveField(fieldPath, parsed)

	// Extract comparison
	rest := strings.TrimSpace(expr[end+1:])
	if strings.HasPrefix(rest, "> 0") || strings.HasPrefix(rest, ">0") {
		if len(val) == 0 {
			return fmt.Errorf("len(%s) == 0, expected > 0", fieldPath)
		}
		return nil
	}

	return nil
}

func evalContainsExpr(expr string, output string, parsed interface{}) error {
	parts := strings.SplitN(expr, " contains ", 2)
	if len(parts) != 2 {
		return nil
	}
	haystack := resolveField(strings.TrimSpace(parts[0]), parsed)
	if haystack == "" {
		haystack = output
	}
	needle := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
	if !strings.Contains(haystack, needle) {
		return fmt.Errorf("%q does not contain %q", parts[0], needle)
	}
	return nil
}

// evalCondition evaluates a simple boolean expression for reject/accept.
func evalCondition(expr string, output string, parsed interface{}) (bool, error) {
	expr = strings.TrimSpace(expr)

	// "output.X == Y"
	if strings.Contains(expr, " == ") {
		parts := strings.SplitN(expr, " == ", 2)
		left := resolveField(strings.TrimSpace(parts[0]), parsed)
		right := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		return left == right, nil
	}

	return false, nil
}

// resolveField extracts a value from parsed JSON by dotted path.
// e.g., "output.action" → parsed["action"]
func resolveField(path string, parsed interface{}) string {
	// Strip "output." prefix if present
	path = strings.TrimPrefix(path, "output.")

	obj, ok := parsed.(map[string]interface{})
	if !ok {
		return ""
	}

	parts := strings.Split(path, ".")
	var current interface{} = obj
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current, ok = m[part]
		if !ok {
			return ""
		}
	}

	switch v := current.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
