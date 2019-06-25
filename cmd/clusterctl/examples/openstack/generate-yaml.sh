#!/bin/bash
set -e

# Function that prints out the help message, describing the script
print_help()
{
  echo "$SCRIPT - generates a provider-configs.yaml file"
  echo ""
  echo "Usage:"
  echo "$SCRIPT [options] <path/to/clouds.yaml> <cloud> <provider os: [centos,ubuntu,coreos]> [output folder]"
  echo "options:"
  echo "-h, --help                    show brief help"
  echo "-f, --force-overwrite         if file to be generated already exists, force script to overwrite it"
  echo ""
}

# Supported Operating Systems
declare -a arr=("centos" "ubuntu" "coreos")
SCRIPT=$(basename $0)
while test $# -gt 0; do
        case "$1" in
          -h|--help)
            print_help
            exit 0
            ;;
          -f|--force-overwrite)
            OVERWRITE=1
            shift
            ;;
          *)
            break
            ;;
        esac
done

# Check if clouds.yaml file provided
if [[ -n "$1" ]] && [[ $1 != -* ]] && [[ $1 != --* ]];then
  CLOUDS_PATH="$1"
else
  echo "Error: No clouds.yaml provided"
  echo "You must provide a valid clouds.yaml script to genereate a cloud.conf"
  echo ""
  print_help
  exit 1
fi

# Check if os cloud is provided
if [[ -n "$2" ]] && [[ $2 != -* ]] && [[ $2 != --* ]]; then
  CLOUD=$2
else
  echo "Error: No cloud specified"
  echo "You must specify which cloud you want to use."
  echo ""
  print_help
  exit 1
fi

# Check that OS is provided
if [[ -n "$3" ]] && [[ $3 != -* ]] && [[ $3 != --* ]]; then
  USER_OS=$(echo $3 | tr '[:upper:]' '[:lower:]')
else
  echo "Error: No provider OS specified"
  echo "You must choose between the following operating systems: centos, ubuntu, coreos"
  echo ""
  print_help
  exit 1
fi

OUTPUT=out
if [[ -n "$4" ]] && [[ $4 != -* ]] && [[ $4 != --* ]]; then
  OUTPUT=$(echo $4 | tr '[:upper:]' '[:lower:]')
else
  echo "no output folder provided, use name 'out' by default"
fi

# Check that OS is supported
for i in "${arr[@]}"
do
  if test "$USER_OS" = "$i"; then
    PROVIDER_OS=$i
    break
  fi
done

if test -z "$PROVIDER_OS"; then
  echo "provider-os error: $USER_OS is not one of the supported operating systems!"
  print_help
  exit 1
fi

if ! hash yq 2>/dev/null; then
  echo "'yq' is not available, please install it. (https://github.com/mikefarah/yq)"
  echo ""
  print_help
  exit 1
fi

yq_type=$(file $(which yq))
if [[ $yq_type == *"Python script"* ]]; then
  echo "Wrong version of 'yq' installed, please install the one from https://github.com/mikefarah/yq"
  echo ""
  print_help
  exit 1
fi

if [ -e $OUTPUT/provider-components.yaml ] && [ "$OVERWRITE" != "1" ]; then
  echo "Can't overwrite provider-components.yaml without user permission. Either run the script again"
  echo "with -f or --force-overwrite, or delete the file in the out/ directory."
  echo ""
  print_help
  exit 1
fi


# Define global variables
PWD=$(cd `dirname $0`; pwd)
TEMPLATES_PATH=${TEMPLATES_PATH:-$PWD/$SUPPORTED_PROVIDER_OS}
HOME_DIR=${PWD%%/cmd/clusterctl/examples/*}
CONFIG_DIR=$PWD/provider-component/clouds-secrets/configs
USERDATA=$PWD/provider-component/user-data
MASTER_USER_DATA=$USERDATA/$PROVIDER_OS/templates/master-user-data.sh
WORKER_USER_DATA=$USERDATA/$PROVIDER_OS/templates/worker-user-data.sh

# Container Linux (simply named CoreOS here) does its configuration a bit different
# so it gets some of its own vars here.
COREOS_COMMON_SECTION=$USERDATA/$PROVIDER_OS/templates/common.yaml
COREOS_MASTER_SECTION=$USERDATA/$PROVIDER_OS/templates/master.yaml
COREOS_WORKER_SECTION=$USERDATA/$PROVIDER_OS/templates/worker.yaml

COREOS_MASTER_USER_DATA=$USERDATA/$PROVIDER_OS/master-user-data.yaml
COREOS_WORKER_USER_DATA=$USERDATA/$PROVIDER_OS/worker-user-data.yaml

OVERWRITE=${OVERWRITE:-0}
CLOUDS_PATH=${CLOUDS_PATH:-""}
OPENSTACK_CLOUD_CONFIG_PLAIN=$(cat "$CLOUDS_PATH")

MACHINE_CONTROLLER_SSH_PRIVATE_FILE=openstack_tmp
MACHINE_CONTROLLER_SSH_HOME=${HOME}/.ssh/
CACERT="/etc/certs/cacert"

# Set up the output dir if it does not yet exist
mkdir -p $PWD/$OUTPUT
cp -n $PWD/cluster.yaml $PWD/$OUTPUT/cluster.yaml || true
cp -n $PWD/machines.yaml.template $PWD/$OUTPUT/machines.yaml || true

# Make the config directory
mkdir -p $CONFIG_DIR

# Check if the ssh key already exists. If not, generate and copy to the .ssh dir.
if [ ! -f $MACHINE_CONTROLLER_SSH_HOME$MACHINE_CONTROLLER_SSH_PRIVATE_FILE ]; then
  echo "Generating SSH key files for machine controller."
  # This is needed because GetKubeConfig assumes the key in the home .ssh dir.
  ssh-keygen -t rsa -f $MACHINE_CONTROLLER_SSH_HOME$MACHINE_CONTROLLER_SSH_PRIVATE_FILE  -N ""
fi

# Just blindly parse the cloud.conf here, overwriting old vars.
AUTH_URL=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.$CLOUD.auth.auth_url)
USERNAME=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.$CLOUD.auth.username)
PASSWORD=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.$CLOUD.auth.password)
REGION=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.$CLOUD.region_name)
PROJECT_ID=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.$CLOUD.auth.project_id)
DOMAIN_NAME=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.$CLOUD.auth.user_domain_name)
CACERT_ORIGINAL=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.$CLOUD.cacert)

# Basic cloud.conf, no LB configuration as that data is not known yet.
OPENSTACK_CLOUD_PROVIDER_CONF_PLAIN="[Global]
auth-url=$AUTH_URL
username=\"$USERNAME\"
password=\"$PASSWORD\"
region=\"$REGION\"
tenant-id=\"$PROJECT_ID\"
domain-name=\"$DOMAIN_NAME\"
"

if [ "$CACERT_ORIGINAL" != "null" ]; then
  OPENSTACK_CLOUD_PROVIDER_CONF_PLAIN="$OPENSTACK_CLOUD_PROVIDER_CONF_PLAIN
  ca-file=\"$CACERT\"
  "
fi

OS=$(uname)
if [[ "$OS" =~ "Linux" ]]; then
  OPENSTACK_CLOUD_PROVIDER_CONF=$(echo "$OPENSTACK_CLOUD_PROVIDER_CONF_PLAIN"|base64 -w0)
  if [ "$CACERT_ORIGINAL" != "null" ]; then
    OPENSTACK_CLOUD_CACERT_CONFIG=$(cat "$CACERT_ORIGINAL"|base64 -w0)
  fi
elif [[ "$OS" =~ "Darwin" ]]; then
  OPENSTACK_CLOUD_PROVIDER_CONF=$(echo "$OPENSTACK_CLOUD_PROVIDER_CONF_PLAIN"|base64)
  if [ "$CACERT_ORIGINAL" != "null" ]; then
    OPENSTACK_CLOUD_CACERT_CONFIG=$(cat "$CACERT_ORIGINAL"|base64)
  fi
else
  echo "Unrecognized OS : $OS"
  exit 1
fi

if [[ "$PROVIDER_OS" == "coreos" ]]; then
  cat $COREOS_COMMON_SECTION \
    | sed -e "s#\$OPENSTACK_CLOUD_PROVIDER_CONF#$OPENSTACK_CLOUD_PROVIDER_CONF#" \
    | sed -e "s#\$OPENSTACK_CLOUD_CACERT_CONFIG#$OPENSTACK_CLOUD_CACERT_CONFIG#" \
    | yq m -a - $COREOS_MASTER_SECTION  \
    > $COREOS_MASTER_USER_DATA
  cat $COREOS_COMMON_SECTION \
    | sed -e "s#\$OPENSTACK_CLOUD_PROVIDER_CONF#$OPENSTACK_CLOUD_PROVIDER_CONF#" \
    | sed -e "s#\$OPENSTACK_CLOUD_CACERT_CONFIG#$OPENSTACK_CLOUD_CACERT_CONFIG#" \
    | yq m -a - $COREOS_WORKER_SECTION  \
    > $COREOS_WORKER_USER_DATA
else
  cat "$MASTER_USER_DATA" \
      | sed -e "s#\$OPENSTACK_CLOUD_PROVIDER_CONF#$OPENSTACK_CLOUD_PROVIDER_CONF#" \
      | sed -e "s#\$OPENSTACK_CLOUD_CACERT_CONFIG#$OPENSTACK_CLOUD_CACERT_CONFIG#" \
      > $USERDATA/$PROVIDER_OS/master-user-data.sh
  cat "$WORKER_USER_DATA" \
    | sed -e "s#\$OPENSTACK_CLOUD_PROVIDER_CONF#$OPENSTACK_CLOUD_PROVIDER_CONF#" \
    | sed -e "s#\$OPENSTACK_CLOUD_CACERT_CONFIG#$OPENSTACK_CLOUD_CACERT_CONFIG#" \
    > $USERDATA/$PROVIDER_OS/worker-user-data.sh
fi

printf $CLOUD > $CONFIG_DIR/os_cloud.txt
echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" > $CONFIG_DIR/clouds.yaml
if [ "$CACERT_ORIGINAL" != "null" ]; then
  cat "$CACERT_ORIGINAL" > $CONFIG_DIR/cacert
else
  echo "dummy" > $CONFIG_DIR/cacert
fi

# Build provider-components.yaml with kustomize
# Coreos has a different kubeadm path (/usr is read-only) so gets a different kustomization.
if [[ "$PROVIDER_OS" == "coreos" ]]; then
  kubectl kustomize $PWD/../../../../overlays-config/coreos > $PWD/$OUTPUT/provider-components.yaml
else
  kubectl kustomize $PWD/../../../../overlays-config/generic > $PWD/$OUTPUT/provider-components.yaml
fi
echo "---" >> $PWD/$OUTPUT/provider-components.yaml
kubectl kustomize $PWD/provider-component/clouds-secrets >> $PWD/$OUTPUT/provider-components.yaml
echo "---" >> $PWD/$OUTPUT/provider-components.yaml

# latest kustomize don't allow include files out of build folder
cp -r $CONFIG_DIR/../../../../../../../vendor/sigs.k8s.io/cluster-api/config $PWD/provider-component/cluster-api
echo "---" >> $PWD/$OUTPUT/provider-components.yaml
kubectl kustomize $PWD/provider-component/cluster-api >> $PWD/$OUTPUT/provider-components.yaml
rm -fr $PWD/provider-component/cluster-api/config

echo "---" >> $PWD/$OUTPUT/provider-components.yaml
kubectl kustomize $USERDATA/$PROVIDER_OS >> $PWD/$OUTPUT/provider-components.yaml
