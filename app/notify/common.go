package notify

import (
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

type BaseNotify struct {
	logger *logrus.Entry
}

func (n *BaseNotify) buildMessages(pattern string, item interface{}) (result string) {
	param := map[string]interface{}{}
	getParamsFromMsg(pattern, param)

	result = pattern
	for p, _ := range param {
		if _, ok := item.(map[string]interface{}); ok {
			if value := GetValue(item.(map[string]interface{}), p); len(value) == 1 {

				valueStr := ""
				switch v := value[0].(type) {
				case float64:
					valueStr = strconv.FormatFloat(v, 'f', -1, 64)
				case string:
					valueStr = v
				}

				result = strings.Replace(result, "%"+p+"%", valueStr, -1)
			}
		}
	}

	return result
}

func getParamsFromMsg(msg string, out map[string]interface{}) {
	start := strings.Index(msg, "%") + 1
	end := strings.Index(msg[start:], "%") + start

	if start < 0 || end < 0 || start > end {
		return
	}

	out[msg[start:end]] = nil
	getParamsFromMsg(msg[end+1:], out)
}

func GetValue(json map[string]interface{}, path string) []interface{} {
	raw := getall(0, strings.Split(path, "."), json, "")
	if v, ok := raw.([]interface{}); ok {
		return v
	}
	return []interface{}{raw}
}

func getall(i int, stack []string, elem interface{}, keychain string) interface{} { // nolint: gocyclo
	if i > len(stack)-1 {
		if list, ok := elem.([]interface{}); ok {
			var mod []interface{}
			for _, e := range list {
				mod = append(mod, addkey(e, keychain))
			}
			return mod
		}
		if m, ok := elem.(map[string]interface{}); ok {
			return addkey(m, keychain)
		}
		return elem
	}

	key := stack[i]

	if m, ok := elem.(map[string]interface{}); ok {
		v, ok := m[key]
		if !ok {
			return nil
		}
		i++
		return getall(i, stack, v, keychain)
	}

	buckets, ok := elem.([]interface{})
	if !ok {
		return nil
	}

	var mod []interface{}
	for _, item := range buckets {
		kc := keychain
		if e, ok := item.(map[string]interface{}); ok {
			if k, ok := e["key"].(string); ok {
				if kc == "" {
					kc = k
				} else {
					kc = kc + " - " + k
				}
			}
		}

		a := getall(i, stack, item, kc)
		switch v := a.(type) {
		case map[string]interface{}:
			mod = append(mod, v)
		case []interface{}:
			mod = append(mod, v...)
		case nil:
		default:
			mod = append(mod, a)
		}
	}
	return mod
}

func addkey(i interface{}, keychain string) interface{} {
	obj, ok := i.(map[string]interface{})
	if !ok {
		return i
	}
	key, ok := obj["key"].(string)
	if !ok {
		return obj
	}
	if key == "" {
		return obj
	}
	if keychain != "" {
		obj["key"] = keychain + " - " + key
	}
	return obj
}
