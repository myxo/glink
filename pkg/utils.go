package glink

import "strings"


func JoinUids(elems []Uid, sep string) string {
	if len(elems) == 0{
		return ""
	}

	var result string
	result = string(elems[0])
	for i := 1; i < len(elems); i++ {
		result += sep + string(elems[i])
	}
	return result
}

func SplitUids(s, sep string) []Uid {
	if len(sep) == 0 {
		panic("cannot have empty separator")
	}

	out := make([]Uid, 5)
	for i:= 0; i < len(s); {
		next := strings.Index(s[i:], sep)
		if next == -1 {
			out = append(out, Uid(s[i:]))
			break
		}
		out = append(out, Uid(s[i:next]))
		i = next
	}
	return out
}
