package checker

import (
	"fmt"
	"strings"
)

func extractJSONValue(data interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil, fmt.Errorf("key '%s' not found", part)
			}
		case []interface{}:
			var idx int
			if _, err := fmt.Sscanf(part, "[%d]", &idx); err == nil {
				if idx < 0 || idx >= len(v) {
					return nil, fmt.Errorf("index %d out of range", idx)
				}
				current = v[idx]
			} else {
				return nil, fmt.Errorf("expected array index, got '%s'", part)
			}
		default:
			return nil, fmt.Errorf("cannot navigate into %T", v)
		}
	}

	return current, nil
}
