package provider

import (
	"strings"
)

// ParseMap takes a string implied to contain multiple items
// separated by commas and returns a map of strings with string keys
// where each key in the map is the key of a tag and the value is the
// tag's corresponding value.
func ParseMap(tags string) map[string]string {
	list := strings.Split(tags, ListSep)
	if len(list) == 0 {
		return nil
	}
	tagMap := make(map[string]string)
	for _, tag := range list {
		keyVal := strings.Split(tag, KeyValSep)
		if len(keyVal) == 2 && keyVal[0] != "" {
			tagMap[keyVal[0]] = keyVal[1]
		}
	}
	return tagMap
}

// ParseList takes a string implied to contain multiple items
// separated by commas and returns a slice where each space in the
// slice contains one of the items.
func ParseList(securityGroups string) []*string {
	list := strings.Split(securityGroups, ListSep)
	if len(list) == 0 {
		return nil
	}
	out := make([]*string, len(list))
	for i := range list {
		out[i] = &list[i]
	}
	return out
}
