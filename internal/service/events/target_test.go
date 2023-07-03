// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package events_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/service/eventbridge"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfevents "github.com/hashicorp/terraform-provider-aws/internal/service/events"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

func TestAccEventsTarget_basic(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"
	snsTopicResourceName := "aws_sns_topic.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "rule", rName),
					resource.TestCheckResourceAttr(resourceName, "event_bus_name", "default"),
					resource.TestCheckResourceAttr(resourceName, "target_id", rName),
					resource.TestCheckResourceAttrPair(resourceName, "arn", snsTopicResourceName, "arn"),

					resource.TestCheckResourceAttr(resourceName, "input", ""),
					resource.TestCheckResourceAttr(resourceName, "input_path", ""),
					resource.TestCheckResourceAttr(resourceName, "role_arn", ""),
					resource.TestCheckResourceAttr(resourceName, "run_command_targets.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "batch_target.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "http_target.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "kinesis_target.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "sqs_target.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "input_transformer.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetNoBusNameImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				Config:   testAccTargetConfig_defaultBusName(rName),
				PlanOnly: true,
			},
		},
	})
}

func TestAccEventsTarget_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					acctest.CheckResourceDisappears(ctx, acctest.Provider, tfevents.ResourceTarget(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccEventsTarget_eventBusName(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_busName(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "rule", rName),
					resource.TestCheckResourceAttr(resourceName, "event_bus_name", rName),
					resource.TestCheckResourceAttr(resourceName, "target_id", rName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_eventBusARN(t *testing.T) {
	ctx := acctest.Context(t)

	// "ValidationException: Adding an EventBus as a target within an account is not allowed."
	if got, want := acctest.Partition(), endpoints.AwsUsGovPartitionID; got == want {
		t.Skipf("EventBridge Target EventBus ARNs are not supported in %s partition", got)
	}

	var v eventbridge.Target
	ruleName := sdkacctest.RandomWithPrefix("tf-acc-test-rule")
	targetID := sdkacctest.RandomWithPrefix("tf-acc-test-target")
	originEventBusName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	destinationEventBusName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_busARN(ruleName, originEventBusName, targetID, destinationEventBusName, sdkacctest.RandomWithPrefix("tf-acc-test-target"), sdkacctest.RandomWithPrefix("tf-acc-test-target")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "rule", ruleName),
					acctest.MatchResourceAttrRegionalARN(resourceName, "arn", "events", regexp.MustCompile(fmt.Sprintf("event-bus/%s", destinationEventBusName))),
					acctest.MatchResourceAttrRegionalARN(resourceName, "event_bus_name", "events", regexp.MustCompile(fmt.Sprintf("event-bus/%s", originEventBusName))),
					resource.TestCheckResourceAttr(resourceName, "target_id", targetID),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_generatedTargetID(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	ruleName := sdkacctest.RandomWithPrefix("tf-acc-cw-event-rule-missing-target-id")
	snsTopicName := sdkacctest.RandomWithPrefix("tf-acc")
	resourceName := "aws_cloudwatch_event_target.test"
	snsTopicResourceName := "aws_sns_topic.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_missingID(ruleName, snsTopicName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "rule", ruleName),
					resource.TestCheckResourceAttrPair(resourceName, "arn", snsTopicResourceName, "arn"),
					acctest.CheckResourceAttrNameGenerated(resourceName, "target_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_RetryPolicy_deadLetter(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"
	kinesisStreamResourceName := "aws_kinesis_stream.test"
	queueResourceName := "aws_sqs_queue.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_retryPolicyDlc(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "rule", rName),
					resource.TestCheckResourceAttr(resourceName, "target_id", rName),
					resource.TestCheckResourceAttrPair(resourceName, "arn", kinesisStreamResourceName, "arn"),
					acctest.CheckResourceAttrEquivalentJSON(resourceName, "input", `{"source": ["aws.cloudtrail"]}`),
					resource.TestCheckResourceAttr(resourceName, "input_path", ""),
					resource.TestCheckResourceAttr(resourceName, "retry_policy.0.maximum_event_age_in_seconds", "60"),
					resource.TestCheckResourceAttr(resourceName, "retry_policy.0.maximum_retry_attempts", "5"),
					resource.TestCheckResourceAttrPair(resourceName, "dead_letter_config.0.arn", queueResourceName, "arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_full(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	ruleName := sdkacctest.RandomWithPrefix("tf-acc-cw-event-rule-full")
	ssmDocumentName := sdkacctest.RandomWithPrefix("tf_ssm_Document")
	targetID := sdkacctest.RandomWithPrefix("tf-acc-cw-target-full")
	resourceName := "aws_cloudwatch_event_target.test"
	kinesisStreamResourceName := "aws_kinesis_stream.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_full(ruleName, targetID, ssmDocumentName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "rule", ruleName),
					resource.TestCheckResourceAttr(resourceName, "target_id", targetID),
					resource.TestCheckResourceAttrPair(resourceName, "arn", kinesisStreamResourceName, "arn"),
					acctest.CheckResourceAttrEquivalentJSON(resourceName, "input", `{"source": ["aws.cloudtrail"]}`),
					resource.TestCheckResourceAttr(resourceName, "input_path", ""),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_ssmDocument(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix("tf_ssm_Document")
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_ssmDocument(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "run_command_targets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "run_command_targets.0.key", "tag:Name"),
					resource.TestCheckResourceAttr(resourceName, "run_command_targets.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "run_command_targets.0.values.0", "acceptance_test"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_http(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix("tf_http_target")
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_http(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "http_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.path_parameter_values.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.header_parameters.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.header_parameters.X-Test", "test"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.query_string_parameters.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.query_string_parameters.Env", "test"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.query_string_parameters.Path", "$.detail.path"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

// https://github.com/hashicorp/terraform-provider-aws/issues/23805
func TestAccEventsTarget_http_params(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix("tf_http_target")
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_httpParameter(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "http_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.path_parameter_values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.path_parameter_values.0", "test"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.header_parameters.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.header_parameters.X-Test", "test"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.query_string_parameters.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.query_string_parameters.Env", "test"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.query_string_parameters.Path", "$.detail.path"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccTargetConfig_httpParameterUpdated(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "http_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.path_parameter_values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.path_parameter_values.0", "test"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.path_parameter_values.1", "test2"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.header_parameters.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.header_parameters.X-Test", "test"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.query_string_parameters.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.query_string_parameters.Env", "test"),
					resource.TestCheckResourceAttr(resourceName, "http_target.0.query_string_parameters.Path", "$.detail.path"),
				),
			},
		},
	})
}

func TestAccEventsTarget_ecs(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"
	iamRoleResourceName := "aws_iam_role.test"
	ecsTaskDefinitionResourceName := "aws_ecs_task_definition.task"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_ecs(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttrPair(resourceName, "role_arn", iamRoleResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.task_count", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "ecs_target.0.task_definition_arn", ecsTaskDefinitionResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.launch_type", "FARGATE"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.network_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.network_configuration.0.subnets.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.ordered_placement_strategy.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_redshift(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_redshift(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "redshift_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "redshift_target.0.database", "redshiftdb"),
					resource.TestCheckResourceAttr(resourceName, "redshift_target.0.sql", "SELECT * FROM table"),
					resource.TestCheckResourceAttr(resourceName, "redshift_target.0.statement_name", "NewStatement"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccEventsTarget_ecsWithoutLaunchType verifies Event Target resources
// can be created without a specified LaunchType
// Reference: https://github.com/hashicorp/terraform-provider-aws/issues/16078
func TestAccEventsTarget_ecsWithoutLaunchType(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"
	iamRoleResourceName := "aws_iam_role.test"
	ecsTaskDefinitionResourceName := "aws_ecs_task_definition.task"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_ecsNoLaunchType(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttrPair(resourceName, "role_arn", iamRoleResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.task_count", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "ecs_target.0.task_definition_arn", ecsTaskDefinitionResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.launch_type", ""),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.network_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.network_configuration.0.subnets.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccTargetConfig_ecs(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.launch_type", "FARGATE"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccTargetConfig_ecsNoLaunchType(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.launch_type", ""),
				),
			},
		},
	})
}

func TestAccEventsTarget_ecsWithBlankLaunchType(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"
	iamRoleResourceName := "aws_iam_role.test"
	ecsTaskDefinitionResourceName := "aws_ecs_task_definition.task"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_ecsBlankLaunchType(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttrPair(resourceName, "role_arn", iamRoleResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.task_count", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "ecs_target.0.task_definition_arn", ecsTaskDefinitionResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.launch_type", ""),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.network_configuration.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.network_configuration.0.subnets.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccTargetConfig_ecs(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.launch_type", "FARGATE"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccTargetConfig_ecsBlankLaunchType(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.launch_type", ""),
				),
			},
		},
	})
}

func TestAccEventsTarget_ecsWithBlankTaskCount(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_ecsBlankTaskCount(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.task_count", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_ecsFull(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_ecsBlankTaskCountFull(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.task_count", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.launch_type", "FARGATE"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.enable_execute_command", "true"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.enable_ecs_managed_tags", "true"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.propagate_tags", "TASK_DEFINITION"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.placement_constraint.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.placement_constraint.0.type", "distinctInstance"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.tags.test", "test1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_ecsCapacityProvider(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_ecsCapacityProvider(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.task_count", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.capacity_provider_strategy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.capacity_provider_strategy.0.base", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.capacity_provider_strategy.0.weight", "100"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.capacity_provider_strategy.0.capacity_provider", "test"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_ecsPlacementStrategy(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_ecsPlacementStrategy(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.task_count", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.ordered_placement_strategy.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.ordered_placement_strategy.0.type", "spread"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.ordered_placement_strategy.0.field", "attribute:ecs.availability-zone"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.ordered_placement_strategy.1.type", "spread"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.ordered_placement_strategy.1.field", "instanceId"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_batch(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix("tf_batch_target")
	resourceName := "aws_cloudwatch_event_target.test"
	batchJobDefinitionResourceName := "aws_batch_job_definition.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_batch(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "batch_target.#", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "batch_target.0.job_definition", batchJobDefinitionResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "batch_target.0.job_name", rName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_kinesis(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix("tf_kinesis_target")
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_kinesis(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "kinesis_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "kinesis_target.0.partition_key_path", "$.detail"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName), ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_sqs(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_sqs(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "sqs_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "sqs_target.0.message_group_id", "event_group"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_Input_transformer(t *testing.T) {
	ctx := acctest.Context(t)
	resourceName := "aws_cloudwatch_event_target.test"
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix("tf_input_transformer")

	tooManyInputPaths := make([]string, 101)
	for i := range tooManyInputPaths {
		tooManyInputPaths[i] = fmt.Sprintf("InvalidField_%d", i)
	}

	validInputPaths := make([]string, 100)
	for i := range validInputPaths {
		validInputPaths[i] = fmt.Sprintf("ValidField_%d", i)
	}

	var expectedInputTemplate strings.Builder
	fmt.Fprintf(&expectedInputTemplate, `{
  "detail-type": "Scheduled Event",
  "source": "aws.events",
`)
	for _, path := range validInputPaths {
		fmt.Fprintf(&expectedInputTemplate, "  \"%[1]s\": <%[1]s>,\n", path)
	}
	fmt.Fprintf(&expectedInputTemplate, `  "detail": {}
}
`)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config:      testAccTargetConfig_inputTransformer(rName, tooManyInputPaths),
				ExpectError: regexp.MustCompile(`.*expected number of items in.* to be less than or equal to.*`),
			},
			{
				Config: testAccTargetConfig_inputTransformer(rName, validInputPaths),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "input_transformer.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "input_transformer.0.input_paths.%", strconv.Itoa(len(validInputPaths))),
					resource.TestCheckResourceAttr(resourceName, "input_transformer.0.input_paths.ValidField_99", "$.ValidField_99"),
					resource.TestCheckResourceAttr(resourceName, "input_transformer.0.input_template", expectedInputTemplate.String()),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_inputTransformerJSONString(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_inputTransformerJSONString(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "input_transformer.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "input_transformer.0.input_paths.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "input_transformer.0.input_paths.instance", "$.detail.instance"),
					resource.TestCheckResourceAttr(resourceName, "input_transformer.0.input_template", "\"<instance> is in state <status>\""),
				),
			},
		},
	})
}

func TestAccEventsTarget_partnerEventBus(t *testing.T) {
	ctx := acctest.Context(t)
	key := "EVENT_BRIDGE_PARTNER_EVENT_BUS_NAME"
	busName := os.Getenv(key)
	if busName == "" {
		t.Skipf("Environment variable %s is not set", key)
	}

	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"
	snsTopicResourceName := "aws_sns_topic.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_partnerBus(rName, busName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "rule", rName),
					resource.TestCheckResourceAttr(resourceName, "event_bus_name", busName),
					resource.TestCheckResourceAttr(resourceName, "target_id", rName),
					resource.TestCheckResourceAttrPair(resourceName, "arn", snsTopicResourceName, "arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccEventsTarget_ecsNoPropagateTags(t *testing.T) {
	ctx := acctest.Context(t)
	var v eventbridge.Target
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_cloudwatch_event_target.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, eventbridge.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTargetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTargetConfig_ecsNoPropagateTags(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTargetExists(ctx, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.task_count", "1"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.launch_type", "FARGATE"),
					resource.TestCheckResourceAttr(resourceName, "ecs_target.0.propagate_tags", ""),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccTargetImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckTargetExists(ctx context.Context, n string, v *eventbridge.Target) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EventsConn(ctx)

		output, err := tfevents.FindTargetByThreePartKey(ctx, conn, rs.Primary.Attributes["event_bus_name"], rs.Primary.Attributes["rule"], rs.Primary.Attributes["target_id"])

		if err != nil {
			return err
		}

		*v = *output

		return nil
	}
}

func testAccCheckTargetDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).EventsConn(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_cloudwatch_event_target" {
				continue
			}

			_, err := tfevents.FindTargetByThreePartKey(ctx, conn, rs.Primary.Attributes["event_bus_name"], rs.Primary.Attributes["rule"], rs.Primary.Attributes["target_id"])

			if tfresource.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("EventBridge Target %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

func testAccTargetImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}

		return fmt.Sprintf("%s/%s/%s", rs.Primary.Attributes["event_bus_name"], rs.Primary.Attributes["rule"], rs.Primary.Attributes["target_id"]), nil
	}
}

func testAccTargetNoBusNameImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}

		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["rule"], rs.Primary.Attributes["target_id"]), nil
	}
}

func testAccTargetConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_event_rule" "test" {
  name                = %[1]q
  schedule_expression = "rate(1 hour)"
}

resource "aws_cloudwatch_event_target" "test" {
  rule      = aws_cloudwatch_event_rule.test.name
  target_id = %[1]q
  arn       = aws_sns_topic.test.arn
}

resource "aws_sns_topic" "test" {
  name = %[1]q
}
`, rName)
}

func testAccTargetConfig_defaultBusName(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_event_rule" "test" {
  name                = %[1]q
  event_bus_name      = "default"
  schedule_expression = "rate(1 hour)"
}

resource "aws_cloudwatch_event_target" "test" {
  rule           = aws_cloudwatch_event_rule.test.name
  event_bus_name = aws_cloudwatch_event_rule.test.event_bus_name
  target_id      = %[1]q
  arn            = aws_sns_topic.test.arn
}

resource "aws_sns_topic" "test" {
  name = %[1]q
}
`, rName)
}

func testAccTargetConfig_busName(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_event_target" "test" {
  rule           = aws_cloudwatch_event_rule.test.name
  event_bus_name = aws_cloudwatch_event_rule.test.event_bus_name
  target_id      = %[1]q
  arn            = aws_sns_topic.test.arn
}

resource "aws_sns_topic" "test" {
  name = %[1]q
}

resource "aws_cloudwatch_event_rule" "test" {
  name           = %[1]q
  event_bus_name = aws_cloudwatch_event_bus.test.name
  event_pattern  = <<PATTERN
{
	"source": [
		"aws.ec2"
	]
}
PATTERN
}

resource "aws_cloudwatch_event_bus" "test" {
  name = %[1]q
}
`, rName)
}

func testAccTargetConfig_busARN(ruleName, originEventBusName, targetID, destinationEventBusName, roleName, policyName string) string {
	return fmt.Sprintf(`
data "aws_partition" "current" {}

resource "aws_cloudwatch_event_bus" "test_origin_bus" {
  name = %[1]q
}

resource "aws_cloudwatch_event_bus" "test_destination_bus" {
  name = %[4]q
}

resource "aws_cloudwatch_event_target" "test" {
  rule           = aws_cloudwatch_event_rule.test.name
  event_bus_name = aws_cloudwatch_event_bus.test_origin_bus.arn
  target_id      = %[3]q
  arn            = aws_cloudwatch_event_bus.test_destination_bus.arn
  role_arn       = aws_iam_role.test.arn
}

resource "aws_cloudwatch_event_rule" "test" {
  name           = %[2]q
  event_bus_name = aws_cloudwatch_event_bus.test_origin_bus.name
  event_pattern  = <<PATTERN
{
  "source": ["aws.ec2"]
}
PATTERN
}

resource "aws_iam_role" "test" {
  name = %[5]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Action": "sts:AssumeRole",
    "Principal": {
      "Service": "events.${data.aws_partition.current.dns_suffix}"
    },
    "Effect": "Allow"
  }]
}
EOF
}
`, originEventBusName, ruleName, targetID, destinationEventBusName, roleName, policyName)
}

func testAccTargetConfig_missingID(ruleName, snsTopicName string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_event_rule" "test" {
  name                = "%s"
  schedule_expression = "rate(1 hour)"
}

resource "aws_cloudwatch_event_target" "test" {
  rule = aws_cloudwatch_event_rule.test.name
  arn  = aws_sns_topic.test.arn
}

resource "aws_sns_topic" "test" {
  name = "%s"
}
`, ruleName, snsTopicName)
}

func testAccTargetConfig_retryPolicyDlc(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_event_rule" "test" {
  name                = %[1]q
  schedule_expression = "rate(1 hour)"
  role_arn            = aws_iam_role.test.arn
}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [{
    "Action": "sts:AssumeRole",
    "Principal": {
      "Service": "events.${data.aws_partition.current.dns_suffix}"
    },
    "Effect": "Allow"
  }]
}
POLICY
}

resource "aws_iam_role_policy" "test" {
  name = %[1]q
  role = aws_iam_role.test.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "kinesis:PutRecord",
        "kinesis:PutRecords"
      ],
      "Resource": [
        "*"
      ],
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_sqs_queue" "test" {
  name = %[1]q

  sqs_managed_sse_enabled = true
}

resource "aws_cloudwatch_event_target" "test" {
  rule      = aws_cloudwatch_event_rule.test.name
  target_id = %[1]q

  input = <<INPUT
{ "source": ["aws.cloudtrail"] }
INPUT

  arn = aws_kinesis_stream.test.arn

  retry_policy {
    maximum_event_age_in_seconds = 60
    maximum_retry_attempts       = 5
  }

  dead_letter_config {
    arn = aws_sqs_queue.test.arn
  }
}

resource "aws_kinesis_stream" "test" {
  name        = %[1]q
  shard_count = 1
}

data "aws_partition" "current" {}
`, rName)
}

func testAccTargetConfig_full(ruleName, targetName, rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_event_rule" "test" {
  name                = %[1]q
  schedule_expression = "rate(1 hour)"
  role_arn            = aws_iam_role.test.arn
}

resource "aws_iam_role" "test" {
  name = %[2]q

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "events.${data.aws_partition.current.dns_suffix}"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy" "test" {
  name = "%[2]s_policy"
  role = aws_iam_role.test.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "kinesis:PutRecord",
        "kinesis:PutRecords"
      ],
      "Resource": [
        "*"
      ],
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_cloudwatch_event_target" "test" {
  rule      = aws_cloudwatch_event_rule.test.name
  target_id = %[3]q

  input = <<INPUT
{ "source": ["aws.cloudtrail"] }
INPUT

  arn = aws_kinesis_stream.test.arn
}

resource "aws_kinesis_stream" "test" {
  name        = "%[2]s_kinesis_test"
  shard_count = 1
}

data "aws_partition" "current" {}
`, ruleName, rName, targetName)
}

func testAccTargetConfig_ssmDocument(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_document" "test" {
  name          = %[1]q
  document_type = "Command"

  content = <<DOC
    {
      "schemaVersion": "1.2",
      "description": "Check ip configuration of a Linux instance.",
      "parameters": {

      },
      "runtimeConfig": {
        "aws:runShellScript": {
          "properties": [
            {
              "id": "0.aws:runShellScript",
              "runCommand": ["ifconfig"]
            }
          ]
        }
      }
    }
DOC
}

resource "aws_cloudwatch_event_rule" "test" {
  name        = %[1]q
  description = "another_test"

  event_pattern = <<PATTERN
{
  "source": [
    "aws.autoscaling"
  ]
}
PATTERN
}

resource "aws_cloudwatch_event_target" "test" {
  arn      = aws_ssm_document.test.arn
  rule     = aws_cloudwatch_event_rule.test.id
  role_arn = aws_iam_role.test.arn

  run_command_targets {
    key    = "tag:Name"
    values = ["acceptance_test"]
  }
}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "events.${data.aws_partition.current.dns_suffix}"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "test" {
  name = %[1]q
  role = aws_iam_role.test.id

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "ssm:*",
            "Effect": "Allow",
            "Resource": [
                "*"
            ]
        }
    ]
}
EOF
}

data "aws_partition" "current" {}
`, rName)
}

func testAccTargetHTTPConfigBase(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_event_rule" "test" {
  name        = %[1]q
  description = "schedule_http_test"

  schedule_expression = "rate(5 minutes)"
}

resource "aws_api_gateway_rest_api" "test" {
  name = %[1]q
  body = jsonencode({
    openapi = "3.0.1"
    info = {
      title   = "example"
      version = "1.0"
    }
    paths = {
      "/" = {
        get = {
          x-amazon-apigateway-integration = {
            httpMethod           = "GET"
            payloadFormatVersion = "1.0"
            type                 = "HTTP_PROXY"
            uri                  = "https://ip-ranges.amazonaws.com"
          }
        }
      }
    }
  })
}

resource "aws_api_gateway_deployment" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id

  triggers = {
    redeployment = sha1(jsonencode(aws_api_gateway_rest_api.test.body))
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_api_gateway_stage" "test" {
  deployment_id = aws_api_gateway_deployment.test.id
  rest_api_id   = aws_api_gateway_rest_api.test.id
  stage_name    = "test"
}

data "aws_partition" "current" {}
`, rName)
}

func testAccTargetConfig_http(rName string) string {
	return testAccTargetHTTPConfigBase(rName) + `
resource "aws_cloudwatch_event_target" "test" {
  arn  = "${aws_api_gateway_stage.test.execution_arn}/GET"
  rule = aws_cloudwatch_event_rule.test.id

  http_target {
    path_parameter_values = []
    query_string_parameters = {
      Env  = "test"
      Path = "$.detail.path"
    }
    header_parameters = {
      X-Test = "test"
    }
  }
}
`
}

func testAccTargetConfig_httpParameter(rName string) string {
	return testAccTargetHTTPConfigBase(rName) + `
resource "aws_cloudwatch_event_target" "test" {
  arn  = "${aws_api_gateway_stage.test.execution_arn}/*/*/GET"
  rule = aws_cloudwatch_event_rule.test.id

  http_target {
    path_parameter_values = ["test"]
    query_string_parameters = {
      Env  = "test"
      Path = "$.detail.path"
    }
    header_parameters = {
      X-Test = "test"
    }
  }
}
`
}

func testAccTargetConfig_httpParameterUpdated(rName string) string {
	return testAccTargetHTTPConfigBase(rName) + `
resource "aws_cloudwatch_event_target" "test" {
  arn  = "${aws_api_gateway_stage.test.execution_arn}/*/*/*/GET"
  rule = aws_cloudwatch_event_rule.test.id

  http_target {
    path_parameter_values = ["test", "test2"]
    query_string_parameters = {
      Env  = "test"
      Path = "$.detail.path"
    }
    header_parameters = {
      X-Test = "test"
    }
  }
}
`
}

func testAccTargetConfig_ecsBase(rName string) string {
	return acctest.ConfigCompose(acctest.ConfigVPCWithSubnets(rName, 1), fmt.Sprintf(`
resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Action": "sts:AssumeRole",
    "Principal": {
      "Service": "events.${data.aws_partition.current.dns_suffix}"
    },
    "Effect": "Allow"
  }]
}
EOF
}

resource "aws_iam_role_policy" "test" {
  name = %[1]q
  role = aws_iam_role.test.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["ecs:RunTask"],
    "Resource": ["*"]
  }]
}
EOF
}

resource "aws_ecs_cluster" "test" {
  name = %[1]q
}

resource "aws_ecs_task_definition" "task" {
  family                   = %[1]q
  cpu                      = 256
  memory                   = 512
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"

  container_definitions = <<EOF
[
  {
    "name": "first",
    "image": "service-first",
    "cpu": 10,
    "memory": 512,
    "essential": true
  }
]
EOF
}

data "aws_partition" "current" {}

resource "aws_cloudwatch_event_rule" "test" {
  name        = %[1]q
  description = "schedule_ecs_test"

  schedule_expression = "rate(5 minutes)"
}
`, rName))
}

func testAccTargetConfig_ecs(rName string) string {
	return acctest.ConfigCompose(testAccTargetConfig_ecsBase(rName), `
resource "aws_cloudwatch_event_target" "test" {
  arn      = aws_ecs_cluster.test.id
  rule     = aws_cloudwatch_event_rule.test.id
  role_arn = aws_iam_role.test.arn

  ecs_target {
    task_count          = 1
    task_definition_arn = aws_ecs_task_definition.task.arn
    launch_type         = "FARGATE"

    network_configuration {
      subnets = aws_subnet.test[*].id
    }
  }
}
`)
}

func testAccTargetConfig_redshift(rName string) string {
	return acctest.ConfigCompose(acctest.ConfigVPCWithSubnets(rName, 2), fmt.Sprintf(`
data "aws_partition" "current" {}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Action": "sts:AssumeRole",
    "Principal": {
      "Service": "events.${data.aws_partition.current.dns_suffix}"
    },
    "Effect": "Allow"
  }]
}
EOF
}

resource "aws_redshift_subnet_group" "test" {
  name       = %[1]q
  subnet_ids = aws_subnet.test[*].id
}

resource "aws_internet_gateway" "test" {
  vpc_id = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_cloudwatch_event_rule" "test" {
  name = %[1]q

  schedule_expression = "rate(5 minutes)"
}

resource "aws_cloudwatch_event_target" "test" {
  arn      = aws_redshift_cluster.test.arn
  rule     = aws_cloudwatch_event_rule.test.id
  role_arn = aws_iam_role.test.arn

  redshift_target {
    database       = "redshiftdb"
    sql            = "SELECT * FROM table"
    statement_name = "NewStatement"
    db_user        = "someUser"
  }

  target_id = %[1]q
}

resource "aws_redshift_cluster" "test" {
  cluster_identifier                  = %[1]q
  cluster_subnet_group_name           = aws_redshift_subnet_group.test.name
  database_name                       = "test"
  master_username                     = "tfacctest"
  master_password                     = "Mustbe8characters"
  node_type                           = "dc2.large"
  automated_snapshot_retention_period = 0
  allow_version_upgrade               = false
  skip_final_snapshot                 = true

  depends_on = [aws_internet_gateway.test]
}
`, rName))
}

func testAccTargetConfig_ecsNoLaunchType(rName string) string {
	return acctest.ConfigCompose(testAccTargetConfig_ecsBase(rName), `
resource "aws_cloudwatch_event_target" "test" {
  arn      = aws_ecs_cluster.test.id
  rule     = aws_cloudwatch_event_rule.test.id
  role_arn = aws_iam_role.test.arn

  ecs_target {
    task_count          = 1
    task_definition_arn = aws_ecs_task_definition.task.arn

    network_configuration {
      subnets = aws_subnet.test[*].id
    }
  }
}
`)
}

func testAccTargetConfig_ecsBlankLaunchType(rName string) string {
	return acctest.ConfigCompose(testAccTargetConfig_ecsBase(rName), `
resource "aws_cloudwatch_event_target" "test" {
  arn      = aws_ecs_cluster.test.id
  rule     = aws_cloudwatch_event_rule.test.id
  role_arn = aws_iam_role.test.arn

  ecs_target {
    task_count          = 1
    task_definition_arn = aws_ecs_task_definition.task.arn
    launch_type         = null

    network_configuration {
      subnets = aws_subnet.test[*].id
    }
  }
}
`)
}

func testAccTargetConfig_ecsBlankTaskCount(rName string) string {
	return acctest.ConfigCompose(testAccTargetConfig_ecsBase(rName), `
resource "aws_cloudwatch_event_target" "test" {
  arn      = aws_ecs_cluster.test.id
  rule     = aws_cloudwatch_event_rule.test.id
  role_arn = aws_iam_role.test.arn

  ecs_target {
    task_definition_arn = aws_ecs_task_definition.task.arn
    launch_type         = "FARGATE"

    network_configuration {
      subnets = aws_subnet.test[*].id
    }
  }
}
`)
}

func testAccTargetConfig_ecsBlankTaskCountFull(rName string) string {
	return acctest.ConfigCompose(testAccTargetConfig_ecsBase(rName), `
resource "aws_cloudwatch_event_target" "test" {
  arn      = aws_ecs_cluster.test.id
  rule     = aws_cloudwatch_event_rule.test.id
  role_arn = aws_iam_role.test.arn

  ecs_target {
    task_definition_arn     = aws_ecs_task_definition.task.arn
    launch_type             = "FARGATE"
    enable_execute_command  = true
    enable_ecs_managed_tags = true
    propagate_tags          = "TASK_DEFINITION"

    placement_constraint {
      type = "distinctInstance"
    }

    tags = {
      test = "test1"
    }

    network_configuration {
      subnets = aws_subnet.test[*].id
    }
  }
}
`)
}

func testAccTargetConfig_ecsCapacityProvider(rName string) string {
	return acctest.ConfigCompose(
		acctest.ConfigLatestAmazonLinuxHVMEBSAMI(),
		testAccTargetConfig_ecsBase(rName),
		fmt.Sprintf(`
resource "aws_cloudwatch_event_target" "test" {
  arn       = aws_ecs_cluster.test.id
  rule      = aws_cloudwatch_event_rule.test.id
  role_arn  = aws_iam_role.test.arn
  target_id = %[1]q

  ecs_target {
    task_definition_arn = aws_ecs_task_definition.task.arn

    capacity_provider_strategy {
      capacity_provider = "test"
      base              = 1
      weight            = 100
    }
  }
}

resource "aws_launch_template" "test" {
  name          = %[1]q
  image_id      = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type = "t2.micro"
}

resource "aws_autoscaling_group" "test" {
  name                = %[1]q
  desired_capacity    = 0
  max_size            = 0
  min_size            = 0
  vpc_zone_identifier = [aws_subnet.test[0].id]

  launch_template {
    id      = aws_launch_template.test.id
    version = "$Latest"
  }

  tag {
    key                 = "AmazonECSManaged"
    value               = ""
    propagate_at_launch = true
  }
}

resource "aws_ecs_capacity_provider" "test" {
  name = %[1]q

  auto_scaling_group_provider {
    auto_scaling_group_arn         = aws_autoscaling_group.test.arn
    managed_termination_protection = "DISABLED"

    managed_scaling {
      maximum_scaling_step_size = 1
      minimum_scaling_step_size = 1
      status                    = "ENABLED"
      target_capacity           = 1
    }
  }
}
`, rName))
}

func testAccTargetConfig_ecsPlacementStrategy(rName string) string {
	return acctest.ConfigCompose(
		acctest.ConfigLatestAmazonLinuxHVMEBSAMI(),
		testAccTargetConfig_ecsBase(rName),
		fmt.Sprintf(`
resource "aws_cloudwatch_event_target" "test" {
  arn       = aws_ecs_cluster.test.id
  rule      = aws_cloudwatch_event_rule.test.id
  role_arn  = aws_iam_role.test.arn
  target_id = %[1]q

  ecs_target {
    task_definition_arn = aws_ecs_task_definition.task.arn
    launch_type         = "EC2"

    ordered_placement_strategy {
      type  = "spread"
      field = "attribute:ecs.availability-zone"
    }

    ordered_placement_strategy {
      type  = "spread"
      field = "instanceId"
    }

  }
}

resource "aws_launch_template" "test" {
  name          = %[1]q
  image_id      = data.aws_ami.amzn-ami-minimal-hvm-ebs.id
  instance_type = "t2.micro"
}

resource "aws_autoscaling_group" "test" {
  name                = %[1]q
  desired_capacity    = 0
  max_size            = 0
  min_size            = 0
  vpc_zone_identifier = [aws_subnet.test[0].id]

  launch_template {
    id      = aws_launch_template.test.id
    version = "$Latest"
  }

  tag {
    key                 = "AmazonECSManaged"
    value               = ""
    propagate_at_launch = true
  }
}

resource "aws_ecs_capacity_provider" "test" {
  name = %[1]q

  auto_scaling_group_provider {
    auto_scaling_group_arn         = aws_autoscaling_group.test.arn
    managed_termination_protection = "DISABLED"

    managed_scaling {
      maximum_scaling_step_size = 1
      minimum_scaling_step_size = 1
      status                    = "ENABLED"
      target_capacity           = 1
    }
  }
}

resource "aws_ecs_cluster_capacity_providers" "test" {
  cluster_name       = aws_ecs_cluster.test.name
  capacity_providers = ["%[1]s"]

  default_capacity_provider_strategy {
    capacity_provider = %[1]q
    base              = 1
    weight            = 100
  }
}


`, rName))
}

func testAccTargetConfig_batch(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_event_rule" "test" {
  name                = "%[1]s"
  description         = "schedule_batch_test"
  schedule_expression = "rate(5 minutes)"
}

resource "aws_cloudwatch_event_target" "test" {
  arn      = aws_batch_job_queue.test.arn
  rule     = aws_cloudwatch_event_rule.test.id
  role_arn = aws_iam_role.event_iam_role.arn

  batch_target {
    job_definition = aws_batch_job_definition.test.arn
    job_name       = "%[1]s"
  }

  depends_on = [
    "aws_batch_job_queue.test",
    "aws_batch_job_definition.test",
    "aws_iam_role.event_iam_role",
  ]
}

data "aws_partition" "current" {}

resource "aws_iam_role" "event_iam_role" {
  name = "event_%[1]s"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Effect": "Allow",
      "Principal": {
        "Service": "events.${data.aws_partition.current.dns_suffix}"
      }
    }
  ]
}
EOF
}

resource "aws_iam_role" "ecs_iam_role" {
  name = "ecs_%[1]s"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.${data.aws_partition.current.dns_suffix}"
      }
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "ecs_policy_attachment" {
  role       = aws_iam_role.ecs_iam_role.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/service-role/AmazonEC2ContainerServiceforEC2Role"
}

resource "aws_iam_instance_profile" "iam_instance_profile" {
  name = "ecs_%[1]s"
  role = aws_iam_role.ecs_iam_role.name
}

resource "aws_iam_role" "batch_iam_role" {
  name = "batch_%[1]s"

  assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
    {
        "Action": "sts:AssumeRole",
        "Effect": "Allow",
        "Principal": {
          "Service": "batch.${data.aws_partition.current.dns_suffix}"
        }
    }
    ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "batch_policy_attachment" {
  role       = aws_iam_role.batch_iam_role.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/service-role/AWSBatchServiceRole"
}

resource "aws_security_group" "security_group" {
  name = "%[1]s"
}

resource "aws_vpc" "vpc" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "subnet" {
  vpc_id     = aws_vpc.vpc.id
  cidr_block = "10.1.1.0/24"
}

resource "aws_batch_compute_environment" "test" {
  compute_environment_name = "%[1]s"

  compute_resources {
    instance_role = aws_iam_instance_profile.iam_instance_profile.arn

    instance_type = [
      "c4.large",
    ]

    max_vcpus = 16
    min_vcpus = 0

    security_group_ids = [
      aws_security_group.security_group.id,
    ]

    subnets = [
      aws_subnet.subnet.id,
    ]

    type = "EC2"
  }

  service_role = aws_iam_role.batch_iam_role.arn
  type         = "MANAGED"
  depends_on   = [aws_iam_role_policy_attachment.batch_policy_attachment]
}

resource "aws_batch_job_queue" "test" {
  name                 = "%[1]s"
  state                = "ENABLED"
  priority             = 1
  compute_environments = [aws_batch_compute_environment.test.arn]
}

resource "aws_batch_job_definition" "test" {
  name = "%[1]s"
  type = "container"

  container_properties = <<CONTAINER_PROPERTIES
{
  "command": ["ls", "-la"],
  "image": "busybox",
  "memory": 512,
  "vcpus": 1,
  "volumes": [ ],
  "environment": [ ],
  "mountPoints": [ ],
  "ulimits": [ ]
}
CONTAINER_PROPERTIES
}
`, rName)
}

func testAccTargetConfig_kinesis(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_event_rule" "test" {
  name                = "%[1]s"
  description         = "schedule_batch_test"
  schedule_expression = "rate(5 minutes)"
}

resource "aws_cloudwatch_event_target" "test" {
  arn      = aws_kinesis_stream.test.arn
  rule     = aws_cloudwatch_event_rule.test.id
  role_arn = aws_iam_role.test.arn

  kinesis_target {
    partition_key_path = "$.detail"
  }
}

resource "aws_iam_role" "test" {
  name = "event_%[1]s"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Effect": "Allow",
      "Principal": {
        "Service": "events.${data.aws_partition.current.dns_suffix}"
      }
    }
  ]
}
EOF
}

resource "aws_kinesis_stream" "test" {
  name        = "%[1]s"
  shard_count = 1
}

data "aws_partition" "current" {}
`, rName)
}

func testAccTargetConfig_sqs(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_event_rule" "test" {
  name                = %[1]q
  description         = "schedule_batch_test"
  schedule_expression = "rate(5 minutes)"
}

resource "aws_cloudwatch_event_target" "test" {
  arn  = aws_sqs_queue.test.arn
  rule = aws_cloudwatch_event_rule.test.id

  sqs_target {
    message_group_id = "event_group"
  }

  target_id = %[1]q
}

resource "aws_sqs_queue" "test" {
  name       = "%[1]s.fifo"
  fifo_queue = true

  sqs_managed_sse_enabled = true
}
`, rName)
}

func testAccTargetConfig_inputTransformer(rName string, inputPathKeys []string) string {
	var inputPaths, inputTemplates strings.Builder

	for _, inputPath := range inputPathKeys {
		fmt.Fprintf(&inputPaths, "      %[1]s = \"$.%[1]s\"\n", inputPath)
		fmt.Fprintf(&inputTemplates, "  \"%[1]s\": <%[1]s>,\n", inputPath)
	}

	return acctest.ConfigCompose(
		testAccTargetLambdaBaseConfig(rName),
		fmt.Sprintf(`
resource "aws_cloudwatch_event_target" "test" {
  arn  = aws_lambda_function.test.arn
  rule = aws_cloudwatch_event_rule.schedule.id

  input_transformer {
    input_paths = {
      %[2]s
    }

    input_template = <<EOF
{
  "detail-type": "Scheduled Event",
  "source": "aws.events",
  %[3]s
  "detail": {}
}
EOF
  }
}

resource "aws_cloudwatch_event_rule" "schedule" {
  name        = "%[1]s"
  description = "test_input_transformer"

  schedule_expression = "rate(5 minutes)"
}
`, rName, inputPaths.String(), strings.TrimSpace(inputTemplates.String())))
}

func testAccTargetConfig_inputTransformerJSONString(name string) string {
	return acctest.ConfigCompose(
		testAccTargetLambdaBaseConfig(name),
		fmt.Sprintf(`
resource "aws_cloudwatch_event_target" "test" {
  arn  = aws_lambda_function.test.arn
  rule = aws_cloudwatch_event_rule.test.id

  input_transformer {
    input_paths = {
      instance = "$.detail.instance",
      status   = "$.detail.status",
    }
    input_template = "\"<instance> is in state <status>\""
  }
}

resource "aws_cloudwatch_event_rule" "test" {
  name        = %[1]q
  description = "test_input_transformer"

  schedule_expression = "rate(5 minutes)"
}
`, name))
}

func testAccTargetLambdaBaseConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_lambda_function" "test" {
  function_name    = %[1]q
  filename         = "test-fixtures/lambdatest.zip"
  source_code_hash = filebase64sha256("test-fixtures/lambdatest.zip")
  role             = aws_iam_role.test.arn
  handler          = "exports.example"
  runtime          = "nodejs16.x"
}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.${data.aws_partition.current.dns_suffix}"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

data "aws_partition" "current" {}
`, name)
}

func testAccTargetConfig_partnerBus(rName, eventBusName string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_event_rule" "test" {
  name           = %[1]q
  event_bus_name = %[2]q

  event_pattern = <<PATTERN
{
  "source": ["aws.ec2"]
}
PATTERN
}

resource "aws_cloudwatch_event_target" "test" {
  rule           = aws_cloudwatch_event_rule.test.name
  event_bus_name = aws_cloudwatch_event_rule.test.event_bus_name
  target_id      = %[1]q
  arn            = aws_sns_topic.test.arn
}

resource "aws_sns_topic" "test" {
  name = %[1]q
}
`, rName, eventBusName)
}

func testAccTargetConfig_ecsNoPropagateTags(rName string) string {
	return acctest.ConfigCompose(testAccTargetConfig_ecsBase(rName), `
resource "aws_cloudwatch_event_target" "test" {
  arn      = aws_ecs_cluster.test.id
  rule     = aws_cloudwatch_event_rule.test.id
  role_arn = aws_iam_role.test.arn

  ecs_target {
    task_definition_arn = aws_ecs_task_definition.task.arn
    launch_type         = "FARGATE"

    network_configuration {
      subnets = aws_subnet.test[*].id
    }
  }
}
`)
}
