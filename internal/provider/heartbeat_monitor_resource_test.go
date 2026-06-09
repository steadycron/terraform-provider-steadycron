package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHeartbeatMonitor_basic(t *testing.T) {
	testAccPreCheck(t)

	var firstID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read
			{
				Config: testAccHeartbeatConfig("acc-heartbeat", "0 2 * * *", 1800),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_heartbeat_monitor.test", "name", "acc-heartbeat"),
					resource.TestCheckResourceAttr("steadycron_heartbeat_monitor.test", "cron_expression", "0 2 * * *"),
					resource.TestCheckResourceAttr("steadycron_heartbeat_monitor.test", "grace_seconds", "1800"),
					resource.TestCheckResourceAttrSet("steadycron_heartbeat_monitor.test", "id"),
					resource.TestCheckResourceAttrSet("steadycron_heartbeat_monitor.test", "ping_url"),
					resource.TestCheckResourceAttrSet("steadycron_heartbeat_monitor.test", "token"),
					resource.TestCheckResourceAttrWith("steadycron_heartbeat_monitor.test", "id", func(id string) error {
						firstID = id
						return nil
					}),
				),
			},
			// ImportState
			{
				ResourceName:      "steadycron_heartbeat_monitor.test",
				ImportState:       true,
				ImportStateVerify: true,
				// token and ping_url may not be fully restored on import (API redacts them).
				ImportStateVerifyIgnore: []string{"token", "ping_url"},
			},
			// Rename — id and token must be unchanged
			{
				Config: testAccHeartbeatConfig("acc-heartbeat-renamed", "0 2 * * *", 1800),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_heartbeat_monitor.test", "name", "acc-heartbeat-renamed"),
					resource.TestCheckResourceAttrWith("steadycron_heartbeat_monitor.test", "id", func(id string) error {
						if id != firstID {
							return fmt.Errorf("id changed after rename: was %q, now %q", firstID, id)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccHeartbeatMonitor_noScheduleFails(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccHeartbeatNoScheduleConfig(),
				ExpectError: regexp.MustCompile(`exactly one of cron_expression or interval_seconds`),
			},
		},
	})
}

func testAccHeartbeatConfig(name, cronExpr string, graceSecs int) string {
	return fmt.Sprintf(`
resource "steadycron_heartbeat_monitor" "test" {
  name            = %q
  cron_expression = %q
  grace_seconds   = %d
}
`, name, cronExpr, graceSecs)
}

func testAccHeartbeatNoScheduleConfig() string {
	return `
resource "steadycron_heartbeat_monitor" "test" {
  name = "no-schedule"
}
`
}
