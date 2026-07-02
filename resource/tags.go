package resource

import (
	"fmt"
	"sort"
	"strings"
)

type Tags map[string]string

func (tags *Tags) String() string {
	if len(*tags) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(*tags))
	for key, value := range *tags {
		// handle colons, like aws:cloudformation:stack-name key
		safeKey := strings.ReplaceAll(key, ":", "_")
		safeValue := strings.ReplaceAll(value, ":", "_")

		pairs = append(pairs, fmt.Sprintf("%s-%s", safeKey, safeValue))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "_")
}

func (tags *Tags) Set(value string) error {
	for _, pair := range strings.Split(value, ",") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key=value pair %q", pair)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" || val == "" {
			return fmt.Errorf("invalid key=value pair %q", pair)
		}
		if *tags == nil {
			*tags = make(Tags)
		}
		(*tags)[key] = val
	}
	return nil
}

func MatchesAllTags(tags map[string]string, want map[string]string) bool {
	if len(want) == 0 {
		return true
	}
	for key, value := range want {
		val, exists := tags[key]
		if !exists || val != value {
			return false
		}
	}
	return true
}
