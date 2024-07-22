// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package route53domains

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53domains"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
)

// @SDKDataSource("aws_route53domains_registered_domains", name="Registered Domains")
func dataSourceRegisteredDomains() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataSourceRegisteredDomainsRead,

		Timeouts: &schema.ResourceTimeout{
			Read: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"domains": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceRegisteredDomainsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).Route53DomainsClient(ctx)

	input := &route53domains.ListDomainsInput{}

	// input.FilterConditions =

	output, err := findDomains(ctx, conn, input)

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading Route53 Registered Domains: %s", err)
	}

	var domains []string

	for _, v := range output.Domains {
		domains = append(domains, aws.ToString(v.DomainName))
	}

	d.SetId(meta.(*conns.AWSClient).Region)
	d.Set("domains", domains)

	return diags
}
