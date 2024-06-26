terraform {
  required_providers {
    tester = {
      source  = "terraform.local/local/lua"
      version = "0.0.1"
    }
  }
}

output "test_simple" {
	value = provider::tester::exec(file("./main.lua"), tomap({"foo": {"bar": 190}}))
}

provider "tester" {
	lua = file("./lib.lua")
}

output "test" {
	value = provider::tester::echo(tomap({"foo": {"bar": 190}}))
}
