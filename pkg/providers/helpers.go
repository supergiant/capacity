package providers

import (
	"strings"
)

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
