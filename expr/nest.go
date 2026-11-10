package expr

import (
	"strings"
)

// Nest returns a flat map with dotted keys as nested maps, the variables a
// [Typed] program reads: "k8s.pod.name" becomes k8s.pod.name. Keys and their
// dotted segments are trimmed of spaces.
//
// When a dotted key and a plain key collide, such as "foo" and "foo.bar",
// the nested map wins and the second return value is true. The same holds
// for a deeper dotted key over a shallower one. Which of two keys that are
// equal after trimming wins is unspecified.
func Nest[K ~string, V ~string](flat map[K]V) (map[string]any, bool) {
	env := map[string]any{}
	conflict := false

	for lk, lv := range flat {
		key := strings.TrimSpace(string(lk))
		value := string(lv)

		if strings.Contains(key, ".") {
			parts := strings.Split(key, ".")
			current := env

			for i, raw := range parts {
				part := strings.TrimSpace(raw)
				if i == len(parts)-1 {
					// A deeper path processed first has made a map here: keep it.
					if _, isMap := current[part].(map[string]any); isMap {
						conflict = true

						break
					}

					current[part] = value

					break
				}

				if nextMap, ok := current[part].(map[string]any); ok {
					current = nextMap

					continue
				}

				// A plain leaf sits on the path: replace it with a map so the
				// deeper path can be built.
				if _, exists := current[part]; exists {
					conflict = true
				}

				newMap := map[string]any{}
				current[part] = newMap
				current = newMap
			}

			continue
		}

		// A nested map already sits at this plain key: keep the map.
		if _, isMap := env[key].(map[string]any); isMap {
			conflict = true

			continue
		}

		env[key] = value
	}

	return env, conflict
}
