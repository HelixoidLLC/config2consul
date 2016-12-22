# Description

_config2consul_ is the one of the tools used in implementation of the "Immutable Configuration" as part of "Immutable Infrastructure" concept.
It is used to "seed" Consul with the configuration from sources like source control and to ensure that there are no
deviations from such configuration.

**Important to understand** the fact that _config2consul_ converges all the rules. This means that it'll ensure that
the state of Consul is exactly matching the rules. If the configuration is not present in Consul, it'll be created
and if the configuration is present in Consul but not in the rules, the setting will be removed and a WARNING will be raised.

The deviations that are hapenning in the configuration are either the natural lifecycle of the system
(ex: deprecation of a setting) or an indentification of a **security breach** (ex: unexpected ACL found).
_config2consul_ is designed to identify such "deviations" and raise a **warning** in such case so the security
monitoring can react to these events.


## Getting started

Converge rules:
```
#> config2consul -config config/config.json rules
```

```
Usage of ./bin/mac/vault_ssh:
  -config string
    	path to the config file (default "./config.json")
  -log.level value
    	Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal, panic].
  -version
    	prints current version
```

### Example of a config file

```
{
  "backend": "consul",
  "scheme": "https",
  "address": "172.20.0.11:8501",
  "token": "56847557-1c68-472c-9d70-ca906be0d288",
  "ca_file": "secrets/ca.crt",
  "cert_file": "secrets/consul_client.crt",
  "key_file": "secrets/consul_client.key",

  "preserve_master_token": true,
  "preserve_vault_acls": true
}
```

### Example of rules

_config2consul_ will load all the files from "rules" directory and will execute all of the policies wihout any particular order

```
---
policies:
  - name: Anonymous Token
    # Hello
    rules: |
      # Deny all access
      key "" {
        policy = "deny"
      }

      # Allow DHCP and REST resolution only
      service "" {
        policy = "read"
      }

      # Deny all access
      event "" {
        policy = "deny"
      }

      # Deny all access
      query "" {
        policy = "deny"
      }

      # Deny all access
      keyring = "deny"
```

## Running tests (on Mac)

1. Launch a Dev docker container
1. Update the Makefile to point to the right Docker instance
1. Generate SSL certificates if needed (requires 'terraform' to be installed)
```
#> cd secrets
#> terraform apply
```
1. Run integration tests
```
make integration
```
