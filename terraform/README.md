# Terraform Azure Infrastructure

This folder provisions a pragmatic Azure setup for the Gin grocery API:

- Azure Container Apps for the main HTTP API.
- Azure Container Registry for the API image.
- Azure SQL Database for users, products, and carts.
- Azure Service Bus for asynchronous events.
- Azure Functions Flex Consumption for scheduled and event-driven background work.
- Log Analytics and Application Insights for observability.

## Why Functions Are Included

The Gin API should keep the request/response work:

- login and token issuing
- product reads/writes
- cart reads/writes
- request validation and API rate limiting

Azure Functions are a better fit for work that is not part of the user waiting for a response:

- scheduled low-stock checks
- scheduled abandoned-cart cleanup
- queue-triggered order receipt generation
- queue-triggered email or external inventory sync
- blob-triggered product import later

This Terraform creates the Function App and the Service Bus queue, but the actual function code would be deployed later as a separate package.

## Files

```text
versions.tf              provider versions
variables.tf             configurable inputs
locals.tf                shared names and computed values
main.tf                  Azure resources
outputs.tf               useful deployment outputs
terraform.tfvars.example example local variables
```

## Run

Use Azure Cloud Shell if your company laptop cannot install Terraform or Docker.

```powershell
cd terraform
Copy-Item terraform.tfvars.example terraform.tfvars
terraform init
terraform plan -out main.tfplan
terraform apply main.tfplan
```

The first apply can use the placeholder public container image. After Azure Container Registry exists, build the real API image in Azure:

```powershell
$acr = terraform output -raw acr_name
az acr build --registry $acr --image gin-grocery-api:v1 ..
```

Then update `terraform.tfvars`:

```hcl
api_image       = "<acr_login_server>/gin-grocery-api:v1"
api_target_port = 8080
```

Apply again:

```powershell
terraform plan -out main.tfplan
terraform apply main.tfplan
```

## Database Migration

Terraform creates the Azure SQL database. Apply the schema from the repo root:

```powershell
sqlcmd -S <sql_server_fqdn> -d GinGrocery -U groceryadmin -P "<password>" -i ..\migrations\001_init.sql
```

You can also run the migration from Azure Data Studio or SQL Server Management Studio.

## Function Runtime

The Function App uses Flex Consumption with `runtime_name = "custom"` and `runtime_version = "1.0"`, which is the Go custom-handler path. Before production, verify runtime availability in your selected Azure region:

```powershell
Get-AzFunctionAppFlexConsumptionRuntime -Location "westeurope" -Runtime custom
```

or:

```powershell
az functionapp list-flexconsumption-runtimes --location westeurope --runtime custom
```

## Security Notes

This first version keeps deployment simple. For production hardening:

- move Terraform state to an Azure Storage backend
- use Key Vault references for secrets
- replace SQL administrator credentials with managed identity access
- use private networking/private endpoints for Azure SQL
- use a distributed API rate limiter, such as Redis, when API replicas scale out

Terraform variables marked as sensitive still end up in Terraform state. Store state somewhere access-controlled.
