package teamcity

import (
	"bytes"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"regexp"
	"strings"

	api "github.com/64mb/go-teamcity/teamcity"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceSshKey() *schema.Resource {
	return &schema.Resource{
		Read:   resourceSshKeyRead,
		Create: resourceSshKeyCreate,
		Delete: resourceSshKeyDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name to identify this SSH Key.",
				ForceNew:    true,
			},
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID for the parent project for this SSH Key. Required.",
				ForceNew:    true,
			},
			"payload": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Payload content of this SSH Key.",
				ForceNew:    true,
			},
			// "type": {
			// 	Type:         schema.TypeString,
			// 	Optional:     true,
			// 	Default:      "rsa",
			// 	ValidateFunc: validation.StringInSlice([]string{"rsa", "ed25519"}, true),
			// 	ForceNew:     true,
			// },
		},
	}
}

func resourceSshKeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	name := d.Get("name").(string)
	projectID := d.Get("project_id").(string)
	payload := d.Get("payload").(string)

	r := strings.NewReader(payload)

	buf := new(bytes.Buffer)
	bw := multipart.NewWriter(buf)

	p1w, _ := bw.CreateFormField("action")
	p1w.Write([]byte("createSshKey"))

	p2w, _ := bw.CreateFormField("projectId")
	p2w.Write([]byte(projectID))

	p3w, _ := bw.CreateFormField("fileName")
	p3w.Write([]byte(name))

	p4w, _ := bw.CreateFormFile("file:fileToUpload", name)
	io.Copy(p4w, r)

	bw.Close()

	request, _ := client.SlingClient().New().Set("content-type", bw.FormDataContentType()).Body(buf).Post("/admin/sshKeys.html").Request()

	res, err := client.HTTPClient.Do(request)

	if err != nil {
		return err
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	bodyString := string(bodyBytes)

	if strings.Contains(bodyString, "parent.BS.SshKeysDialog.error") {
		regex := regexp.MustCompile(`parent.BS.SshKeysDialog.error\("([^\)]+)"\)`)
		findError := regex.FindAllStringSubmatch(bodyString, -1)

		if len(findError) > 0 {
			return errors.New(findError[0][1])
		}
	}

	reqProjectId, _ := client.SlingClient().New().Get("/admin/editProject.html?projectId=" + projectID + "&tab=ssh-manager").Request()

	resProjectId, err := client.HTTPClient.Do(reqProjectId)

	if err != nil {
		return err
	}

	bodyBytesProject, err := io.ReadAll(resProjectId.Body)
	if err != nil {
		return err
	}
	bodyStringProject := string(bodyBytesProject)

	regex := regexp.MustCompile(`BS.SshKeysDialog.deleteKey\('([^']+)'`)
	findProject := regex.FindAllStringSubmatch(bodyStringProject, -1)

	projectInternalId := findProject[0][1]

	d.SetId(projectInternalId + "___ssh_key___" + name)

	return nil
}

func resourceSshKeyRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceSshKeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	seed := strings.Split(d.Id(), "___ssh_key___")

	name := ""
	projectID := ""

	if len(seed) > 1 {
		name = seed[1]
		projectID = seed[0]
	}

	log.Printf("[DEBUG]: resourceSshKeyDelete - Destroying ssh key %v", d.Id())

	buf := new(bytes.Buffer)
	bw := multipart.NewWriter(buf)

	p1w, _ := bw.CreateFormField("action")
	p1w.Write([]byte("deleteSshKey"))

	p2w, _ := bw.CreateFormField("projectId")
	p2w.Write([]byte(projectID))

	p3w, _ := bw.CreateFormField("keyName")
	p3w.Write([]byte(name))

	bw.Close()

	request, _ := client.SlingClient().New().Set("content-type", bw.FormDataContentType()).Body(buf).Post("/admin/sshKeysActions.html").Request()

	_, err := client.HTTPClient.Do(request)

	if err != nil {
		return err
	}

	log.Printf("[INFO]: resourceSshKeyDelete - Destroyed ssh key %v", d.Id())
	return nil
}
