#!/usr/bin/env python3

import urllib.request
from typing import Optional
from fastapi import FastAPI
import uvicorn
import logging
import functools
import socket
import platform
import os
import json
from  pathlib import Path

def logging_initialize():
  logging.basicConfig(level=logging.INFO)
  logging.info("starting")

class Cached2:
  """read and cache these properties"""
  def read_log_environment(self, environment_variable, log_contents=True, warn_not_found=False):
    """read an environment variable.  Log that it was read"""
    if environment_variable in os.environ:
      ret = os.environ[environment_variable]
      logging.info(f'{environment_variable} found in environment.  Value is {ret if log_contents else "hidden"}')
    else:
      if warn_not_found:
        logging.warning(f"environment variable {environment_variable} not in environment")
      else:
        logging.info(f"environment variable {environment_variable} not in environment")
      ret = None
    return ret

  @property
  @functools.lru_cache()
  def external_ip(self):
    """fip"""
    logging.info("external_ip check")
    try:
      ret = urllib.request.urlopen('https://ident.me', data=None, timeout=1).read().decode('utf8')
    except Exception as e:
      logging.warning(e)
      ret = "unknown"
    return ret

  @property
  @functools.lru_cache()
  def private_ip(self):
    """10.x"""
    try:
      s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
      s.connect(("8.8.8.8", 80))
      ret = s.getsockname()[0]
      s.close()
    except Exception as e:
      logging.warning(e)
      ret = "unkown"
    return ret

  @property
  @functools.lru_cache()
  def name(self):
    try:
      return platform.uname().node
    except Exception as e:
      logging.warning(e)
      ret = "unkown"
    return ret

  @property
  @functools.lru_cache()
  def remote_url(self):
    return self.read_log_environment("REMOTE_URL")

  @property
  @functools.lru_cache()
  def port(self):
    port = self.read_log_environment("PORT")
    return int(port) if port else 8000

  @property
  @functools.lru_cache()
  def front_back(self):
    """Expecting front or back"""
    return self.read_log_environment("FRONT_BACK", log_contents=True, warn_not_found=True)

  @property
  @functools.lru_cache()
  def front(self):
    return self.front_back == "front"

  @property
  @functools.lru_cache()
  def back(self):
    return self.front_back == "back"

class G:
  cache = Cached2()

def id():
  return {"uname": G.cache.name, "floatin_ip": G.cache.external_ip, "private_ip": G.cache.private_ip}

def remote_get(path):
  try:
    remote_url = f"{G.cache.remote_url}/{path}"
    ret_str = urllib.request.urlopen(remote_url, data=None, timeout=1).read().decode('utf8')
    try:
      ret = json.loads(ret_str)
    except Exception as e_inner:
      logging.warning(e_inner)
      logging.warning(ret_str)
      ret = {"notjson": str(ret_str)}
  except Exception as e:
    logging.warning(e)
    ret = {"error": f"error accessing {remote_url}"}
  return ret

# GLOBAL
count = 0
app = FastAPI()

@app.get("/")
def read_root():
  return id()

@app.get("/health")
def read_health():
  return {"status": "healthy"}

def id_increment_count():
  """Return the id() dict with an incremented count global variable"""
  global count
  count = count + 1
  return {**id(), "count": count}

@app.get("/increment")
def read_increment():
  ret = id_increment_count()
  if G.cache.remote_url:
    ret["remote"] = remote_get("increment")
  return ret

def load_pi(load:int):
  """https://en.wikipedia.org/wiki/Leibniz_formula_for_%CF%80"""
  k = 1
  power = 10 ** load
  # Initialize sum
  pi = 0
  for i in range(power):
      # even index elements are positive
      if i % 2 == 0:
          pi += 4/k
      else:
          # odd index elements are negative
          pi -= 4/k
      # denominator is odd
      k += 2
  return pi
     

@app.get("/cpu_load")
def cpu_load(load:int = 1):
  ret = dict(pi=load_pi(load))
  return ret

postgresql_import = None
def postgresql_initialize():
  global postgresql_import
  try:
    logging.info("check for the postgresql module, it would be in the postgresql.py file")
    import postgresql
    logging.info("postgresql module found")
    postgresql_import = postgresql
  except Exception as e:
    logging.info("no postgresql.py file")

@app.get("/postgresql")
def read_increment_postgresqlincrement():
  """Everything in /increment along with a postgresql database read and a remote postgresql read"""
  ret = id_increment_count()
  if G.cache.front:
    logging.info("application configured for front end, no postgresql")
  else:
    ret["postgresql"] = postgresql_import.get_increment_postgresql() if postgresql_import else "no postgresql module add postgresql.py to add support to this application"
  if G.cache.remote_url:
    ret["remote"] = remote_get("postgresql")
  return ret

logging_initialize()
postgresql_initialize()
if __name__ == "__main__":
  uvicorn.run(app, host="0.0.0.0", port=G.cache.port)