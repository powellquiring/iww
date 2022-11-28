package main

import (
	"log"
	"os"

	"github.com/powellquiring/iww/iww"
	"github.com/urfave/cli/v2"
)

func main() {
	var apikey string
	var resourceGroup string
	var fileName string
	var region string
	var vpcid string
	app := &cli.App{
		Name:  "iww",
		Usage: "ibm cloud world wide operations on existing resources",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "apikey",
				Usage:       "apikey key to access resources",
				Required:    true,
				Destination: &apikey,
				EnvVars:     []string{"APIKEY"},
			},
		},
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
						Name:    "verbose",
						Usage:   "fast as possible do not read resource specific attributes",
						Aliases: []string{"v"},
					},
					&cli.StringFlag{
						Name:        "group",
						Aliases:     []string{"g"},
						Usage:       "resource group for resources",
						Required:    false,
						Destination: &resourceGroup,
					},
					&cli.StringFlag{
						Name:        "region",
						Aliases:     []string{"r"},
						Usage:       "restrict resources to be from one specific region, us-south, ....",
						Required:    false,
						Destination: &region,
					},
					&cli.StringFlag{
						Name:        "vpcid",
						Aliases:     []string{"vpc"},
						Usage:       "restrict resources to be from one vpc id",
						Required:    false,
						Destination: &vpcid,
					},
				},
				Action: func(c *cli.Context) error {
					return iww.Ls(apikey, region, resourceGroup, vpcid, c.Bool("fast"), c.Bool("verbose"))
				},
			},
			{
				Name:  "rm",
				Usage: "remove resources",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "verbose",
						Usage:   "fast as possible do not read resource specific attributes",
						Aliases: []string{"v"},
					},
					&cli.StringFlag{
						Name:        "group",
						Aliases:     []string{"g"},
						Usage:       "resource group for resources",
						Required:    false,
						Destination: &resourceGroup,
					},
					&cli.StringFlag{
						Name:        "region",
						Aliases:     []string{"r"},
						Usage:       "restrict resources to be from one specific region, us-south, ....",
						Required:    false,
						Destination: &region,
					},
					&cli.StringFlag{
						Name:        "file",
						Usage:       "take the list of resources to rm from a file, first word in each line must be a crn",
						Required:    false,
						Destination: &fileName,
					},
					&cli.StringFlag{
						Name:        "vpcid",
						Aliases:     []string{"vpc"},
						Usage:       "restrict resources to be from one vpc id",
						Required:    false,
						Destination: &vpcid,
					},
				},
				Action: func(c *cli.Context) error {
					return iww.Rm(apikey, region, resourceGroup, fileName, vpcid, c.Bool("verbose"))
				},
			},
			{
				Name:  "test",
				Usage: "test existence of resources",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "group",
						Aliases:     []string{"g"},
						Usage:       "resource group for resources",
						Required:    false,
						Destination: &resourceGroup,
					},
					&cli.StringFlag{
						Name:        "region",
						Aliases:     []string{"r"},
						Usage:       "restrict resources to be from one specific region, us-south, ....",
						Required:    false,
						Destination: &region,
					},
					&cli.StringFlag{
						Name:        "file",
						Usage:       "take the list of resources to rm from a file, first word in each line must be a crn",
						Required:    false,
						Destination: &fileName,
					},
				},
				Action: func(c *cli.Context) error {
					return iww.Tst(apikey, region, resourceGroup)
				},
			},
			{
				Name:  "tag",
				Usage: "tag matching resources",
				Action: func(c *cli.Context) error {
					return iww.Tag(apikey, c.Args().First())
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

/*
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/iamidentityv1"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
)

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/IBM/go-sdk-core/core"
)


func badFlags() {
	fmt.Fprintln(os.Stderr, "cleanrg resource-group-name")
	flag.PrintDefaults()
	os.Exit(1)
}
func parseFlags() (rgName string, apikey string, force bool) {
	_force := flag.Bool("force", false, "force a delete")
	_apikey := flag.String("apikey", "", "apikey must be provided or in the APIKEY environment variable")
	flag.Parse()
	rgName = flag.Arg(0)
	if rgName == "" {
		badFlags()
	}
	force = *_force
	fmt.Println("rgName:", rgName)
	fmt.Println("force:", force)

	apikey = *_apikey
	if apikey == "" {
		apikey = os.Getenv("APIKEY")
	}
	if apikey == "" {
		badFlags()
	}
	return
}

func exit() {
	os.Exit(1)
}

func printErrExit(err error) {
	if err != nil {
		fmt.Println(err)
		exit()
	}
}

func getIamIdentityService(authenticator *core.IamAuthenticator) (iamIdentity *iamidentityv1.IamIdentityV1, err error) {
	iamClientOptions := &iamidentityv1.IamIdentityV1Options{
		Authenticator: authenticator,
	}
	return iamidentityv1.NewIamIdentityV1UsingExternalConfig(iamClientOptions)
}

func getResourceManagerService(authenticator *core.IamAuthenticator) (iamIdentity *resourcemanagerv2.ResourceManagerV2, err error) {
	iamClientOptions := &resourcemanagerv2.ResourceManagerV2Options{
		Authenticator: authenticator,
	}
	return resourcemanagerv2.NewResourceManagerV2(iamClientOptions)
}

func resourceGroupFind(returnListResourceGroups *resourcemanagerv2.ResourceGroupList, rgName string) (resourcemanagerv2.ResourceGroup, bool) {
	for _, resourceGroup := range returnListResourceGroups.Resources {
		fmt.Println(*resourceGroup.Name)
		if *resourceGroup.Name == rgName {
			return resourceGroup, true
		}
	}
	return resourcemanagerv2.ResourceGroup{}, false
}

func main() {
	rgName, apikey, force := parseFlags()
	fmt.Println(rgName, force)

	authenticator := &core.IamAuthenticator{
		ApiKey: apikey,
	}

	iamIdentityService, err := getIamIdentityService(authenticator)
	printErrExit(err)

	do := &iamidentityv1.GetAPIKeysDetailsOptions{
		IamAPIKey: &apikey,
	}
	apiKeyDetails, _, err := iamIdentityService.GetAPIKeysDetails(do)
	printErrExit(err)

	resourceManagerService, err := getResourceManagerService(authenticator)
	groupOptions := &resourcemanagerv2.ListResourceGroupsOptions{
		AccountID: apiKeyDetails.AccountID,
	}
	returnListResourceGroups, _, err := resourceManagerService.ListResourceGroups(groupOptions)
	printErrExit(err)

	resourceGroup, found := resourceGroupFind(returnListResourceGroups, rgName)
	if !found {
		fmt.Println("resource group not found:", rgName)
		exit()
	}
	fmt.Println(resourceGroup.Name)
	fmt.Println(resourceGroup.Name)
}

/*



	authenticator := &core.GetAuthenticatorFromEnvironment("IAM_IDENTITY")
	exampleUserAccountID = authenticator.

	serviceClientOptions := &resourcemanagerv2.ResourceManagerV2Options{}
	// serviceClient, err := resourcemanagerv2.NewResourceManagerV2UsingExternalConfig(serviceClientOptions)
	resourceManagerService, err := resourcemanagerv2.NewResourceManagerV2UsingExternalConfig(serviceClientOptions)
	if err != nil {
		fmt.Println(err)
		return
	}

	getAPIKeysDetailsOptions := iamIdentityService.NewGetAPIKeysDetailsOptions()
	getAPIKeysDetailsOptions.SetIamAPIKey(iamAPIKey)
	getAPIKeysDetailsOptions.SetIncludeHistory(false)

	apiKey, response, err := iamIdentityService.GetAPIKeysDetails(getAPIKeysDetailsOptions)
	if err != nil {
		panic(err)
	}
	b, _ := json.MarshalIndent(apiKey, "", "  ")
	fmt.Println(string(b))

	listResourceGroupsOptions := resourceManagerService.NewListResourceGroupsOptions()
	listResourceGroupsOptions.SetAccountID(exampleUserAccountID)
	listResourceGroupsOptions.SetIncludeDeleted(true)

	resourceGroupList, response, err := resourceManagerService.ListResourceGroups(listResourceGroupsOptions)
	if err != nil {
		panic(err)
	}
	b, _ := json.MarshalIndent(resourceGroupList, "", "  ")
	fmt.Println(string(b))

}

*/
