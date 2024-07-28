package teamcity

import (
	"fmt"
	"strings"

	api "github.com/64mb/go-teamcity/teamcity"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceFeatureSshAgent() *schema.Resource {
	return &schema.Resource{
		Create: resourceFeatureSshAgentCreate,
		Read:   resourceFeatureSshAgentRead,
		Delete: resourceFeatureSshAgentDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"build_config_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ssh_key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceFeatureSshAgentCreate(d *schema.ResourceData, meta interface{}) error {
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

	dt, err := buildSshAgent(d)
	if err != nil {
		return err
	}
	out, err := srv.Create(dt)

	if err != nil {
		return err
	}

	d.SetId(out.ID())

	return resourceFeatureSshAgentRead(d, meta)
}

func resourceFeatureSshAgentRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client).BuildFeatureService(d.Get("build_config_id").(string))

	dt, err := getBuildFeatureSshAgent(d, client, d.Id())

	if dt == nil && err == nil {
		return nil
	}

	if err != nil {
		return err
	}

	if err := d.Set("build_config_id", dt.BuildTypeID()); err != nil {
		return err
	}

	opt := dt.Options.(*api.SshAgentOptions)

	d.Set("ssh_key", opt.SshKey)

	return nil
}

func resourceFeatureSshAgentDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	svr := client.BuildFeatureService(d.Get("build_config_id").(string))

	return svr.Delete(d.Id())
}

func buildSshAgent(d *schema.ResourceData) (api.BuildFeature, error) {
	var opt api.SshAgentOptions

	sshKey := d.Get("ssh_key").(string)
	opt = api.NewSshAgentOptions(sshKey)

	return api.NewFeatureSshAgent(opt)
}

func getBuildFeatureSshAgent(d *schema.ResourceData, c *api.BuildFeatureService, id string) (*api.FeatureSshAgent, error) {
	dt, err := c.GetByID(id)

	if err != nil && strings.Contains(err.Error(), "404") {
		d.SetId("")
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	fcsp := dt.(*api.FeatureSshAgent)
	return fcsp, nil
}
