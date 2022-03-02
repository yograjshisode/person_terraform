package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/persontest/person_terraform/person"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: person.Provider})
}
