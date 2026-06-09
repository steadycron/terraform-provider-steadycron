package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAlertRule_basic(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertRuleConfig("on_failure", "P1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_alert_rule.test", "trigger", "on_failure"),
					resource.TestCheckResourceAttr("steadycron_alert_rule.test", "severity", "P1"),
					resource.TestCheckResourceAttrSet("steadycron_alert_rule.test", "id"),
					resource.TestCheckResourceAttrSet("steadycron_alert_rule.test", "job_id"),
					resource.TestCheckResourceAttrSet("steadycron_alert_rule.test", "channel_id"),
				),
			},
			{
				ResourceName:      "steadycron_alert_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAlertRule_consecutiveRequiresThreshold(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAlertRuleNoThresholdConfig(),
				ExpectError: regexp.MustCompile(`threshold is required`),
			},
		},
	})
}

func testAccAlertRuleConfig(trigger, severity string) string {
	return fmt.Sprintf(`
resource "steadycron_alert_channel" "test" {
  name     = "acc-rule-channel"
  kind     = "email"
  email_to = "test@example.com"
}

resource "steadycron_http_job" "test" {
  name             = "acc-rule-job"
  method           = "GET"
  url              = "https://httpbin.org/get"
  interval_seconds = 3600
}

resource "steadycron_alert_rule" "test" {
  job_id     = steadycron_http_job.test.id
  channel_id = steadycron_alert_channel.test.id
  trigger    = %q
  severity   = %q
}
`, trigger, severity)
}

func testAccAlertRuleNoThresholdConfig() string {
	return `
resource "steadycron_alert_channel" "test" {
  name     = "acc-rule-channel-nt"
  kind     = "email"
  email_to = "test@example.com"
}

resource "steadycron_http_job" "test" {
  name             = "acc-rule-job-nt"
  method           = "GET"
  url              = "https://httpbin.org/get"
  interval_seconds = 3600
}

resource "steadycron_alert_rule" "test" {
  job_id     = steadycron_http_job.test.id
  channel_id = steadycron_alert_channel.test.id
  trigger    = "on_n_consecutive"
  # threshold is required but omitted — should fail at plan time
}
`
}
