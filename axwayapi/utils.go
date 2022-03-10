package axwayapi

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	client "github.com/axway-techlab/axwayapi_client/axwayapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type flattenMap = map[string]interface{}

func syncImage(d *schema.ResourceData, object client.WithId, c *client.Client) (diags diag.Diagnostics) {
	// image is a special case.
	if d.HasChange("image_jpg") {
		err := c.UpdateImageFor(object, d.Get("image_jpg").(string))
		if nil != err {
			diags = warn(diags, "updating image for %T %s failed: %v", object, object.GetId(), err)
		}
	}
	return diags
}

func guard(diags diag.Diagnostics, f func(*client.Frontend) error, frontend *client.Frontend) diag.Diagnostics {
	if err := f(frontend); nil != err {
		diags = append(diags, diag.FromErr(err)...)
	}
	return diags
}
func warn(diags diag.Diagnostics, warn string, params ...interface{}) diag.Diagnostics {
	return append(diags, diag.Diagnostic{Severity: diag.Warning, Summary: fmt.Sprintf(warn, params...)})
}
func toParameters(params map[string]interface{}) map[string]interface{} {
	r := make(map[string]interface{}, len(params))
	for k, v := range params {
		var a interface{}
		e := json.Unmarshal([]byte(v.(string)), &a)
		if e != nil {
			panic(e)
		}
		r[k] = a
	}
	return r
}
// -- Tags

var TFTag = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name":  required(_string()),
		"values": required(_plistMin(1, schema.TypeString)),
	},
}

func flattenTags(tags map[string][]string) *schema.Set {
	r := make([]interface{}, 0)
	for k, v := range tags {
		t := make(map[string]interface{}, 2)
		t["name"] = k
		t["values"] = v
		r = append(r, t)
	}
	return schema.NewSet(func (i interface{}) int {
		f := fnv.New32a()
		f.Write([]byte(i.(map[string]interface{})["name"].(string)))
		return int(f.Sum32())
	},r)
}
func toTags(tags *schema.Set) map[string][]string {
	r := make(map[string][]string, tags.Len())
	for _, v := range tags.List() {
		t := v.(map[string]interface{})
		name := t["name"].(string)
		values := toStringArray(t["values"])
		r[name] = values
	}
	return r
}
//--
func toStringArray(array interface{}) []string {
	// Joys of Golang...
	a := array.([]interface{})
	s := make([]string, len(a))
	for i, o := range a {
		s[i] = o.(string)
	}
	return s
}

//--
func toStringMap(array interface{}) map[string]string {
	// Joys of Golang...
	a := array.(map[string]interface{})
	s := make(map[string]string, len(a))
	for i, o := range a {
		s[i] = o.(string)
	}
	return s
}
