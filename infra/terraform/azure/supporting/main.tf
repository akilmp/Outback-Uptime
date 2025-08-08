terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

provider "azurerm" {
  features {}
}

resource "azurerm_network_security_group" "nsg" {
  name                = var.name
  location            = var.location
  resource_group_name = var.resource_group
}

variable "name" {}
variable "location" {}
variable "resource_group" {}

output "nsg_id" {
  value = azurerm_network_security_group.nsg.id
}
