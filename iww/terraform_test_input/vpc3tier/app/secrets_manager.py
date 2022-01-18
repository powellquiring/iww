from ibm_cloud_sdk_core.authenticators.iam_authenticator import IAMAuthenticator
from ibm_secrets_manager_sdk.secrets_manager_v1 import *
import ibm_secrets_manager_sdk

class SecretsManagerDeleteme:
  def service_url(region, instance_id):
    return f"https://{instance_id}.{region}.secrets-manager.appdomain.cloud"

  def my_service_url():
    region = "us-south"
    # ibm_resource_instance.secrets_manager.guid in terraform
    instance_id = "192ff5df-3f9e-4a32-965e-1b70adddd53c"
    return service_url(region, instance_id)

  def api_key():
    return "SOMEAPIKEY"

  def environ_check() -> bool():
    apikey_var =  "SECRETS_MANAGER_APIKEY" 
    if not apikey_var in os.environ:
      print(f"Environment must contai {apikey_var}")
      return False
    return True

  # Documented:
  # sm = ibm_secrets_manager_sdk.IbmCloudSecretsManagerApiV1()
  # sm = ibm_secrets_manager_sdk.SecretsManagerV1()

  # authenticator=IAMAuthenticator(apikey=api_key())

  # Environment
  #      "env": {
  #        "SECRETS_MANAGER_AUTH_TYPE": "iam",
  #        "SECRETS_MANAGER_APIKEY": "SOMEAPIKEY"
  #      }

  def investigate_secrets_manager():
    if not environ_check():
      exit(1)

    authenticator=get_authenticator_from_environment(SecretsManagerV1.DEFAULT_SERVICE_NAME)

    secretsManager = SecretsManagerV1(
      authenticator=authenticator
    )

    secretsManager.set_service_url(my_service_url())
    response = secretsManager.list_secret_groups().get_result()

    print(json.dumps(response, indent=2))
    print("done")

