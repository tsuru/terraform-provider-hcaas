package hcaas

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceHcaasURL() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceHcaasURLCreate,
		ReadContext:   resourceHcaasURLRead,
		DeleteContext: resourceHcaasURLDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"instance": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "HCaaS Instance Name",
			},
			"service_name": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "healthcheck",
				Description: "HCaaS Service Name",
			},
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "URL of monitoring",
			},
			"expected_string": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Expected body string",
			},
			"comment": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Comment of alert",
			},
		},
	}
}

func resourceHcaasURLCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*hcaasProvider)

	r := &CreateURLResource{
		URL:            d.Get("url").(string),
		ExpectedString: d.Get("expected_string").(string),
		Comment:        d.Get("comment").(string),
	}

	url := provider.serviceURL(d.Get("service_name").(string), d.Get("instance").(string), "url")

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(r)
	if err != nil {
		return diag.FromErr(err)
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return diag.FromErr(err)
	}

	req.Header.Set("Authorization", provider.Token)

	err = retryRequestOnEventLock(ctx, d, req)

	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(r.URL)
	return nil
}

func resourceHcaasURLRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*hcaasProvider)
	url := provider.serviceURL(d.Get("service_name").(string), d.Get("instance").(string), "url")

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	req.Header.Set("Authorization", provider.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := ioutil.ReadAll(resp.Body)
		return diag.Errorf("Bad status code: %d, body: %q", resp.StatusCode, string(body))
	}

	var listObj ListURLResource
	err = json.NewDecoder(resp.Body).Decode(&listObj)
	if err != nil {
		return diag.FromErr(err)
	}

	haasURLItem := d.Id()
	for _, item := range listObj {
		if item.URL == haasURLItem {
			d.Set("url", item.URL)
			d.Set("comment", item.Comment)

			return nil
		}
	}

	d.SetId("") // when not found
	return nil
}

func resourceHcaasURLDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*hcaasProvider)
	url := provider.serviceURL(d.Get("service_name").(string), d.Get("instance").(string), "url")

	r := &DeleteURLResource{
		URL: d.Id(),
	}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(r)
	if err != nil {
		return diag.FromErr(err)
	}

	req, err := http.NewRequest(http.MethodDelete, url, &buf)
	if err != nil {
		return diag.FromErr(err)
	}

	req.Header.Set("Authorization", provider.Token)

	err = retryRequestOnEventLock(ctx, d, req)

	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

type CreateURLResource struct {
	URL            string `json:"url"`
	ExpectedString string `json:"expected_string"`
	Comment        string `json:"comment"`
}

type ListURLResource []struct {
	URL     string `json:"url"`
	Comment string `json:"comment"`
}

type DeleteURLResource struct {
	URL string `json:"url"`
}
