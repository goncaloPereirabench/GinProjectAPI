resource "random_string" "suffix" {
  length  = 6
  special = false
  upper   = false
}

resource "azurerm_resource_group" "main" {
  name     = "rg-${local.resource_name}"
  location = var.location
  tags     = local.common_tags
}

resource "azurerm_log_analytics_workspace" "main" {
  name                = "log-${local.resource_name}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
  tags                = local.common_tags
}

resource "azurerm_application_insights" "main" {
  name                = "appi-${local.resource_name}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  application_type    = "web"
  workspace_id        = azurerm_log_analytics_workspace.main.id
  tags                = local.common_tags
}

resource "azurerm_container_registry" "main" {
  name                = "acr${local.clean_project}${local.clean_env}${local.name_suffix}"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  sku                 = "Basic"
  admin_enabled       = false
  tags                = local.common_tags
}

resource "azurerm_container_app_environment" "main" {
  name                       = "cae-${local.resource_name}"
  location                   = azurerm_resource_group.main.location
  resource_group_name        = azurerm_resource_group.main.name
  log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
  tags                       = local.common_tags
}

resource "azurerm_user_assigned_identity" "api" {
  name                = "id-api-${local.resource_name}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tags                = local.common_tags
}

resource "azurerm_role_assignment" "api_acr_pull" {
  scope                = azurerm_container_registry.main.id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_user_assigned_identity.api.principal_id
}

resource "azurerm_mssql_server" "main" {
  name                         = "sql-${local.resource_name}"
  resource_group_name          = azurerm_resource_group.main.name
  location                     = azurerm_resource_group.main.location
  version                      = "12.0"
  administrator_login          = var.sql_admin_login
  administrator_login_password = var.sql_admin_password
  minimum_tls_version          = "1.2"
  tags                         = local.common_tags
}

resource "azurerm_mssql_database" "main" {
  name      = "GinGrocery"
  server_id = azurerm_mssql_server.main.id
  sku_name  = var.sql_database_sku_name
  tags      = local.common_tags
}

resource "azurerm_mssql_firewall_rule" "allow_azure_services" {
  name             = "AllowAzureServices"
  server_id        = azurerm_mssql_server.main.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "0.0.0.0"
}

resource "azurerm_servicebus_namespace" "main" {
  name                = "sb-${local.resource_name}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "Basic"
  tags                = local.common_tags
}

resource "azurerm_servicebus_queue" "order_events" {
  name         = "order-events"
  namespace_id = azurerm_servicebus_namespace.main.id
}

resource "azurerm_servicebus_namespace_authorization_rule" "api_sender" {
  name         = "api-send"
  namespace_id = azurerm_servicebus_namespace.main.id
  listen       = false
  send         = true
  manage       = false
}

resource "azurerm_servicebus_namespace_authorization_rule" "functions_listener" {
  name         = "functions-listen"
  namespace_id = azurerm_servicebus_namespace.main.id
  listen       = true
  send         = false
  manage       = false
}

resource "azurerm_container_app" "api" {
  name                         = "ca-api-${local.resource_name}"
  container_app_environment_id = azurerm_container_app_environment.main.id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  tags                         = local.common_tags

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.api.id]
  }

  registry {
    server   = azurerm_container_registry.main.login_server
    identity = azurerm_user_assigned_identity.api.id
  }

  secret {
    name  = "jwt-secret"
    value = var.jwt_secret
  }

  secret {
    name  = "database-dsn"
    value = local.database_dsn
  }

  secret {
    name  = "servicebus-send-connection"
    value = azurerm_servicebus_namespace_authorization_rule.api_sender.primary_connection_string
  }

  template {
    min_replicas = var.api_min_replicas
    max_replicas = var.api_max_replicas

    container {
      name   = "api"
      image  = var.api_image
      cpu    = var.api_cpu
      memory = var.api_memory

      env {
        name  = "APP_ENV"
        value = "production"
      }

      env {
        name  = "PORT"
        value = tostring(var.api_target_port)
      }

      env {
        name        = "JWT_SECRET"
        secret_name = "jwt-secret"
      }

      env {
        name        = "DATABASE_DSN"
        secret_name = "database-dsn"
      }

      env {
        name  = "JWT_ACCESS_TTL"
        value = var.jwt_access_ttl
      }

      env {
        name  = "RATE_LIMIT_REQUESTS_PER_MINUTE"
        value = tostring(var.rate_limit_requests_per_minute)
      }

      env {
        name  = "RATE_LIMIT_BURST"
        value = tostring(var.rate_limit_burst)
      }

      env {
        name        = "SERVICEBUS_SEND_CONNECTION"
        secret_name = "servicebus-send-connection"
      }

      env {
        name  = "ORDER_EVENTS_QUEUE_NAME"
        value = azurerm_servicebus_queue.order_events.name
      }

      env {
        name  = "APPLICATIONINSIGHTS_CONNECTION_STRING"
        value = azurerm_application_insights.main.connection_string
      }
    }
  }

  ingress {
    external_enabled = true
    target_port      = var.api_target_port
    transport        = "auto"

    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }

  depends_on = [azurerm_role_assignment.api_acr_pull]
}

resource "azurerm_storage_account" "functions" {
  name                            = "st${local.clean_project}${local.clean_env}${local.name_suffix}"
  resource_group_name             = azurerm_resource_group.main.name
  location                        = azurerm_resource_group.main.location
  account_tier                    = "Standard"
  account_replication_type        = "LRS"
  min_tls_version                 = "TLS1_2"
  allow_nested_items_to_be_public = false
  tags                            = local.common_tags
}

resource "azurerm_storage_container" "functions_package" {
  name                  = "function-packages"
  storage_account_id    = azurerm_storage_account.functions.id
  container_access_type = "private"
}

resource "azurerm_service_plan" "functions" {
  name                = "asp-func-${local.resource_name}"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  os_type             = "Linux"
  sku_name            = "FC1"
  tags                = local.common_tags
}

resource "azurerm_function_app_flex_consumption" "worker" {
  name                = "func-${local.resource_name}"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  service_plan_id     = azurerm_service_plan.functions.id

  storage_container_type      = "blobContainer"
  storage_container_endpoint  = "${azurerm_storage_account.functions.primary_blob_endpoint}${azurerm_storage_container.functions_package.name}"
  storage_authentication_type = "StorageAccountConnectionString"
  storage_access_key          = azurerm_storage_account.functions.primary_access_key

  runtime_name           = var.functions_runtime_name
  runtime_version        = var.functions_runtime_version
  maximum_instance_count = var.functions_max_instances
  instance_memory_in_mb  = var.functions_instance_memory_mb

  app_settings = {
    APP_ENV                       = "production"
    DATABASE_DSN                  = local.database_dsn
    FUNCTIONS_WORKER_RUNTIME      = "custom"
    SERVICEBUS_CONNECTION         = azurerm_servicebus_namespace_authorization_rule.functions_listener.primary_connection_string
    ORDER_EVENTS_QUEUE_NAME       = azurerm_servicebus_queue.order_events.name
    LOW_STOCK_TIMER_SCHEDULE      = var.low_stock_timer_schedule
    ABANDONED_CART_TIMER_SCHEDULE = var.abandoned_cart_timer_schedule
  }

  site_config {
    application_insights_connection_string = azurerm_application_insights.main.connection_string
  }

  tags = local.common_tags
}
