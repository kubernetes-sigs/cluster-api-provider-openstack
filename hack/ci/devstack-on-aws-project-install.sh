#!/usr/bin/env bash

# Copyright 2021 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# hack script for preparing AWS to run cluster-api-provider-openstack e2e

set -o errexit -o nounset -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/../../
cd "${REPO_ROOT}" || exit 1
REPO_ROOT_ABSOLUTE=$(pwd)

CLUSTER_NAME=${CLUSTER_NAME:-"capo-e2e"}

AWS_REGION=${AWS_REGION:-"eu-central-1"}
AWS_ZONE=${AWS_ZONE:-"eu-central-1a"}
# AMIs:
# * capa-ami-ubuntu-20.04-1.20.4-00-1613898574 id: ami-0120656d38c206057
# * ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-20210223 id: ami-0767046d1677be5a0
AWS_AMI=${AWS_AMI:-"ami-0767046d1677be5a0"}
# Choose via: https://eu-central-1.console.aws.amazon.com/ec2/v2/home?region=eu-central-1#InstanceTypes:
AWS_MACHINE_TYPE=${AWS_MACHINE_TYPE:-"c5.metal"}
AWS_NETWORK_NAME=${AWS_NETWORK_NAME:-"${CLUSTER_NAME}-mynetwork"}
# prepare with:
# * create key pair:
# aws ec2 create-key-pair --key-name capo-e2e --query 'KeyMaterial' --region "${AWS_REGION}" --output text > ~/.ssh/aws-capo-e2e
# * add to local agent and generate public key:
# ssh-add ~/.ssh/aws-capo-e2e
# ssh-keygen -y -f ~/.ssh/aws-capo-e2e > ~/.ssh/aws-capo-e2e.pub
AWS_KEY_PAIR=${AWS_KEY_PAIR:-"capo-e2e"}
# disable pagination of AWS cli
export AWS_PAGER=""

OPENSTACK_RELEASE=${OPENSTACK_RELEASE:-"victoria"}
OPENSTACK_ENABLE_HORIZON=${OPENSTACK_ENABLE_HORIZON:-"false"}
# Flavors are default or preinstalled: (AWS script currently only supports default)
# * default: installs devstack via cloud-init
#   * OPENSTACK_RELEASE only works on default
# * preinstalled: uses a already installed devstack
FLAVOR=${FLAVOR:="default"}

echo "Using: AWS_REGION: ${AWS_REGION} AWS_NETWORK_NAME: ${AWS_NETWORK_NAME}"

# retry $1 times with $2 sleep in between
function retry {
  attempt=0
  max_attempts=${1}
  interval=${2}
  shift; shift
  until [[ ${attempt} -ge "${max_attempts}" ]] ; do
    attempt=$((attempt+1))
    set +e
    eval "$*" && return || echo "failed ${attempt} times: $*"
    set -e
    sleep "${interval}"
  done
  echo "error: reached max attempts at retry($*)"
  return 1
}

function init_networks() {
  if [[ ${AWS_NETWORK_NAME} != "default" ]]; then
    if [[ $(aws ec2 describe-vpcs --filters Name=tag:Name,Values=capo-e2e-mynetwork --region="${AWS_REGION}" --query 'length(*[0])') = "0" ]];
    then
      aws ec2 create-vpc --cidr-block 10.0.0.0/16 --tag-specifications "ResourceType=vpc,Tags=[{Key=Name,Value=${AWS_NETWORK_NAME}}]" --region="${AWS_REGION}"
      AWS_VPC_ID=$(aws ec2 describe-vpcs --filters Name=tag:Name,Values=capo-e2e-mynetwork --region "${AWS_REGION}" --query '*[0].VpcId' --output text)

      aws ec2 create-subnet --cidr-block 10.0.0.0/20 --vpc-id "${AWS_VPC_ID}" --tag-specifications "ResourceType=subnet,Tags=[{Key=Name,Value=${AWS_NETWORK_NAME}}]" --region "${AWS_REGION}" --availability-zone "${AWS_ZONE}"
      AWS_SUBNET_ID=$(aws ec2 describe-subnets --filters Name=tag:Name,Values=capo-e2e-mynetwork --region "${AWS_REGION}" --query '*[0].SubnetId' --output text)
      # It's also the route table of the VPC
      AWS_SUBNET_ROUTE_TABLE_ID=$(aws ec2 describe-route-tables --filters "Name=vpc-id,Values=${AWS_VPC_ID}" --region "${AWS_REGION}" --query '*[0].RouteTableId' --output text)

      aws ec2 create-security-group --group-name "${AWS_NETWORK_NAME}" --description "${AWS_NETWORK_NAME}" --vpc-id "${AWS_VPC_ID}" --tag-specifications "ResourceType=security-group,Tags=[{Key=Name,Value=${AWS_NETWORK_NAME}}]" --region="${AWS_REGION}"
      AWS_SECURITY_GROUP_ID=$(aws ec2 describe-security-groups --filters Name=tag:Name,Values=capo-e2e-mynetwork --region "${AWS_REGION}" --query '*[0].GroupId' --output text)

      aws ec2 authorize-security-group-ingress --group-id "${AWS_SECURITY_GROUP_ID}" --protocol tcp --port 22 --cidr 0.0.0.0/0 --region="${AWS_REGION}"

      # Documentation to enable internet access for subnet:
      # https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/TroubleshootingInstancesConnecting.html#TroubleshootingInstancesConnectionTimeout
      aws ec2 create-internet-gateway --tag-specifications "ResourceType=internet-gateway,Tags=[{Key=Name,Value=${AWS_NETWORK_NAME}}]" --region="${AWS_REGION}"
      aws ec2 attach-internet-gateway --internet-gateway-id "${AWS_INTERNET_GATEWAY_ID}" --vpc-id "${AWS_VPC_ID}" --region="${AWS_REGION}"
      AWS_INTERNET_GATEWAY_ID=$(aws ec2 describe-internet-gateways --filters Name=tag:Name,Values=capo-e2e-mynetwork --region "${AWS_REGION}" --query '*[0].InternetGatewayId' --output text)

      aws ec2 create-route --route-table-id "${AWS_SUBNET_ROUTE_TABLE_ID}" --destination-cidr-block 0.0.0.0/0 --gateway-id "${AWS_INTERNET_GATEWAY_ID}" --region "${AWS_REGION}"
      aws ec2 create-route --route-table-id "${AWS_SUBNET_ROUTE_TABLE_ID}" --destination-ipv6-cidr-block ::/0 --gateway-id "${AWS_INTERNET_GATEWAY_ID}" --region "${AWS_REGION}"
    fi
  fi
}

main() {
  # Initialize the necessary network requirements
  if [[ -n "${SKIP_INIT_NETWORK:-}" ]]; then
    echo "Skipping network initialization..."
  else
    init_networks
  fi

  if [[ $(aws ec2 describe-instances --filters Name=tag:Name,Values=openstack --region="${AWS_REGION}" --query 'length(*[0])') = "0" ]];
  then
    SSH_PUBLIC_KEY="- $(cat ~/.ssh/aws-capo-e2e.pub)"
    < ./hack/ci/devstack-${FLAVOR}-cloud-init.yaml.tpl \
	    sed "s|\${OPENSTACK_ENABLE_HORIZON}|${OPENSTACK_ENABLE_HORIZON}|" | \
      sed "s|\${OPENSTACK_RELEASE}|${OPENSTACK_RELEASE}|" | \
      sed "s|\${SSH_PUBLIC_KEY}|${SSH_PUBLIC_KEY}|" \
	    > ./hack/ci/devstack-${FLAVOR}-cloud-init.yaml

    AWS_SUBNET_ID=$(aws ec2 describe-subnets --filters Name=tag:Name,Values=capo-e2e-mynetwork --region "${AWS_REGION}" --query '*[0].SubnetId' --output text)
    AWS_SECURITY_GROUP_ID=$(aws ec2 describe-security-groups --filters Name=tag:Name,Values=capo-e2e-mynetwork --region "${AWS_REGION}" --query '*[0].GroupId' --output text)

    # /dev/sda1 is renamed to /dev/nvme0n1 by AWS
    aws ec2 run-instances --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=openstack}]" \
      --region "${AWS_REGION}" \
      --placement "AvailabilityZone=${AWS_ZONE}" \
      --image-id "${AWS_AMI}" \
      --instance-type "${AWS_MACHINE_TYPE}" \
      --block-device-mappings 'DeviceName=/dev/sda1,Ebs={VolumeSize=300}' \
      --subnet-id "${AWS_SUBNET_ID}"  \
      --private-ip-address 10.0.2.15 \
      --count 1 \
      --associate-public-ip-address \
      --security-group-ids "${AWS_SECURITY_GROUP_ID}"  \
      --key-name "${AWS_KEY_PAIR}" \
      --user-data file://hack/ci/devstack-${FLAVOR}-cloud-init.yaml \
      --no-paginate
  fi

  # Install some local dependencies we later need in the meantime (we have to wait for cloud init anyway)
  if ! command -v sshuttle;
  then
    # Install sshuttle from source because we need: https://github.com/sshuttle/sshuttle/pull/606
    # TODO(sbueringer) install via pip after the next release after 1.0.5 via:
    # pip3 install sshuttle
    cd /tmp
    git clone https://github.com/sshuttle/sshuttle.git
    cd sshuttle
    pip3 install .
    cd "${REPO_ROOT_ABSOLUTE}" || exit 1
  fi
  if ! command -v openstack;
  then
    apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y python3-dev
    # install PyYAML first because otherwise we get an error because pip3 doesn't upgrade PyYAML to the correct version
    # ERROR: Cannot uninstall 'PyYAML'. It is a distutils installed project and thus we cannot accurately determine which
    # files belong to it which would lead to only a partial uninstall.
    pip3 install --ignore-installed PyYAML
    pip3 install python-cinderclient python-glanceclient python-keystoneclient python-neutronclient python-novaclient python-openstackclient python-octaviaclient
  fi

  # wait a bit so the server has time to get a public ip
  sleep 30

  PUBLIC_IP=$(aws ec2 describe-instances --filters Name=tag:Name,Values=openstack --region "${AWS_REGION}" --query 'Reservations[*].Instances[*].PublicIpAddress' --output text)
  PRIVATE_IP=$(aws ec2 describe-instances --filters Name=tag:Name,Values=openstack --region "${AWS_REGION}" --query 'Reservations[*].Instances[*].PrivateIpAddress' --output text)

  # Wait until cloud-init is done
  retry 120 30 "ssh ubuntu@${PUBLIC_IP} -o 'StrictHostKeyChecking no' -o 'UserKnownHostsFile=/dev/null' -- cat /var/lib/cloud/instance/boot-finished"

  # Open tunnel
  echo "Opening tunnel to ${PRIVATE_IP} via ${PUBLIC_IP}"
  # Packets from the Prow Pod or the Pods in Kind have TTL 63 or 64.
  # We need a ttl of 65 (default 63), so all of our packets are captured by sshuttle.
  sshuttle -r "ubuntu@${PUBLIC_IP}" "${PRIVATE_IP}/32" 172.24.4.0/24 --ttl=65 --ssh-cmd='ssh -o "StrictHostKeyChecking no" -o "UserKnownHostsFile=/dev/null"' -l 0.0.0.0 -D

  export OS_REGION_NAME=RegionOne
  export OS_PROJECT_DOMAIN_ID=default
  export OS_AUTH_URL=http://${PRIVATE_IP}/identity
  export OS_TENANT_NAME=admin
  export OS_USER_DOMAIN_ID=default
  export OS_USERNAME=admin
  export OS_PROJECT_NAME=admin
  export OS_PASSWORD=secretadmin
  export OS_IDENTITY_API_VERSION=3

  # Wait until the OpenStack API is reachable
  retry 120 30 "openstack versions show"

  nova hypervisor-stats
  openstack host list
  openstack usage list
  openstack project list
  openstack network list
  openstack subnet list
  openstack image list
  openstack flavor list
  openstack server list
  openstack availability zone list
  openstack domain list

  # the flavors are created in a way that we can execute at least 2 e2e tests in parallel (overall we have 32 vCPUs)
  openstack flavor delete m1.tiny
  openstack flavor create --ram 512 --disk 1 --vcpus 1 --public --id 1 m1.tiny --property hw_rng:allowed='True'
  openstack flavor delete m1.small
  openstack flavor create --ram 4192 --disk 10 --vcpus 2 --public --id 2 m1.small --property hw_rng:allowed='True'
  openstack flavor delete m1.medium
  openstack flavor create --ram 6144 --disk 10 --vcpus 4 --public --id 3 m1.medium --property hw_rng:allowed='True'

  # Adjust the CPU quota
  openstack quota set --cores 32 demo
  openstack quota set --secgroups 50 demo

  export OS_TENANT_NAME=demo
  export OS_USERNAME=demo
  export OS_PROJECT_NAME=demo

  cat << EOF > "${REPO_ROOT_ABSOLUTE}/clouds.yaml"
clouds:
  ${CLUSTER_NAME}:
    auth:
      username: ${OS_USERNAME}
      password: ${OS_PASSWORD}
      user_domain_id: ${OS_USER_DOMAIN_ID}
      auth_url: ${OS_AUTH_URL}
      domain_id: default
      project_name: demo
    verify: false
    region_name: RegionOne
EOF
  echo "${REPO_ROOT_ABSOLUTE}/clouds.yaml:"
  cat "${REPO_ROOT_ABSOLUTE}/clouds.yaml"
}

main "$@"
