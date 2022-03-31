package axwayapi

import (
	"context"
	"time"

	client "github.com/axway-techlab/axwayapi_client/axwayapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var TFBackendSchema = schemaMap{
	"swagger":                 _FORCENEW(required(_hashedString())),
	"org_id":                  _FORCENEW(required(_string())),
	"name":                    required(_string()),
	"base_path":               desc(inOut(_string()), "If none is given, will be read from the Swagger"),
	"summary":                 desc(inOut(_string()), "If none is given, will be read from the Swagger"),
	"description":             desc(inOut(_string()), "If none is given, will be read from the Swagger"),
	"resource_path":           desc(inOut(_string()), "If none is given, will be read from the Swagger"),
	"version":                 readonly(_string()),
	"consumes":                readonly(_plist(schema.TypeString)),
	"produces":                readonly(_plist(schema.TypeString)),
	"integral":                readonly(_bool()),
	"created_on":              readonly(_int()),
	"created_by":              readonly(_string()),
	"service_type":            readonly(_string()),
	"has_original_definition": readonly(_bool()),
	"import_url":              readonly(_string()),
	"properties":              readonly(_map(schema.TypeString)),
	"models":                  readonly(_string()),
}

func resourceBackend() *schema.Resource {
	return &schema.Resource{
		Schema:        TFBackendSchema,
		CreateContext: resourceBackendCreate,
		ReadContext:   resourceBackendRead,
		UpdateContext: resourceBackendUpdate,
		DeleteContext: resourceBackendDelete,
	}
}

func resourceBackendCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}
	orgId := d.Get("org_id").(string)
	name := d.Get("name").(string)
	file := d.Get("swagger").(string)

	backend, err := c.CreateBackend(orgId, name, "swagger", file)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	// doing an update with the writable fields that might be set
	// in the configuration.
	memo := map[string]string{}
	keys := []string{"base_path", "summary", "description", "resource_path"}
	for _, k := range keys {
		if v, ok := d.GetOk(k); ok {
			memo[k] = v.(string)
		}
	}
	flattenBackend(backend, d)
	for k, v := range memo {
		d.Set(k, v)
	}
	// Update only if needed.
	if len(memo) > 0 {
		diags = append(diags, resourceBackendUpdate(ctx, d, m)...)
	}

	return diags
}

func resourceBackendRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	backend, err := c.GetBackend(d.Id())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	flattenBackend(backend, d)

	return diags
}

func resourceBackendUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	backend := &client.Backend{}
	expandBackend(d, backend)

	err = c.UpdateBackend(backend)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	flattenBackend(backend, d)

	return diags
}

func resourceBackendDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteBackend(d.Id())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	return diags
}

func flattenBackend(backend *client.Backend, d *schema.ResourceData) {
	d.SetId(backend.Id)
	d.Set("base_path", backend.BasePath)
	d.Set("org_id", backend.OrganizationId)
	d.Set("id", backend.Id)
	d.Set("name", backend.Name)
	d.Set("version", backend.Version)
	d.Set("resource_path", backend.ResourcePath)
	d.Set("summary", backend.Summary)
	d.Set("description", backend.Description)
	d.Set("consumes", backend.Consumes)
	d.Set("produces", backend.Produces)
	d.Set("integral", backend.Integral)
	d.Set("created_on", backend.CreatedOn)
	d.Set("created_by", backend.CreatedBy)
	d.Set("service_type", backend.ServiceType)
	d.Set("has_original_definition", backend.HasOriginalDefinition)
	d.Set("import_url", backend.ImportUrl)
	d.Set("properties", backend.Properties)
	d.Set("models", serMap(backend.Models))
	d.Set("last_updated", time.Now().Format(time.RFC850))
}

func expandBackend(d *schema.ResourceData, backend *client.Backend) {
	backend.Id = d.Id()
	backend.BasePath = d.Get("base_path").(string)
	backend.OrganizationId = d.Get("org_id").(string)
	backend.Name = d.Get("name").(string)
	backend.Version = d.Get("version").(string)
	backend.ResourcePath = d.Get("resource_path").(string)
	backend.Summary = d.Get("summary").(string)
	backend.Description = d.Get("description").(string)
	backend.Consumes = toStringArray(d.Get("consumes"))
	backend.Produces = toStringArray(d.Get("produces"))
	backend.Integral = d.Get("integral").(bool)
	backend.CreatedOn = d.Get("created_on").(int)
	backend.CreatedBy = d.Get("created_by").(string)
	backend.ServiceType = d.Get("service_type").(string)
	backend.HasOriginalDefinition = d.Get("has_original_definition").(bool)
	backend.ImportUrl = d.Get("import_url").(string)
	backend.Properties = d.Get("properties").(map[string]interface{})
	backend.Models = deserMap(d.Get("models").(string))
}
