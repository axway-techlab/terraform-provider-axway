package axwayapi

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSecurityProfile() *schema.Resource {
	return TFSecurityProfile
}

var TFSecurityProfile = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name":       required(_string()),
		"is_default": required(_bool()),
		"devices":    inOut(_listMin(1, TFDevice)),
	},
	ReadContext: noop,
}

func noop(context.Context, *schema.ResourceData, interface{}) (diags diag.Diagnostics) {
	return diags
}

var TFDevice = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name":  required(inOut(_string())),
		"type":  required(inOut(_string())),
		"order": required(inOut(_int())),
		/**/ "properties": required(_map(schema.TypeString)),
	},
}