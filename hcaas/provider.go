package hcaas

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	tsuruclient "github.com/tsuru/go-tsuruclient/pkg/client"
	"github.com/tsuru/go-tsuruclient/pkg/config"
)

func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Description: "Target to tsuru API",
				Optional:    true,
			},
			"token": {
				Type:        schema.TypeString,
				Description: "Token to authenticate on tsuru API (optional)",
				Optional:    true,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"hcaas_url":     resourceHcaasURL(),
			"hcaas_watcher": resourceHcaasWatcher(),
			"hcaas_group":   resourceHcaasGroup(),
		},
	}

	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, d, p.TerraformVersion)
	}
	return p
}

type hcaasProvider struct {
	Host  string
	Token string
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, terraformVersion string) (interface{}, diag.Diagnostics) {
	p := &hcaasProvider{}

	p.Host = d.Get("host").(string)

	if p.Host == "" {
		target, err := config.GetTarget()
		if err != nil {
			return nil, diag.FromErr(err)
		}
		p.Host = target
	}

	p.Token = d.Get("token").(string)
	if p.Token == "" {
		token, err := readToken()
		if err != nil {
			return nil, diag.FromErr(err)
		}
		p.Token = token
	}

	return p, nil
}

func (h *hcaasProvider) serviceURL(serviceName, instance, path string) string {
	q := url.Values{}
	q.Set("callback", fmt.Sprintf("/resources/%s/%s", instance, strings.TrimLeft(path, "/")))
	return fmt.Sprintf("%s/services/%s/proxy/%s?%s", h.Host, serviceName, instance, q.Encode())
}

// retryRequestOnEventLock will retry the same request if the response is a 5xx containing a "event locked" in the response
func retryRequestOnEventLock(ctx context.Context, d *schema.ResourceData, req *http.Request) error {
	return retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate)-time.Minute, func() *retry.RetryError {
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
}

func readToken() (string, error) {
	_, tokenProvider, err := tsuruclient.RoundTripperAndTokenProvider()
	if err != nil {
		return "", err
	}
	return tokenProvider.Token()
}
