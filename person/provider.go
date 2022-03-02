// Copyright 2019 VMware, Inc.

package person

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"person_service_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Person API service url.",
			},
			"person_service_port": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Person API service port.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"person_person": resourcePerson(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	person_service_url := d.Get("person_service_url").(string)
	person_service_port := d.Get("person_service_port").(string)
	personsess, err := NewPersonSession(person_service_url + ":" + person_service_port)
	log.Printf("Person session created for service url %s and port %s\n", person_service_url, person_service_port)
	return personsess, err
}
