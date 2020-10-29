package common

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Map ...
type Map map[string]interface{}

// Value ...
func (m Map) Value(key string) (v interface{}, ok bool) {
	v, ok = m[key]
	return
}

// Int ...
func (m Map) Int(key string) int {
	if oldv, ok := m[key]; ok {
		if v, ok := oldv.(int); ok {
			return v
		} else if v, ok := oldv.(float64); ok {
			return int(v)
		} else {
			return 0
		}
	} else {
		return 0
	}
}

// Int64 ...
func (m Map) Int64(key string) int64 {
	if oldv, ok := m[key]; ok {
		if v, ok := oldv.(int64); ok {
			return v
		} else if v, ok := oldv.(float64); ok {
			return int64(v)
		} else {
			return 0
		}
	} else {
		return 0
	}
}

// GetEnvvar ...
func (m Map) GetEnvvar(varName string) (string, bool) {
	if v, ok := m[varName]; ok {
		return fmt.Sprintf("%v", v), ok
	}
	return "", false
}

// Map ...
func (m Map) Map(key string) Map {
	v, ok := m[key].(map[string]interface{})
	if ok {
		return Map(v)
	}
	tmp, ok := IntfToMap(m[key])
	if !ok {
		return nil
	}
	return tmp
}

// Str ...
func (m Map) Str(key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", v)

}

// Float ...
func (m Map) Float(key string) float64 {
	if oldv, ok := m[key]; ok {
		if v, ok := oldv.(float64); ok {
			return v
		} else if v, ok := oldv.(int); ok {
			return float64(v)
		} else {
			return 0.0
		}
	} else {
		return 0.0
	}
}

// IncFloatValue ...
func (m Map) IncFloatValue(key string, value float64) {
	m[key] = m.Float(key) + value
}

// IncIntValue ...
func (m Map) IncIntValue(key string, value int) {
	m[key] = m.Int(key) + value
}

// IntfToMap ...
func IntfToMap(v interface{}) (Map, bool) {
	mi, ok := v.(Map)
	if ok {
		return mi, true
	}
	m, ok := v.(map[string]interface{})
	if ok {
		return Map(m), true
	}
	m2, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, false
	}
	m = make(map[string]interface{})
	for k, v := range m2 {
		m[fmt.Sprintf("%v", k)] = v
	}
	return Map(m), true
}

// MapSlice ...
func MapSlice(v interface{}) []Map {
	mlist, ok := v.([]Map)
	if ok {
		return mlist
	}
	sl, ok := v.([]map[string]interface{})
	if !ok {
		ilist := v.([]interface{})
		for _, v := range ilist {
			m, ok := IntfToMap(v)
			if ok {
				sl = append(sl, m)
			}
		}
	}
	r := []Map{}
	for _, v := range sl {
		r = append(r, Map(v))
	}
	return r
}

// MapSlice ...
func (m Map) MapSlice(key string) []Map {
	v, ok := m[key]
	if !ok {
		return nil
	}
	return MapSlice(v)
}

// FloatSlice ...
func (m Map) FloatSlice(key string) (flist []float64) {
	vlist := m[key].([]interface{})
	for _, v := range vlist {
		flist = append(flist, v.(float64))
	}
	return
}

// StringSlice ...
func StringSlice(v interface{}) []string {
	slist, ok := v.([]string)
	if ok {
		return slist
	}
	vlist, ok := v.([]interface{})
	if !ok {
		return nil
	}
	for _, v := range vlist {
		slist = append(slist, v.(string))
	}
	return slist
}

// StringSlice ...
func (m Map) StringSlice(key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	return StringSlice(v)
}

const (
	cBaseObjNameKeyName = "__base_obj_name__"
	cObjNameKeyName     = "__name__"
)

func copyValue(v interface{}, f getBaseObjFunc) interface{} {
	if mv, ok := v.(map[string]interface{}); ok {
		return inheritObj(mv, map[string]interface{}{}, f)
	} else if av, ok := v.([]interface{}); ok {
		nv := make([]interface{}, len(av))
		for _, it := range av {
			nv = append(nv, copyValue(it, f))
		}
		return nv
	} else {
		return v
	}
}

type getBaseObjFunc func(key string) map[string]interface{}

func newGetBaseObjFunc(m map[string]interface{}, pf getBaseObjFunc) getBaseObjFunc {
	return func(key string) map[string]interface{} {
		if v, ok := m[key]; ok {
			return v.(map[string]interface{})
		} else if pf != nil {
			return pf(key)
		} else {
			return nil
		}
	}
}

func inheritObj(baseObj, m map[string]interface{}, f getBaseObjFunc) map[string]interface{} {
	if v, ok := baseObj[cBaseObjNameKeyName]; ok {
		if v.(string) != "" {
			inheritObj(f(v.(string)), baseObj, f)
		}
	}
	for k, v := range baseObj {
		if _, ok := m[k]; !ok {
			m[k] = copyValue(v, f)
		}
	}
	delete(m, cBaseObjNameKeyName)
	return m
}

func isPrivateKey(k string) bool {
	return k == cObjNameKeyName
}

func processMapTree(m map[string]interface{}, f getBaseObjFunc) {
	nextLevels := []map[string]interface{}{}
	for k, v := range m {
		if isPrivateKey(k) {
			continue
		}
		if _, ok := v.(map[string]interface{}); ok {
			o := v.(map[string]interface{})
			o[cObjNameKeyName] = k
			nextLevels = append(nextLevels, o)
		}
	}

	getBaseObj := newGetBaseObjFunc(m, f)
	for k, v := range m {
		if k == cBaseObjNameKeyName {
			baseObj := getBaseObj(v.(string))
			if baseObj == nil {
				panic(fmt.Errorf("%v obj can not found", v.(string)))
			}
			inheritObj(baseObj, m, f)
		}
	}

	for _, no := range nextLevels {
		processMapTree(no, getBaseObj)
	}
}

// ReadMap ...
func ReadMap(reader io.Reader, parse bool) (Map, error) {
	var m map[string]interface{}
	err := json.NewDecoder(reader).Decode(&m)
	if err != nil {
		return nil, err
	}
	if parse {
		processMapTree(m, nil)
	}
	return Map(m), nil
}

// ReadFileToMap ...
func ReadFileToMap(fpath string, parse bool) (Map, error) {
	f, err := os.Open(fpath)
	CheckError(err)
	defer f.Close()
	return ReadMap(f, parse)
}

// WriteFileToMap ...
func WriteFileToMap(fpath string, m Map) {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0)
	CheckError(err)
	defer f.Close()
	je := json.NewEncoder(f)
	je.SetIndent("", "\t")
	CheckError(je.Encode(m))
}
