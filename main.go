// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/tsuru/terraform-provider-hcaas/hcaas"
)

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: hcaas.Provider})
}
