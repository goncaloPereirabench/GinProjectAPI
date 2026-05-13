locals {
  clean_project = substr(replace(lower(var.project_name), "/[^a-z0-9]/", ""), 0, 12)
  clean_env     = substr(replace(lower(var.environment), "/[^a-z0-9]/", ""), 0, 5)
  name_suffix   = random_string.suffix.result

  resource_name = "${var.project_name}-${var.environment}-${local.name_suffix}"

  common_tags = merge(
    {
      project     = var.project_name
      environment = var.environment
      managed_by  = "terraform"
    },
    var.tags
  )

  database_dsn = "sqlserver://${var.sql_admin_login}:${urlencode(var.sql_admin_password)}@${azurerm_mssql_server.main.fully_qualified_domain_name}:1433?database=${azurerm_mssql_database.main.name}&encrypt=true&guid+conversion=true"
}
