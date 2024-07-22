// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package route53domains_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccEC2EIPsDataSource_basic(t *testing.T) {
	ctx := acctest.Context(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.Route53DomainsServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEIPsDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					acctest.CheckResourceAttrGreaterThanValue("data.aws_route53domains_registered_domains.all", "domains.#", 1),
				),
			},
		},
	})
}

func testAccEIPsDataSourceConfig_basic() string {
	return `
resource "aws_route53domains_registered_domain" "test1" {
  domain_name = "foo.com"
}

resource "aws_route53domains_registered_domain" "test2" {
  domain_name = "bar.com"
}

data "aws_route53domains_registered_domains" "all" {
  depends_on = [aws_route53domains_registered_domain.test1, aws_route53domains_registered_domain.test2]
}
`
}
