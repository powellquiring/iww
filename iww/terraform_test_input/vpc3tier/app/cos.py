import ibm_boto3
import ibm_botocore
from ibm_botocore.client import Config, ClientError
import functools
import json

COS_ENDPOINT = None
COS_API_KEY_ID = None
COS_INSTANCE_CRN = None
COS_BUCKET_NAME = None
COS_OBJECT_KEY = None

def initialize_cos():
  # logging.basicConfig(encoding='utf-8', level=logging.DEBUG)
  global COS_ENDPOINT, COS_API_KEY_ID, COS_INSTANCE_CRN, COS_BUCKET_NAME, COS_OBJECT_KEY
  COS_ENDPOINT = "https://s3.us-south.cloud-object-storage.appdomain.cloud" # Current list avaiable at https://control.cloud-object-storage.cloud.ibm.com/v2/endpoints
  logging.info(f"COS_ENDPOINT '{COS_ENDPOINT}'")
  COS_API_KEY_ID = "5kiZrIDEAzPMvZrgMf5n2-MVkRYS-ehgo4S5RN0--XTe" # eg "W00YixxxxxxxxxxMB-odB-2ySfTrFBIQQWanc--P3byk"
  logging.info("COS_API_KEY:'HIDDEN'")
  COS_INSTANCE_CRN = "crn:v1:bluemix:public:cloud-object-storage:global:a/713c783d9a507a53135fe6793c37cc74:017a0d19-ded7-436e-bb36-c4984488a6c6::" # eg "crn:v1:bluemix:public:cloud-object-storage:global:a/3bf0d9003xxxxxxxxxx1c3e97696b71c:d6f04d83-6c4f-4a62-a165-696756d63903::"
  logging.info(f"COS_INSTANCE_CRN '{COS_INSTANCE_CRN}'")
  COS_BUCKET_NAME = 'vpc3tier-000-data'
  logging.info(f"COS_BUCKET_NAME '{COS_BUCKET_NAME}'")
  COS_OBJECT_KEY = "data"
  logging.info(f"COS_OBJECT_KEY '{COS_OBJECT_KEY}'")

def bucket_test():
  logging.info(f"bucket test {COS_BUCKET_NAME}")
  logging.info(f"bucket creation date {data_bucket().creation_date}")

@functools.lru_cache()
def s3_resource():
  # Constants for IBM COS values

  # Create resource
  s3 = ibm_boto3.resource("s3",
      ibm_api_key_id=COS_API_KEY_ID,
      ibm_service_instance_id=COS_INSTANCE_CRN,
      config=Config(signature_version="oauth"),
      endpoint_url=COS_ENDPOINT
  )
  return s3

def data_bucket():
  global COS_BUCKET_NAME
  return s3_resource().Bucket(COS_BUCKET_NAME)

def data_object():
  global COS_BUCKET_NAME, COS_OBJECT_KEY
  return s3_resource().Object(COS_BUCKET_NAME, COS_OBJECT_KEY)

def s3_increment_object(o):
  """Return the dict {count: num} where num is the number of times this function has been called"""
  initial = dict(count=0)
  def put_contents(body_str):
    o.put(Body=body_str.encode())
  def get_contents():
    response = o.get()
    body = response['Body'].read()
    body_str = body.decode()
    return body_str
  def increment_file():
    body_str = get_contents()
    start = json.loads(body_str)
    start['count'] = start['count'] + 1
    end_str = json.dumps(start)
    put_contents(end_str)
    return json.loads(end_str)
  def put_first_file():
    put_contents(json.dumps(initial))
    o.wait_until_exists()
  try:
    o.load()
  except ibm_botocore.exceptions.ClientError as e:
      if e.response['Error']['Code'] == "404":
          # The object does not exist.
          put_first_file()
          return increment_file()
      else:
          raise
  else:
    return increment_file()



@app.get("/cos")
def read_cos():
  bucket_test()
  return s3_increment_object(data_object())
