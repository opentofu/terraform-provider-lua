# terraform-provider-lua

This is a OpenTofu and Terraform function provider based on terraform-plugin-go.

It provides an "exec" function which takes a lua program as the first parameter and passes all additional parameters to the function defined in the lua file.

```hcl
locals {
    lua_echo = <<EOT

function echo( input )
    return input
end

return echo

EOT
}

output "example" {
    value = provider::lua::exec(local.lua_echo, {"foo": {"bar": 42}})
}
```

In OpenTofu 1.7.0-beta1 you may configure the provider and pass it a lua library to load.  Any functions exposed in this library
will be available as functions within the tofu configuration.  This feature is an experimental preview and is subject to change
before the OpenTofu 1.7.0 release.

```hcl
provider "lua" {
    lua = file("./lib.lua")
}

output "example" {
    value = provider::lua::echo({"message": "Hello Functions!"})
}

```
