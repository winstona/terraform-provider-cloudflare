package cloudflare

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCloudFlareRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudFlareRecordCreate,
		Read:   resourceCloudFlareRecordRead,
		Update: resourceCloudFlareRecordUpdate,
		Delete: resourceCloudFlareRecordDelete,
		Importer: &schema.ResourceImporter{
			State: resourceCloudFlareRecordImport,
		},

		SchemaVersion: 1,
		MigrateState:  resourceCloudFlareRecordMigrateState,
		Schema: map[string]*schema.Schema{
			"domain": {
				Type:     schema.TypeString,
				Required: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"hostname": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"value": {
				Type:     schema.TypeString,
				Required: true,
			},

			"ttl": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"priority": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"proxied": {
				Default:  false,
				Optional: true,
				Type:     schema.TypeBool,
			},

			"zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudFlareRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)

	newRecord := cloudflare.DNSRecord{
		Type:     d.Get("type").(string),
		Name:     d.Get("name").(string),
		Content:  d.Get("value").(string),
		Proxied:  d.Get("proxied").(bool),
		ZoneName: d.Get("domain").(string),
	}

	if priority, ok := d.GetOk("priority"); ok {
		newRecord.Priority = priority.(int)
	}

	if ttl, ok := d.GetOk("ttl"); ok {
		newRecord.TTL = ttl.(int)
	}

	// Validate value based on type
	if err := validateRecordName(newRecord.Type, newRecord.Content); err != nil {
		return fmt.Errorf("Error validating record name %q: %s", newRecord.Name, err)
	}

	// Validate type
	if err := validateRecordType(newRecord.Type, newRecord.Proxied); err != nil {
		return fmt.Errorf("Error validating record type %q: %s", newRecord.Type, err)
	}

	zoneId, err := client.ZoneIDByName(newRecord.ZoneName)
	if err != nil {
		return fmt.Errorf("Error finding zone %q: %s", newRecord.ZoneName, err)
	}

	d.Set("zone_id", zoneId)
	newRecord.ZoneID = zoneId

	log.Printf("[DEBUG] CloudFlare Record create configuration: %#v", newRecord)

	r, err := client.CreateDNSRecord(zoneId, newRecord)
	if err != nil {
		return fmt.Errorf("Failed to create record: %s", err)
	}

	// In the Event that the API returns an empty DNS Record, we verify that the
	// ID returned is not the default ""
	if r.Result.ID == "" {
		return fmt.Errorf("Failed to find record in Creat response; Record was empty")
	}

	d.SetId(r.Result.ID)

	log.Printf("[INFO] CloudFlare Record ID: %s", d.Id())

	return resourceCloudFlareRecordRead(d, meta)
}

func resourceCloudFlareRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	domain := d.Get("domain").(string)

	zoneId, err := client.ZoneIDByName(domain)
	if err != nil {
		return fmt.Errorf("Error finding zone %q: %s", domain, err)
	}

	record, err := client.DNSRecord(zoneId, d.Id())
	if err != nil {
		return err
	}

	d.SetId(record.ID)
	d.Set("hostname", record.Name)
	d.Set("type", record.Type)
	d.Set("value", record.Content)
	d.Set("ttl", record.TTL)
	d.Set("priority", record.Priority)
	d.Set("proxied", record.Proxied)
	d.Set("zone_id", zoneId)

	return nil
}

func resourceCloudFlareRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)

	updateRecord := cloudflare.DNSRecord{
		ID:       d.Id(),
		Type:     d.Get("type").(string),
		Name:     d.Get("name").(string),
		Content:  d.Get("value").(string),
		ZoneName: d.Get("domain").(string),
		Proxied:  false,
	}

	if priority, ok := d.GetOk("priority"); ok {
		updateRecord.Priority = priority.(int)
	}

	if proxied, ok := d.GetOk("proxied"); ok {
		updateRecord.Proxied = proxied.(bool)
	}

	if ttl, ok := d.GetOk("ttl"); ok {
		updateRecord.TTL = ttl.(int)
	}

	zoneId, err := client.ZoneIDByName(updateRecord.ZoneName)
	if err != nil {
		return fmt.Errorf("Error finding zone %q: %s", updateRecord.ZoneName, err)
	}

	updateRecord.ZoneID = zoneId

	log.Printf("[DEBUG] CloudFlare Record update configuration: %#v", updateRecord)
	err = client.UpdateDNSRecord(zoneId, d.Id(), updateRecord)
	if err != nil {
		return fmt.Errorf("Failed to update CloudFlare Record: %s", err)
	}

	return resourceCloudFlareRecordRead(d, meta)
}

func resourceCloudFlareRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	domain := d.Get("domain").(string)

	zoneId, err := client.ZoneIDByName(domain)
	if err != nil {
		return fmt.Errorf("Error finding zone %q: %s", domain, err)
	}

	log.Printf("[INFO] Deleting CloudFlare Record: %s, %s", domain, d.Id())

	err = client.DeleteDNSRecord(zoneId, d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting CloudFlare Record: %s", err)
	}

	return nil
}

func resourceCloudFlareRecordImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*cloudflare.API)
	id := d.Id()
	re := regexp.MustCompile("^(.*?)/(.*?)/(.*?)(/(.*?))?$")
	matches := re.FindAllStringSubmatch(id, -1)

	match_domain := matches[0][1]
	match_record := matches[0][2]
	match_type := matches[0][3]
	match_index := 0
	if match_type == "MX" {
		parsed_index, err := strconv.ParseInt(matches[0][5], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse index for MX record type, ensure id follows <domain>/<hostname>/<type>/<index>. Error: %s", err)
		}
		match_index = int(parsed_index)
		log.Printf("[INFO] using matchindex: %v", match_index)
	}

	// Get list of Zones to check for record in
	zones, err := client.ListZones()
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch list of Zones when importing ID: %s. Error: %s", id, err)
	}
	log.Printf("[INFO] Found %d zones to check", len(zones))

	// Itterate through Zones to find zone with record for given id
	found := false
	results := make([]*schema.ResourceData, 0)
	for _, zone := range zones {
		log.Printf("[INFO] Checking Zone: %s (%s)", zone.Name, zone.ID)
		if zone.Name == match_domain {

			dnsRecords, err := client.DNSRecords(zone.ID, cloudflare.DNSRecord{Type: match_type})
			if err != nil {
				log.Printf("[ERROR] got error when getting all records: %+v", dnsRecords)
			}

			for _, record := range dnsRecords {
				log.Printf("[INFO] Checking record: %v: %v", record.Name, record.Type)
				if record.Name == match_record && record.Type == match_type {
					found = true
					resource := resourceCloudFlareRecord()
					d := resource.Data(nil)
					d.SetType("cloudflare_record")

					d.SetId(record.ID)

					log.Printf("[INFO] Found record: %+v", record)
					name := strings.TrimSuffix(record.Name, "."+zone.Name)
					d.Set("domain", zone.Name)
					d.Set("hostname", record.Name)
					d.Set("name", name)
					d.Set("priority", record.Priority)
					d.Set("proxied", record.Proxied)
					d.Set("ttl", record.TTL)
					d.Set("type", record.Type)
					d.Set("value", record.Content)
					d.Set("zone_id", zone.ID)
					results = append(results, d)

				}

			}

			break
		}

	}
	if !found {
		return nil, fmt.Errorf("Unable to find record for ID: %s. Checked %d zones", id, len(zones))
	}

	if len(results) < match_index+1 {
		return nil, fmt.Errorf("Unable to find record index %s for ID: %s. Checked %d zones", match_index, id, len(zones))
	}

	return []*schema.ResourceData{results[match_index]}, nil
}
