package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

var _ resource.Resource = &TagResource{}
var _ resource.ResourceWithImportState = &TagResource{}

func NewTagResource() resource.Resource {
	return &TagResource{}
}

type TagResource struct {
	client *client.Client
}

type tagModel struct {
	ID        types.String `tfsdk:"id"`
	Key       types.String `tfsdk:"key"`
	Value     types.String `tfsdk:"value"`
	Color     types.String `tfsdk:"color"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *TagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *TagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SteadyCron tag (`key=value` pair with an optional color). " +
			"Tags are account-scoped — the same tag is shared across all jobs that reference it. " +
			"Reference a tag's `id` in `steadycron_http_job.tags` or `steadycron_heartbeat_monitor.tags`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Server-assigned UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Tag key (e.g. `env`). Allowed characters: `[a-z0-9_-]`, max 32 characters.",
			},
			"value": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Tag value (e.g. `prod`). Allowed characters: `[a-z0-9_-]`, max 64 characters.",
			},
			"color": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Named hue token from the SteadyCron design palette (e.g. `green`, `blue`). " +
					"When empty the server auto-derives a color.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 creation timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *TagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag, err := r.client.CreateTag(ctx, client.UpsertTagRequest{
		Key:   plan.Key.ValueString(),
		Value: plan.Value.ValueString(),
		Color: plan.Color.ValueString(),
	})
	if err != nil {
		appendAPIError(resp.Diagnostics, "creating tag", err)
		return
	}

	tagResponseToModel(tag, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag, err := r.client.GetTag(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		appendAPIError(resp.Diagnostics, "reading tag", err)
		return
	}

	tagResponseToModel(tag, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan tagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state tagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag, err := r.client.UpdateTag(ctx, state.ID.ValueString(), client.UpsertTagRequest{
		Key:   plan.Key.ValueString(),
		Value: plan.Value.ValueString(),
		Color: plan.Color.ValueString(),
	})
	if err != nil {
		appendAPIError(resp.Diagnostics, "updating tag", err)
		return
	}

	tagResponseToModel(tag, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteTag(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			appendAPIError(resp.Diagnostics, "deleting tag", err)
		}
	}
}

func (r *TagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tag, err := r.client.GetTag(ctx, req.ID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Tag not found", fmt.Sprintf("No tag with id %q was found.", req.ID))
			return
		}
		appendAPIError(resp.Diagnostics, "importing tag", err)
		return
	}

	var state tagModel
	tagResponseToModel(tag, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func tagResponseToModel(tag *client.TagResponse, m *tagModel) {
	m.ID = types.StringValue(tag.ID)
	m.Key = types.StringValue(tag.Key)
	m.Value = types.StringValue(tag.Value)
	m.Color = types.StringValue(tag.Color)
	m.CreatedAt = types.StringValue(tag.CreatedAt)
}
