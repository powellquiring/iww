package main

import (
	"errors"
	"strings"

	"github.com/IBM-Cloud/ibm-cloud-cli-sdk/bluemix/terminal"
	"github.com/IBM-Cloud/ibm-cloud-cli-sdk/plugin"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/powellquiring/iww/iww"
	"github.com/urfave/cli/v2"
)

type IwwPlugin struct{}

var ui = terminal.NewStdUI()
var context plugin.PluginContext

func main() {
	plugin.Start(new(IwwPlugin))
}

func sanitizeToken(token string) string {
	return strings.TrimPrefix(token, "Bearer ")
}

func getActiveIAMToken() string {
	/*
		// no point in looking for a token if the user isn't logged in
		if !context.IsLoggedIn() {
			return ""
		}
	*/

	// read current token
	token := sanitizeToken(context.IAMToken())
	// the token should never be empty while logged in,
	// but check for that just in case
	if token != "" {
		return token
	}

	/*
		TODO
		// check if token is still active
		tokenInfo := core_config.NewIAMTokenInfo(token)
		expireTime := tokenInfo.Expiry.Unix()
		thirtySeconds := int64(30)
		// if token is nearing expiration, refresh it
		// allow a 30 second buffer to ensure the token does
		// not expire while the rest of the code is executing
		if core.GetCurrentTime() > (expireTime - thirtySeconds) {
			newToken, err := context.RefreshIAMToken()
			if err != nil {
				return ""
			}

			token = sanitizeToken(newToken)
		}
	*/

	return token
}

func GetAuthenticator() (core.Authenticator, error) {
	if token := getActiveIAMToken(); token != "" {
		return core.NewBearerTokenAuthenticator(token)
	}
	return nil, errors.New("no-credentials")
}

func mainer(token, accountID, region, resourceGroupName, resourceGroupGUID string, args []string) {
	var vpcid string
	var crn string
	app := &cli.App{
		Name:  "iww",
		Usage: "ibm cloud world wide operations on existing resources",
		Commands: []*cli.Command{
			{
				Name:  "ls",
				Usage: "list matching resources",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "fast",
						Usage: "fast as possible do not read resource specific attributes",
					},
					&cli.BoolFlag{
						Name:    "all-resource-groups",
						Aliases: []string{"ag"},
						Usage:   "all resource groups not just the one configured (try: ibmcloud target)",
					},
					&cli.BoolFlag{
						Name:    "all-regions",
						Aliases: []string{"ar"},
						Usage:   "all regions not just the one configured (try: ibmcloud target)",
					},
					&cli.BoolFlag{
						Name:  "verbose",
						Usage: "fast as possible do not read resource specific attributes",
					},
					&cli.StringFlag{
						Name:        "vpcid",
						Aliases:     []string{"v"},
						Usage:       "restrict resources to be from one vpc id",
						Required:    false,
						Destination: &vpcid,
					},
				},
				Action: func(c *cli.Context) error {
					if c.Bool("all-resource-groups") {
						resourceGroupName = ""
						resourceGroupGUID = ""
					}
					if c.Bool("all-regions") {
						region = ""
					}
					return iww.LsWithToken(token, accountID, region, resourceGroupName, resourceGroupGUID, vpcid, c.Bool("fast"), c.Bool("verbose"))
				},
			},
			{
				Name:  "rm",
				Usage: "remove resources",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "all-regions",
						Aliases: []string{"ar"},
						Usage:   "all regions not just the one configured (try: ibmcloud target)",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Usage:   "fast as possible do not read resource specific attributes",
						Aliases: []string{"v"},
					},
					&cli.BoolFlag{
						Name:    "force",
						Usage:   "do not prompt with y/n just assume y and rm resources",
						Aliases: []string{"f"},
					},
					&cli.StringFlag{
						Name:        "vpcid",
						Usage:       "restrict resources to be from one vpc id",
						Required:    false,
						Destination: &vpcid,
					},
					&cli.StringFlag{
						Name:        "crn",
						Aliases:     []string{"c"},
						Usage:       "Delete on resource based on the crn",
						Required:    false,
						Destination: &crn,
					},
				},
				Action: func(c *cli.Context) error {
					if c.Bool("all-regions") {
						region = ""
					}
					return iww.RmWithToken(token, accountID, region, resourceGroupName, resourceGroupGUID, vpcid, crn, c.Bool("force"), c.Bool("verbose"))
				},
			},
		},
	}
	err := app.Run(append(args))
	if err != nil {
		ui.Failed(err.Error())
	}
}
func (p *IwwPlugin) Run(localContext plugin.PluginContext, args []string) {
	context = localContext
	token := sanitizeToken(context.IAMToken())
	if token == "" {
		ui.Failed("no-credentials")
		return
	}
	var resourceGroupName string
	var resourceGroupGUID string

	accountID := context.CurrentAccount().GUID

	if context.HasTargetedResourceGroup() {
		resourceGroupName = context.CurrentResourceGroup().Name
		resourceGroupGUID = context.CurrentResourceGroup().GUID
	}
	region := context.CurrentRegion()
	mainer(token, accountID, region.Name, resourceGroupName, resourceGroupGUID, args)
}
func (p *IwwPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "iww",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 10,
		},
		Commands: []plugin.Command{
			{
				Name:        "iww",
				Description: "IBM World Wide resources management.  Currently list and remove",
				Usage:       "ibmcloud iww",
			},
		},
	}
}
