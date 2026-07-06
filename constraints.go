package kern

import "net/http"

// IntPathConstraint validates signed base-10 integers.
func IntPathConstraint(value string) bool {
	if value == "" {
		return false
	}
	start := 0
	if value[0] == '-' {
		if len(value) == 1 {
			return false
		}
		start = 1
	}
	for i := start; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

// UintPathConstraint validates unsigned base-10 integers.
func UintPathConstraint(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

// SlugPathConstraint validates URL-safe slugs ([a-zA-Z0-9_-]+).
func SlugPathConstraint(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			continue
		}
		return false
	}
	return true
}

// UUIDPathConstraint validates canonical UUID format.
func UUIDPathConstraint(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i := 0; i < len(value); i++ {
		ch := value[i]
		switch i {
		case 8, 13, 18, 23:
			if ch != '-' {
				return false
			}
		default:
			if !isHex(ch) {
				return false
			}
		}
	}
	return true
}

func isHex(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func clonePathConstraints(constraints PathConstraints) PathConstraints {
	if len(constraints) == 0 {
		return nil
	}
	out := make(PathConstraints, len(constraints))
	for name, fn := range constraints {
		if name == "" || fn == nil {
			continue
		}
		out[name] = fn
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func validatePathConstraints(r *http.Request, constraints PathConstraints) bool {
	if len(constraints) == 0 {
		return true
	}
	for name, fn := range constraints {
		if !fn(r.PathValue(name)) {
			return false
		}
	}
	return true
}
