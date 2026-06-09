// Package provider implements the SteadyCron Terraform provider.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

var _ provider.Provider = &SteadyCronProvider{}

// SteadyCronProvider implements the Terraform provider.
type SteadyCronProvider struct {
	version string
}

// SteadyCronProviderModel holds the provider configuration state.
type SteadyCronProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	APIKey   types.String `tfsdk:"api_key"`
}

// New returns a new provider factory function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SteadyCronProvider{version: version}
	}
}

func (p *SteadyCronProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "steadycron"
	resp.Version = p.version
}

func (p *SteadyCronProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The SteadyCron provider manages HTTP jobs, heartbeat monitors, alert channels, " +
			"alert rules, tags, and template-variable names via the SteadyCron REST API.\n\n" +
			"A **Full**-scope API key is required for write operations; a **Read-only** key suffices for data sources.\n\n" +
			"Per-key rate limit is 120 req/min. The provider retries 429 responses automatically.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Base URL of the SteadyCron API. Defaults to `https://api.steadycron.com`. " +
					"Can also be set via the `STEADYCRON_ENDPOINT` environment variable.",
				Optional: true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "SteadyCron API key (`sc_...`). Requires a **Full**-scope key for writes. " +
					"Can also be set via the `STEADYCRON_API_KEY` environment variable. **Never commit this value.**",
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func (p *SteadyCronProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config SteadyCronProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("STEADYCRON_ENDPOINT")
	if !config.Endpoint.IsNull() && !config.Endpoint.IsUnknown() {
		endpoint = config.Endpoint.ValueString()
	}

	apiKey := os.Getenv("STEADYCRON_API_KEY")
	if !config.APIKey.IsNull() && !config.APIKey.IsUnknown() {
		apiKey = config.APIKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API key",
			"The SteadyCron provider requires an API key. Set it via the `api_key` provider attribute "+
				"or the `STEADYCRON_API_KEY` environment variable.",
		)
		return
	}

	c := client.New(endpoint, apiKey, p.version)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *SteadyCronProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewHTTPJobResource,
		NewHeartbeatMonitorResource,
		NewAlertChannelResource,
		NewAlertRuleResource,
		NewTagResource,
		NewTemplateVariableResource,
	}
}

func (p *SteadyCronProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewHTTPJobDataSource,
		NewHeartbeatMonitorDataSource,
		NewTagDataSource,
		NewAlertChannelDataSource,
	}
}
