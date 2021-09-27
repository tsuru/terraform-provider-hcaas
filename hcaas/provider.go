package hcaas

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	tsuruCmd "github.com/tsuru/tsuru/cmd"
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
			"hcaas_url": resourceHcaasURL(),
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
		target, err := tsuruCmd.GetTarget()
		if err != nil {
			return nil, diag.FromErr(err)
		}
		p.Host = target
	}

	p.Token = d.Get("token").(string)
	if p.Token == "" {
		token, err := tsuruCmd.ReadToken()
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
