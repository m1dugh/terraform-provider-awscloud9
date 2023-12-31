package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/cloud9"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/m1dugh/terraform-provider-awscloud9/internal/aws"
)

var (
	_ resource.Resource                = &SSHEnvironmentResource{}
	_ resource.ResourceWithConfigure   = &SSHEnvironmentResource{}
	_ resource.ResourceWithImportState = &SSHEnvironmentResource{}
)

type SSHEnvironmentResource struct {
	client *aws.AWSCloud9Client
}

func NewSSHEnvironmentResource() resource.Resource {
	return &SSHEnvironmentResource{}
}

type SSHEnvironmentResourceModel = SSHEnvironmentModel

func (rs *SSHEnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_environment"
}

func (rs *SSHEnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates a cloud 9 SSH environment",
		Attributes: map[string]schema.Attribute{
			"arn": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				MarkdownDescription: "The arn of the environment",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"id": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				MarkdownDescription: "The id of the environment",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the environment",
			},
			"description": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				MarkdownDescription: "The description of the environment",
			},
			"login_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The login name of the user to use the environment",
			},
			"hostname": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The hostname of the remote machine",
			},
			"port": schema.Int64Attribute{
				Default:  int64default.StaticInt64(22),
				Computed: true,
			},
			"environment_path": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The path for the environment",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"node_path": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The path to node.js on the remote host",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bastion_url": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				MarkdownDescription: "The ssh url to a bastion host",
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "A list of tags to attach",
				ElementType:         types.StringType,
				Required:            false,
				Optional:            true,
			},
		},
	}
}

func (rs *SSHEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*aws.AWSCloud9Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *aws.AWSCloud9Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	rs.client = client
}

func (rs *SSHEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SSHEnvironmentResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var request aws.CreateEnvironmentSSHRequest

	request.Name = plan.Name.ValueString()
	request.LoginName = plan.LoginName.ValueString()
	request.Hostname = plan.Hostname.ValueString()
	request.Port = int16(plan.Port.ValueInt64())

	calculatedTags := make([]aws.Tag, 0)
	for key, val := range plan.Tags.Elements() {
		tfVal, err := val.ToTerraformValue(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Convert Error", "Error converting value from tag")
		}
		var strVal string
		if err = tfVal.As(&strVal); err != nil {
			resp.Diagnostics.AddError("Convert Error", "Error converting value from tag")
		}
		calculatedTags = append(calculatedTags, aws.Tag{
			Key:   key,
			Value: strVal,
		})
	}
	request.Tags = calculatedTags

	if !plan.BastionURL.IsNull() {
		request.BastionHost = plan.BastionURL.ValueString()
	}

	if !plan.NodePath.IsNull() {
		request.NodePath = plan.NodePath.ValueString()
	}

	if !plan.EnvironmentPath.IsNull() {
		request.EnvironmentPath = plan.ID.ValueString()
	} else {
		plan.EnvironmentPath = types.StringValue("")
	}

	environment, err := rs.client.CreateEnvironmentSSH(&request)
	if err != nil {
		resp.Diagnostics.AddError("Client error", fmt.Sprintf("Unable to create environment %s, got error: %s", request.Name, err))
		return
	}

	plan.ID = types.StringValue(environment.EnvironmentId)
	readResults, err := rs.client.GetSSHEnvironments(environment.EnvironmentId)
	if err != nil {
		resp.Diagnostics.AddError("Client error", fmt.Sprintf("Could not read environment %s: %s", environment.EnvironmentId, err.Error()))
		return
	} else if len(readResults) == 0 {
		resp.Diagnostics.AddError("Client error", fmt.Sprintf("Could not read environment %s", environment.EnvironmentId))
		return
	}
	readResult := readResults[0]
	plan.NodePath = types.StringValue(readResult.NodePath)
	plan.EnvironmentPath = types.StringValue(readResult.EnvironmentPath)
	plan.Arn = types.StringValue(readResult.Arn)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func convertModelToPlan(state *SSHEnvironmentResourceModel, environment *aws.Cloud9SSHEnvironment) diag.Diagnostics {
	state.Arn = types.StringValue(environment.Arn)
	state.ID = basetypes.NewStringValue(environment.EnvironmentId)
	if len(environment.BastionHost) > 0 {
		state.BastionURL = types.StringValue(environment.BastionHost)
	} else {
		state.BastionURL = types.StringNull()
	}
	state.Name = basetypes.NewStringValue(environment.Name)
	state.LoginName = basetypes.NewStringValue(environment.LoginName)

	if len(environment.Description) > 0 {
		state.Description = types.StringValue(environment.Description)
	} else {
		state.Description = types.StringNull()
	}
	state.Port = basetypes.NewInt64Value(int64(environment.Port))
	state.Hostname = basetypes.NewStringValue(environment.Hostname)
	state.EnvironmentPath = basetypes.NewStringValue(environment.EnvironmentPath)
	state.NodePath = basetypes.NewStringValue(environment.NodePath)
	typedTags := make(map[string]attr.Value)
	for _, tag := range environment.Tags {
		typedTags[tag.Key] = basetypes.NewStringValue(tag.Value)
	}

	var diags diag.Diagnostics

	state.Tags, diags = basetypes.NewMapValue(types.StringType, typedTags)
	return diags
}

func (rs *SSHEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SSHEnvironmentResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	envId := state.ID.ValueString()
	environments, err := rs.client.GetSSHEnvironments(envId)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching env", fmt.Sprintf("Could not fetch env %s: %s", envId, err.Error()))
		return
	}

	if len(environments) == 0 {
		resp.Diagnostics.AddError("Environment not found", fmt.Sprintf("Environment not found: %s", envId))
		return
	}

	environment := environments[0]
	diags = convertModelToPlan(&state, &environment)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (rs *SSHEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SSHEnvironmentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	envId := state.ID.ValueString()
	_, err := rs.client.Cloud9.DeleteEnvironment(&cloud9.DeleteEnvironmentInput{
		EnvironmentId: &envId,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting env", fmt.Sprintf("Could not delete environment %s: %s", envId, err.Error()))
		return
	}
}

func (rs *SSHEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SSHEnvironmentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	planTags := make(map[string]string)
	diags = plan.Tags.ElementsAs(ctx, &planTags, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SSHEnvironmentResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateTags := make(map[string]string)
	diags = state.Tags.ElementsAs(ctx, &stateTags, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	removedTags := make([]*string, 0)
	addedTags := make([]*cloud9.Tag, 0)

	for stateKey, stateValue := range stateTags {
		found := false
		for planKey, planValue := range planTags {
			if planKey == stateKey {
				if stateValue != planValue {
					key := strings.Clone(planKey)
					val := strings.Clone(planValue)
					addedTags = append(addedTags, &cloud9.Tag{
						Key:   &key,
						Value: &val,
					})
				}
				found = true
				break
			}
		}
		if !found {
			key := strings.Clone(stateKey)
			removedTags = append(removedTags, &key)
		}
	}

	for planKey, planValue := range planTags {
		if _, ok := stateTags[planKey]; !ok {
			key := strings.Clone(planKey)
			val := strings.Clone(planValue)
			addedTags = append(addedTags, &cloud9.Tag{
				Key:   &key,
				Value: &val,
			})
		}
	}

	envId := plan.ID.ValueString()
	arn := plan.Arn.ValueString()
	updatedEnv := aws.Cloud9SSHEnvironment{
		EnvironmentId: envId,
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		LoginName:     plan.LoginName.ValueString(),
		Hostname:      plan.Hostname.ValueString(),
	}
	if !plan.Port.IsNull() {
		updatedEnv.Port = int16(plan.Port.ValueInt64())
	}
	if !plan.EnvironmentPath.IsNull() {
		updatedEnv.EnvironmentPath = plan.EnvironmentPath.ValueString()
	}
	if !plan.NodePath.IsNull() {
		updatedEnv.NodePath = plan.NodePath.ValueString()
	}
	if !plan.BastionURL.IsNull() {
		updatedEnv.BastionHost = plan.BastionURL.ValueString()
	}

	err := rs.client.UpdateEnvironment(updatedEnv)
	if err != nil {
		resp.Diagnostics.AddError("Error updating environment", fmt.Sprintf("Error updating environment %s: %s", envId, err.Error()))
		return
	}

	_, err = rs.client.Cloud9.UntagResource(&cloud9.UntagResourceInput{
		ResourceARN: &arn,
		TagKeys:     removedTags,
	})

	if err != nil {
		resp.Diagnostics.AddError("Error untagging environment", fmt.Sprintf("Error untagging environment %s: %s", envId, err.Error()))
		return
	}

	_, err = rs.client.Cloud9.TagResource(&cloud9.TagResourceInput{
		ResourceARN: &arn,
		Tags:        addedTags,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error tagging environment", fmt.Sprintf("Error tagging environment %s: %s", envId, err.Error()))
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (rs *SSHEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
