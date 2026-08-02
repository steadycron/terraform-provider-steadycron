package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAgentMonitor_basic(t *testing.T) {
	testAccPreCheck(t)

	var firstID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read
			{
				Config: testAccAgentConfig("acc-agent", "tickets", 0.5),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_agent_monitor.test", "name", "acc-agent"),
					resource.TestCheckResourceAttr("steadycron_agent_monitor.test", "cron_expression", "0 3 * * *"),
					resource.TestCheckResourceAttr("steadycron_agent_monitor.test", "items_label", "tickets"),
					resource.TestCheckResourceAttr("steadycron_agent_monitor.test", "rule_max_cost_usd_per_run", "0.5"),
					// Both default on — they are the point of the kind.
					resource.TestCheckResourceAttr("steadycron_agent_monitor.test", "report_required", "true"),
					resource.TestCheckResourceAttr("steadycron_agent_monitor.test", "rule_empty_result_enabled", "true"),
					resource.TestCheckResourceAttrSet("steadycron_agent_monitor.test", "id"),
					// A run is an ordered pair, so both ends of it must be addressable.
					resource.TestCheckResourceAttrSet("steadycron_agent_monitor.test", "ping_url"),
					resource.TestCheckResourceAttrSet("steadycron_agent_monitor.test", "start_ping_url"),
					resource.TestCheckResourceAttrSet("steadycron_agent_monitor.test", "fail_ping_url"),
					resource.TestCheckResourceAttrSet("steadycron_agent_monitor.test", "token"),
					resource.TestCheckResourceAttrWith("steadycron_agent_monitor.test", "id", func(id string) error {
						firstID = id
						return nil
					}),
				),
			},
			// ImportState
			{
				ResourceName:      "steadycron_agent_monitor.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The API redacts the ping URLs and token on GET.
				ImportStateVerifyIgnore: []string{"token", "ping_url", "start_ping_url", "fail_ping_url"},
			},
			// Rename plus a changed ceiling — an in-place update; id and token must survive.
			{
				Config: testAccAgentConfig("acc-agent-renamed", "rows", 1.25),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_agent_monitor.test", "name", "acc-agent-renamed"),
					resource.TestCheckResourceAttr("steadycron_agent_monitor.test", "items_label", "rows"),
					resource.TestCheckResourceAttr("steadycron_agent_monitor.test", "rule_max_cost_usd_per_run", "1.25"),
					resource.TestCheckResourceAttrWith("steadycron_agent_monitor.test", "id", func(id string) error {
						if id != firstID {
							return fmt.Errorf("id changed after rename: was %q, now %q", firstID, id)
						}
						return nil
					}),
				),
			},
			// Removing the ceiling must actually clear it, not silently leave it in place.
			{
				Config: testAccAgentNoCeilingConfig("acc-agent-renamed"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("steadycron_agent_monitor.test", "rule_max_cost_usd_per_run"),
					resource.TestCheckResourceAttrWith("steadycron_agent_monitor.test", "id", func(id string) error {
						if id != firstID {
							return fmt.Errorf("id changed after clearing a ceiling: was %q, now %q", firstID, id)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccAgentMonitor_noScheduleFails(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAgentNoScheduleConfig(),
				ExpectError: regexp.MustCompile(`exactly one of cron_expression or interval_seconds`),
			},
		},
	})
}

func TestAccAgentMonitor_costPeriodWithoutCeilingFails(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccAgentCostPeriodOnlyConfig(),
				ExpectError: regexp.MustCompile(`rule_cost_period requires rule_max_cost_usd_per_period`),
			},
		},
	})
}

func TestAccAgentMonitor_reportRequiredForcesReplacement(t *testing.T) {
	testAccPreCheck(t)

	var firstID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAgentReportRequiredConfig("acc-agent-replace", true),
				Check: resource.TestCheckResourceAttrWith("steadycron_agent_monitor.test", "id", func(id string) error {
					firstID = id
					return nil
				}),
			},
			{
				// The API accepts report_required on create only, so flipping it must recreate
				// the monitor rather than produce an update the server silently ignores.
				Config: testAccAgentReportRequiredConfig("acc-agent-replace", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_agent_monitor.test", "report_required", "false"),
					resource.TestCheckResourceAttrWith("steadycron_agent_monitor.test", "id", func(id string) error {
						if id == firstID {
							return fmt.Errorf("id unchanged (%q) — report_required should have forced replacement", id)
						}
						return nil
					}),
				),
			},
		},
	})
}

func testAccAgentConfig(name, itemsLabel string, maxCostPerRun float64) string {
	return fmt.Sprintf(`
resource "steadycron_agent_monitor" "test" {
  name                      = %q
  cron_expression           = "0 3 * * *"
  grace_seconds             = 600
  max_run_duration_seconds  = 1800
  items_label               = %q
  rule_max_cost_usd_per_run = %g
}
`, name, itemsLabel, maxCostPerRun)
}

func testAccAgentNoCeilingConfig(name string) string {
	return fmt.Sprintf(`
resource "steadycron_agent_monitor" "test" {
  name                     = %q
  cron_expression          = "0 3 * * *"
  grace_seconds            = 600
  max_run_duration_seconds = 1800
  items_label              = "rows"
}
`, name)
}

func testAccAgentNoScheduleConfig() string {
	return `
resource "steadycron_agent_monitor" "test" {
  name = "no-schedule"
}
`
}

func testAccAgentCostPeriodOnlyConfig() string {
	return `
resource "steadycron_agent_monitor" "test" {
  name             = "cost-period-only"
  cron_expression  = "0 3 * * *"
  rule_cost_period = "month"
}
`
}

func testAccAgentReportRequiredConfig(name string, reportRequired bool) string {
	return fmt.Sprintf(`
resource "steadycron_agent_monitor" "test" {
  name            = %q
  cron_expression = "0 3 * * *"
  report_required = %t
}
`, name, reportRequired)
}
