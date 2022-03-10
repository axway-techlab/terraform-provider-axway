package main

import (

	prov "github.com/axway-techlab/terraform-provider-axwayapi/axwayapi"
	sdk "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *sdk.Provider {
			return prov.Provider()
		},
	})
}
