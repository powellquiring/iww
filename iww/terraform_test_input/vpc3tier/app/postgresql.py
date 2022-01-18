import logging
import psycopg2
import functools
from  pathlib import Path
import json
import base64

# Globals
POSTGRESQL_TABLE = "count"

class Cache:
  def read_log_terraform_postgresql_credentials(self, key, log_contents=True):
    """read a key from the terraform postgresql credentials.  Log it"""
    if key in self.terraform_postgresql_credentials:
      ret = self.terraform_postgresql_credentials[key]
      logging.info(f'{key} found in terraform_service_credentials.json.  Value is {ret if log_contents else "hidden"}')
    else:
      logging.warning(f"key {key} not in terraform_service_credentials.json")
      ret = None
    return ret
  def read_log_normal_postgresql_credentials(self, key, log_contents=True):
    """read a key from the normal postgresql credentials.  Log it.  The key is expressed in dot notation"""
    def get(keys, d):
      key = keys[0]
      try:
        n = d.get(key) if isinstance(d, dict) else d[int(key)]
      except Exception as e:
        return None
      if len(keys) == 1:
        return n
      else:
        return get(keys[1:], n)
    ret = get(key.split("."), self.normal_postgresql_credentials)
    if ret == None:
      logging.warning(f"key {key} not in terraform_service_credentials.json")
    else:
      logging.info(f'{key} found in terraform_service_credentials.json.  Value is {ret if log_contents else "hidden"}')
    return ret
  
  @property
  @functools.lru_cache()
  def python_directory(self):
    return Path(__file__).parent

  def __init__(self):
    self.postgresql_credentials_available = False
    self._connection = None
    service_credentials_path = Path(self.python_directory / "service_credentials.json")
    logging.info(f"looking for postgresql service credentials file in file {str(service_credentials_path.resolve())}")
    # choose the find_f function based on the type of credentials file found.  Terraform uses a funky json file format
    if service_credentials_path.exists():
      logging.info("using the postgresql service credentials json file from the ibm cloud console")
      self.postgresql_credentials_available = True
      logging.info("loading normal postgresql credentials")
      postgresql_credentials_s = service_credentials_path.read_text()
      self.normal_postgresql_credentials = json.loads(postgresql_credentials_s)
      find_f = self.read_log_normal_postgresql_credentials
    else:
      terraform_service_credentials_path = Path(self.python_directory / "terraform_service_credentials.json")
      logging.info(f"looking for terraform generated postgresql service credentials file in file {str(terraform_service_credentials_path.resolve())}")
      if terraform_service_credentials_path.exists():
        logging.info("using the terraform generated credentials file terraform_service_credentials.json")
        postgresql_credentials_s = terraform_service_credentials_path.read_text()
        if postgresql_credentials_s.strip() == f'__{"POSTGRESQL_CREDENTIALS"}__':
          logging.info(f"no terraform postgresql credentials in file {terraform_service_credentials_path.resolve()}")
          return
        else:
          logging.info("loading terraform postgresql credentials")
          self.terraform_postgresql_credentials = json.loads(postgresql_credentials_s)
          find_f = self.read_log_terraform_postgresql_credentials
        
    # use the find function to resolve the cached values
    self.postgresql_host = find_f("connection.postgres.hosts.0.hostname")
    self.postgresql_port = find_f("connection.postgres.hosts.0.port")
    self.postgresql_user = find_f("connection.postgres.authentication.username")
    self.postgresql_password = find_f("connection.cli.environment.PGPASSWORD", log_contents=False)
    self.postgresql_certificate_base64_s = find_f("connection.postgres.certificate.certificate_base64")
    self.postgresql_credentials_available = True
    certificate_path = self.python_directory / "cert"
    self.postgresql_certificate_file = str(certificate_path.resolve())
    if certificate_path.exists():
      logging.info(f"using the existing postgresql certificate file: {self.postgresql_certificate_file}")
    else:
      logging.info(f"creating a new the existing postgresql certificate file: {self.postgresql_certificate_file}")
      base64_s = self.postgresql_certificate_base64_s
      certificate_s = base64.b64decode(base64_s)
      certificate_path.write_bytes(certificate_s)

  @property
  def connection(self):
    if self._connection:
      return self._connection
    else:
      try:
        logging.info("postgresql connect")
        self._connection = psycopg2.connect(
          host=self.postgresql_host,
          port=self.postgresql_port,
          user=self.postgresql_user,
          password=self.postgresql_password,
          sslmode="verify-full", # not parsing queryoptions = "?sslmode=verify-full"
          sslrootcert=self.postgresql_certificate_file,
          database="ibmclouddb" # fixed
          )
        return self._connection
      except Exception as e: 
        return None

class G:
  cache = Cache()

def postgresql_table_create():
  """Validate or create a new table with one row"""
  table_exists = True
  with G.cache.connection:
    with G.cache.connection.cursor() as cur:
      try:
        cur.execute(f"SELECT * FROM {POSTGRESQL_TABLE};")
      except psycopg2.errors.UndefinedTable as eee:
        table_exists = False

  if not table_exists:
    with G.cache.connection:
      with G.cache.connection.cursor() as cur:
        try:
          sql = f"CREATE TABLE {POSTGRESQL_TABLE} (id INTEGER PRIMARY KEY, count INTEGER);"
          cur.execute(sql)
        except Exception as e:
          print(e)
          raise
  
  with G.cache.connection:
    with G.cache.connection.cursor() as cur:
      delete_table = False
      cur.execute(f"SELECT * from {POSTGRESQL_TABLE};")
      initial_rowcount = cur.rowcount
      if initial_rowcount == 1:
        row = cur.fetchone()
        logging.info(f"initial table contents {row}")
        if row[0] != 0:
          logging.info(f"expecting id to be 0, will delete table contents")
          delete_table = True
      if initial_rowcount > 1:
        logging.warning(f"expected 1 row in table got {cur.rowcount}, will delete table")
        delete_table = True
      if delete_table:
        cur.execute(f"DELETE FROM {POSTGRESQL_TABLE};")
      if delete_table or initial_rowcount == 0:
        cur.execute(f"INSERT INTO {POSTGRESQL_TABLE} (id, count) VALUES (%s, %s)", (0, 0))


def postgresql_increment():
  """Return the dict {count: num} where num is the number of times this function has been called update the table with the new count"""
  ret = dict(count=-1)
  logging.info(f"postgresql_increment increment count in row in table")
  with G.cache.connection:
    with G.cache.connection.cursor() as cur:
      cur.execute(f"SELECT * FROM {POSTGRESQL_TABLE};")
      assert cur.rowcount == 1
      row = cur.fetchone()
      assert row[0] == 0
      count = row[1] + 1
      cur.execute(f"UPDATE {POSTGRESQL_TABLE} SET count=(%s) WHERE id = (%s)", (count, 0))
      ret = dict(count=count)
  return ret

def teststuff():
  logging.info(f"server_version: {G.cache.connection.server_version}")
  for parameter in ("server_version", "server_encoding", "client_encoding", "is_superuser", "session_authorization", "DateStyle", "TimeZone", "integer_datetimes"):
    logging.info(f"connection parameter {parameter}: {G.cache.connection.get_parameter_status(parameter)}")
  # verify databases can be read
  with G.cache.connection:
    with G.cache.connection.cursor() as cur:
      logging.info(f"cursor scrollable: {cur.scrollable}")

    logging.info("postgresql read databases")
    with G.cache.connection.cursor() as cur:
      cur.execute("""SELECT datname from pg_database""")
      logging.info(f"cursor description: {cur.description}")
      rows = cur.fetchall()
      for row in rows:
        logging.info(f"row {row[0]}")
    with G.cache.connection.cursor() as cur:
      cur.execute("""SELECT datname from pg_database""")
      for row in cur:
        logging.info(f"row {row[0]}")



def get_increment_postgresql():
  """return a dictionary for the count"""
  if G.cache._connection == None:
    postgresql_table_create()
  return postgresql_increment()

#if __name__ == "__main__":
#  get_increment_postgresql() 