package teamcity

import (
	"bytes"
	"fmt"
	"strings"

	api "github.com/64mb/go-teamcity/teamcity"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceFeaturePullRequests() *schema.Resource {
	return &schema.Resource{
		Create: resourceFeaturePullRequestsCreate,
		Read:   resourceFeaturePullRequestsRead,
		Delete: resourceFeaturePullRequestsDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"build_config_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"hosting": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"github"}, true),
			},
			"filter_author_role": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"MEMBER", "MEMBER_OR_COLLABORATOR", "EVERYBODY"}, true),
			},
			"github": {
				Type:     schema.TypeSet,
				ForceNew: true,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auth_type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"token", "password"}, true),
							ForceNew:     true,
						},
						"username": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"password": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
							Computed:  true,
							ForceNew:  true,
						},
						"access_token": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
							Computed:  true,
							ForceNew:  true,
						},
					},
				},
				Set: githubProviderOptionsHash,
			},
		},
	}
}

func resourceFeaturePullRequestsCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	var buildConfigID string

	if v, ok := d.GetOk("build_config_id"); ok {
		buildConfigID = v.(string)
	}

	// validates the Build Configuration exists
	if _, err := client.BuildTypes.GetByID(buildConfigID); err != nil {
		return fmt.Errorf("invalid build_config_id '%s' - Build configuration does not exist", buildConfigID)
	}

	srv := client.BuildFeatureService(buildConfigID)

	//Only Github publisher for now - Add support for more publishers later

	dt, err := buildGithubPullRequests(d)
	if err != nil {
		return err
	}
	out, err := srv.Create(dt)

	if err != nil {
		return err
	}

	d.SetId(out.ID())

	return resourceFeaturePullRequestsRead(d, meta)
}

func resourceFeaturePullRequestsRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client).BuildFeatureService(d.Get("build_config_id").(string))

	dt, err := getBuildFeaturePullRequests(d, client, d.Id())

	if dt == nil && err == nil {
		return nil
	}

	if err != nil {
		return err
	}

	if err := d.Set("build_config_id", dt.BuildTypeID()); err != nil {
		return err
	}

	//TODO: Implement other publishers
	if err := d.Set("hosting", "github"); err != nil {
		return err
	}

	opt := dt.Options.(*api.PullRequestsGithubOptions)

	var optsToSave []map[string]interface{}
	m := make(map[string]interface{})
	m["auth_type"] = opt.AuthenticationType

	if opt.AuthenticationType == "password" {
		m["username"] = opt.Username
	}

	optsToSave = append(optsToSave, m)
	return d.Set("github", optsToSave)
}

func resourceFeaturePullRequestsDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	svr := client.BuildFeatureService(d.Get("build_config_id").(string))

	return svr.Delete(d.Id())
}

func buildGithubPullRequests(d *schema.ResourceData) (api.BuildFeature, error) {
	var opt api.PullRequestsGithubOptions
	// MaxItems ensure at most 1 github element
	local := d.Get("github").(*schema.Set).List()[0].(map[string]interface{})
	authType := local["auth_type"].(string)
	filterAuthorRole := d.Get("filter_author_role").(string)
	switch strings.ToLower(authType) {
	case "token":
		opt = api.NewPullRequestsGithubOptionsToken(local["access_token"].(string), filterAuthorRole)
	case "password":
		opt = api.NewPullRequestsGithubOptionsPassword(local["username"].(string), local["password"].(string), filterAuthorRole)
	}

	return api.NewFeaturePullRequestsGithub(opt, "")
}

func getBuildFeaturePullRequests(d *schema.ResourceData, c *api.BuildFeatureService, id string) (*api.FeaturePullRequests, error) {
	dt, err := c.GetByID(id)

	if err != nil && strings.Contains(err.Error(), "404") {
		d.SetId("")
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	fcsp := dt.(*api.FeaturePullRequests)
	return fcsp, nil
}

func githubProviderOptionsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["auth_type"].(string)))

	if v, ok := m["username"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}
