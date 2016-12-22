variable "out_path" {
  default = "out"
}

//-------------------------------------------------------------------
// Organization settings
//-------------------------------------------------------------------

variable "organization_name" {
  default = "My Org"
  type = "string"
  description = "Org Name"
}

variable "organization_unit" {
  default = "DevOps"
  type = "string"
  description = "Org Unit"
}

variable "organization_street" {
  default = "1 Some Str"
  type = "string"
  description = "Office Street Name"
}

variable "organization_locality" {
  default = "Boston"
  type = "string"
  description = "Office City"
}

variable "organization_province" {
  default = "MA"
  type = "string"
  description = "Office State"
}

variable "organization_country" {
  default = "US"
  type = "string"
  description = "Office Country"
}

variable "organization_zip" {
  default = "01234"
  type = "string"
  description = "Office Zip Code"
}

//-------------------------------------------------------------------
// Consul settings
//-------------------------------------------------------------------

variable "consul_instance_count" {
  default = "3"
  type = "string"
  description = "Consul # of instances"
}

variable "consul_data_center" {
  default = {
    "0" = "east-aws"
  }
  type = "map"
  description = "Consul data center"
}

variable "consul_domain" {
  default = "consul"
  type = "string"
  description = "Consul domain"
}
