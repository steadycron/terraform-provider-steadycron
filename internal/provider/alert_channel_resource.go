package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

var _ resource.Resource = &AlertChannelResource{}
var _ resource.ResourceWithImportState = &AlertChannelResource{}

func NewAlertChannelResource() resource.Resource {
	return &AlertChannelResource{}
}

type AlertChannelResource struct {
	client *client.Client
}

// alertChannelModel is the Terraform state for steadycron_alert_channel.
// Config fields are flattened per-kind to avoid complex nested objects.
// Only the fields that match `kind` are used; others are left null.
type alertChannelModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Kind types.String `tfsdk:"kind"`

	// email config
	EmailTo types.String `tfsdk:"email_to"`

	// slack config
	SlackWebhookURL types.String `tfsdk:"slack_webhook_url"`

	// discord config
	DiscordWebhookURL types.String `tfsdk:"discord_webhook_url"`

	// webhook config
	WebhookURL    types.String `tfsdk:"webhook_url"`
	WebhookSecret types.String `tfsdk:"webhook_secret"`

	// telegram config
	TelegramBotToken types.String `tfsdk:"telegram_bot_token"`
	TelegramChatID   types.String `tfsdk:"telegram_chat_id"`

	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *AlertChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_channel"
}

func (r *AlertChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SteadyCron alert channel. " +
			"The required fields depend on the `kind`:\n\n" +
			"| kind | Required fields |\n" +
			"|---|---|\n" +
			"| `email` | `email_to` |\n" +
			"| `slack` | `slack_webhook_url` |\n" +
			"| `discord` | `discord_webhook_url` |\n" +
			"| `webhook` | `webhook_url` |\n" +
			"| `telegram` | `telegram_bot_token`, `telegram_chat_id` |\n\n" +
			"Secret fields (`slack_webhook_url`, `discord_webhook_url`, `webhook_secret`, `telegram_bot_token`) " +
			"are marked **Sensitive** and are never shown in plan output. The API does not return them on reads — " +
			"the provider preserves them from state to avoid perpetual diffs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Server-assigned UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable name for this channel.",
			},
			"kind": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Channel type. One of: `email`, `slack`, `discord`, `webhook`, `telegram`.",
				Validators: []validator.String{
					stringvalidator.OneOf("email", "slack", "discord", "webhook", "telegram"),
				},
			},
			// email
			"email_to": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Recipient email address. Required when `kind = \"email\"`.",
			},
			// slack
			"slack_webhook_url": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Slack incoming webhook URL. Required when `kind = \"slack\"`. **Sensitive.**",
			},
			// discord
			"discord_webhook_url": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Discord webhook URL. Required when `kind = \"discord\"`. **Sensitive.**",
			},
			// webhook
			"webhook_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Target URL for the webhook. Required when `kind = \"webhook\"`.",
			},
			"webhook_secret": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Optional HMAC signing secret for the webhook. **Sensitive.**",
			},
			// telegram
			"telegram_bot_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Telegram bot token. Required when `kind = \"telegram\"`. **Sensitive.**",
			},
			"telegram_chat_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Telegram chat ID or `@channel_username`. Required when `kind = \"telegram\"`.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 creation timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *AlertChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *AlertChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, err := channelModelToRequest(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid channel config", err.Error())
		return
	}

	ch, apiErr := r.client.CreateAlertChannel(ctx, apiReq)
	if apiErr != nil {
		appendAPIError(&resp.Diagnostics, "creating alert channel", apiErr)
		return
	}

	channelResponseToModel(ch, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AlertChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save secrets from state before overwriting with GET response (which redacts them).
	saved := state

	ch, err := r.client.GetAlertChannel(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		appendAPIError(&resp.Diagnostics, "reading alert channel", err)
		return
	}

	channelResponseToModel(ch, &state)
	restoreSecrets(&state, &saved)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AlertChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state alertChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, err := channelModelToRequest(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid channel config", err.Error())
		return
	}

	ch, apiErr := r.client.UpdateAlertChannel(ctx, state.ID.ValueString(), apiReq)
	if apiErr != nil {
		appendAPIError(&resp.Diagnostics, "updating alert channel", apiErr)
		return
	}

	channelResponseToModel(ch, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AlertChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAlertChannel(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			appendAPIError(&resp.Diagnostics, "deleting alert channel", err)
		}
	}
}

func (r *AlertChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ch, err := r.client.GetAlertChannel(ctx, req.ID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Alert channel not found", fmt.Sprintf("No alert channel with id %q was found.", req.ID))
			return
		}
		appendAPIError(&resp.Diagnostics, "importing alert channel", err)
		return
	}

	var state alertChannelModel
	channelResponseToModel(ch, &state)

	resp.Diagnostics.AddWarning(
		"Secret fields not imported",
		"The alert channel was imported successfully, but secret fields (webhook URLs, bot tokens) "+
			"are not returned by the API. Add them to your configuration to avoid a perpetual diff on next plan.",
	)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func channelModelToRequest(m alertChannelModel) (client.UpsertAlertChannelRequest, error) {
	kind := m.Kind.ValueString()
	req := client.UpsertAlertChannelRequest{
		Name: m.Name.ValueString(),
		Kind: kind,
	}

	switch kind {
	case "email":
		if m.EmailTo.IsNull() || m.EmailTo.ValueString() == "" {
			return req, fmt.Errorf("email_to is required when kind = \"email\"")
		}
		req.Config = map[string]string{"to": m.EmailTo.ValueString()}
	case "slack":
		if m.SlackWebhookURL.IsNull() || m.SlackWebhookURL.ValueString() == "" {
			return req, fmt.Errorf("slack_webhook_url is required when kind = \"slack\"")
		}
		req.Config = map[string]string{"webhook_url": m.SlackWebhookURL.ValueString()}
	case "discord":
		if m.DiscordWebhookURL.IsNull() || m.DiscordWebhookURL.ValueString() == "" {
			return req, fmt.Errorf("discord_webhook_url is required when kind = \"discord\"")
		}
		req.Config = map[string]string{"webhook_url": m.DiscordWebhookURL.ValueString()}
	case "webhook":
		if m.WebhookURL.IsNull() || m.WebhookURL.ValueString() == "" {
			return req, fmt.Errorf("webhook_url is required when kind = \"webhook\"")
		}
		cfg := map[string]string{"url": m.WebhookURL.ValueString()}
		if !m.WebhookSecret.IsNull() && m.WebhookSecret.ValueString() != "" {
			cfg["secret"] = m.WebhookSecret.ValueString()
		}
		req.Config = cfg
	case "telegram":
		if m.TelegramBotToken.IsNull() || m.TelegramBotToken.ValueString() == "" {
			return req, fmt.Errorf("telegram_bot_token is required when kind = \"telegram\"")
		}
		if m.TelegramChatID.IsNull() || m.TelegramChatID.ValueString() == "" {
			return req, fmt.Errorf("telegram_chat_id is required when kind = \"telegram\"")
		}
		req.Config = map[string]string{
			"bot_token": m.TelegramBotToken.ValueString(),
			"chat_id":   m.TelegramChatID.ValueString(),
		}
	default:
		return req, fmt.Errorf("unknown kind %q", kind)
	}

	return req, nil
}

func channelResponseToModel(ch *client.AlertChannelResponse, m *alertChannelModel) {
	m.ID = types.StringValue(ch.ID)
	m.Name = types.StringValue(ch.Name)
	m.Kind = types.StringValue(ch.Kind)
	m.CreatedAt = types.StringValue(normalizeTimestamp(ch.CreatedAt))

	// Parse config — non-secret fields are populated; secrets stay null if redacted.
	if ch.Config != nil {
		switch ch.Kind {
		case "email":
			if v, ok := ch.Config["to"]; ok {
				m.EmailTo = types.StringValue(v)
			}
		case "webhook":
			if v, ok := ch.Config["url"]; ok {
				m.WebhookURL = types.StringValue(v)
			}
		case "telegram":
			if v, ok := ch.Config["chat_id"]; ok {
				m.TelegramChatID = types.StringValue(v)
			}
		// slack and discord: webhook_url is the only field and it's redacted — nothing to populate.
		}
	}
}

// restoreSecrets copies secret field values from `from` to `to` when `to` has a null/empty value
// (indicating the API redacted the field on GET).
func restoreSecrets(to, from *alertChannelModel) {
	if to.SlackWebhookURL.IsNull() {
		to.SlackWebhookURL = from.SlackWebhookURL
	}
	if to.DiscordWebhookURL.IsNull() {
		to.DiscordWebhookURL = from.DiscordWebhookURL
	}
	if to.WebhookSecret.IsNull() {
		to.WebhookSecret = from.WebhookSecret
	}
	if to.TelegramBotToken.IsNull() {
		to.TelegramBotToken = from.TelegramBotToken
	}
}
