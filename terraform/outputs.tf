output "resource_group_name" {
  description = "Azure resource group."
  value       = azurerm_resource_group.main.name
}

output "acr_name" {
  description = "Azure Container Registry name."
  value       = azurerm_container_registry.main.name
}

output "acr_login_server" {
  description = "Azure Container Registry login server."
  value       = azurerm_container_registry.main.login_server
}

output "api_url" {
  description = "Public URL for the Container App API."
  value       = "https://${azurerm_container_app.api.latest_revision_fqdn}"
}

output "sql_server_fqdn" {
  description = "Azure SQL Server FQDN."
  value       = azurerm_mssql_server.main.fully_qualified_domain_name
}

output "sql_database_name" {
  description = "Azure SQL database name."
  value       = azurerm_mssql_database.main.name
}

output "function_app_name" {
  description = "Azure Functions app for scheduled and event-driven work."
  value       = azurerm_function_app_flex_consumption.worker.name
}

output "function_app_url" {
  description = "Azure Functions app URL."
  value       = "https://${azurerm_function_app_flex_consumption.worker.name}.azurewebsites.net"
}

output "servicebus_namespace_name" {
  description = "Service Bus namespace used for asynchronous jobs."
  value       = azurerm_servicebus_namespace.main.name
}

output "order_events_queue_name" {
  description = "Service Bus queue intended for order-related async jobs."
  value       = azurerm_servicebus_queue.order_events.name
}
