package person

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourcePersonSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"person_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"name": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"address": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"email": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"mobile_number": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}
}

func resourcePerson() *schema.Resource {
	return &schema.Resource{
		Create: resourcePersonCreate,
		Read:   ResourcePersonRead,
		Update: resourcePersonUpdate,
		Delete: resourcePersonDelete,
		Schema: ResourcePersonSchema(),
	}
}

func ResourcePersonRead(d *schema.ResourceData, meta interface{}) error {
	log.Println("ResourcePersonRead")
	client := meta.(*PersonSession)

	var robj interface{}
	id := d.Get("person_id").(string)
	err := client.Get("api/person/"+id, &robj)
	respMap := robj.(map[string]interface{})
	person_id := respMap["person_id"].(float64)
	id = fmt.Sprintf("%.0f", person_id)
	d.SetId(id)
	d.Set("person_id", id)
	return err
}

func resourcePersonCreate(d *schema.ResourceData, meta interface{}) error {
	log.Println("resourcePersonCreate")
	client := meta.(*PersonSession)

	person := make(map[string]string)
	person["name"] = d.Get("name").(string)
	person["address"] = d.Get("address").(string)
	person["email"] = d.Get("email").(string)
	person["mobile_number"] = d.Get("mobile_number").(string)

	var pres interface{}
	err := client.Post("api/person", &person, &pres)
	respMap := pres.(map[string]interface{})
	person_id := respMap["person_id"].(float64)
	id := fmt.Sprintf("%.0f", person_id)
	d.SetId(id)
	d.Set("person_id", id)
	return err
}

func resourcePersonUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Println("resourcePersonUpdate")
	client := meta.(*PersonSession)
	var robj interface{}
	id := d.Get("person_id").(string)
	err := client.Get("api/person/"+id, &robj)
	respMap := robj.(map[string]interface{})

	respMap["name"] = d.Get("name").(string)
	respMap["address"] = d.Get("address").(string)
	respMap["email"] = d.Get("email").(string)
	respMap["mobile_number"] = d.Get("mobile_number").(string)

	var pres interface{}
	err = client.Put("api/person/"+id, &respMap, &pres)
	putRespMap := pres.(map[string]interface{})
	person_id := putRespMap["person_id"].(float64)
	id = fmt.Sprintf("%.0f", person_id)
	d.SetId(id)
	d.Set("person_id", id)
	return err
}

func resourcePersonDelete(d *schema.ResourceData, meta interface{}) error {
	log.Println("resourcePersonDelete")
	id := d.Get("person_id").(string)
	client := meta.(*PersonSession)
	err := client.Delete("api/person/" + id)
	return err
}
