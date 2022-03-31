package axwayapi

import (
	"context"
	"time"

	client "github.com/axway-techlab/axwayapi_client/axwayapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var TFOrgSchema = schemaMap{
	"name":             required(_string()),
	"description":      inOut(_string()),
	"email":            inOut(_string()),
	"image_jpg":        inOut(_hashedString()),
	"restricted":       readonly(_bool()),
	"virtual_host":     inOut(_string()),
	"phone":            inOut(_string()),
	"enabled":          required(_bool()),
	"development":      inOut(_bool()),
	"dn":               readonly(_string()),
	"created_on":       readonly(_int()),
	"start_trial_date": readonly(_int()),
	"end_trial_date":   readonly(_int()),
	"trial_duration":   readonly(_int()),
	"is_trial":         readonly(_bool()),
}

func resourceOrganization() *schema.Resource {
	return &schema.Resource{
		Schema:        TFOrgSchema,
		CreateContext: resourceOrgCreate,
		ReadContext:   resourceOrgRead,
		UpdateContext: resourceOrgUpdate,
		DeleteContext: resourceOrgDelete,
	}
}

func resourceOrgCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	org := &client.Org{}
	expandOrg(d, org)
	err = c.CreateOrg(org)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	diags = append(diags, syncImage(d, org, c)...)

	flattenOrg(org, d)

	return diags
}

func resourceOrgRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	org, err := c.GetOrg(d.Id())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	flattenOrg(org, d)

	return diags
}

func resourceOrgUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	org := &client.Org{}
	expandOrg(d, org)
	err = c.UpdateOrg(org)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	diags = append(diags, syncImage(d, org, c)...)

	flattenOrg(org, d)

	return diags
}

func resourceOrgDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteOrg(d.Id())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	return diags
}

func flattenOrg(org *client.Org, d *schema.ResourceData) {
	d.SetId(org.Id)
	d.Set("name", org.Name)
	d.Set("description", org.Description)
	d.Set("email", org.Email)
	d.Set("restricted", org.Restricted)
	d.Set("virtual_host", org.VirtualHost)
	d.Set("phone", org.Phone)
	d.Set("enabled", org.Enabled)
	d.Set("development", org.Development)
	d.Set("dn", org.Dn)
	d.Set("created_on", org.CreatedOn)
	d.Set("start_trial_date", org.StartTrialDate)
	d.Set("end_trial_date", org.EndTrialDate)
	d.Set("trial_duration", org.TrialDuration)
	d.Set("is_trial", org.IsTrial)
	d.Set("last_updated", time.Now().Format(time.RFC850))
}

func expandOrg(d *schema.ResourceData, org *client.Org) {
	org.Id = d.Id()
	org.Name = d.Get("name").(string)
	org.Description = d.Get("description").(string)
	org.Email = d.Get("email").(string)
	org.VirtualHost = d.Get("virtual_host").(string)
	org.Phone = d.Get("phone").(string)
	org.Enabled = d.Get("enabled").(bool)
	org.Development = d.Get("development").(bool)
}
