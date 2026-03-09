package subzero

import (
	"regexp"
	"strings"
)

var refPattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ResolveRefs replaces {{ref}}, {{ref.field}}, and {{param.name}} in text.
func ResolveRefs(text string, outputs map[string]*CellResult, params map[string]string) string {
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

		// ref.field
		if idx := strings.IndexByte(ref, '.'); idx > 0 {
			cellName := ref[:idx]
			field := ref[idx+1:]
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
