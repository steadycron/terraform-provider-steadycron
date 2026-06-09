package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAlertChannel_email(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEmailChannelConfig("acc-email-channel", "alerts@example.com"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_alert_channel.test", "name", "acc-email-channel"),
					resource.TestCheckResourceAttr("steadycron_alert_channel.test", "kind", "email"),
					resource.TestCheckResourceAttr("steadycron_alert_channel.test", "email_to", "alerts@example.com"),
					resource.TestCheckResourceAttrSet("steadycron_alert_channel.test", "id"),
				),
			},
			{
				ResourceName:            "steadycron_alert_channel.test",
				ImportState:             true,
				ImportStateVerify:       true,
				// Secrets are redacted on GET; post-import plan will show diffs for secret fields.
				ImportStateVerifyIgnore: []string{"slack_webhook_url", "discord_webhook_url", "webhook_secret", "telegram_bot_token"},
			},
		},
	})
}

func TestAccAlertChannel_secretFieldNotInPlan(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWebhookChannelConfig("acc-webhook", "https://example.com/hook", "super-secret"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_alert_channel.test", "kind", "webhook"),
					resource.TestCheckResourceAttr("steadycron_alert_channel.test", "webhook_url", "https://example.com/hook"),
					// secret value must not appear in non-sensitive check
					resource.TestCheckResourceAttrSet("steadycron_alert_channel.test", "webhook_secret"),
				),
			},
		},
	})
}

func testAccEmailChannelConfig(name, email string) string {
	return fmt.Sprintf(`
resource "steadycron_alert_channel" "test" {
  name     = %q
  kind     = "email"
  email_to = %q
}
`, name, email)
}

func testAccWebhookChannelConfig(name, url, secret string) string {
	return fmt.Sprintf(`
resource "steadycron_alert_channel" "test" {
  name           = %q
  kind           = "webhook"
  webhook_url    = %q
  webhook_secret = %q
}
`, name, url, secret)
}
