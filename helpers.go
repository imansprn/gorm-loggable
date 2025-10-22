package loggable

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gorm.io/gorm"
)

func toJSONMap(stmt *gorm.Statement) JSONB {
	result := JSONB{}
	if stmt == nil || stmt.Schema == nil || !stmt.ReflectValue.IsValid() {
		return result
	}
	for _, f := range stmt.Schema.Fields {
		if !isExportedField(f.Name) {
			continue
		}
		if f.Name == "LoggableModel" { // do not include embedded marker
			continue
		}
		v, _ := f.ValueOf(stmt.Context, stmt.ReflectValue)
		result[f.Name] = v
	}
	return result
}

func isExportedField(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)[0]
	return r >= 'A' && r <= 'Z'
}

func callMetaIfPresent(stmt *gorm.Statement) any {
	if stmt == nil || !stmt.ReflectValue.IsValid() {
		return nil
	}
	method := stmt.ReflectValue.MethodByName("Meta")
	if !method.IsValid() || method.Type().NumIn() != 0 || method.Type().NumOut() != 1 {
		return nil
	}
	out := method.Call(nil)
	if len(out) != 1 {
		return nil
	}
	return out[0].Interface()
}

func structToJSONB(v any) (JSONB, bool) {
	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, false
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, false
	}
	res := JSONB{}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" { // unexported
			continue
		}
		res[field.Name] = val.Field(i).Interface()
	}
	return res, true
}

// computeDiffFromTags reads struct fields tagged with `gorm-loggable:true` and returns their values
func computeDiffFromTags(stmt *gorm.Statement) JSONB {
	res := JSONB{}
	if stmt == nil || stmt.Schema == nil || !stmt.ReflectValue.IsValid() {
		return res
	}
	for _, f := range stmt.Schema.Fields {
		if f.Tag.Get("gorm-loggable") != "true" {
			continue
		}
		// For now, always include tagged fields in diff (we'll optimize later)
		v, _ := f.ValueOf(stmt.Context, stmt.ReflectValue)
		res[f.Name] = v
	}
	if len(res) == 0 {
		return nil
	}
	return res
}

// significantChanges returns names of fields that changed excluding ignored ones
func significantChanges(stmt *gorm.Statement, ignore []string) []string {
	if stmt == nil || stmt.Schema == nil {
		return nil
	}
	ignoreSet := map[string]struct{}{}
	for _, n := range ignore {
		ignoreSet[n] = struct{}{}
	}
	var changed []string
	for _, f := range stmt.Schema.Fields {
		if stmt.Changed(f.Name) || stmt.Changed(f.DBName) {
			name := f.DBName
			if _, skip := ignoreSet[name]; skip {
				continue
			}
			changed = append(changed, name)
		}
	}
	sort.Strings(changed)
	return changed
}

// primaryKeyString returns a printable primary key from statement
func primaryKeyString(stmt *gorm.Statement) string {
	if stmt == nil || stmt.Schema == nil {
		return ""
	}

	var parts []string

	// First try to get primary key from ReflectValue
	if stmt.ReflectValue.IsValid() {
		for _, pf := range stmt.Schema.PrimaryFields {
			v, ok := pf.ValueOf(stmt.Context, stmt.ReflectValue)
			if !ok || v == nil {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s=%v", pf.Name, v))
		}
	}

	// If no primary key found and we have a where clause, try to extract from there
	if len(parts) == 0 && stmt.Schema.PrioritizedPrimaryField != nil && stmt.Dest != nil {
		if model, ok := stmt.Dest.(map[string]interface{}); ok {
			if id, exists := model[stmt.Schema.PrioritizedPrimaryField.DBName]; exists {
				parts = append(parts, fmt.Sprintf("%s=%v", stmt.Schema.PrioritizedPrimaryField.Name, id))
			}
		}
	}

	return strings.Join(parts, ",")
}
