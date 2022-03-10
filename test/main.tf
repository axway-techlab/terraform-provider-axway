terraform {
  required_providers {
    axwayapi = {
      source  = "axway.com/axway-techlabs/axwayapi"
      version = "0.0.1"
    }
  }
  required_version = ">= 0.14"
}

locals {
  image_path = "./cat.jpg"
}

provider "axwayapi" {
  proxy               = "http://localhost:8080/"
  host                = "https://manager.testing.axway-techlabs.com/api/portal/v1.4"
  username            = "apiadmin"
  password            = "changeme"
  skip_tls_cert_verif = true
}

resource "axwayapi_config" "config" {
  portal_name                    = "API Manager"
  api_import_editable            = true
  change_password_on_first_login = false
  session_idle_timeout_millis    = 86400000 // 5 days
  session_timeout_millis         = 86400000 // 5 days
  api_default_virtual_host       = "traffic.testing.axway-techlabs.com"
  lock_user_account {
    enabled               = true
    attempts              = 10
    time_period_unit      = "second"
    time_period           = 30
    lock_time_period      = 5
    lock_time_period_unit = "minute"
  }
  system_default_quota {
    restriction {
      api_id = axwayapi_frontend.simplefe.id
      limit  = "2000 MB per seconds"
    }
    restriction {
      api_id = axwayapi_frontend.simplefe2.id
      limit  = "1800 MB per seconds"
    }
  }
  application_default_quota {
//    restriction {
//      api_id = axwayapi_frontend.simplefe.id
//      limit  = "13000 MB per seconds"
//    }
//    restriction {
//      api_id = axwayapi_frontend.simplefe2.id
//      limit  = "4000 MB per seconds"
//    }
  }
}

resource "axwayapi_organization" "fff" {
  name        = "fff-fff"
  development = true
  enabled     = true
  image_jpg   = filebase64(local.image_path)
}

resource "axwayapi_organization" "hhh" {
  name        = "other"
  development = true
  enabled     = true
  image_jpg   = filebase64(local.image_path)
}

resource "axwayapi_user" "bob" {
  name       = "bob smithers"
  image_jpg  = filebase64(local.image_path)
  enabled    = true
  login_name = "bob"
  email      = "bob@sponge.com"
  password   = "Password123#"
  main_role {
    org_id = axwayapi_organization.fff.id
    role   = "user"
  }
  additional_roles = {
    //    (axwayapi_organization.ggg.id) = "oadmin"
    (axwayapi_organization.hhh.id) = "user"
  }
}

resource "axwayapi_backend" "petstore" {
  name    = "petstore"
  summary = "summary"
  org_id  = axwayapi_organization.fff.id
  swagger = file("./swagger.json")
}

resource "axwayapi_frontend" "simplefe" {
  name      = "tatame"
  org_id    = axwayapi_organization.fff.id
  api_id    = axwayapi_backend.petstore.id
  image_jpg = filebase64(local.image_path)
  path      = "/v1"
  state     = "published"
  inbound_profile {
    name             = "_default"
    security_profile = "_default"
  }
  security_profile {
    name       = "_default"
    is_default = true
    devices {
      name  = "passthrough"
      type  = "passThrough"
      order = 1
      properties = {
        "subjectIdFieldName" = "eee"
      }
    }
  }
  authentication_profile {
    is_default = true
    name       = "_default"
    type       = "none"
  }
  tag {
    name   = "mytag"
    values = ["myvalue", "my second value", "my third value"]
  }
  tag {
    name   = "my other tag"
    values = ["my other value"]
  }
}

resource "axwayapi_backend" "petstore2" {
  name    = "petstore2"
  summary = "summary"
  org_id  = axwayapi_organization.fff.id
  swagger = file("./swagger.json")
}

resource "axwayapi_frontend" "simplefe2" {
  name               = "tatatata"
  org_id             = axwayapi_organization.fff.id
  api_id             = axwayapi_backend.petstore2.id
  description_type   = "manual"
  description_manual = <<-EOD
  ## description overriden
  EOD

  path      = "/v2"
  image_jpg = filebase64(local.image_path)
  state     = "published"
  inbound_profile {
    name             = "_default"
    security_profile = "_default"
  }
  security_profile {
    name       = "_default"
    is_default = true
    devices {
      name  = "passthrough"
      type  = "passThrough"
      order = 1
      properties = {
        "subjectIdFieldName" = "eee"
      }
    }
  }
  authentication_profile {
    is_default = true
    name       = "_default"
    type       = "none"
  }
}

resource "axwayapi_application" "myApp" {
  name        = "my app"
  description = "test application"
  org_id      = axwayapi_organization.fff.id
  enabled     = true
  image_jpg   = filebase64(local.image_path)
  apis = [
    axwayapi_frontend.simplefe2.id,
    axwayapi_frontend.simplefe.id,
  ]
  apikey {
    id           = "2dca1e5f-d417-4b11-9745-d110c15c6095"
    enabled      = true
    cors_origins = ["*"]
  }
  apikey {
    id           = "4720a6e5-1a09-4244-87c8-6dc350f1e01f"
    cors_origins = ["*"]
  }
//  quota {
//    restriction {
//      api_id = axwayapi_frontend.simplefe.id
//      limit  = "500 MB per 15 seconds"
//    }
//    restriction {
//      api_id = axwayapi_frontend.simplefe2.id
//      limit  = "150 MB per 130 seconds"
//    }
//    restriction {
//      api_id = axwayapi_frontend.simplefe2.id
//      limit  = "200 MB per 130 seconds"
//    }
//  }
}
