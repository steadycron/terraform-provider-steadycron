package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/steadycron/terraform-provider-steadycron/internal/client"
)

var _ datasource.DataSource = &TagDataSource{}

func NewTagDataSource() datasource.DataSource {
	return &TagDataSource{}
}

type TagDataSource struct {
	client *client.Client
}

func (d *TagDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (d *TagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a SteadyCron tag by its server-assigned `id`.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Required: true},
			"key":        schema.StringAttribute{Computed: true},
			"value":      schema.StringAttribute{Computed: true},
			"color":      schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *TagDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TagDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config struct {
		ID        types.String `tfsdk:"id"`
		Key       types.String `tfsdk:"key"`
		Value     types.String `tfsdk:"value"`
		Color     types.String `tfsdk:"color"`
		CreatedAt types.String `tfsdk:"created_at"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag, err := d.client.GetTag(ctx, config.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Tag not found", fmt.Sprintf("No tag with id %q was found.", config.ID.ValueString()))
			return
		}
		appendAPIError(&resp.Diagnostics, "reading tag data source", err)
		return
	}

	config.Key = types.StringValue(tag.Key)
	config.Value = types.StringValue(tag.Value)
	config.Color = types.StringValue(tag.Color)
	config.CreatedAt = types.StringValue(normalizeTimestamp(tag.CreatedAt))

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
