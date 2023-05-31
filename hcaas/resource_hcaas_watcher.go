package hcaas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceHcaasWatcher() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceHcaasWatcherCreate,
		ReadContext:   resourceHcaasWatcherRead,
		DeleteContext: resourceHcaasWatcherDelete,
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
			"email": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Email of watcher",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Sensitive:   true,
				Description: "Password of watcher",
			},
		},
	}
}

func resourceHcaasWatcherCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*hcaasProvider)

	r := &CreateWatcherResource{
		Watcher:  d.Get("email").(string),
		Password: d.Get("password").(string),
	}

	url := provider.serviceURL(d.Get("service_name").(string), d.Get("instance").(string), "watcher")

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

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate)-time.Minute, func() *retry.RetryError {
		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			if strings.Contains(err.Error(), "event locked") {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)

		if resp.StatusCode >= http.StatusInternalServerError && strings.Contains(string(body), "event locked") {
			return retry.RetryableError(err)
		}

		if resp.StatusCode >= http.StatusBadRequest {
			return retry.NonRetryableError(fmt.Errorf("bad status code: %d, body: %q", resp.StatusCode, string(body)))
		}
		return nil
	})

	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(r.Watcher)
	return nil
}

func resourceHcaasWatcherRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*hcaasProvider)
	url := provider.serviceURL(d.Get("service_name").(string), d.Get("instance").(string), "watcher")

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

	var users []string
	err = json.NewDecoder(resp.Body).Decode(&users)
	if err != nil {
		return diag.FromErr(err)
	}

	currentUser := d.Id()
	for _, user := range users {
		if user == currentUser {
			d.Set("email", user)
			return nil
		}
	}

	d.SetId("") // when not found
	return nil
}

func resourceHcaasWatcherDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*hcaasProvider)
	url := provider.serviceURL(d.Get("service_name").(string), d.Get("instance").(string), "watcher/"+d.Id())

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	req.Header.Set("Authorization", provider.Token)

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate)-time.Minute, func() *retry.RetryError {
		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			if strings.Contains(err.Error(), "event locked") {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)

		if resp.StatusCode >= http.StatusInternalServerError && strings.Contains(string(body), "event locked") {
			return retry.RetryableError(err)
		}

		if resp.StatusCode >= http.StatusBadRequest {
			return retry.NonRetryableError(fmt.Errorf("bad status code: %d, body: %q", resp.StatusCode, string(body)))
		}
		return nil
	})

	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

type CreateWatcherResource struct {
	Watcher  string `json:"watcher"`
	Password string `json:"password"`
}
