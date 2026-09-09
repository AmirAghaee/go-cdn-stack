package cache

import (
	"math"
	"strconv"
	"strings"
	"time"
)

func parseAge(header map[string][]string) (time.Duration, bool) {
	values := headerValues(header, "Age")
	if len(values) == 0 {
		return 0, true
	}
	if len(values) != 1 {
		return 0, false
	}
	seconds, err := strconv.ParseUint(strings.TrimSpace(values[0]), 10, 63)
	if err != nil || seconds > uint64(math.MaxInt64/int64(time.Second)) {
		return 0, false
	}
	return time.Duration(seconds) * time.Second, true
}

func parseCacheControl(header map[string][]string) (map[string][]string, bool) {
	directives := make(map[string][]string)
	for _, value := range headerValues(header, "Cache-Control") {
		if !hasBalancedQuotes(value) {
			return nil, false
		}
		for _, part := range splitHeaderList(value) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			name, directiveValue, hasValue := strings.Cut(part, "=")
			name = strings.ToLower(strings.TrimSpace(name))
			if !isToken(name) {
				return nil, false
			}
			if hasValue {
				directiveValue = strings.TrimSpace(directiveValue)
				if strings.HasPrefix(directiveValue, `"`) {
					unquoted, err := strconv.Unquote(directiveValue)
					if err != nil {
						return nil, false
					}
					directiveValue = unquoted
				} else if !isToken(directiveValue) {
					return nil, false
				}
			} else {
				directiveValue = ""
			}
			directives[name] = append(directives[name], directiveValue)
		}
	}
	return directives, true
}

func hasBalancedQuotes(value string) bool {
	inQuotes := false
	escaped := false
	for _, character := range value {
		switch {
		case escaped:
			escaped = false
		case inQuotes && character == '\\':
			escaped = true
		case character == '"':
			inQuotes = !inQuotes
		}
	}
	return !inQuotes && !escaped
}

func isToken(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if !(character >= 'a' && character <= 'z') &&
			!(character >= 'A' && character <= 'Z') &&
			!(character >= '0' && character <= '9') &&
			!strings.ContainsRune("!#$%&'*+-.^_`|~", character) {
			return false
		}
	}
	return true
}

func splitHeaderList(value string) []string {
	var parts []string
	start := 0
	inQuotes := false
	escaped := false
	for index, character := range value {
		switch {
		case escaped:
			escaped = false
		case inQuotes && character == '\\':
			escaped = true
		case character == '"':
			inQuotes = !inQuotes
		case character == ',' && !inQuotes:
			parts = append(parts, value[start:index])
			start = index + 1
		}
	}
	return append(parts, value[start:])
}

func hasHeader(header map[string][]string, name string) bool {
	for existingName := range header {
		if strings.EqualFold(existingName, name) {
			return true
		}
	}
	return false
}

func headerValues(header map[string][]string, name string) []string {
	var values []string
	for existingName, existingValues := range header {
		if strings.EqualFold(existingName, name) {
			values = append(values, existingValues...)
		}
	}
	return values
}

func firstHeader(header map[string][]string, name string) string {
	values := headerValues(header, name)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func cloneHeader(header map[string][]string) map[string][]string {
	cloned := make(map[string][]string, len(header))
	for key, values := range header {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}
