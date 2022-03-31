package axwayapi

import (
	"context"
	"fmt"
	"time"

	client "github.com/axway-techlab/axwayapi_client/axwayapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var TFConfigSchema = schemaMap{
	"advisory_banner_enabled":             inOut(_bool()),
	"advisory_banner_text":                inOut(_string()),
	"api_default_virtual_host":            inOut(_string()),
	"api_import_editable":                 inOut(_bool()),
	"api_import_mime_validation":          inOut(_bool()),
	"api_import_timeout":                  inOut(_int()),
	"api_portal_hostname":                 inOut(_string()),
	"api_portal_name":                     inOut(_string()),
	"api_routing_key_enabled":             inOut(_bool()),
	"api_routing_key_location":            inOut(_string()),
	"application_scope_restrictions":      inOut(_bool()),
	"architecture":                        readonly(_string()),
	"auto_approve_applications":           inOut(_bool()),
	"auto_approve_user_registration":      inOut(_bool()),
	"base_o_auth":                         inOut(_bool()),
	"change_password_on_first_login":      inOut(_bool()),
	"default_trial_duration":              inOut(_int()),
	"delegate_application_administration": inOut(_bool()),
	"delegate_user_administration":        inOut(_bool()),
	"email_bounce_address":                inOut(_string()),
	"email_from":                          inOut(_string()),
	"fault_handlers_enabled":              inOut(_bool()),
	"global_fault_handler_policy":         inOut(_string()),
	"global_policies_enabled":             inOut(_bool()),
	"global_request_policy":               inOut(_string()),
	"global_response_policy":              inOut(_string()),
	"is_api_portal_configured":            inOut(_bool()),
	"is_trial":                            inOut(_bool()),
	"login_name_regex":                    inOut(_string()),
	"login_response_time":                 inOut(_int()),
	"minimum_password_length":             inOut(_int()),
	"oadmin_self_service_enabled":         inOut(_bool()),
	"os":                                  readonly(_string()),
	"password_expiry_enabled":             inOut(_bool()),
	"password_lifetime_days":              inOut(_int()),
	"portal_hostname":                     inOut(_string()),
	"portal_name":                         inOut(_string()),
	"product_version":                     readonly(_string()),
	"promote_api_via_policy":              inOut(_bool()),
	"reg_token_email_enabled":             inOut(_bool()),
	"registration_enabled":                readonly(_bool()),
	"reset_password_enabled":              inOut(_bool()),
	"server_certificate_verification":     inOut(_bool()),
	"session_idle_timeout_millis":         inOut(_int()),
	"session_timeout_millis":              inOut(_int()),
	"strict_certificate_checking":         inOut(_bool()),
	"system_o_auth_scopes_enabled":        inOut(_bool()),
	"user_name_regex":                     inOut(_string()),
	"lock_user_account": inOut(_listMax(1, &schema.Resource{
		Schema: map[string]*schema.Schema{
			"enabled":               inOut(_bool()),
			"attempts":              inOut(_int()),
			"time_period":           inOut(_int()),
			"time_period_unit":      inOut(_string()),
			"lock_time_period":      inOut(_int()),
			"lock_time_period_unit": inOut(_string()),
		},
	})),
	"application_default_quota": inOut(_listMax(1, &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id":          readonly(_string()),
			"restriction": inOut(_set(TFRestriction)),
		},
	})),
	"system_default_quota": inOut(_listMax(1, &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id":          readonly(_string()),
			"restriction": inOut(_set(TFRestriction)),
		},
	})),
}

// This whole resource works slightly differently than regular ones.
// The CREATE part does in fact only read the configuration from the gateway
// The READ and UPDATE works as expected
// The DELETE is a no-op.
// It is so because the config is not in itself a resource to be created.
// In a regular workflow, it should be imported.
// But just to have a nice one-step workflow, we simulate creation and deletion.
func resourceConfig() *schema.Resource {
	return &schema.Resource{
		Schema:        TFConfigSchema,
		CreateContext: resourceConfigCreate,
		ReadContext:   resourceConfigRead,
		UpdateContext: resourceConfigUpdate,
		DeleteContext: resourceConfigDelete,
	}
}

func resourceConfigCreate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	// read config from API Gateway
	config, err := c.GetConfig()
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	// apply the tf configuration on the config read from server
	expandConfig(d, config, false)
	// The config object is updated with the latest configs
	err = c.UpdateConfig(config)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	diags = append(diags, syncDefaultQuota(d, c)...)
	// update our state from the freshest config read from server.

	flattenConfig(config, d)

	d.SetId(fmt.Sprintf("%s/config", c.HostURL))
	return diags
}

func resourceConfigRead(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	// Read conf from server
	config, err := c.GetConfig()
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	// apply the read conf onto our state
	flattenConfig(config, d)

	// Reading the quota
	defapp, err := c.GetQuota("00000000-0000-0000-0000-000000000001")
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	err = d.Set("application_default_quota", flattenQuota(defapp))
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	// Reading the quota
	defsys, err := c.GetQuota("00000000-0000-0000-0000-000000000000")
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	err = d.Set("system_default_quota", flattenQuota(defsys))
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	return diags
}
func flattenQuota(quota *client.Quota) []flattenMap {
	return []flattenMap{{
		"id":          quota.Id,
		"restriction": flattenRestriction(quota.Restrictions),
	}}
}

func resourceConfigUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	c, err := m.(*ProviderState).GetClient()
	if err != nil {
		return diag.FromErr(err)
	}

	// Start from blank config object
	config := &client.Config{}
	// forcibly apply our state onto that object
	expandConfig(d, config, true)
	// update the server with this new state
	err = c.UpdateConfig(config)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	diags = append(diags, syncDefaultQuota(d, c)...)

	// update our state from the freshest config read from server.
	flattenConfig(config, d)

	d.Set("last_updated", time.Now().Format(time.RFC850))
	return diags
}

func syncDefaultQuota(d *schema.ResourceData, c *client.Client) (diags diag.Diagnostics) {

	defsys, err := c.GetQuota("00000000-0000-0000-0000-000000000000")
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	if q, ok := d.GetOk("system_default_quota"); ok {
		quota := &client.Quota{}
		expandQuota2(q, "", quota)
		defsys.Restrictions = quota.Restrictions
	} else {
		defsys.Restrictions = []client.Constraint{}
	}
	err = c.UpdateQuota(defsys)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	if diags.HasError() {
		return diags
	}
	d.Set("system_default_quota", flattenQuota(defsys))

	defapp, err := c.GetQuota("00000000-0000-0000-0000-000000000001")
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	if q, ok := d.GetOk("application_default_quota"); ok {
		quota := &client.Quota{}
		expandQuota2(q, "", quota)
		defapp.Restrictions = quota.Restrictions
	} else {
		defapp.Restrictions = []client.Constraint{}
	}
	err = c.UpdateQuota(defapp)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	if diags.HasError() {
		return diags
	}
	d.Set("application_default_quota", flattenQuota(defapp))

	return diags
}

func resourceConfigDelete(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	return diags
}

// There was a version based on reflection, but this is easier to maintain.
func flattenConfig(axconfig *client.Config, d *schema.ResourceData) {
	d.Set("registration_enabled", axconfig.RegistrationEnabled)
	d.Set("reg_token_email_enabled", axconfig.RegTokenEmailEnabled)
	d.Set("api_import_timeout", axconfig.ApiImportTimeout)
	d.Set("is_trial", axconfig.IsTrial)
	d.Set("promote_api_via_policy", axconfig.PromoteApiViaPolicy)
	d.Set("system_o_auth_scopes_enabled", axconfig.SystemOAuthScopesEnabled)
	d.Set("oadmin_self_service_enabled", axconfig.OadminSelfServiceEnabled)
	d.Set("product_version", axconfig.ProductVersion)
	d.Set("portal_name", axconfig.PortalName)
	d.Set("global_response_policy", axconfig.GlobalResponsePolicy)
	d.Set("auto_approve_applications", axconfig.AutoApproveApplications)
	d.Set("global_request_policy", axconfig.GlobalRequestPolicy)
	d.Set("auto_approve_user_registration", axconfig.AutoApproveUserRegistration)
	d.Set("delegate_application_administration", axconfig.DelegateApplicationAdministration)
	d.Set("api_default_virtual_host", axconfig.ApiDefaultVirtualHost)
	d.Set("api_routing_key_location", axconfig.ApiRoutingKeyLocation)
	d.Set("application_scope_restrictions", axconfig.ApplicationScopeRestrictions)
	d.Set("base_o_auth", axconfig.BaseOAuth)
	d.Set("email_bounce_address", axconfig.EmailBounceAddress)
	d.Set("advisory_banner_enabled", axconfig.AdvisoryBannerEnabled)
	d.Set("user_name_regex", axconfig.UserNameRegex)
	d.Set("api_import_mime_validation", axconfig.ApiImportMimeValidation)
	d.Set("session_idle_timeout_millis", axconfig.SessionIdleTimeout)
	d.Set("is_api_portal_configured", axconfig.IsApiPortalConfigured)
	d.Set("change_password_on_first_login", axconfig.ChangePasswordOnFirstLogin)
	d.Set("session_timeout_millis", axconfig.SessionTimeout)
	d.Set("email_from", axconfig.EmailFrom)
	d.Set("api_routing_key_enabled", axconfig.ApiRoutingKeyEnabled)
	d.Set("login_response_time", axconfig.LoginResponseTime)
	d.Set("server_certificate_verification", axconfig.ServerCertificateVerification)
	d.Set("reset_password_enabled", axconfig.ResetPasswordEnabled)
	d.Set("advisory_banner_text", axconfig.AdvisoryBannerText)
	d.Set("api_import_editable", axconfig.ApiImportEditable)
	d.Set("api_portal_hostname", axconfig.ApiPortalHostname)
	d.Set("api_portal_name", axconfig.ApiPortalName)
	d.Set("fault_handlers_enabled", axconfig.FaultHandlersEnabled)
	d.Set("architecture", axconfig.Architecture)
	d.Set("strict_certificate_checking", axconfig.StrictCertificateChecking)
	d.Set("global_policies_enabled", axconfig.GlobalPoliciesEnabled)
	d.Set("minimum_password_length", axconfig.MinimumPasswordLength)
	d.Set("password_expiry_enabled", axconfig.PasswordExpiryEnabled)
	d.Set("os", axconfig.Os)
	d.Set("login_name_regex", axconfig.LoginNameRegex)
	d.Set("default_trial_duration", axconfig.DefaultTrialDuration)
	d.Set("global_fault_handler_policy", axconfig.GlobalFaultHandlerPolicy)
	d.Set("password_lifetime_days", axconfig.PasswordLifetimeDays)
	d.Set("delegate_user_administration", axconfig.DelegateUserAdministration)
	d.Set("portal_hostname", axconfig.PortalHostname)

	d.Set("lock_user_account", flattenLua(&axconfig.LockUserAccount))
}

func flattenLua(lua *client.LockUserAccount) (res []map[string]interface{}) {
	data := make(map[string]interface{})
	data["enabled"] = lua.Enabled
	data["attempts"] = lua.Attempts
	data["lock_time_period"] = lua.LockTimePeriod
	data["lock_time_period_unit"] = lua.LockTimePeriodUnit
	data["time_period"] = lua.TimePeriod
	data["time_period_unit"] = lua.TimePeriodUnit
	res = append(res, data)
	return res
}

// Tedious but much easier to maintain and understand...
func expandConfig(tfconfig *schema.ResourceData, axconfig *client.Config, full bool) (msg string) {
	if full || tfconfig.HasChange("registration_enabled") {
		axconfig.RegistrationEnabled = tfconfig.Get("registration_enabled").(bool)
	}
	if full || tfconfig.HasChange("reg_token_email_enabled") {
		axconfig.RegTokenEmailEnabled = tfconfig.Get("reg_token_email_enabled").(bool)
	}
	if full || tfconfig.HasChange("api_import_timeout") {
		axconfig.ApiImportTimeout = tfconfig.Get("api_import_timeout").(int)
	}
	if full || tfconfig.HasChange("is_trial") {
		axconfig.IsTrial = tfconfig.Get("is_trial").(bool)
	}
	if full || tfconfig.HasChange("promote_api_via_policy") {
		axconfig.PromoteApiViaPolicy = tfconfig.Get("promote_api_via_policy").(bool)
	}
	if full || tfconfig.HasChange("system_o_auth_scopes_enabled") {
		axconfig.SystemOAuthScopesEnabled = tfconfig.Get("system_o_auth_scopes_enabled").(bool)
	}
	if full || tfconfig.HasChange("oadmin_self_service_enabled") {
		axconfig.OadminSelfServiceEnabled = tfconfig.Get("oadmin_self_service_enabled").(bool)
	}
	if full || tfconfig.HasChange("product_version") {
		axconfig.ProductVersion = tfconfig.Get("product_version").(string)
	}
	if full || tfconfig.HasChange("portal_name") {
		axconfig.PortalName = tfconfig.Get("portal_name").(string)
	}
	if full || tfconfig.HasChange("global_response_policy") {
		axconfig.GlobalResponsePolicy = tfconfig.Get("global_response_policy").(string)
	}
	if full || tfconfig.HasChange("auto_approve_applications") {
		axconfig.AutoApproveApplications = tfconfig.Get("auto_approve_applications").(bool)
	}
	if full || tfconfig.HasChange("global_request_policy") {
		axconfig.GlobalRequestPolicy = tfconfig.Get("global_request_policy").(string)
	}
	if full || tfconfig.HasChange("auto_approve_user_registration") {
		axconfig.AutoApproveUserRegistration = tfconfig.Get("auto_approve_user_registration").(bool)
	}
	if full || tfconfig.HasChange("delegate_application_administration") {
		axconfig.DelegateApplicationAdministration = tfconfig.Get("delegate_application_administration").(bool)
	}
	if full || tfconfig.HasChange("api_default_virtual_host") {
		axconfig.ApiDefaultVirtualHost = tfconfig.Get("api_default_virtual_host").(string)
	}
	if full || tfconfig.HasChange("api_routing_key_location") {
		axconfig.ApiRoutingKeyLocation = tfconfig.Get("api_routing_key_location").(string)
	}
	if full || tfconfig.HasChange("application_scope_restrictions") {
		axconfig.ApplicationScopeRestrictions = tfconfig.Get("application_scope_restrictions").(bool)
	}
	if full || tfconfig.HasChange("base_o_auth") {
		axconfig.BaseOAuth = tfconfig.Get("base_o_auth").(bool)
	}
	if full || tfconfig.HasChange("email_bounce_address") {
		axconfig.EmailBounceAddress = tfconfig.Get("email_bounce_address").(string)
	}
	if full || tfconfig.HasChange("advisory_banner_enabled") {
		axconfig.AdvisoryBannerEnabled = tfconfig.Get("advisory_banner_enabled").(bool)
	}
	if full || tfconfig.HasChange("user_name_regex") {
		axconfig.UserNameRegex = tfconfig.Get("user_name_regex").(string)
	}
	if full || tfconfig.HasChange("api_import_mime_validation") {
		axconfig.ApiImportMimeValidation = tfconfig.Get("api_import_mime_validation").(bool)
	}
	if full || tfconfig.HasChange("session_idle_timeout_millis") {
		axconfig.SessionIdleTimeout = tfconfig.Get("session_idle_timeout_millis").(int)
	}
	if full || tfconfig.HasChange("is_api_portal_configured") {
		axconfig.IsApiPortalConfigured = tfconfig.Get("is_api_portal_configured").(bool)
	}
	if full || tfconfig.HasChange("change_password_on_first_login") {
		axconfig.ChangePasswordOnFirstLogin = tfconfig.Get("change_password_on_first_login").(bool)
	}
	if full || tfconfig.HasChange("session_timeout_millis") {
		axconfig.SessionTimeout = tfconfig.Get("session_timeout_millis").(int)
	}
	if full || tfconfig.HasChange("email_from") {
		axconfig.EmailFrom = tfconfig.Get("email_from").(string)
	}
	if full || tfconfig.HasChange("api_routing_key_enabled") {
		axconfig.ApiRoutingKeyEnabled = tfconfig.Get("api_routing_key_enabled").(bool)
	}
	if full || tfconfig.HasChange("login_response_time") {
		axconfig.LoginResponseTime = tfconfig.Get("login_response_time").(int)
	}
	if full || tfconfig.HasChange("server_certificate_verification") {
		axconfig.ServerCertificateVerification = tfconfig.Get("server_certificate_verification").(bool)
	}
	if full || tfconfig.HasChange("reset_password_enabled") {
		axconfig.ResetPasswordEnabled = tfconfig.Get("reset_password_enabled").(bool)
	}
	if full || tfconfig.HasChange("advisory_banner_text") {
		axconfig.AdvisoryBannerText = tfconfig.Get("advisory_banner_text").(string)
	}
	if full || tfconfig.HasChange("api_import_editable") {
		axconfig.ApiImportEditable = tfconfig.Get("api_import_editable").(bool)
	}
	if full || tfconfig.HasChange("api_portal_hostname") {
		axconfig.ApiPortalHostname = tfconfig.Get("api_portal_hostname").(string)
	}
	if full || tfconfig.HasChange("api_portal_name") {
		axconfig.ApiPortalName = tfconfig.Get("api_portal_name").(string)
	}
	if full || tfconfig.HasChange("fault_handlers_enabled") {
		axconfig.FaultHandlersEnabled = tfconfig.Get("fault_handlers_enabled").(bool)
	}
	if full || tfconfig.HasChange("architecture") {
		axconfig.Architecture = tfconfig.Get("architecture").(string)
	}
	if full || tfconfig.HasChange("strict_certificate_checking") {
		axconfig.StrictCertificateChecking = tfconfig.Get("strict_certificate_checking").(bool)
	}
	if full || tfconfig.HasChange("global_policies_enabled") {
		axconfig.GlobalPoliciesEnabled = tfconfig.Get("global_policies_enabled").(bool)
	}
	if full || tfconfig.HasChange("minimum_password_length") {
		axconfig.MinimumPasswordLength = tfconfig.Get("minimum_password_length").(int)
	}
	if full || tfconfig.HasChange("password_expiry_enabled") {
		axconfig.PasswordExpiryEnabled = tfconfig.Get("password_expiry_enabled").(bool)
	}
	if full || tfconfig.HasChange("os") {
		axconfig.Os = tfconfig.Get("os").(string)
	}
	if full || tfconfig.HasChange("login_name_regex") {
		axconfig.LoginNameRegex = tfconfig.Get("login_name_regex").(string)
	}
	if full || tfconfig.HasChange("default_trial_duration") {
		axconfig.DefaultTrialDuration = tfconfig.Get("default_trial_duration").(int)
	}
	if full || tfconfig.HasChange("global_fault_handler_policy") {
		axconfig.GlobalFaultHandlerPolicy = tfconfig.Get("global_fault_handler_policy").(string)
	}
	if full || tfconfig.HasChange("password_lifetime_days") {
		axconfig.PasswordLifetimeDays = tfconfig.Get("password_lifetime_days").(int)
	}
	if full || tfconfig.HasChange("delegate_user_administration") {
		axconfig.DelegateUserAdministration = tfconfig.Get("delegate_user_administration").(bool)
	}
	if full || tfconfig.HasChange("portal_hostname") {
		axconfig.PortalHostname = tfconfig.Get("portal_hostname").(string)
	}
	if full || tfconfig.HasChange("lock_user_account.0.enabled") {
		axconfig.LockUserAccount.Enabled = tfconfig.Get("lock_user_account.0.enabled").(bool)
	}
	if full || tfconfig.HasChange("lock_user_account.0.attempts") {
		axconfig.LockUserAccount.Attempts = tfconfig.Get("lock_user_account.0.attempts").(int)
	}
	if full || tfconfig.HasChange("lock_user_account.0.lock_time_period") {
		axconfig.LockUserAccount.LockTimePeriod = tfconfig.Get("lock_user_account.0.lock_time_period").(int)
	}
	if full || tfconfig.HasChange("lock_user_account.0.lock_time_period_unit") {
		msg = msg + "changed lock_user_account.0.lock_time_period_unit"
		axconfig.LockUserAccount.LockTimePeriodUnit = tfconfig.Get("lock_user_account.0.lock_time_period_unit").(string)
	}
	if full || tfconfig.HasChange("lock_user_account.0.time_period") {
		msg = msg + "changed lock_user_account.0.time_period"
		axconfig.LockUserAccount.TimePeriod = tfconfig.Get("lock_user_account.0.time_period").(int)
	}
	if full || tfconfig.HasChange("lock_user_account.0.time_period_unit") {
		msg = msg + "changed lock_user_account.0.time_period_unit"
		axconfig.LockUserAccount.TimePeriodUnit = tfconfig.Get("lock_user_account.0.time_period_unit").(string)
	} else {
		msg = msg + "NOT changeing lock_user_account.0.time_period_unit"
	}
	return msg + fmt.Sprintf("expanded to %v", axconfig)
}
