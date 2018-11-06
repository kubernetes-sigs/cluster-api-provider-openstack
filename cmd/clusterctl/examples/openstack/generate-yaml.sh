#!/bin/bash
set -e

# Function that prints out the help message, describing the script 
print_help()
{
  echo "$SCRIPT - generates input yaml files for Cluster API on openstack"
  echo " "
  echo "$SCRIPT [options]"
  echo " "
  echo "options:"
  echo "-h, --help                show brief help"
  echo "-f, --force-overwrite     if file to be generated already exists, force script to overwrite it"
  echo "-c, --clouds [File]       specifies an existing clouds.yaml file to use rather than generating one interactively"
  echo "--provider-os [os name]   Required: select the operating system of your provider environment"
  echo "                            Supported Operating Systems: ubuntu, centos"
  echo ""
}

# Supported Operating Systems
declare -a arr=("centos" "ubuntu")
SUPPORTED_PROVIDER_OS=""

SCRIPT=$(basename $0)
while test $# -gt 0; do
        case "$1" in
          -h|--help)
            print_help
            exit 0
            ;;
          -t)
              TEMPLATES_PATH=`realpath $2`
              shift
              shift
              ;;
          -f)
            OVERWRITE=1
            shift
            ;;
          --force-overwrite)
            OVERWRITE=1
            shift
            ;;
          -c|--clouds)
            if [[ -z "$2" ]] || [[ $2 == -* ]] || [[ $2 == --* ]];then
              echo "Error: No cloud path was provided!"
              print_help
              exit 1
            fi
            CLOUDS_PATH=$2
            shift
            shift
            ;;
          --provider-os)
            if [[ -z "$2" ]] || [[ $2 == -* ]] || [[ $2 == --* ]];then
              echo "provider-os error: No operating system was provided!"
              print_help
              exit 1
            fi
            PROVIDER_OS=$(echo $2 | tr '[:upper:]' '[:lower:]')
            for i in "${arr[@]}"
            do
              if test "$PROVIDER_OS" = "$i"; then
                SUPPORTED_PROVIDER_OS=$i
                break
              fi
            done
            if test -z "$SUPPORTED_PROVIDER_OS"; then
              echo "provider-os error: $PROVIDER_OS is not one of the supported operating systems!"
              print_help
              exit 1
            fi
            shift
            shift
            ;;
          *)
            break
            ;;
        esac
done

if test -z "$SUPPORTED_PROVIDER_OS"; then
  echo "Missing argument: provider-os is a required argument"
  print_help
  exit 1
fi

# Define global variables
PWD=$(cd `dirname $0`; pwd)
TEMPLATES_PATH=${TEMPLATES_PATH:-$PWD/$SUPPORTED_PROVIDER_OS/}
HOME_DIR=${PWD%%/cmd/clusterctl/examples/*}
OUTPUT_DIR="${TEMPLATES_PATH}/out"
PROVIDER_CRD_DIR="${HOME_DIR}/config/crd"
PROVIDER_RBAC_DIR="${HOME_DIR}/config/rbac"
PROVIDER_MANAGER_DIR="${HOME_DIR}/config/manager"
CLUSTER_CRD_DIR="${HOME_DIR}/vendor/sigs.k8s.io/cluster-api/config/crds"
CLUSTER_RBAC_DIR="${HOME_DIR}/vendor/sigs.k8s.io/cluster-api/config/rbac"
CLUSTER_MANAGER_DIR="${HOME_DIR}/vendor/sigs.k8s.io/cluster-api/config/manager"

MACHINE_TEMPLATE_FILE=${TEMPLATES_PATH}/machines.yaml.template
MACHINE_GENERATED_FILE=${OUTPUT_DIR}/machines.yaml
CLUSTER_TEMPLATE_FILE=${TEMPLATES_PATH}/cluster.yaml.template
CLUSTER_GENERATED_FILE=${OUTPUT_DIR}/cluster.yaml
PROVIDERCOMPONENT_TEMPLATE_FILE=${TEMPLATES_PATH}/provider-components.yaml.template
PROVIDERCOMPONENT_GENERATED_FILE=${OUTPUT_DIR}/provider-components.yaml

MACHINE_CONTROLLER_SSH_PUBLIC_FILE=openstack_tmp.pub
MACHINE_CONTROLLER_SSH_PUBLIC=
MACHINE_CONTROLLER_SSH_PRIVATE_FILE=openstack_tmp
MACHINE_CONTROLLER_SSH_PRIVATE=
MACHINE_CONTROLLER_SSH_HOME=${HOME}/.ssh/

OVERWRITE=${OVERWRITE:-0}
CLOUDS_PATH=${CLOUDS_PATH:-""}
CLOUD="${OS_CLOUD}" 

if [ $OVERWRITE -ne 1 ] && [ -f "$MACHINE_GENERATED_FILE" ]; then
  echo "File $MACHINE_GENERATED_FILE already exists. Delete it manually before running this script."
  exit 1
fi

if [ $OVERWRITE -ne 1 ] && [ -f "$CLUSTER_GENERATED_FILE" ]; then
  echo "File $CLUSTER_GENERATED_FILE already exists. Delete it manually before running this script."
  exit 1
fi

if [ $OVERWRITE -ne 1 ] && [ -f "$PROVIDERCOMPONENT_GENERATED_FILE" ]; then
  echo "File "$PROVIDERCOMPONENT_GENERATED_FILE" already exists. Delete it manually before running this script."
  exit 1
fi

if [ -z "$CLOUD" ]; then
  CLOUD=openstack
fi

mkdir -p "${OUTPUT_DIR}"

if [ -n "$CLOUDS_PATH" ]; then
  # Read clouds.yaml from file if a path is provided 
  OPENSTACK_CLOUD_CONFIG_PLAIN=$(cat "$CLOUDS_PATH")
else
  # Collect user input to generate a clouds.yaml file
  read -p "Enter your username:" username
  read -p "Enter your domainname:" domain_name
  read -p "Enter your project id:" project_id
  read -p "Enter region name:" region
  read -p "Enter authurl:" authurl
  read -s -p "Enter your password:" password
  OPENSTACK_CLOUD_CONFIG_PLAIN="clouds:
  openstack:
    auth:
      username: $username
      password: $password
      user_domain_name: $domain_name
      project_id: $project_id
      auth_url: $authurl
    interface: public
    identity_api_version: 3
    region: $region"
fi

# Check if the ssh key already exists. If not, generate and copy to the .ssh dir.
if [ ! -f $MACHINE_CONTROLLER_SSH_HOME$MACHINE_CONTROLLER_SSH_PRIVATE_FILE ]; then
  echo "Generating SSH key files for machine controller."
  # This is needed because GetKubeConfig assumes the key in the home .ssh dir.
  ssh-keygen -t rsa -f $MACHINE_CONTROLLER_SSH_HOME$MACHINE_CONTROLLER_SSH_PRIVATE_FILE  -N ""
fi
MACHINE_CONTROLLER_SSH_PLAIN=clusterapi

OS=$(uname)
if [[ "$OS" =~ "Linux" ]]; then
  OPENSTACK_CLOUD_CONFIG=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN"|base64 -w0)
  MACHINE_CONTROLLER_SSH_USER=$(echo -n $MACHINE_CONTROLLER_SSH_PLAIN|base64 -w0)
  MACHINE_CONTROLLER_SSH_PUBLIC=$(cat "$MACHINE_CONTROLLER_SSH_HOME$MACHINE_CONTROLLER_SSH_PUBLIC_FILE"|base64 -w0)
  MACHINE_CONTROLLER_SSH_PRIVATE=$(cat "$MACHINE_CONTROLLER_SSH_HOME$MACHINE_CONTROLLER_SSH_PRIVATE_FILE"|base64 -w0)
elif [[ "$OS" =~ "Darwin" ]]; then
  OPENSTACK_CLOUD_CONFIG=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN"|base64)
  MACHINE_CONTROLLER_SSH_USER=$(printf $MACHINE_CONTROLLER_SSH_PLAIN|base64)
  MACHINE_CONTROLLER_SSH_PUBLIC=$(cat "$MACHINE_CONTROLLER_SSH_HOME$MACHINE_CONTROLLER_SSH_PUBLIC_FILE"|base64)
  MACHINE_CONTROLLER_SSH_PRIVATE=$(cat "$MACHINE_CONTROLLER_SSH_HOME$MACHINE_CONTROLLER_SSH_PRIVATE_FILE"|base64)
else
  echo "Unrecognized OS : $OS"
  exit 1
fi

# write config file to PROVIDERCOMPONENT_GENERATED_FILE
for file in `ls "${PROVIDER_CRD_DIR}"`
do
    cat "${PROVIDER_CRD_DIR}/${file}" > "$PROVIDERCOMPONENT_GENERATED_FILE"
    echo "---" >> "$PROVIDERCOMPONENT_GENERATED_FILE"
done
for file in `ls "${PROVIDER_RBAC_DIR}"`
do
    cat "${PROVIDER_RBAC_DIR}/${file}" >> "$PROVIDERCOMPONENT_GENERATED_FILE"
    echo "---" >> "$PROVIDERCOMPONENT_GENERATED_FILE"
done
for file in `ls "${PROVIDER_MANAGER_DIR}"`
do
    sed "s/{OS_CLOUD}/$CLOUD/g" "${PROVIDER_MANAGER_DIR}/${file}" >> "$PROVIDERCOMPONENT_GENERATED_FILE"
    echo "---" >> "$PROVIDERCOMPONENT_GENERATED_FILE"
done
for file in `ls "${CLUSTER_MANAGER_DIR}"`
do
    cat "${CLUSTER_MANAGER_DIR}/${file}" >> "$PROVIDERCOMPONENT_GENERATED_FILE"
    echo "---" >> "$PROVIDERCOMPONENT_GENERATED_FILE"
done
for file in `ls "${CLUSTER_CRD_DIR}"`
do
    cat "${CLUSTER_CRD_DIR}/${file}" >> "$PROVIDERCOMPONENT_GENERATED_FILE"
    echo "---" >> "$PROVIDERCOMPONENT_GENERATED_FILE"
done
for file in `ls "${CLUSTER_RBAC_DIR}"`
do
    cat "${CLUSTER_RBAC_DIR}/${file}" >> "$PROVIDERCOMPONENT_GENERATED_FILE"
    echo "---" >> "$PROVIDERCOMPONENT_GENERATED_FILE"
done

cat "$PROVIDERCOMPONENT_TEMPLATE_FILE" \
  | sed -e "s/\$OPENSTACK_CLOUD_CONFIG/$OPENSTACK_CLOUD_CONFIG/" \
  | sed -e "s/\$MACHINE_CONTROLLER_SSH_USER/$MACHINE_CONTROLLER_SSH_USER/" \
  | sed -e "s/\$MACHINE_CONTROLLER_SSH_PUBLIC/$MACHINE_CONTROLLER_SSH_PUBLIC/" \
  | sed -e "s/\$MACHINE_CONTROLLER_SSH_PRIVATE/$MACHINE_CONTROLLER_SSH_PRIVATE/" \
  >> "$PROVIDERCOMPONENT_GENERATED_FILE"

if [[ "$OS" =~ "Linux" ]]; then
  sed -i "s#image: controller:latest#image: gcr.io/k8s-cluster-api/cluster-api-controller:latest#" "$PROVIDERCOMPONENT_GENERATED_FILE"
elif [[ "$OS" =~ "Darwin" ]]; then
  sed -i '' -e "s#image: controller:latest#image: gcr.io/k8s-cluster-api/cluster-api-controller:latest#" "$PROVIDERCOMPONENT_GENERATED_FILE"
else
  echo "Unrecognized OS : $OS"
  exit 1
fi

cat "$MACHINE_TEMPLATE_FILE" \
  > "$MACHINE_GENERATED_FILE"

cat "$CLUSTER_TEMPLATE_FILE" \
  > "$CLUSTER_GENERATED_FILE"


echo "Done generating $PROVIDERCOMPONENT_GENERATED_FILE $MACHINE_GENERATED_FILE $CLUSTER_GENERATED_FILE"
echo "You should now manually change your cluster configuration by editing the generated files."

