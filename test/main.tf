terraform {
  required_providers {
    axwayapi = {
      source  = "axway-techlab/axwayapi"
      version = "0.1.1"
    }
  }
  required_version = ">= 0.14"
}

locals {
  image_path = "./cat.jpg"
}

provider "axwayapi" {
  proxy               = "http://localhost:8080/"
  host                = "https://manager.debug.axway-techlabs.com/api/portal/v1.4"
  username            = "apiadmin"
  password            = "changeme"
  skip_tls_cert_verif = true
}

resource "axwayapi_config" "config" {
  portal_name                    = "API Manager"
  api_import_editable            = true
  change_password_on_first_login = false
  password_expiry_enabled        = false
  session_idle_timeout_millis    = 86400000 // 5 days
  session_timeout_millis         = 86400000 // 5 days
  api_default_virtual_host       = "traffic.testing.axway-techlabs.com"
  lock_user_account {
    enabled               = false
    attempts              = 13
    time_period_unit      = "second"
    time_period           = 34
    lock_time_period      = 5
    lock_time_period_unit = "minute"
  }
}

##resource "axwayapi_organization" "fff" {
##  name        = "todelete"
##  development = true
##  enabled     = true
##  image_jpg   = filebase64(local.image_path)
##}
##
##resource "axwayapi_backend" "petstore" {
##  name    = "todelete"
##  summary = "summary"
##  org_id  = axwayapi_organization.fff.id
##  swagger = file("./swagger.json")
##}
##
##resource "axwayapi_frontend" "todelete" {
##  name      = "todelete"
##  org_id    = axwayapi_organization.fff.id
##  api_id    = axwayapi_backend.petstore.id
##  image_jpg = filebase64(local.image_path)
##  state     = "published"
##  inbound_profile {
##    name             = "_default"
##    security_profile = "a"
##  }
##  security_profile {
##    name       = "_default"
##    is_default = true
##  }
##  security_profile {
##    name       = "a"
##    device {
##      name = "passthrough"
##      passthrough { subject_id_field_name         = "first" }
##    }
##    device {
##      name = "basic"
##      basic { realm = "realm"}
##    }
##    device {
##      name = "apikey"
##      api_key {
##        take_from          = "HEADER"
##        api_key_field_name = "field_name"
##      }
##    }
##  }
##  security_profile {
##    name       = "b"
##    device {
##      name = "aws_header"
##      aws_header {}
##    }
##    device {
##      name = "apikey2"
##      oauth {
##        token_store          = "store"
##        access_token_location = "HEADER"
##        authorization_header_prefix = "Bearer"
##        access_token_location_query_string = "query"
##        scopes_must_match = "All"
##        scopes = "resource.WRITE, resource.READ"
##        implicit_grant {
##          login_endpoint_url = "http://localhost:9999"
##          login_token_name = "access_token"
##        }
##      }
##    }
##    device {
##      name = "aws query"
##      aws_query { api_key_field_name = "query" }
##    }
##  }
##  authentication_profile {
##    is_default = true
##    name       = "_default"
##    type       = "none"
##  }
##  tag {
##    name   = "mytag"
##    values = ["myvalue", "my second value", "my third value"]
##  }
##  tag {
##    name   = "my other tag"
##    values = ["my other value"]
##  }
##}

//resource "axwayapi_backend" "petstore2" {
//  name    = "petstore2"
//  summary = "summary"
//  org_id  = axwayapi_organization.fff.id
//  swagger = file("./swagger.json")
//}
//
//resource "axwayapi_frontend" "simplefe2" {
//  name               = "tatatata"
//  org_id             = axwayapi_organization.fff.id
//  api_id             = axwayapi_backend.petstore2.id
//  description_type   = "manual"
//  description_manual = <<-EOD
//  ## description overriden
//  EOD
//
//  path      = "/v2"
//  image_jpg = filebase64(local.image_path)
//  state     = "published"
//  inbound_profile {
//    name             = "_default"
//    security_profile = "_default"
//  }
//  security_profile {
//    name       = "_default"
//    is_default = true
//    devices {
//      name  = "passthrough"
//      type  = "passThrough"
//      order = 1
//      basic {
//        subject_id_field_name = "eee"
//      }
//    }
//  }
//  authentication_profile {
//    is_default = true
//    name       = "_default"
//    type       = "none"
//  }
//}
//
//resource "axwayapi_application" "myApp" {
//  name        = "my app"
//  description = "test application"
//  org_id      = axwayapi_organization.fff.id
//  enabled     = true
//  image_jpg   = filebase64(local.image_path)
//  apis = [
//    axwayapi_frontend.simplefe2.id,
//    axwayapi_frontend.simplefe.id,
//  ]
//  apikey {
//    id           = "2dca1e5f-d417-4b11-9745-d110c15c6095"
//    enabled      = true
//    cors_origins = ["*"]
//  }
//  apikey {
//    id           = "4720a6e5-1a09-4244-87c8-6dc350f1e01f"
//    cors_origins = ["*"]
//  }
////  quota {
////    restriction {
////      api_id = axwayapi_frontend.simplefe.id
////      limit  = "500 MB per 15 seconds"
////    }
////    restriction {
////      api_id = axwayapi_frontend.simplefe2.id
////      limit  = "150 MB per 130 seconds"
////    }
////    restriction {
////      api_id = axwayapi_frontend.simplefe2.id
////      limit  = "200 MB per 130 seconds"
////    }
////  }
//}
//
