#!/usr/bin/env bash

# Copyright 2021 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# 	http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# hack script for preparing AWS to run cluster-api-provider-openstack e2e

set -x -o errexit -o nounset -o pipefail

function cloud_init {
  AWS_REGION=${AWS_REGION:-"eu-central-1"}
  AWS_ZONE=${AWS_ZONE:-"eu-central-1a"}
  # AMIs:
  # * capa-ami-ubuntu-20.04-1.20.4-00-1613898574 id: ami-0120656d38c206057
  # * ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-arm64-server-20231207 id: ami-05d47d29a4c2d19e1
  AWS_AMI=${AWS_AMI:-"ami-05d47d29a4c2d19e1"}
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

  echo "Using: AWS_REGION: ${AWS_REGION} AWS_NETWORK_NAME: ${AWS_NETWORK_NAME}"
}

function init_infrastructure() {
  if [[ ${AWS_NETWORK_NAME} != "default" ]]; then
    if [[ $(aws ec2 describe-vpcs --filters Name=tag:Name,Values=capo-e2e-mynetwork --region="${AWS_REGION}" --query 'length(*[0])') = "0" ]];
    then
      aws ec2 create-vpc --cidr-block "$PRIVATE_NETWORK_CIDR" --tag-specifications "ResourceType=vpc,Tags=[{Key=Name,Value=${AWS_NETWORK_NAME}}]" --region="${AWS_REGION}"
      AWS_VPC_ID=$(aws ec2 describe-vpcs --filters Name=tag:Name,Values=capo-e2e-mynetwork --region "${AWS_REGION}" --query '*[0].VpcId' --output text)

      aws ec2 create-subnet --cidr-block "$PRIVATE_NETWORK_CIDR" --vpc-id "${AWS_VPC_ID}" --tag-specifications "ResourceType=subnet,Tags=[{Key=Name,Value=${AWS_NETWORK_NAME}}]" --region "${AWS_REGION}" --availability-zone "${AWS_ZONE}"
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

function create_vm {
  local name=$1 && shift
  local ip=$1 && shift
  local userdata=$1 && shift
  local public=$1 && shift # Unused by AWS

  if [[ $(aws ec2 describe-instances --filters Name=tag:Name,Values="${name}" --region="${AWS_REGION}" --query 'length(*[0])') = "0" ]];
  then
    AWS_SUBNET_ID=$(aws ec2 describe-subnets --filters Name=tag:Name,Values=capo-e2e-mynetwork --region "${AWS_REGION}" --query '*[0].SubnetId' --output text)
    AWS_SECURITY_GROUP_ID=$(aws ec2 describe-security-groups --filters Name=tag:Name,Values=capo-e2e-mynetwork --region "${AWS_REGION}" --query '*[0].GroupId' --output text)

    # /dev/sda1 is renamed to /dev/nvme0n1 by AWS
    aws ec2 run-instances --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=${name}}]" \
      --region "${AWS_REGION}" \
      --placement "AvailabilityZone=${AWS_ZONE}" \
      --image-id "${AWS_AMI}" \
      --instance-type "${AWS_MACHINE_TYPE}" \
      --block-device-mappings 'DeviceName=/dev/sda1,Ebs={VolumeSize=300}' \
      --subnet-id "${AWS_SUBNET_ID}"  \
      --private-ip-address "${ip}" \
      --count 1 \
      --associate-public-ip-address \
      --security-group-ids "${AWS_SECURITY_GROUP_ID}"  \
      --key-name "${AWS_KEY_PAIR}" \
      --user-data "file://${userdata}" \
      --no-paginate
  fi

  # wait a bit so the server has time to get a public ip
  sleep 30
}

function get_public_ip {
  aws ec2 describe-instances --filters "Name=tag:Name,Values=${CLUSTER_NAME}-controller" --region "${AWS_REGION}" \
      --query 'Reservations[*].Instances[*].PublicIpAddress' --output text
}

function get_mtu {
    # https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/network_mtu.html
    echo 1300
}

function get_ssh_public_key_file {
  echo "${SSH_PUBLIC_KEY_FILE}"
}

function get_ssh_private_key_file {
  echo "${SSH_PRIVATE_KEY_FILE}"
}

function cloud_cleanup {
  echo Not implemented
  exit 1
}
