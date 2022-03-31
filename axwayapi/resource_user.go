package axwayapi

import (
	"context"
	"time"

	client "github.com/axway-techlab/axwayapi_client/axwayapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var TFUserSchema = schemaMap{
	"login_name": required(_string()),
	"name":       required(_string()),
	"email":      required(_string()),
	"enabled":    required(_bool()),
	"main_role": required(_listExact(1,
		&schema.Resource{
			Schema: map[string]*schema.Schema{
				"org_id": required(_string()),
				"role":   required(_string()),
			}})),
	"password":         _sensitive(required(_string())),
	"description":      inOut(_string()),
	"phone":            inOut(_string()),
	"mobile":           inOut(_string()),
	"image_jpg":        inOut(_hashedString()),
	"additional_roles": inOut(_map(schema.TypeString)),
	"created_on":       readonly(_int()),
	"state":            readonly(_string()),
	"type":             readonly(_string()),
	"auth_attrs":       readonly(_map(schema.TypeString)),
	"dn":               readonly(_string()),
}

func resourceUser() *schema.Resource {
	return &schema.Resource{
		Schema:        TFUserSchema,
		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,
	}
}

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	user := &client.User{}

	expandUser(d, user)

	err = c.CreateUser(user)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	diags = append(diags, syncImage(d, user, c)...)
	diags = append(diags, syncPassword(d, user, c)...)

	flattenUser(user, d)

	return diags
}

func syncPassword(d *schema.ResourceData, user *client.User, c *client.Client) (diags diag.Diagnostics) {
	if pwd, ok := d.GetOk("password"); ok {
		err := c.SetPassword(user.Id, pwd.(string))
		if nil != err {
			diags = warn(diags, "updating password for %T %s failed: %v", user, user.Id, err)
		}
	}
	return diags
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	user, err := c.GetUser(d.Id())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	flattenUser(user, d)

	return diags
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	user := &client.User{}
	expandUser(d, user)
	err = c.UpdateUser(user)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	diags = append(diags, syncImage(d, user, c)...)
	diags = append(diags, syncPassword(d, user, c)...)
	flattenUser(user, d)

	d.Set("last_updated", time.Now().Format(time.RFC850))
	return diags
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.DeleteUser(d.Id())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	return diags
}

func flattenUser(user *client.User, d *schema.ResourceData) {
	d.SetId(user.Id)
	d.Set("organization_id", user.OrganizationId)
	d.Set("name", user.Name)
	d.Set("description", user.Description)
	d.Set("login_name", user.LoginName)
	d.Set("email", user.Email)
	d.Set("phone", user.Phone)
	d.Set("mobile", user.Mobile)
	d.Set("role", user.Role)
	d.Set("enabled", user.Enabled)
	d.Set("created_on", user.CreatedOn)
	d.Set("state", user.State)
	d.Set("type", user.Type)
	d.Set("dn", user.Dn)
	d.Set("main_role", map[string]string{"org_id": user.OrganizationId, "role": user.Role})
	d.Set("auth_attrs", &user.AuthAttrs)
	d.Set("additional_roles", user.Orgs2Role)
}

func expandUser(d *schema.ResourceData, user *client.User) {
	//---
	user.Id = d.Id()
	user.OrganizationId = d.Get("main_role.0.org_id").(string)
	user.Name = d.Get("name").(string)
	user.Description = d.Get("description").(string)
	user.LoginName = d.Get("login_name").(string)
	user.Email = d.Get("email").(string)
	user.Phone = d.Get("phone").(string)
	user.Mobile = d.Get("mobile").(string)
	user.Role = d.Get("main_role.0.role").(string)
	user.Enabled = d.Get("enabled").(bool)
	user.CreatedOn = d.Get("created_on").(int)
	user.State = d.Get("state").(string)
	user.Type = d.Get("type").(string)
	user.Dn = d.Get("dn").(string)
	user.Orgs2Role = toStringMap(d.Get("additional_roles"))
	//---
}
