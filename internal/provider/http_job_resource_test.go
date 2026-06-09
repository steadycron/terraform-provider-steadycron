package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccHTTPJob_basic(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccHTTPJobConfig("acc-http-job", "GET", "https://httpbin.org/get", "0 12 * * *"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_http_job.test", "name", "acc-http-job"),
					resource.TestCheckResourceAttr("steadycron_http_job.test", "method", "GET"),
					resource.TestCheckResourceAttr("steadycron_http_job.test", "url", "https://httpbin.org/get"),
					resource.TestCheckResourceAttr("steadycron_http_job.test", "cron_expression", "0 12 * * *"),
					resource.TestCheckResourceAttrSet("steadycron_http_job.test", "id"),
					resource.TestCheckResourceAttrSet("steadycron_http_job.test", "created_at"),
				),
			},
			// ImportState
			{
				ResourceName:      "steadycron_http_job.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name in-place
			{
				Config: testAccHTTPJobConfig("acc-http-job-renamed", "GET", "https://httpbin.org/get", "0 12 * * *"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_http_job.test", "name", "acc-http-job-renamed"),
					// id must be unchanged (in-place update)
					testCheckIDUnchanged("steadycron_http_job.test"),
				),
			},
		},
	})
}

func TestAccHTTPJob_intervalSchedule(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPJobIntervalConfig("acc-http-interval", 3600),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_http_job.test", "interval_seconds", "3600"),
					resource.TestCheckNoResourceAttr("steadycron_http_job.test", "cron_expression"),
				),
			},
		},
	})
}

func TestAccHTTPJob_bothScheduleFieldsFail(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccHTTPJobBothSchedulesConfig(),
				ExpectError: regexp.MustCompile(`exactly one of cron_expression or interval_seconds`),
			},
		},
	})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func testAccHTTPJobConfig(name, method, url, cronExpr string) string {
	return fmt.Sprintf(`
resource "steadycron_http_job" "test" {
  name            = %q
  method          = %q
  url             = %q
  cron_expression = %q
}
`, name, method, url, cronExpr)
}

func testAccHTTPJobIntervalConfig(name string, intervalSecs int) string {
	return fmt.Sprintf(`
resource "steadycron_http_job" "test" {
  name             = %q
  method           = "GET"
  url              = "https://httpbin.org/get"
  interval_seconds = %d
}
`, name, intervalSecs)
}

func testAccHTTPJobBothSchedulesConfig() string {
	return `
resource "steadycron_http_job" "test" {
  name             = "both-schedules"
  method           = "GET"
  url              = "https://httpbin.org/get"
  cron_expression  = "0 12 * * *"
  interval_seconds = 3600
}
`
}

// testCheckIDUnchanged captures the id before and checks it is stable across updates.
// Uses a package-level variable for simplicity across steps.
var capturedID string

func testCheckIDUnchanged(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %q not found", resourceName)
		}
		id := rs.Primary.ID
		if capturedID == "" {
			capturedID = id
			return nil
		}
		if id != capturedID {
			return fmt.Errorf("id changed from %q to %q — expected in-place update", capturedID, id)
		}
		return nil
	}
}
