package provider_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/steadycron/terraform-provider-steadycron/internal/provider"
)

// testAccProtoV6ProviderFactories is used in acceptance tests.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"steadycron": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// testAccPreCheck skips the test when STEADYCRON_API_KEY is absent.
// Call this from every acceptance test's PreCheck.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv("STEADYCRON_API_KEY"); v == "" {
		t.Skip("STEADYCRON_API_KEY must be set for acceptance tests")
	}
}
