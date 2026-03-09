package subzero

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

var refPattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ResolveRefs replaces {{ref}}, {{ref.field}}, {{ref.*}}, and {{param.name}} in text.
// For {{ref.*}}: returns JSON object of all fields from ref's output.
// For self-ref gathering, use ResolveRefsWithContext.
func ResolveRefs(text string, outputs map[string]*CellResult, params map[string]string) string {
	return resolveRefs(text, outputs, params, "", nil)
}

// ResolveRefsWithContext resolves refs with awareness of the current cell.
// When {{currentCell.*}} is encountered, it gathers all dependency outputs into a JSON array.
func ResolveRefsWithContext(text string, outputs map[string]*CellResult, params map[string]string, currentCell string, depNames []string) string {
	return resolveRefs(text, outputs, params, currentCell, depNames)
}

func resolveRefs(text string, outputs map[string]*CellResult, params map[string]string, currentCell string, depNames []string) string {
	return refPattern.ReplaceAllStringFunc(text, func(match string) string {
		ref := strings.TrimSpace(match[2 : len(match)-2])

		// param.X
		if strings.HasPrefix(ref, "param.") {
			key := ref[len("param."):]
			if v, ok := params[key]; ok {
				return v
			}
			return match
		}

		// ref.field or ref.*
		if idx := strings.IndexByte(ref, '.'); idx > 0 {
			cellName := ref[:idx]
			field := ref[idx+1:]

			// Wildcard: {{cellname.*}}
			if field == "*" {
				// Self-ref gather: {{currentCell.*}} → JSON array of all dep outputs
				if currentCell != "" && cellName == currentCell && depNames != nil {
					return gatherDepOutputs(outputs, depNames)
				}
				// Normal wildcard: {{other.*}} → JSON object of all fields
				if r, ok := outputs[cellName]; ok && r.Fields != nil {
					return fieldsToJSON(r.Fields)
				}
				// Fallback: return raw output
				if r, ok := outputs[cellName]; ok {
					return r.Output
				}
				return match
			}

			if r, ok := outputs[cellName]; ok && r.Fields != nil {
				if v, ok := r.Fields[field]; ok {
					return v
				}
			}
			return match
		}

		// plain ref
		if r, ok := outputs[ref]; ok {
			return r.Output
		}
		return match
	})
}

// gatherDepOutputs collects all dependency outputs into a JSON array.
func gatherDepOutputs(outputs map[string]*CellResult, depNames []string) string {
	var items []string
	for _, name := range depNames {
		if r, ok := outputs[name]; ok {
			items = append(items, r.Output)
		}
	}
	b, _ := json.Marshal(items)
	return string(b)
}

// fieldsToJSON converts a Fields map to a JSON object string.
func fieldsToJSON(fields map[string]string) string {
	// Sort keys for deterministic output
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	obj := make(map[string]string, len(fields))
	for _, k := range keys {
		obj[k] = fields[k]
	}
	b, _ := json.Marshal(obj)
	return string(b)
}
