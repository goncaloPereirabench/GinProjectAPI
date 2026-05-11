variable "project_name" {
  type        = string
  description = "Short project name used in Azure resource names."
  default     = "gin-grocery"
}

variable "environment" {
  type        = string
  description = "Deployment environment name."
  default     = "dev"
}

variable "location" {
  type        = string
  description = "Azure region."
  default     = "westeurope"
}

variable "tags" {
  type        = map(string)
  description = "Extra tags to apply to resources."
  default     = {}
}

variable "api_image" {
  type        = string
  description = "Container image for the Gin API. Use the ACR login server output after building the image."
  default     = "mcr.microsoft.com/azuredocs/containerapps-helloworld:latest"
}

variable "api_target_port" {
  type        = number
  description = "Container port exposed by the API image. Use 8080 for this Go API."
  default     = 8080
}

variable "api_cpu" {
  type        = number
  description = "CPU allocated to the API container."
  default     = 0.5
}

variable "api_memory" {
  type        = string
  description = "Memory allocated to the API container."
  default     = "1Gi"
}

variable "api_min_replicas" {
  type        = number
  description = "Minimum API replicas. Use 0 to reduce idle cost when cold starts are acceptable."
  default     = 0
}

variable "api_max_replicas" {
  type        = number
  description = "Maximum API replicas."
  default     = 3
}

variable "jwt_secret" {
  type        = string
  description = "JWT signing secret used by the API."
  sensitive   = true
}

variable "jwt_access_ttl" {
  type        = string
  description = "JWT access-token lifetime."
  default     = "15m"
}

variable "rate_limit_requests_per_minute" {
  type        = number
  description = "Per-user/IP request limit applied by the API."
  default     = 60
}

variable "rate_limit_burst" {
  type        = number
  description = "Token-bucket burst size for request limiting."
  default     = 20
}

variable "sql_admin_login" {
  type        = string
  description = "Azure SQL administrator login."
  default     = "groceryadmin"
}

variable "sql_admin_password" {
  type        = string
  description = "Azure SQL administrator password."
  sensitive   = true
}

variable "sql_database_sku_name" {
  type        = string
  description = "Azure SQL database SKU."
  default     = "Basic"
}

variable "functions_runtime_name" {
  type        = string
  description = "Azure Functions Flex runtime. Use custom for Go custom handlers."
  default     = "custom"
}

variable "functions_runtime_version" {
  type        = string
  description = "Azure Functions Flex runtime version for custom handlers."
  default     = "1.0"
}

variable "functions_instance_memory_mb" {
  type        = number
  description = "Memory per Azure Functions Flex instance."
  default     = 2048
}

variable "functions_max_instances" {
  type        = number
  description = "Maximum Azure Functions Flex instances."
  default     = 10
}

variable "low_stock_timer_schedule" {
  type        = string
  description = "CRON schedule for a future low-stock timer function."
  default     = "0 0 6 * * *"
}

variable "abandoned_cart_timer_schedule" {
  type        = string
  description = "CRON schedule for a future abandoned-cart cleanup function."
  default     = "0 0 3 * * *"
}
