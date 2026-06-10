package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

var _ resource.Resource = &TemplateVariableResource{}
var _ resource.ResourceWithImportState = &TemplateVariableResource{}

func NewTemplateVariableResource() resource.Resource {
	return &TemplateVariableResource{}
}

type TemplateVariableResource struct {
	client *client.Client
}

type templateVariableModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *TemplateVariableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template_variable"
}

func (r *TemplateVariableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the **existence** of a SteadyCron template variable name. " +
			"Template variables are used as `{{name}}` placeholders in HTTP job URL, headers, and body fields — " +
			"the server substitutes the stored value at execution time.\n\n" +
			"**Values are write-only server-side** and are never returned by the API, so this resource manages " +
			"the name only. Set values via the SteadyCron dashboard or `steadycron vars set <name> <value>` in the CLI.\n\n" +
			"This resource will show **no perpetual diff** because only existence (the name) is tracked.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Server-assigned UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Variable name used in `{{name}}` placeholders. Must be unique within the account.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 creation timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 last-updated timestamp.",
			},
		},
	}
}

func (r *TemplateVariableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TemplateVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan templateVariableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tv, err := r.client.CreateTemplateVariable(ctx, client.UpsertTemplateVariableRequest{
		Name: plan.Name.ValueString(),
	})
	if err != nil {
		appendAPIError(&resp.Diagnostics, "creating template variable", err)
		return
	}

	tvResponseToModel(tv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TemplateVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state templateVariableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tv, err := r.client.GetTemplateVariable(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		appendAPIError(&resp.Diagnostics, "reading template variable", err)
		return
	}

	tvResponseToModel(tv, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TemplateVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan templateVariableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state templateVariableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tv, err := r.client.UpdateTemplateVariable(ctx, state.ID.ValueString(), client.UpsertTemplateVariableRequest{
		Name: plan.Name.ValueString(),
	})
	if err != nil {
		appendAPIError(&resp.Diagnostics, "updating template variable", err)
		return
	}

	tvResponseToModel(tv, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TemplateVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state templateVariableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteTemplateVariable(ctx, state.ID.ValueString()); err != nil {
		if !client.IsNotFound(err) {
			appendAPIError(&resp.Diagnostics, "deleting template variable", err)
		}
	}
}

func (r *TemplateVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tv, err := r.client.GetTemplateVariable(ctx, req.ID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Template variable not found", fmt.Sprintf("No template variable with id %q was found.", req.ID))
			return
		}
		appendAPIError(&resp.Diagnostics, "importing template variable", err)
		return
	}

	var state templateVariableModel
	tvResponseToModel(tv, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func tvResponseToModel(tv *client.TemplateVariableResponse, m *templateVariableModel) {
	m.ID = types.StringValue(tv.ID)
	m.Name = types.StringValue(tv.Name)
	m.CreatedAt = types.StringValue(tv.CreatedAt)
	m.UpdatedAt = types.StringValue(tv.UpdatedAt)
}
