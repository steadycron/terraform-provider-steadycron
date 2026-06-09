package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTag_basic(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTagConfig("env", "acc-test", "blue"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_tag.test", "key", "env"),
					resource.TestCheckResourceAttr("steadycron_tag.test", "value", "acc-test"),
					resource.TestCheckResourceAttr("steadycron_tag.test", "color", "blue"),
					resource.TestCheckResourceAttrSet("steadycron_tag.test", "id"),
					resource.TestCheckResourceAttrSet("steadycron_tag.test", "created_at"),
				),
			},
			{
				ResourceName:      "steadycron_tag.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccTagConfig(key, value, color string) string {
	return fmt.Sprintf(`
resource "steadycron_tag" "test" {
  key   = %q
  value = %q
  color = %q
}
`, key, value, color)
}
