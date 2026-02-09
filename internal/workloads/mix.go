package workloads

import (
	"fmt"
	"strconv"
	"strings"
)

type Mix map[string]int32

func ParseMix(input string) (Mix, error) {
	mix := Mix{}
	input = strings.TrimSpace(input)
	if input == "" {
		return mix, nil
	}
	parts := strings.Split(input, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid mix entry %q", part)
		}
		key := normalizeMixKey(kv[0])
		if key == "" {
			return nil, fmt.Errorf("unknown size class %q", kv[0])
		}
		val, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil || val < 0 {
			return nil, fmt.Errorf("invalid mix count for %q", kv[0])
		}
		mix[key] = int32(val)
	}
	return mix, nil
}

func normalizeMixKey(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "small", "s":
		return "small"
	case "medium", "med", "m":
		return "medium"
	case "large", "l":
		return "large"
	default:
		return ""
	}
}

func MixTotal(mix Mix) int32 {
	var total int32
	for _, v := range mix {
		total += v
	}
	return total
}
