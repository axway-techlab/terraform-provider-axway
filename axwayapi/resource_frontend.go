package axwayapi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	client "github.com/axway-techlab/axwayapi_client/axwayapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var TFFrontendSchema = &schemaMap{
	"id":                   readonly(_string()),
	"org_id":               _FORCENEW(inOut(_string())),
	"api_id":               _FORCENEW(inOut(_string())),
	"name":                 required(_string()),
	"version":              inOut(_string()),
	"api_routing_key":      inOut(_string()),
	"vhost":                inOut(_string()),
	"path":                 inOut(_string(r(`^(/[a-zA-Z0-9_.+-]+)+$`))),
	"description_type":     inOut(_string()),
	"description_manual":   inOut(_string()),
	"description_markdown": inOut(_string()),
	"description_url":      inOut(_string()),
	"summary":              inOut(_string()),
	"retired":              inOut(_bool()),
	"expired":              inOut(_bool()),
	"image_jpg":            inOut(_hashedString()),
	"retirement_date":      inOut(_int()),
	"state": desc(inOut(_string(oneOf(published, unpublished, deprecated))),
		`Can be 'unpublished', 'published' or 'deprecated'.
		Published and deprecated frontends must be temporarily 
		unpublished to apply some changes. A warning will be displayed when this occurs.`),
	"cors_profile":           inOut(_list(TFCorsProfile)),
	"security_profile":       inOut(_list(TFSecurityProfile)),
	"authentication_profile": inOut(_list(TFAuthenticationProfile)),
	"inbound_profile":        inOut(_list(TFInboudProfile)),
	"outbound_profile":       inOut(_list(TFOutboundProfile)),
	"service_profile":        inOut(_list(TFServiceProfile)),
	"ca_cert":                inOut(_list(TFCACert)),
	"tag":                    optional(_setMin(1, TFTag)),
	"custom_properties":      inOut(_map(schema.TypeString)),
	"created_on":             readonly(_int()),
	"created_by":             readonly(_string()),
}

var TFCorsProfile = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name":                required(_string()),
		"is_default":          required(_bool()),
		"origins":             required(_plist(schema.TypeString)),
		"allowed_headers":     required(_plist(schema.TypeString)),
		"exposed_headers":     required(_plist(schema.TypeString)),
		"support_credentials": required(_bool()),
		"max_age_seconds":     inOut(_int()),
	},
}

var TFAuthenticationProfile = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name":       inOut(_string()),
		"type":       inOut(_string()),
		"is_default": inOut(_bool()),
		"parameters": inOut(_map(schema.TypeString)),
	},
}

var TFInboudProfile = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name":             required(_string()),
		"security_profile": inOut(_string()),
		"cors_profile":     inOut(_string()),
		"monitor_api":      inOut(_bool()),
		"monitor_subject":  inOut(_string()),
	},
}

var TFOutboundProfile = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name":                   required(_string()),
		"authentication_profile": inOut(_string()),
		"route_type":             inOut(_string()),
		"request_policy":         inOut(_string()),
		"response_policy":        inOut(_string()),
		"route_policy":           inOut(_string()),
		"fault_handler_policy":   inOut(_string()),
		"api_id":                 inOut(_string()),
		"api_method_id":          inOut(_string()),
		"parameters":             inOut(_list(TFParamValue)),
	},
}
var TFParamValue = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name":       inOut(_string()),
		"param_type": inOut(_string()),
		"type":       inOut(_string()),
		"format":     inOut(_string()),
		"value":      inOut(_string()),
		"required":   inOut(_bool()),
		"exclude":    inOut(_bool()),
		"additional": inOut(_bool()),
	},
}
var TFServiceProfile = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name":      required(_string()),
		"api_id":    required(_string()),
		"base_path": required(_string()),
	},
}
var TFCACert = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"cert_blob":           required(_string()),
		"name":                required(_string()),
		"alias":               required(_string()),
		"subject":             required(_string()),
		"issuer":              required(_string()),
		"version":             required(_int()),
		"not_valid_before":    required(_int()),
		"not_valid_after":     required(_int()),
		"signature_algorithm": required(_string()),
		"sha1_fingerprint":    required(_string()),
		"md5_fingerprint":     required(_string()),
		"expired":             required(_bool()),
		"not_yet_valid":       required(_bool()),
		"inbound":             required(_bool()),
		"outbound":            required(_bool()),
	},
}

func resourceFrontend() *schema.Resource {
	return &schema.Resource{
		Schema:        *TFFrontendSchema,
		CreateContext: resourceFrontendCreate,
		ReadContext:   resourceFrontendRead,
		UpdateContext: resourceFrontendUpdate,
		DeleteContext: resourceFrontendDelete,
	}
}

func resourceFrontendCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	frontend := &client.Frontend{}
	expandFrontendForCreate(d, frontend)

	err = c.CreateFrontend(frontend)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	diags = append(diags, syncImage(d, frontend, c)...)

	flattenFrontend(frontend, d)
	return diags
}

func resourceFrontendRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	frontend, err := c.GetFrontend(d.Id())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	diags = append(diags, syncImage(d, frontend, c)...)

	flattenFrontend(frontend, d)

	return diags
}

func resourceFrontendUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	sis, swant := d.GetChange("state")
	// we must unpublish if the api is
	// - not unpublished, AND
	// - has changes on fields that cannot be forced.
	mustUnpublish := sis.(string) != unpublished &&
		d.HasChangesExcept("state",
			"description_type", "description_manual", "description_markdown", "description_url",
			"summary",
			"image_jpg")

	// Unpublish the API, and undeprecate it if needed.
	var frontend *client.Frontend
	if mustUnpublish {
		// A stub object carrying only the id, which is enough for the next few calls
		frontend = &client.Frontend{Id: d.Id()}
		s := sis
		if s == deprecated {
			// This call also refreshes the frontend object.
			err := c.UndeprecateFrontend(frontend)
			if err != nil {
				diags = append(diags, diag.FromErr(err)...)
				return diags
			}
			s = frontend.State
		}
		if s == published {
			// This call also refreshes the frontend object.
			err := c.UnpublishFrontend(frontend)
			if err != nil {
				diags = append(diags, diag.FromErr(err)...)
				return diags
			}
		}
		diags = warn(diags, "The api %s has been temporarily unpublished to allow some changes", frontend.Name)
	} else {
		frontend, err = c.GetFrontend(d.Id())
		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
			return diags
		}
	}

	// Apply the desired changes onto the frontend object
	if d.HasChangesExcept("state") {
		expandFrontendForUpdate(d, frontend)
		err = c.UpdateFrontend(frontend)
		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
			return diags
		}
	}

	diags = append(diags, syncImage(d, frontend, c)...)

	flattenFrontend(frontend, d)

	// Fix the state of the proxy
	diags = append(diags, adaptStates(c, d, swant.(string), frontend)...)
	if diags.HasError() {
		return diags
	}
	flattenFrontend(frontend, d)
	d.Set("last_updated", time.Now().Format(time.RFC850))

	return diags
}

const (
	published   = "published"
	unpublished = "unpublished"
	deprecated  = "deprecated"
)

func adaptStates(c *client.Client, d *schema.ResourceData, state string, frontend *client.Frontend) (diags diag.Diagnostics) {
	d0 := frontend.Deprecated
	s0 := frontend.State
	s1 := state
	if d0 && s0 == published {
		s0 = deprecated
	}
	if !diags.HasError() {
		transition := fmt.Sprintf("%s -> %s", s0, s1)
		switch transition {
		case unpublished + " -> " + published:
			diags = guard(diags, c.PublishFrontend, frontend)
		case unpublished + " -> " + deprecated:
			diags = guard(diags, c.PublishFrontend, frontend)
			diags = guard(diags, c.DeprecateFrontend, frontend)
		case published + " -> " + unpublished:
			diags = guard(diags, c.UnpublishFrontend, frontend)
		case published + " -> " + deprecated:
			diags = guard(diags, c.DeprecateFrontend, frontend)
		case deprecated + " -> " + published:
			diags = guard(diags, c.UndeprecateFrontend, frontend)
		case deprecated + " -> " + unpublished:
			diags = guard(diags, c.UndeprecateFrontend, frontend)
			diags = guard(diags, c.UnpublishFrontend, frontend)
		case deprecated + " -> " + deprecated,
			published + " -> " + published,
			unpublished + " -> " + unpublished:
			// nothing to do, the api is already in the expected state.
		default:
			// could be e.g. anything starting from pending
			diags = warn(diags, "transition (%s) is not implemented: ignoring this change", transition)
		}
	}
	return diags
}
func resourceFrontendDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	if d.Get("state") != unpublished {
		// unpublish the frontend prior to deletion
		frontend := &client.Frontend{}
		expandFrontendForUpdate(d, frontend)
		c.UnpublishFrontend(frontend)
	}

	err = c.DeleteFrontend(d.Id())
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	return diags
}

func flattenFrontend(c *client.Frontend, d *schema.ResourceData) {
	d.SetId(c.Id)
	d.Set("org_id", c.OrganizationId)                    //inOut(_string())
	d.Set("api_id", c.ApiId)                             //inOut(_string())
	d.Set("name", c.Name)                                //inOut(_string())
	d.Set("version", c.Version)                          //inOut(_string())
	d.Set("api_routing_key", c.ApiRoutingKey)            //inOut(_string())
	d.Set("vhost", c.Vhost)                              //inOut(_string())
	d.Set("path", c.Path)                                //inOut(_string())
	d.Set("description_type", c.DescriptionType)         //inOut(_string())
	d.Set("description_manual", c.DescriptionManual)     //inOut(_string())
	d.Set("description_markdown", c.DescriptionMarkdown) //inOut(_string())
	d.Set("description_url", c.DescriptionUrl)           //inOut(_string())
	d.Set("summary", c.Summary)                          //inOut(_string())
	d.Set("retired", c.Retired)                          //inOut(_bool())
	d.Set("expired", c.Expired)                          //inOut(_bool())
	d.Set("retirement_date", c.RetirementDate)           //inOut(_int())
	if c.Deprecated {
		d.Set("state", deprecated)
	} else {
		d.Set("state", c.State)
	}
	d.Set("cors_profile", flattenCorsProfiles(c.CorsProfiles))                               //inOut(_list(TFCorsProfile))
	d.Set("security_profile", flattenSecurityProfiles(c.SecurityProfiles))                   //inOut(_list(TFSecurityProfile))
	d.Set("authentication_profile", flattenAuthenticationProfiles(c.AuthenticationProfiles)) //inOut(_list(TFAuthenticationProfile))
	d.Set("inbound_profile", flattenInboundProfiles(c.InboundProfiles))                      //inOut(_namedMap(TFInboundProfile))
	d.Set("outbound_profile", flattenOutboundProfiles(c.OutboundProfiles))                   //inOut(_namedMap(TFOutboundProfile))
	d.Set("service_profile", flattenServiceProfiles(c.ServiceProfiles))                      //inOut(_namedMap(TFServiceProfile))
	d.Set("ca_cert", flattenCACerts(c.CACerts))                                              //inOut(_list(TFCACert))
	d.Set("tag", flattenTags(c.Tags))                                                        //inOut(_pnamedMap(_plist(schema.TypeString)))
	d.Set("custom_properties", c.CustomProperties)                                           //inOut(_map(schema.TypeString)),
	d.Set("created_by", c.CreatedBy)                                                         //inOut(_string())
	d.Set("created_on", c.CreatedOn)                                                         //inOut(_int())
}

func flattenCorsProfiles(c []client.CorsProfile) []flattenMap {
	r := make([]flattenMap, len(c))
	for i, a := range c {
		r[i] = flattenMap{
			"name":                a.Name,               //required(_string())
			"is_default":          a.IsDefault,          //required(_bool())
			"origins":             a.Origins,            //required(_plist(schema.TypeString))
			"allowed_headers":     a.AllowedHeaders,     //required(_plist(schema.TypeString))
			"exposed_headers":     a.ExposedHeaders,     //required(_plist(schema.TypeString))
			"support_credentials": a.SupportCredentials, //required(_bool())
			"max_age_seconds":     a.MaxAgeSeconds,      //inOut(_int())
		}
	}
	return r
}
func flattenSecurityProfiles(c []client.SecurityProfile) []flattenMap {
	r := make([]flattenMap, len(c))
	for i, a := range c {
		r[i] = flattenMap{
			"name":       a.Name,                    //required(_string())
			"is_default": a.IsDefault,               //required(_bool())
			"devices":    flattenDevices(a.Devices), // inOut(_listMin(1, TFDevice)),
		}
	}
	return r
}
func flattenDevices(c []client.Device) []flattenMap {
	r := make([]flattenMap, len(c))
	for i, a := range c {
		r[i] = flattenMap{
			"name":  a.Name,  //required(inOut(_string()))
			"type":  a.Type,  //required(inOut(_string()))
			"order": a.Order, //required(inOut(_int()))
			/**/ "properties": a.Properties, //required(_map(schema.TypeString)),
		}
	}
	return r
}

func flattenAuthenticationProfiles(c []client.AuthenticationProfile) []flattenMap {
	r := make([]flattenMap, len(c))
	for i, a := range c {
		r[i] = flattenMap{
			"name":       a.Name,      //inOut(_string())
			"type":       a.Type,      //inOut(_string())
			"is_default": a.IsDefault, //inOut(_bool())
			/**/ "parameters": flattenParameters(a.Parameters), //required(_map(schema.TypeString)),
		}
	}
	return r
}
func flattenParameters(tags map[string]interface{}) map[string]interface{} {
	r := make(map[string]interface{}, len(tags))
	for k, v := range tags {
		a, e := json.Marshal(v)
		if e != nil {
			panic(e)
		}
		r[k] = string(a)
	}
	return r
}
func flattenInboundProfiles(c map[string]client.InboundProfile) []flattenMap {
	r := make([]flattenMap, 0)
	for name, a := range c {
		r = append(r, flattenMap{
			"name":             name,
			"security_profile": a.SecurityProfile, //inOut(_string())
			"cors_profile":     a.CorsProfile,     //inOut(_string())
			"monitor_api":      a.MonitorAPI,      //inOut(_bool())
			"monitor_subject":  a.MonitorSubject,  //inOut(_string())
		})
	}
	return r
}
func flattenOutboundProfiles(c map[string]client.OutboundProfile) []flattenMap {
	r := make([]flattenMap, 0)
	for name, a := range c {
		r = append(r, flattenMap{
			"name":                   name,
			"authentication_profile": a.AuthenticationProfile,          //inOut(_string())
			"route_type":             a.RouteType,                      //inOut(_string())
			"request_policy":         a.RequestPolicy,                  //inOut(_string())
			"response_policy":        a.ResponsePolicy,                 //inOut(_string())
			"route_policy":           a.RoutePolicy,                    //inOut(_string())
			"fault_handler_policy":   a.FaultHandlerPolicy,             //inOut(_string())
			"api_id":                 a.ApiId,                          //inOut(_string())
			"api_method_id":          a.ApiMethodId,                    //inOut(_string())
			"parameters":             flattenParamValues(a.Parameters), //inOut(_list(TFParamValue))
		})
	}
	return r
}
func flattenParamValues(c []client.ParamValue) []flattenMap {
	r := make([]flattenMap, len(c))
	for i, a := range c {
		r[i] = flattenMap{
			"name":       a.Name,       //inOut(_string())
			"param_type": a.ParamType,  //inOut(_string())
			"type":       a.Type,       //inOut(_string())
			"format":     a.Format,     //inOut(_string())
			"value":      a.Value,      //inOut(_string())
			"required":   a.Required,   //inOut(_bool())
			"exclude":    a.Exclude,    //inOut(_bool())
			"additional": a.Additional, //inOut(_bool())
		}
	}
	return r
}
func flattenServiceProfiles(c map[string]client.ServiceProfile) []flattenMap {
	r := make([]flattenMap, 0)
	for name, a := range c {
		r = append(r, flattenMap{
			"name":      name,
			"api_id":    a.ApiId,    //required(_string())
			"base_path": a.BasePath, //required(_string())
		})
	}
	return r
}
func flattenCACerts(c []client.CACert) []flattenMap {
	r := make([]flattenMap, len(c))
	for i, a := range c {
		r[i] = flattenMap{
			"cert_blob":           a.CertBlob,           //required(_string())
			"name":                a.Name,               //required(_string())
			"alias":               a.Alias,              //required(_string())
			"subject":             a.Subject,            //required(_string())
			"issuer":              a.Issuer,             //required(_string())
			"version":             a.Version,            //required(_int())
			"not_valid_before":    a.NotValidBefore,     //required(_int())
			"not_valid_after":     a.NotValidAfter,      //required(_int())
			"signature_algorithm": a.SignatureAlgorithm, //required(_string())
			"sha1_fingerprint":    a.Sha1Fingerprint,    //required(_string())
			"md5_fingerprint":     a.Md5Fingerprint,     //required(_string())
			"expired":             a.Expired,            //required(_bool())
			"not_yet_valid":       a.NotYetValid,        //required(_bool())
			"inbound":             a.Inbound,            //required(_bool())
			"outbound":            a.Outbound,           //required(_bool())
		}
	}
	return r
}

// ####### //
func expandFrontendForCreate(d *schema.ResourceData, frontend *client.Frontend) {
	expandFrontendForUpdate(d, frontend)
	switch state := d.Get("state"); state {
	case unpublished, published:
		frontend.Deprecated = false
		frontend.State = state.(string)
	case deprecated:
		frontend.Deprecated = true
		frontend.State = published
	}
}
func expandFrontendForUpdate(d *schema.ResourceData, frontend *client.Frontend) {
	frontend.Id = d.Id()
	frontend.OrganizationId = d.Get("org_id").(string)                    //inOut(_string())
	frontend.ApiId = d.Get("api_id").(string)                             //inOut(_string())
	frontend.Name = d.Get("name").(string)                                //inOut(_string())
	frontend.Version = d.Get("version").(string)                          //inOut(_string())
	frontend.ApiRoutingKey = d.Get("api_routing_key").(string)            //inOut(_string())
	frontend.Vhost = d.Get("vhost").(string)                              //inOut(_string())
	frontend.Path = d.Get("path").(string)                                //inOut(_string())
	frontend.DescriptionType = d.Get("description_type").(string)         //inOut(_string())
	frontend.DescriptionManual = d.Get("description_manual").(string)     //inOut(_string())
	frontend.DescriptionMarkdown = d.Get("description_markdown").(string) //inOut(_string())
	frontend.DescriptionUrl = d.Get("description_url").(string)           //inOut(_string())
	frontend.Summary = d.Get("summary").(string)                          //inOut(_string())
	frontend.Retired = d.Get("retired").(bool)                            //inOut(_bool())
	frontend.Expired = d.Get("expired").(bool)                            //inOut(_bool())
	frontend.RetirementDate = d.Get("retirement_date").(int)              //inOut(_int())
	if v, ok := d.GetOk("cors_profile"); ok {
		frontend.CorsProfiles = expandCorsProfiles(v)
	}
	if v, ok := d.GetOk("security_profile"); ok {
		frontend.SecurityProfiles = expandSecurityProfiles(v) //inOut(_list(TFSecurityProfile))
	}
	if v, ok := d.GetOk("authentication_profile"); ok {
		frontend.AuthenticationProfiles = expandAuthenticationProfiles(v) //inOut(_list(TFAuthenticationProfile))
	}
	if v, ok := d.GetOk("inbound_profile"); ok {
		frontend.InboundProfiles = expandInboundProfiles(v) //inOut(_namedMap(TFInboundProfile))
	}
	if v, ok := d.GetOk("outbound_profile"); ok {
		frontend.OutboundProfiles = expandOutboundProfiles(v) //inOut(_namedMap(TFOutboundProfile))
	}
	if v, ok := d.GetOk("service_profile"); ok {
		frontend.ServiceProfiles = expandServiceProfiles(v) //inOut(_namedMap(TFServiceProfile))
	}
	if v, ok := d.GetOk("ca_cert"); ok {
		frontend.CACerts = expandCACerts(v) //inOut(_list(TFCACert))
	}
	if v, ok := d.GetOk("tag"); ok {
		frontend.Tags = toTags(v.(*schema.Set)) //inOut(_pnamedMap(_plist(schema.TypeString)))
	}
	frontend.CustomProperties = d.Get("custom_properties").(map[string]interface{}) //inOut(_map(schema.TypeString)),
	frontend.CreatedBy = d.Get("created_by").(string)                               //readonly(_string())
	frontend.CreatedOn = d.Get("created_on").(int)                                  //readonly(_int())
}

func expandCorsProfiles(v interface{}) []client.CorsProfile {
	c := v.([]interface{})
	r := make([]client.CorsProfile, len(c))
	for i, b := range c {
		a := b.(map[string]interface{})
		r[i].Name = a["name"].(string)                            //required(_string())
		r[i].IsDefault = a["is_default"].(bool)                   //required(_bool())
		r[i].Origins = toStringArray(a["origins"])                //required(_plist(schema.TypeString))
		r[i].AllowedHeaders = toStringArray(a["allowed_headers"]) //required(_plist(schema.TypeString))
		r[i].ExposedHeaders = toStringArray(a["exposed_headers"]) //required(_plist(schema.TypeString))
		r[i].SupportCredentials = a["support_credentials"].(bool) //required(_bool())
		r[i].MaxAgeSeconds = a["max_age_seconds"].(int)           //inOut(_int())
	}
	return r
}

func expandSecurityProfiles(v interface{}) []client.SecurityProfile {
	c := v.([]interface{})
	r := make([]client.SecurityProfile, len(c))
	for i, b := range c {
		a := b.(map[string]interface{})
		r[i].Name = a["name"].(string)             //required(_string())
		r[i].IsDefault = a["is_default"].(bool)    //required(_bool())
		r[i].Devices = expandDevices(a["devices"]) // inOut(_listMin(1, TFDevice)),
	}
	return r
}
func expandDevices(v interface{}) []client.Device {
	c := v.([]interface{})
	r := make([]client.Device, len(c))
	for i, b := range c {
		a := b.(map[string]interface{})
		r[i].Name = a["name"].(string) //required(inOut(_string()))
		r[i].Type = a["type"].(string) //required(inOut(_string()))
		r[i].Order = a["order"].(int)  //required(inOut(_int()))
		/**/ r[i].Properties = a["properties"].(flattenMap) //required(_map(schema.TypeString)),
	}
	return r
}

func expandAuthenticationProfiles(v interface{}) []client.AuthenticationProfile {
	c := v.([]interface{})
	r := make([]client.AuthenticationProfile, len(c))
	for i, b := range c {
		a := b.(map[string]interface{})
		r[i].Name = a["name"].(string)                                           //inOut(_string())
		r[i].Type = a["type"].(string)                                           //inOut(_string())
		r[i].IsDefault = a["is_default"].(bool)                                  //inOut(_bool())
		r[i].Parameters = toParameters(a["parameters"].(map[string]interface{})) //required(_map(schema.TypeString)),
	}
	return r
}
func expandInboundProfiles(v interface{}) map[string]client.InboundProfile {
	c := v.([]interface{})
	r := make(map[string]client.InboundProfile, len(c))
	for _, b := range c {
		a := b.(map[string]interface{})
		r[a["name"].(string)] = client.InboundProfile{
			SecurityProfile: a["security_profile"].(string), //inOut(_string())
			CorsProfile:     a["cors_profile"].(string),     //inOut(_string())
			MonitorAPI:      a["monitor_api"].(bool),        //inOut(_bool())
			MonitorSubject:  a["monitor_subject"].(string),  //inOut(_string())
		}
	}
	return r
}
func expandOutboundProfiles(v interface{}) map[string]client.OutboundProfile {
	c := v.([]interface{})
	r := make(map[string]client.OutboundProfile, len(c))
	for _, b := range c {
		a := b.(map[string]interface{})
		r[a["name"].(string)] = client.OutboundProfile{
			AuthenticationProfile: a["authentication_profile"].(string), //inOut(_string())
			RouteType:             a["route_type"].(string),             //inOut(_string())
			RequestPolicy:         a["request_policy"].(string),         //inOut(_string())
			ResponsePolicy:        a["response_policy"].(string),        //inOut(_string())
			RoutePolicy:           a["route_policy"].(string),           //inOut(_string())
			FaultHandlerPolicy:    a["fault_handler_policy"].(string),   //inOut(_string())
			ApiId:                 a["api_id"].(string),                 //inOut(_string())
			ApiMethodId:           a["api_method_id"].(string),          //inOut(_string())
			Parameters:            expandParamValues(a["parameters"]),   //inOut(_list(TFParamValue))
		}
	}
	return r
}
func expandParamValues(v interface{}) []client.ParamValue {
	c := v.([]interface{})
	r := make([]client.ParamValue, len(c))
	for i, b := range c {
		a := b.(map[string]interface{})
		r[i].Name = a["name"].(string)            //inOut(_string())
		r[i].ParamType = a["param_type"].(string) //inOut(_string())
		r[i].Type = a["type"].(string)            //inOut(_string())
		r[i].Format = a["format"].(string)        //inOut(_string())
		r[i].Value = a["value"].(string)          //inOut(_string())
		r[i].Required = a["required"].(bool)      //inOut(_bool())
		r[i].Exclude = a["exclude"].(bool)        //inOut(_bool())
		r[i].Additional = a["additional"].(bool)  //inOut(_bool())
	}
	return r
}
func expandServiceProfiles(v interface{}) map[string]client.ServiceProfile {
	c := v.([]interface{})
	r := make(map[string]client.ServiceProfile, len(c))
	for _, b := range c {
		a := b.(map[string]interface{})
		r[a["name"].(string)] = client.ServiceProfile{
			ApiId:    a["api_id"].(string),    //required(_string())
			BasePath: a["base_path"].(string), //required(_string())
		}
	}
	return r
}
func expandCACerts(v interface{}) []client.CACert {
	c := v.([]interface{})
	r := make([]client.CACert, len(c))
	for i, b := range c {
		a := b.(map[string]interface{})
		r[i].CertBlob = a["cert_blob"].(string)                     //required(_string())
		r[i].Name = a["name"].(string)                              //required(_string())
		r[i].Alias = a["alias"].(string)                            //required(_string())
		r[i].Subject = a["subject"].(string)                        //required(_string())
		r[i].Issuer = a["issuer"].(string)                          //required(_string())
		r[i].Version = a["version"].(int)                           //required(_int())
		r[i].NotValidBefore = a["not_valid_before"].(int)           //required(_int())
		r[i].NotValidAfter = a["not_valid_after"].(int)             //required(_int())
		r[i].SignatureAlgorithm = a["signature_algorithm"].(string) //required(_string())
		r[i].Sha1Fingerprint = a["sha1_fingerprint"].(string)       //required(_string())
		r[i].Md5Fingerprint = a["md5_fingerprint"].(string)         //required(_string())
		r[i].Expired = a["expired"].(bool)                          //required(_bool())
		r[i].NotYetValid = a["not_yet_valid"].(bool)                //required(_bool())
		r[i].Inbound = a["inbound"].(bool)                          //required(_bool())
		r[i].Outbound = a["outbound"].(bool)                        //required(_bool())
	}
	return r
}
