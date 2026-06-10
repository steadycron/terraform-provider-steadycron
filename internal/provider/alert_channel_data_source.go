package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

var _ datasource.DataSource = &AlertChannelDataSource{}

func NewAlertChannelDataSource() datasource.DataSource {
	return &AlertChannelDataSource{}
}

type AlertChannelDataSource struct {
	client *client.Client
}

func (d *AlertChannelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_channel"
}

func (d *AlertChannelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a SteadyCron alert channel by its server-assigned `id`.\n\n" +
			"**Note:** secret fields (webhook URLs, bot tokens) are redacted by the API and will be null in the result.",
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Required: true},
			"name":             schema.StringAttribute{Computed: true},
			"kind":             schema.StringAttribute{Computed: true},
			"email_to":         schema.StringAttribute{Computed: true},
			"webhook_url":      schema.StringAttribute{Computed: true},
			"telegram_chat_id": schema.StringAttribute{Computed: true},
			"created_at":       schema.StringAttribute{Computed: true},
		},
	}
}

func (d *AlertChannelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *AlertChannelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config struct {
		ID             types.String `tfsdk:"id"`
		Name           types.String `tfsdk:"name"`
		Kind           types.String `tfsdk:"kind"`
		EmailTo        types.String `tfsdk:"email_to"`
		WebhookURL     types.String `tfsdk:"webhook_url"`
		TelegramChatID types.String `tfsdk:"telegram_chat_id"`
		CreatedAt      types.String `tfsdk:"created_at"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ch, err := d.client.GetAlertChannel(ctx, config.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Alert channel not found", fmt.Sprintf("No alert channel with id %q was found.", config.ID.ValueString()))
			return
		}
		appendAPIError(&resp.Diagnostics, "reading alert channel data source", err)
		return
	}

	config.Name = types.StringValue(ch.Name)
	config.Kind = types.StringValue(ch.Kind)
	config.CreatedAt = types.StringValue(ch.CreatedAt)

	if ch.Config != nil {
		switch ch.Kind {
		case "email":
			if v, ok := ch.Config["to"]; ok {
				config.EmailTo = types.StringValue(v)
			}
		case "webhook":
			if v, ok := ch.Config["url"]; ok {
				config.WebhookURL = types.StringValue(v)
			}
		case "telegram":
			if v, ok := ch.Config["chat_id"]; ok {
				config.TelegramChatID = types.StringValue(v)
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
