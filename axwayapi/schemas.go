package axwayapi

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type schemaMap map[string]*schema.Schema

func resource(s schemaMap) *schema.Resource {
	return &schema.Resource{Schema: s}
}

//--
func inOut(schema *schema.Schema) *schema.Schema {
	schema.Required = false
	schema.Optional = true
	schema.Computed = true
	return schema
}
func required(schema *schema.Schema) *schema.Schema {
	schema.Required = true
	schema.Optional = false
	schema.Computed = false
	return schema
}
func exactlyOneOfResource(schema *schema.Schema, choices ...string) *schema.Schema {
	schema.Required = false
	schema.Optional = true
	schema.Computed = false
	schema.ExactlyOneOf = choices
	return schema
}
func atLeastOneOfResource(schema *schema.Schema, choices ...string) *schema.Schema {
	schema.Required = false
	schema.Optional = true
	schema.Computed = false
	schema.AtLeastOneOf = choices
	return schema
}
func optional(schema *schema.Schema, defValue ...interface{}) *schema.Schema {
	schema.Required = false
	schema.Optional = true
	schema.Computed = false
	if len(defValue) > 0 {
		schema.Default = defValue[0]
	}
	return schema
}
func readonly(schema *schema.Schema) *schema.Schema {
	schema.Required = false
	schema.Optional = false
	schema.Computed = true
	return schema
}
func desc(schema *schema.Schema, description string) *schema.Schema {
	schema.Description = description
	return schema
}

//--
func _computed(schema *schema.Schema) *schema.Schema {
	schema.Computed = true
	return schema
}
func _sensitive(schema *schema.Schema) *schema.Schema {
	schema.Sensitive = true
	return schema
}
func _optional(schema *schema.Schema) *schema.Schema {
	schema.Optional = true
	return schema
}
func _FORCENEW(schema *schema.Schema) *schema.Schema {
	schema.ForceNew = true
	return schema
}

func _asBlock(s *schema.Schema) *schema.Schema {
	s.ConfigMode = schema.SchemaConfigModeBlock
	return s
}

//--
func _int() *schema.Schema {
	return &schema.Schema{Type: schema.TypeInt}
}
func _string(f ...func(interface{}, cty.Path) diag.Diagnostics) *schema.Schema {
	s := &schema.Schema{Type: schema.TypeString}
	if len(f) == 0 {
		return s
	} else if len(f) == 1 {
		s.ValidateDiagFunc = f[0]
	} else {
		s.ValidateDiagFunc = all(f...)
	}
	return s
}
func oneOf(allowed ...string) func(interface{}, cty.Path) diag.Diagnostics {
	return func(value interface{}, path cty.Path) diag.Diagnostics {
		for _, a := range allowed {
			if value.(string) == a {
				return nil
			}
		}
		return diag.Errorf("Value '%s' is not in allowed list (%q) (%#+q)", value.(string), allowed, path)
	}
}
func r(pattern string) func(interface{}, cty.Path) diag.Diagnostics {
	return func(value interface{}, path cty.Path) diag.Diagnostics {
		b, err := regexp.MatchString(pattern, value.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		if !b {
			return diag.Errorf("Value '%s' does not match pattern '%s' (%#+v)", value.(string), pattern, path)
		}
		return nil
	}
}
func any(f ...func(interface{}, cty.Path) diag.Diagnostics) func(interface{}, cty.Path) diag.Diagnostics {
	return func(value interface{}, path cty.Path) diag.Diagnostics {
		for _, ff := range f {
			d := ff(value, path)
			if d.HasError() {
				return d
			}
		}
		return diag.Errorf("No condition was met for value '%s' (%#+v)", value, path)
	}
}
func all(f ...func(interface{}, cty.Path) diag.Diagnostics) func(interface{}, cty.Path) diag.Diagnostics {
	if len(f) == 1 {
		return f[0]
	}
	return func(value interface{}, path cty.Path) diag.Diagnostics {
		for _, ff := range f {
			d := ff(value, path)
			if d.HasError() {
				return d
			}
		}
		return nil
	}
}
func not(f func(interface{}, cty.Path) diag.Diagnostics) func(interface{}, cty.Path) diag.Diagnostics {
	return func(value interface{}, path cty.Path) diag.Diagnostics {
		d := f(value, path)
		if !d.HasError() {
			return diag.Errorf("A condition is not met for value '%v' (%#+v)", value, path)
		}
		return nil
	}
}
func _hashedString() *schema.Schema {
	return _apply(_hash, _string())
}
func _bool() *schema.Schema {
	return &schema.Schema{Type: schema.TypeBool}
}
func _map(mapValuesType schema.ValueType) *schema.Schema {
	return &schema.Schema{Type: schema.TypeMap, Elem: &schema.Schema{Type: mapValuesType}}
}
func _apply(ser func(interface{}) string, s *schema.Schema) *schema.Schema {
	s.StateFunc = ser
	return s
}

//--
func _pset(setValuesType schema.ValueType) *schema.Schema {
	return &schema.Schema{Type: schema.TypeSet, Elem: &schema.Schema{Type: setValuesType}}
}
func _set(s *schema.Resource) *schema.Schema {
	return &schema.Schema{Type: schema.TypeSet, Elem: s}
}
func _setMax(max int, s *schema.Resource) *schema.Schema {
	return &schema.Schema{Type: schema.TypeSet, MaxItems: max, Elem: s}
}
func _setMin(min int, s *schema.Resource) *schema.Schema {
	return &schema.Schema{Type: schema.TypeSet, MinItems: min, Elem: s}
}
func _setBounded(min, max int, s *schema.Resource) *schema.Schema {
	return &schema.Schema{Type: schema.TypeSet, MinItems: min, MaxItems: max, Elem: s}
}

//--
func _list(s *schema.Resource) *schema.Schema {
	return &schema.Schema{Type: schema.TypeList, Elem: s}
}
func _listMax(max int, s *schema.Resource) *schema.Schema {
	return &schema.Schema{Type: schema.TypeList, MaxItems: max, Elem: s}
}
func _listMin(min int, s *schema.Resource) *schema.Schema {
	return &schema.Schema{Type: schema.TypeList, MinItems: min, Elem: s}
}
func _listBounded(min, max int, s *schema.Resource) *schema.Schema {
	return &schema.Schema{Type: schema.TypeList, MinItems: min, MaxItems: max, Elem: s}
}
func _listExact(nb int, s *schema.Resource) *schema.Schema {
	return &schema.Schema{Type: schema.TypeList, MinItems: nb, MaxItems: nb, Elem: s}
}
func _singleton(s *schema.Resource) *schema.Schema {
	return _listExact(1, s)
}

//-- avoiding generics for the nonce
func _plist(innerType schema.ValueType) *schema.Schema {
	return &schema.Schema{Type: schema.TypeList, Elem: &schema.Schema{Type: innerType}}
}
func _plistMax(max int, innerType schema.ValueType) *schema.Schema {
	return &schema.Schema{Type: schema.TypeList, MaxItems: max, Elem: &schema.Schema{Type: innerType}}
}
func _plistMin(min int, innerType schema.ValueType) *schema.Schema {
	return &schema.Schema{Type: schema.TypeList, MinItems: min, Elem: &schema.Schema{Type: innerType}}
}
func _plistBounded(min, max int, innerType schema.ValueType) *schema.Schema {
	return &schema.Schema{Type: schema.TypeList, MinItems: min, MaxItems: max, Elem: &schema.Schema{Type: innerType}}
}
func _plistExact(nb int, innerType schema.ValueType) *schema.Schema {
	return &schema.Schema{Type: schema.TypeList, MinItems: nb, MaxItems: nb, Elem: &schema.Schema{Type: innerType}}
}

//---
func _namedMap(s *schema.Resource) *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: schemaMap{
				"name":  required(_string()),
				"value": required(_singleton(s)),
			},
		},
	}
}
func serMap(m map[string]interface{}) string {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(b)
}

//---
func deserMap(s string) (m map[string]interface{}) {
	err := json.Unmarshal([]byte(s), &m)
	if err != nil {
		panic(fmt.Sprintf("%#+v: %#+v", m, err))
	}
	return m
}

func _hash(v interface{}) string {
	if v != nil {
		s := v.(string)
		return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
	}
	return ""
}
