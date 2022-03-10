package axwayapi

import (
	"context"
	"net/url"

	"github.com/axway-techlab/axwayapi_client/axwayapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider -
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AXWAYAPI_HOST", nil),
			},
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AXWAYAPI_USERNAME", nil),
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("AXWAYAPI_PASSWORD", nil),
			},
			"proxy": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("AXWAYAPI_PROXY", nil),
			},
			"skip_tls_cert_verif": {
				Type:        schema.TypeBool,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("AXWAYAPI_SKIP_TLS_CERT_VERIF", false),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"axwayapi_config":       resourceConfig(),
			"axwayapi_organization": resourceOrganization(),
			"axwayapi_user":         resourceUser(),
			"axwayapi_backend":      resourceBackend(),
			"axwayapi_frontend":     resourceFrontend(),
			"axwayapi_application":  resourceApplication(),
		},
		DataSourcesMap:       map[string]*schema.Resource{},
		ConfigureContextFunc: providerConfigure,
	}
}

type ProviderState struct {
	Client *axwayapi.Client
	Cache  map[string]interface{}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	username := d.Get("username").(string)
	password := d.Get("password").(string)
	p, ok := d.GetOk("proxy")
	var proxy *url.URL
	var err error
	if ok {
		proxy, err = url.Parse(p.(string))
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to create axwayapi client",
				Detail:   "When a proxy is given, it should be a valid URL.",
			})
		}
	}
	skipTlsCertVerif := d.Get("skip_tls_cert_verif").(bool)

	host := d.Get("host").(string)

	if (username != "") && (password != "") {
		c, err := axwayapi.NewClient(host, username, password, proxy, skipTlsCertVerif)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to create axwayapi client",
				Detail:   "Unable to authenticate user for authenticated axwayapi client",
			})

			return nil, diags
		}

		return &ProviderState{c, make(map[string]interface{}, 10)}, diags
	} else {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create axwayapi client",
			Detail:   "missing username and/or password cofiguration",
		})
		return nil, diags
	}
}
