package axwayapi

import (
	"context"
	"time"

	client "github.com/axway-techlab/axwayapi_client/axwayapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TODO: Deal with tags.
// They are ignored for now, since they are encoded as
// anonymous fields in the application directly.
// Decoding them is hacky, and error-prone.
var TFApplicationSchema = &schemaMap{
	"id":          readonly(_string()),
	"name":        required(_string()),
	"description": optional(_string(), ""),
	"org_id":      _FORCENEW(inOut(_string())),
	"phone":       optional(_string()),
	"email":       optional(_string()),
	"enabled":     optional(_bool(), true),
	"image_jpg":   optional(_hashedString()),
	"state":       readonly(_string()),
	//	"tags":        inOut(_map(schema.TypeString)),
	"managed_by": readonly(_plist(schema.TypeString)),
	"created_by": readonly(_string()),
	"created_on": readonly(_int()),
	"apis":       desc(optional(_plist(schema.TypeString)), "A list of APIs that this application can reach"),
	"apikey":     desc(optional(_list(TFApiKey)), "The API keys this application holds"),
	"quota": desc(optional(_singleton(&schema.Resource{
		Schema: TFQuotaSchema,
	})), "Overrides the default quota for applications, if any"),
}
var TFApiKey = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"id":           desc(required(_string()), "the actual api key here"),
		"secret":       desc(inOut(_string()), "If no value is given, the gateway will generate one for you"),
		"enabled":      desc(optional(_bool(), true), "defaults to 'true'"),
		"cors_origins": required(_plist(schema.TypeString)),
		"created_by":   readonly(_string()),
		"created_on":   readonly(_int()),
		"deleted_on":   readonly(_int()),
	},
}

func resourceApplication() *schema.Resource {
	return &schema.Resource{
		Schema:        *TFApplicationSchema,
		CreateContext: resourceApplicationCreate,
		ReadContext:   resourceApplicationRead,
		UpdateContext: resourceApplicationUpdate,
		DeleteContext: resourceApplicationDelete,
	}
}

func resourceApplicationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	application := &client.Application{}
	expandApplication(d, application)

	err = c.CreateApplication(application)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	diags = append(diags, syncImage(d, application, c)...)
	diags = append(diags, syncApplicationApis(d, application, c)...)
	diags = append(diags, syncApplicationApiKeys(d, application, c)...)
	diags = append(diags, syncQuota(d, application, c)...)

	flattenApplication(application, d)

	return diags
}

func resourceApplicationRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	application, err := c.GetApplication(d.Id())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	flattenApplication(application, d)

	return diags
}

func resourceApplicationUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	application := &client.Application{}
	expandApplication(d, application)

	err = c.UpdateApplication(application)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	diags = append(diags, syncImage(d, application, c)...)
	diags = append(diags, syncApplicationApis(d, application, c)...)
	diags = append(diags, syncApplicationApiKeys(d, application, c)...)
	diags = append(diags, syncQuota(d, application, c)...)

	flattenApplication(application, d)

	return diags
}

func resourceApplicationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteApplication(d.Id())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	return diags
}

func syncQuota(d *schema.ResourceData, application *client.Application, c *client.Client) (diags diag.Diagnostics) {
	quota := &client.Quota{}
	err := c.GetQuotaForApplication(application.Id, quota)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	hasQuota := quota.Id != "" && !quota.System

	if wanted, ok := d.GetOk("quota"); ok {
		quota := &client.Quota{}
		expandQuota2(wanted, "quota for "+application.Name, quota)
		if hasQuota {
			c.UpdateQuotaForApplication(application, quota)
		} else {
			c.AddQuotaToApplication(application, quota)
		}
	} else {
		// Quota has gone in conf, must be deleted.
		err := c.DeleteQuotaFromApplication(application.Id)
		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
			return diags
		}
	}
	return diags
}
func expandQuota2(data interface{}, name string, quota *client.Quota) {
	d := data.([]interface{})[0].(map[string]interface{})
	quota.Name = name
	if a, ok := d["description"]; ok {
		quota.Description = a.(string)
	}
	quota.Type = "APPLICATION"
	quota.System = false
	quota.Restrictions = expandRestrictions(d["restriction"])
}

func syncApplicationApiKeys(d *schema.ResourceData, application *client.Application, c *client.Client) (diags diag.Diagnostics) {
	existing, err := c.ListApiKeysInApplication(application.Id)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	w := d.Get("apikey").([]interface{})
	wanted := make([]interface{}, len(w))
	copy(wanted, w)
	for _, w := range wanted {
		apiKey := expandApiKey(w, application.Id)
		found := false
		for i, exists := range existing {
			if exists == apiKey.Id {
				existing[i] = existing[len(existing)-1]
				existing = existing[:len(existing)-1]
				found = true
				break
			}
		}
		if !found {
			c.AddApiKeyToApplication(application, apiKey)
		}
	}
	for _, toDelete := range existing {
		c.DeleteApiFromApplication(application, toDelete)
	}
	return diags
}

func syncApplicationApis(d *schema.ResourceData, application *client.Application, c *client.Client) (diags diag.Diagnostics) {
	existing, err := c.ListApisInApplication(application.Id)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	w := d.Get("apis").([]interface{})
	wanted := make([]interface{}, len(w))
	copy(wanted, w)
	for _, w := range wanted {
		apiId := w.(string)
		found := false
		for i, exists := range existing {
			if exists == apiId {
				// the existing api exists also in the wanted state.
				// removing the api from the expected state,
				// so that in the end, 'existing' contains only
				// the APIs to remove.
				existing[i] = existing[len(existing)-1]
				existing = existing[:len(existing)-1]
				found = true
				break
			}
		}
		if !found {
			// The wanted api does not exist: adding it it
			c.AddApiToApplication(application, apiId)
		}
	}
	// Only the apis to remove remains in this array
	for _, toDelete := range existing {
		c.DeleteApiFromApplication(application, toDelete)
	}
	return diags
}

func flattenApplication(c *client.Application, d *schema.ResourceData) {
	d.SetId(c.Id)
	d.Set("name", c.Name)
	d.Set("description", c.Description)
	d.Set("org_id", c.OrganizationId)
	d.Set("phone", c.Phone)
	d.Set("email", c.Email)
	d.Set("enabled", c.Enabled)
	d.Set("state", c.State)
	//	d.Set("tags", c.Tags)
	d.Set("created_by", c.CreatedBy)
	d.Set("managed_by", c.ManagedBy)
	d.Set("created_on", c.CreatedOn)
	d.Set("last_updated", time.Now().Format(time.RFC850))
}

// ####### //
func expandApplication(d *schema.ResourceData, application *client.Application) {
	application.Id = d.Id()
	application.Name = d.Get("name").(string)
	application.Description = d.Get("description").(string)
	application.OrganizationId = d.Get("org_id").(string)
	application.Phone = d.Get("phone").(string)
	application.Email = d.Get("email").(string)
	application.Enabled = d.Get("enabled").(bool)
	application.State = d.Get("state").(string)
	//	if v, ok := d.GetOk("tags"); ok {
	//		application.Tags = toStringMap(v.(map[string]interface{}))
	//	}
	application.ManagedBy = toStringArray(d.Get("managed_by"))
}

func expandApiKey(m interface{}, appId string) (key *client.ApiKey) {
	asMap := m.(map[string]interface{})
	return &client.ApiKey{
		Id:            asMap["id"].(string),
		ApplicationId: appId,
		Enabled:       asMap["enabled"].(bool),
		Secret:        asMap["secret"].(string),
		CorsOrigins:   toStringArray(asMap["cors_origins"]),
	}
}
