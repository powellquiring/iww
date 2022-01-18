#!/bin/bash

set -x

# parameters via substitution in the form __a__ done in terraform, look for all occurrances
# FRONT_BACK values of FRONT or BACK indicating the type of instance
# REMOTE_URL instances can reach out to a remote if provided
# MAIN_PY contents of the main.py python program
# POSTGRESQL_CREDENTIALS contents of the postgresql credentials

# ubuntu has a /root directory

# these will be empty or a values
__BASH_VARIABLES__

cat > /root/terraform_service_credentials.json << '__EOF__'
__POSTGRESQL_CREDENTIALS__
__EOF__

cat > /root/main.py << 'EOF'
__MAIN_PY__
EOF

# only include the postgresql file if there is postgres configured
if [ x$POSTGRESQL = xtrue ]; then
  cat > /root/postgresql.py << '__EOF'
__POSTGRESQL_PY__
__EOF
fi
# fix apt install it is prompting: Restart services during package upgrades without asking? <Yes><No>
export DEBIAN_FRONTEND=noninteractive

while ! ping -c 2 pypi.python.org; do
  sleep 1
done
apt update -y
apt install python3-pip -y
pip3 install --upgrade pip
pip3 install fastapi uvicorn psycopg2-binary

cat > /etc/systemd/system/threetier.service  << 'EOF'
[Service]
Environment="FRONT_BACK=__FRONT_BACK__"
Environment="REMOTE_URL=__REMOTE_URL__"
WorkingDirectory=/root
ExecStart=uvicorn main:app --host 0.0.0.0 --port 8000
EOF

systemctl start threetier

# logdna
if [ x$LOGDNA_INGESTION_KEY != x ]; then
  echo "deb https://repo.logdna.com stable main" | sudo tee /etc/apt/sources.list.d/logdna.list
  wget -O- https://repo.logdna.com/logdna.gpg | sudo apt-key add -
  sudo apt-get update
  sudo apt-get install logdna-agent < "/dev/null"
  sudo logdna-agent -k $LOGDNA_INGESTION_KEY
  sudo logdna-agent -s LOGDNA_APIHOST=api.private.${REGION}.logging.cloud.ibm.com
  sudo logdna-agent -s LOGDNA_LOGHOST=logs.private.${REGION}.logging.cloud.ibm.com
  # sudo logdna-agent -d /path/to/log/folders
  # sudo logdna-agent -t mytag,myothertag
  sudo update-rc.d logdna-agent defaults
  sudo /etc/init.d/logdna-agent start
fi
# sysdig
if [ x$SYSDIG_INGESTION_KEY != x ]; then
  curl -sL https://ibm.biz/install-sysdig-agent | sudo bash -s -- -a $SYSDIG_INGESTION_KEY -c ingest.private.$REGION.monitoring.cloud.ibm.com --collector_port 6443 --secure true -ac "sysdig_capture_enabled: false"
  apt-get -y install linux-headers-$(uname -r)
fi