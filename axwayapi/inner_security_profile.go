package axwayapi

import (
	"fmt"
	"strconv"

	client "github.com/axway-techlab/axwayapi_client/axwayapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var TFSecurityProfile = resource(schemaMap{
	"name":       required(_string()),
	"is_default": optional(_bool(), false),
	"device":     inOut(_listMin(1, TFDevice)),
})

func flattenSecurityProfiles(c []client.SecurityProfile) []flattenMap {
	r := make([]flattenMap, len(c))
	for i, a := range c {
		r[i] = flattenMap{
			"name":       a.Name,                    //required(_string())
			"is_default": a.IsDefault,               //required(_bool())
			"device":     flattenDevices(a.Devices), // inOut(_listMin(1, TFDevice)),
		}
	}
	return r
}
func flattenDevices(c []client.Device) []flattenMap {
	r := make([]flattenMap, len(c))
	for i, a := range c {
		r[i] = flattenMap{
			"name":  a.Name,  //required(inOut(_string()))
			"order": a.Order, //required(inOut(_int()))
		}
		switch a.Type {
		case "apiKey":
			r[i]["api_key"] = flattenMap{
				"remove_credentials_on_success": a.Properties["removeCredentialsOnSuccess"] == "true",
				"api_key_field_name":            a.Properties["apiKeyFieldName"],
				"take_from":                     a.Properties["takeFrom"],
			}
		case "awsHeader":
			r[i]["aws_header"] = flattenMap{
				"remove_credentials_on_success": a.Properties["removeCredentialsOnSuccess"] == "true",
			}
		case "awsQuery":
			r[i]["aws_query"] = flattenMap{
				"remove_credentials_on_success": a.Properties["removeCredentialsOnSuccess"] == "true",
				"api_key_field_name":            a.Properties["apiKeyFieldName"],
			}
		case "basic":
			r[i]["basic"] = flattenMap{
				"remove_credentials_on_success": a.Properties["removeCredentialsOnSuccess"] == "true",
				"realm":                         a.Properties["realm"], //required(inOut(_string())),
			}
		case "oauth":
			oauth := flattenMap{
				"remove_credentials_on_success":      a.Properties["removeCredentialsOnSuccess"] == "true",
				"token_store":                        a.Properties["tokenStore"],                     //required(inOut(_string())),                               // "<key type='OAuth2StoresGroup'><id field='name' value='OAuth2 Stores'/><key type='AccessTokenStoreGroup'><id field='name' value='Access Token Stores'/><key type='AccessTokenPersist'><id field='name' value='OAuth Access Token Store'/></key></key></key>",
				"access_token_location":              a.Properties["accessTokenLocation"],            //required(inOut(_string(oneOf("HEADER", "QUERYSTRING")))), // "HEADER",
				"authorization_header_prefix":        a.Properties["authorizationHeaderPrefix"],      //required(inOut(_string())),                               // "Bearer",
				"access_token_location_query_string": a.Properties["accessTokenLocationQueryString"], //optional(inOut(_string())),                               // "",
				"scopes_must_match":                  a.Properties["scopesMustMatch"],                //required(inOut(_string(oneOf("Any", "All")))),            // "Any",
				"scopes":                             a.Properties["scopes"],                         //required(inOut(_string())),                               // "resource.WRITE, resource.READ",
			}
			implicit, _ := strconv.ParseBool(a.Properties["implicitGrantEnabled"].(string))
			if implicit {
				oauth["implicit_grant"] = flattenMap{
					"login_endpoint_url": a.Properties["implicitGrantLoginEndpointUrl"],
					"login_token_name":   a.Properties["implicitGrantLoginTokenName"],
				}
			}
			authCode, _ := strconv.ParseBool(a.Properties["authCodeGrantTypeEnabled"].(string))
			if authCode {
				oauth["implicit_grant"] = flattenMap{
					"request_endpoint_url":      a.Properties["authCodeGrantTypeRequestEndpointUrl"],
					"request_client_id_name":    a.Properties["authCodeGrantTypeRequestClientIdName"],
					"request_secret_name":       a.Properties["authCodeGrantTypeRequestSecretName"],
					"token_endpoint_url":        a.Properties["authCodeGrantTypeTokenEndpointUrl"],
					"token_endpoint_token_name": a.Properties["authCodeGrantTypeTokenEndpointTokenName"],
				}
			}
			clientCred, _ := strconv.ParseBool(a.Properties["clientCredentialsGrantTypeEnabled"].(string))
			if clientCred {
				oauth["implicit_grant"] = flattenMap{
					"token_endpoint_url": a.Properties["clientCredentialsGrantTypeTokenEndpointUrl"],
					"token_name":         a.Properties["clientCredentialsGrantTypeTokenName"],
				}
			}
			r[i]["oauth"] = oauth
		case "twoWaySSL":
			r[i]["two_ways_ssl"] = flattenMap{
				"remove_credentials_on_success": a.Properties["removeCredentialsOnSuccess"] == "true",
				"api_key_field_name":            a.Properties["apiKeyFieldName"],
			}
		case "passThrough":
			r[i]["passthrough"] = flattenMap{
				"remove_credentials_on_success": a.Properties["removeCredentialsOnSuccess"] == "true",
				"subject_id_field_name":         a.Properties["subjectIdFieldName"],
			}
		}
	}
	return r
}

var TFDevice = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name": required(inOut(_string())),
		// "order":        required(inOut(_int())),
		"api_key":      optional(_singleton(TFApiKeyProperties)),      // exactlyOneOfResource(_singleton(TFApiKeyProperties), excl...),
		"aws_header":   optional(_singleton(TFAwsHeaderProperties)),   // exactlyOneOfResource(_singleton(TFAwsHeaderProperties), excl...),
		"aws_query":    optional(_singleton(TFAwsQueryProperties)),    // exactlyOneOfResource(_singleton(TFAwsQueryProperties), excl...),
		"basic":        optional(_singleton(TFBasicProperties)),       // exactlyOneOfResource(_singleton(TFBasicProperties), excl...),
		"oauth":        optional(_singleton(TFOAuthProperties)),       // exactlyOneOfResource(_singleton(TFOAuthProperties), excl...),
		"two_ways_ssl": optional(_singleton(TFTwoWaysSslProperties)),  // exactlyOneOfResource(_singleton(TFTwoWaysSslProperties), excl...),
		"passthrough":  optional(_singleton(TFPassthroughProperties)), // exactlyOneOfResource(_singleton(TFPassthroughProperties), excl...),
	},
}

func expandSecurityProfiles(v interface{}) []client.SecurityProfile {
	c := v.([]interface{})
	r := make([]client.SecurityProfile, len(c))
	for i, b := range c {
		a := b.(map[string]interface{})
		r[i].Name = a["name"].(string)            //required(_string())
		r[i].IsDefault = a["is_default"].(bool)   //required(_bool())
		r[i].Devices = expandDevices(a["device"]) // inOut(_listMin(1, TFDevice)),
	}
	return r
}
func expandDevices(v interface{}) []client.Device {
	c := v.([]interface{})
	r := make([]client.Device, len(c))
	for i, b := range c {
		a := b.(map[string]interface{})
		r[i].Name = a["name"].(string) //required(inOut(_string()))
		r[i].Order = i                 //required(inOut(_int()))
		nb := 0
		var params map[string]interface{}
		if v, ok := a["api_key"]; ok && len(v.([]interface{})) > 0 {
			nb = nb + 1
			params = v.([]interface{})[0].(map[string]interface{})
			r[i].Type = "apiKey"
			r[i].Properties = flattenMap{
				"removeCredentialsOnSuccess": params["remove_credentials_on_success"],
				"apiKeyFieldName":            params["api_key_field_name"],
				"takeFrom":                   params["take_from"],
			}
		}
		if v, ok := a["aws_header"]; ok && len(v.([]interface{})) > 0 {
			nb = nb + 1
			params = v.([]interface{})[0].(map[string]interface{})
			r[i].Type = "awsHeader"
			r[i].Properties = flattenMap{
				// No params for this type
			}
		}
		if v, ok := a["aws_query"]; ok && len(v.([]interface{})) > 0 {
			nb = nb + 1
			params = v.([]interface{})[0].(map[string]interface{})
			r[i].Type = "awsQuery"
			r[i].Properties = flattenMap{
				"apiKeyFieldName": params["api_key_field_name"],
			}
		}
		if v, ok := a["basic"]; ok && len(v.([]interface{})) > 0 {
			nb = nb + 1
			params = v.([]interface{})[0].(map[string]interface{})
			r[i].Type = "basic"
			r[i].Properties = flattenMap{
				"realm": params["realm"],
			}
		}
		if v, ok := a["two_ways_ssl"]; ok && len(v.([]interface{})) > 0 {
			nb = nb + 1
			params = v.([]interface{})[0].(map[string]interface{})
			r[i].Type = "twoWaySSL"
			r[i].Properties = flattenMap{
				"apiKeyFieldName": params["api_key_field_name"],
			}
		}
		if v, ok := a["passthrough"]; ok && len(v.([]interface{})) > 0 {
			nb = nb + 1
			params = v.([]interface{})[0].(map[string]interface{})
			r[i].Type = "passThrough"
			r[i].Properties = flattenMap{
				"subjectIdFieldName": params["subject_id_field_name"],
			}
		}
		if v, ok := a["oauth"]; ok && len(v.([]interface{})) > 0 {
			nb = nb + 1
			params = v.([]interface{})[0].(map[string]interface{})
			r[i].Type = "oauth"
			props := flattenMap{
				"tokenStore":                     params["token_store"],                        //"<key type='OAuth2StoresGroup'><id field='name' value='OAuth2 Stores'/><key type='AccessTokenStoreGroup'><id field='name' value='Access Token Stores'/><key type='AccessTokenPersist'><id field='name' value='OAuth Access Token Store'/></key></key></key>",
				"accessTokenLocation":            params["access_token_location"],              //"QUERYSTRING",
				"authorizationHeaderPrefix":      params["authorization_header_prefix"],        //"Bearer",
				"accessTokenLocationQueryString": params["access_token_location_query_string"], //"aaa",
				"scopesMustMatch":                params["scopes_must_match"],                  //"Any",
				"scopes":                         params["scopes"],                             //"resource.WRITE, resource.READ",
			}
			if grant, has := params["implicit_grant"]; has && len(grant.([]interface{})) > 0 {
				g := grant.([]interface{})[0].(map[string]interface{})
				props = merge(props, flattenMap{
					"implicitGrantEnabled":          "true",
					"implicitGrantLoginEndpointUrl": g["login_endpoint_url"],
					"implicitGrantLoginTokenName":   g["login_token_name"],
				})
			} else {
				props = merge(props, flattenMap{
					"implicitGrantEnabled": "false",
				})
			}
			if grant, has := params["auth_code_grant"]; has && len(grant.([]interface{})) > 0 {
				g := grant.([]interface{})[0].(map[string]interface{})
				props = merge(props, flattenMap{
					"authCodeGrantTypeEnabled":                "true",
					"authCodeGrantTypeRequestEndpointUrl":     g["request_endpoint_url"],
					"authCodeGrantTypeRequestClientIdName":    g["request_client_id_name"],
					"authCodeGrantTypeRequestSecretName":      g["request_secret_name"],
					"authCodeGrantTypeTokenEndpointUrl":       g["token_endpoint_url"],
					"authCodeGrantTypeTokenEndpointTokenName": g["token_endpoint_token_name"],
				})
			} else {
				props = merge(props, flattenMap{
					"authCodeGrantTypeEnabled": "false",
				})
			}
			if grant, has := params["client_credentials_grant"]; has && len(grant.([]interface{})) > 0 {
				g := grant.([]interface{})[0].(map[string]interface{})
				props = merge(props, flattenMap{
					"clientCredentialsGrantTypeEnabled":          "true",
					"clientCredentialsGrantTypeTokenEndpointUrl": g["token_endpoint_url"],
					"clientCredentialsGrantTypeTokenName":        g["token_name"],
				})
			} else {
				props = merge(props, flattenMap{
					"clientCredentialsGrantTypeEnabled": "false",
				})
			}
			r[i].Properties = props
		}
		if nb != 1 {
			panic(fmt.Errorf("exactly one of 'api_key', 'aws_header', 'aws_query', 'basic', 'oauth', 'two_ways_ssl', 'passthrough' can be defined for a device, found %d here", nb))
		}
		r[i].Properties["removeCredentialsOnSuccess"] = params["remove_credentials_on_success"]
	}
	return r
}

func merge(maps ...flattenMap) flattenMap {
	r := flattenMap{}
	for _, m := range maps {
		for k, v := range m {
			r[k] = v
		}
	}
	return r
}

var TFApiKeyProperties = resource(schemaMap{
	"remove_credentials_on_success": optional(inOut(_bool()), true),
	"api_key_field_name":            required(inOut(_string())),
	"take_from":                     required(inOut(_string(oneOf("HEADER", "QUERY")))),
})

var TFAwsHeaderProperties = resource(schemaMap{
	"remove_credentials_on_success": optional(inOut(_bool()), true),
})

var TFAwsQueryProperties = resource(schemaMap{
	"remove_credentials_on_success": optional(inOut(_bool()), true),
	"api_key_field_name":            required(inOut(_string())),
})

var TFBasicProperties = resource(schemaMap{
	"remove_credentials_on_success": optional(inOut(_bool()), true),
	"realm":                         required(inOut(_string())),
})

var TFTwoWaysSslProperties = resource(schemaMap{
	"remove_credentials_on_success": optional(inOut(_bool()), true),
	"api_key_field_name":            required(inOut(_string())),
})

var TFPassthroughProperties = resource(schemaMap{
	"remove_credentials_on_success": optional(inOut(_bool()), true),
	"subject_id_field_name":         required(inOut(_string())),
})

var TFOAuthProperties = resource(schemaMap{
	"remove_credentials_on_success":      optional(inOut(_bool()), true),
	"token_store":                        required(inOut(_string())),                               // "<key type='OAuth2StoresGroup'><id field='name' value='OAuth2 Stores'/><key type='AccessTokenStoreGroup'><id field='name' value='Access Token Stores'/><key type='AccessTokenPersist'><id field='name' value='OAuth Access Token Store'/></key></key></key>",
	"access_token_location":              required(inOut(_string(oneOf("HEADER", "QUERYSTRING")))), // "HEADER",
	"authorization_header_prefix":        required(inOut(_string())),                               // "Bearer",
	"access_token_location_query_string": optional(inOut(_string())),                               // "",
	"scopes_must_match":                  required(inOut(_string(oneOf("Any", "All")))),            // "Any",
	"scopes":                             required(inOut(_string())),                               // "resource.WRITE, resource.READ",
	"implicit_grant":                     optional(_singleton(TFOAuthImplicitGrant)),               //atLeastOneOfResource(optional(_singleton(TFOAuthImplicitGrant)), oauthGrantTypes...),
	"auth_code_grant":                    optional(_singleton(TFOAuthCodeGrant)),                   //atLeastOneOfResource(optional(_singleton(TFOAuthCodeGrant)), oauthGrantTypes...),
	"client_credentials_grant":           optional(_singleton(TFOAuthClientCredentialsGrant)),      //atLeastOneOfResource(optional(_singleton(TFOAuthClientCredentialsGrant)), oauthGrantTypes...),
})

var TFOAuthImplicitGrant = resource(schemaMap{
	"login_endpoint_url": optional(inOut(_string())), // "https://localhost:8089/api/oauth/authorize",
	"login_token_name":   optional(inOut(_string())), // "access_token",
})
var TFOAuthCodeGrant = resource(schemaMap{
	"request_endpoint_url":      required(inOut(_string())), // "https://localhost:8089/api/oauth/authorize",
	"request_client_id_name":    required(inOut(_string())), // "client_id",
	"request_secret_name":       required(inOut(_string())), // "client_secret",
	"token_endpoint_url":        required(inOut(_string())), // "https://localhost:8089/api/oauth/token",
	"token_endpoint_token_name": required(inOut(_string())), // "access_code",
})
var TFOAuthClientCredentialsGrant = resource(schemaMap{
	"token_endpoint_url": required(inOut(_string())), // "https://localhost:8089/api/oauth/token",
	"token_name":         required(inOut(_string())), // "access_token",
})
