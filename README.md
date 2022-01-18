# iww
ibm cli with a world wide view of resources, list (ls), remove (rm)
## iww cli
A release is currently not available, you will need to build it, sorry.
If you would rather use the `ibmcloud` cli see [Plugin](#Plugin) below
### Build
```
git clone https://github.com/powellquiring/iww
cd iww/cmd/iww
make
```

The iww command should have been created.  Use it in the next section

### Usage
To use the iww command directly an apikey is required.  The `ls` command lists resources.  Take a look and replace with the `rm` command to remove them.  Here is a typical usage pattern

Take a look at the resources in my account
```
export APIKEY=x
iww ls
```

At the top there may be a section of `#Missing resource instances`  this would call out resources that are in the Resource Controller, RC, but do not really exist.  File a support ticket to get rid of these.

Next you will see resources sorted by resource group:

```
#Resource instances
# 1b10f091c2ab4e1f8fa8e0100fb988ac ( usc4 )
is image usc4-travis-61938296 vpc crn:v1:bluemix:public:is:us-south:a/713c783d9a507a53135fe6793c37cc74::image:r006-f6fe0eb5-70ac-4d0e-87b7-205f3f6a47a0
# 074c1474b26e4118a3771e70d2affe19 ( Default )
schematics environment VSI AIO App Environment US - crn:v1:bluemix:public:schematics:us-south:a/713c783d9a507a53135fe6793c37cc74:f1eb1305-6add-4f84-baa2-17001a935b44:environment:us-east.ENVIRONMENT.VSI-AIO-App-Environment-US.5e72c0e2
schematics environment pfqfeedback - crn:v1:bluemix:public:schematics:us-south:a/713c783d9a507a53135fe6793c37cc74:f1eb1305-6add-4f84-baa2-17001a935b44:environment:us-east.ENVIRONMENT.pfqfeedback.d4a21b69
schematics workspace config - crn:v1:bluemix:public:schematics:us-south:a/713c783d9a507a53135fe6793c37cc74:f1eb1305-6add-4f84-baa2-17001a935b44:workspace:us-east.workspace.config.29dc96a9
schematics workspace enterprise - crn:v1:bluemix:public:schematics:us-south:a/713c783d9a507a53135fe6793c37cc74:f1eb1305-6add-4f84-baa2-17001a935b44:workspace:us-east.workspace.enterprise.209060d2
# b6503f25836d49029966ab5be7fe50b5 ( default )
internet-svcs  cis-master - crn:v1:bluemix:public:internet-svcs:global:a/713c783d9a507a53135fe6793c37cc74:142daab2-b230-4b6b-9d6c-16c89e28a2a0::
...
```

Now I notice that I want to delete all of the `usc4` resources so I list them one more time, then rm them:

```
$ ./iww ls --group usc4
#Resource instances
# 1b10f091c2ab4e1f8fa8e0100fb988ac ( usc4 )
is image usc4-travis-61938296 vpc crn:v1:bluemix:public:is:us-south:a/713c783d9a507a53135fe6793c37cc74::image:r006-f6fe0eb5-70ac-4d0e-87b7-205f3f6a47a0
$ ./iww rm --group usc4
start: is image -- vpc crn:v1:bluemix:public:is:us-south:a/713c783d9a507a53135fe6793c37cc74::image:r006-f6fe0eb5-70ac-4d0e-87b7-205f3f6a47a0
destroying is image usc4-travis-61938296 vpc crn:v1:bluemix:public:is:us-south:a/713c783d9a507a53135fe6793c37cc74::image:r006-f6fe0eb5-70ac-4d0e-87b7-205f3f6a47a0
destroying is image usc4-travis-61938296 vpc crn:v1:bluemix:public:is:us-south:a/713c783d9a507a53135fe6793c37cc74::image:r006-f6fe0eb5-70ac-4d0e-87b7-205f3f6a47a0
2022/01/18 17:42:45 VpcGenericOperation.Destroy Destroy err:An action was requested on a resource which is not supported at the current status of the resource.
```

It is in a loop trying to destroy resources until they no longer exist.  Although there were error messages generated in the above example the resource was deleted.  Try the `ls` or `rm` again to verify they are gone.

## Plugin
### Build
```
cd plugin/
make
```

### Usage
Then to use the plugin see the help:

```
plugin $ ibmcloud iww
NAME:
   iww - ibm cloud world wide operations on existing resources

USAGE:
   iww [global options] command [command options] [arguments...]

COMMANDS:
   ls       list matching resources
   rm       remove resources
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help (default: false)
plugin $ ibmcloud iww ls --help
NAME:
   iww ls - list matching resources

USAGE:
   iww ls [command options] [arguments...]

OPTIONS:
   --fast                       fast as possible do not read resource specific attributes (default: false)
   --all-resource-groups, --ag  all resource groups not just the one configured (try: ibmcloud target) (default: false)
   --all-regions, --ar          all regions not just the one configured (try: ibmcloud target) (default: false)
   --help, -h                   show help (default: false)

plugin $ ibmcloud iww rm --help
NAME:
   iww rm - remove resources

USAGE:
   iww rm [command options] [arguments...]

OPTIONS:
   --all-regions, --ar  all regions not just the one configured (try: ibmcloud target) (default: false)
   --help, -h           show help (default: false)
```

## Testing
In progress, not ready for general consumption, sorry ....
```
export TF_VAR_x=y
go test -v ./...
```

# Problems
## Issues with this code
File issues in this repository