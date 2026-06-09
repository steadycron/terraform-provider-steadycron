package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTemplateVariable_basic(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTemplateVariableConfig("acc_test_token"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("steadycron_template_variable.test", "name", "acc_test_token"),
					resource.TestCheckResourceAttrSet("steadycron_template_variable.test", "id"),
					resource.TestCheckResourceAttrSet("steadycron_template_variable.test", "created_at"),
				),
			},
			{
				ResourceName:      "steadycron_template_variable.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// No perpetual diff on second apply
			{
				Config:   testAccTemplateVariableConfig("acc_test_token"),
				PlanOnly: true,
			},
		},
	})
}

func testAccTemplateVariableConfig(name string) string {
	return fmt.Sprintf(`
resource "steadycron_template_variable" "test" {
  name = %q
}
`, name)
}
