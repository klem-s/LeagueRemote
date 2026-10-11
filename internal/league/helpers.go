package league

// The LCU API is unofficial and loosely specified, so responses are decoded
// to interface{} and read defensively with these helpers rather than fixed
// structs. Typed DTOs (see models.go) are built from this raw data at the
// boundary, before anything leaves the service.

func asMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func asList(v interface{}) []interface{} {
	if l, ok := v.([]interface{}); ok {
		return l
	}
	return nil
}

func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asBool(v interface{}) bool {
	b, _ := v.(bool)
	return b
}

func asInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

func toIntSlice(v interface{}) []int {
	out := []int{}
	for _, x := range asList(v) {
		out = append(out, asInt(x))
	}
	return out
}

func intSliceOrFallback(v interface{}, fallback []int) []int {
	s := toIntSlice(v)
	if len(s) > 0 {
		return s
	}
	return fallback
}

func filterPositive(ids []int) []int {
	out := []int{}
	for _, id := range ids {
		if id > 0 {
			out = append(out, id)
		}
	}
	return out
}
