#!/bin/sh
set -ex
echo onprem.sh

check_vars() {
  all_vars_set=true
  for var in DNS_SERVER_IPS; do 
    echo $var $(eval echo \$$var)
    if [ x = "x$(eval echo \$$var)" ]; then
      echo $var not initialized
      all_vars_set=false
    fi
  done
  if [ $all_vars_set = false ]; then
    exit 1
  fi
}

#edit the /etc/nplan/50-cloud-init.yaml file.  The result will be something like this
#where the 10.1.0.5 and 10.1.1.5 are the DNS IP addrsses of the private dns location in the cloud
#network:
#  ethernets:
#    ens3:
#      dhcp4: true
#      match:
#        macaddress: 02:00:0e:3e:fa:c3
#      nameservers:
#        addresses:
#        - 10.1.0.5
#        - 10.1.1.11
#      dhcp4-overrides:
#        use-dns: false
#      set-name: ens3
#  version: 2
dns() {
  cd /etc/netplan
  netplan_file=50-cloud-init.yaml
  python_script=$(cat <<__EOF
import yaml
f = open("$netplan_file")
y = yaml.safe_load(f)
y['network']['ethernets']['ens3']['nameservers'] = {'addresses': "$DNS_SERVER_IPS".split(" ")}
y['network']['ethernets']['ens3']['dhcp4-overrides'] = {'use-dns': False}
print(yaml.dump(y, default_flow_style=False))
__EOF
)
  python3 -c "$python_script" > $netplan_file.new
  mv $netplan_file $netplan_file.bu
  mv $netplan_file.new $netplan_file
  netplan apply
  systemd-resolve --flush-caches
}
dns2() {
  cat >> /etc/systemd/resolved.conf <<__EOF
DNS=$DNS_SERVER_IPS
__EOF
  systemctl restart systemd-resolved
  systemd-resolve --flush-caches
}

main() {
  echo onprem.sh main called
  check_vars
  #dns
  #dns2
  echo $DNS_SERVER_IPS > /root/DNS_SERVER_IPS
}

# -- variables will be concatinated here and then a call to main
