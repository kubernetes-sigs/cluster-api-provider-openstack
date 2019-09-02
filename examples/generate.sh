#!/bin/bash
# Copyright 2019 The Kubernetes Authors.
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

set -o errexit

# Directories.
SOURCE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
OUTPUT_DIR=${OUTPUT_DIR:-${SOURCE_DIR}/_out}

# Binaries
ENVSUBST=${ENVSUBST:-envsubst}
command -v "${ENVSUBST}" >/dev/null 2>&1 || echo -v "Cannot find ${ENVSUBST} in path."

# Cluster.
export CLUSTER_NAME="${CLUSTER_NAME:-test-cluster}"
export KUBERNETES_VERSION="${KUBERNETES_VERSION:-v1.15.0}"

# Machine settings.
export CONTROL_PLANE_MACHINE_TYPE="${CONTROL_PLANE_MACHINE_TYPE:-m1.medium}"
export NODE_MACHINE_TYPE="${CONTROL_PLANE_MACHINE_TYPE:-m1.medium}"
export SSH_KEY_NAME="${SSH_KEY_NAME:-cluster-api-provider-openstack}"

# Overwrite flag.
OVERWRITE=0

SCRIPT=$(basename "$0")
while test $# -gt 0; do
        case "$1" in
          -h|--help)
            echo "$SCRIPT - generates input yaml files for Cluster API on OpenStack"
            echo " "
            echo "$SCRIPT [options] <path/to/clouds.yaml> <cloud> [output folder]"
            echo " "
            echo "options:"
            echo "-h, --help                show brief help"
            echo "-f, --force-overwrite     if file to be generated already exists, force script to overwrite it"
            exit 0
            ;;
          -f)
            OVERWRITE=1
            shift
            ;;
          --force-overwrite)
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
  echo "You must provide a valid clouds.yaml script to generate a cloud.conf"
  echo ""
  exit 1
fi

# Check if os cloud is provided
if [[ -n "$2" ]] && [[ $2 != -* ]] && [[ $2 != --* ]]; then
  export CLOUD=$2
else
  echo "Error: No cloud specified"
  echo "You must specify which cloud you want to use."
  echo ""
  exit 1
fi

if [[ -n "$3" ]] && [[ $3 != -* ]] && [[ $3 != --* ]]; then
  OUTPUT_DIR=$(echo $3 | tr '[:upper:]' '[:lower:]')
else
  echo "no output folder provided, use name '_out' by default"
fi

if [[ ${OVERWRITE} -ne 1 ]] && [[ -d "$OUTPUT_DIR" ]]; then
  echo "ERR: Folder ${OUTPUT_DIR} already exists. Delete it manually before running this script."
  exit 1
fi

yq_type=$(file "$(which yq)")
if [[ ${yq_type} == *"Python script"* ]]; then
  echo "Wrong version of 'yq' installed, please install the one from https://github.com/mikefarah/yq"
  echo ""
  exit 1
fi


# Outputs.
COMPONENTS_CLUSTER_API_GENERATED_FILE=${SOURCE_DIR}/provider-components/provider-components-cluster-api.yaml
COMPONENTS_KUBEADM_GENERATED_FILE=${SOURCE_DIR}/provider-components/provider-components-kubeadm.yaml
COMPONENTS_OPENSTACK_GENERATED_FILE=${SOURCE_DIR}/provider-components/provider-components-openstack.yaml
COMPONENTS_OPENSTACK_CLOUDS_SECRETS_GENERATED_FILE=${SOURCE_DIR}/provider-components/provider-components-openstack-clouds-secrets.yaml
CLOUDS_SECRETS_CONFIG_DIR=${SOURCE_DIR}/clouds-secrets/configs
MACHINE_CONTROLLER_SSH_PRIVATE_FILE=${HOME}/.ssh/openstack_tmp

PROVIDER_COMPONENTS_GENERATED_FILE=${OUTPUT_DIR}/provider-components.yaml
CLUSTER_GENERATED_FILE=${OUTPUT_DIR}/cluster.yaml
CONTROLPLANE_GENERATED_FILE=${OUTPUT_DIR}/controlplane.yaml
MACHINEDEPLOYMENT_GENERATED_FILE=${OUTPUT_DIR}/machinedeployment.yaml
MACHINES_GENERATED_FILE=${OUTPUT_DIR}/machines.yaml

rm -rf "${OUTPUT_DIR}"
rm -rf "${CLOUDS_SECRETS_CONFIG_DIR}"
mkdir -p "${OUTPUT_DIR}"
mkdir -p "${CLOUDS_SECRETS_CONFIG_DIR}"

# Check if the ssh key already exists. If not, generate and copy to the .ssh dir.
if [[ ! -f ${MACHINE_CONTROLLER_SSH_PRIVATE_FILE} ]]; then
  echo "Generating SSH key files for machine controller."
  # This is needed because GetKubeConfig assumes the key in the home .ssh dir.
  ssh-keygen -t rsa -f ${MACHINE_CONTROLLER_SSH_PRIVATE_FILE}  -N ""
fi
export MACHINE_CONTROLLER_SSH_PUBLIC_FILE_CONTENT
MACHINE_CONTROLLER_SSH_PUBLIC_FILE_CONTENT=$(cat ${MACHINE_CONTROLLER_SSH_PRIVATE_FILE}.pub)

CLOUDS_PATH=${CLOUDS_PATH:-""}
OPENSTACK_CLOUD_CONFIG_PLAIN=$(cat "$CLOUDS_PATH")

# Just blindly parse the cloud.conf here, overwriting old vars.
AUTH_URL=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.${CLOUD}.auth.auth_url)
USERNAME=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.${CLOUD}.auth.username)
PASSWORD=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.${CLOUD}.auth.password)
REGION=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.${CLOUD}.region_name)
PROJECT_ID=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.${CLOUD}.auth.project_id)
DOMAIN_NAME=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.${CLOUD}.auth.user_domain_name)
if [[ "$DOMAIN_NAME" = "null" ]]; then
  DOMAIN_NAME=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.${CLOUD}.auth.domain_name)
fi
CACERT_ORIGINAL=$(echo "$OPENSTACK_CLOUD_CONFIG_PLAIN" | yq r - clouds.${CLOUD}.cacert)

# Basic cloud.conf, no LB configuration as that data is not known yet.
export OPENSTACK_CLOUD_PROVIDER_CONF="[Global]
          auth-url=$AUTH_URL
          username=\"$USERNAME\"
          password=\"$PASSWORD\"
          tenant-id=\"$PROJECT_ID\"
          domain-name=\"$DOMAIN_NAME\"
"
if [[ "$CACERT_ORIGINAL" != "null" ]]; then
  OPENSTACK_CLOUD_PROVIDER_CONF="$OPENSTACK_CLOUD_PROVIDER_CONF
          ca-file=\"${CACERT_ORIGINAL}\"
  "
fi
if [[ "$REGION" != "null" ]]; then
  OPENSTACK_CLOUD_PROVIDER_CONF="$OPENSTACK_CLOUD_PROVIDER_CONF
          region=\"${REGION}\"
  "
fi
OS=$(uname)
if [[ "$OS" =~ "Linux" ]]; then
#  export OPENSTACK_CLOUD_PROVIDER_CONF=$(echo "$OPENSTACK_CLOUD_PROVIDER_CONF_PLAIN"|base64 -w0)
  if [[ "$CACERT_ORIGINAL" != "null" ]]; then
    export OPENSTACK_CLOUD_CACERT_CONFIG
    OPENSTACK_CLOUD_CACERT_CONFIG=$(cat "$CACERT_ORIGINAL"|base64 -w0)
  fi
elif [[ "$OS" =~ "Darwin" ]]; then
#  export OPENSTACK_CLOUD_PROVIDER_CONF=$(echo "$OPENSTACK_CLOUD_PROVIDER_CONF_PLAIN"|base64)
  if [[ "$CACERT_ORIGINAL" != "null" ]]; then
    export OPENSTACK_CLOUD_CACERT_CONFIG
    OPENSTACK_CLOUD_CACERT_CONFIG=$(cat "$CACERT_ORIGINAL"|base64)
  fi
else
  echo "Unrecognized OS : $OS"
  exit 1
fi

echo "${OPENSTACK_CLOUD_CONFIG_PLAIN}" > ${CLOUDS_SECRETS_CONFIG_DIR}/clouds.yaml
if [[ "$CACERT_ORIGINAL" != "null" ]]; then
  cat "$CACERT_ORIGINAL" > ${CLOUDS_SECRETS_CONFIG_DIR}/cacert
else
  echo "dummy" > ${CLOUDS_SECRETS_CONFIG_DIR}/cacert
fi

# Generate cluster resources.
kustomize build "${SOURCE_DIR}/cluster" --reorder=none | envsubst > "${CLUSTER_GENERATED_FILE}"
echo "Generated ${CLUSTER_GENERATED_FILE}"

# Generate controlplane resources.
kustomize build "${SOURCE_DIR}/controlplane" --reorder=none | envsubst > "${CONTROLPLANE_GENERATED_FILE}"
echo "Generated ${CONTROLPLANE_GENERATED_FILE}"

# Generate machinedeployment resources.
kustomize build "${SOURCE_DIR}/machinedeployment" --reorder=none | envsubst >> "${MACHINEDEPLOYMENT_GENERATED_FILE}"
echo "Generated ${MACHINEDEPLOYMENT_GENERATED_FILE}"

# combine control plane and regular machines in ${MACHINES_GENERATED_FILE}
cat ${CONTROLPLANE_GENERATED_FILE} > ${MACHINES_GENERATED_FILE}
echo "---" >> ${MACHINES_GENERATED_FILE}
#cat ${MACHINEDEPLOYMENT_GENERATED_FILE} >> ${MACHINES_GENERATED_FILE}
echo "---" >> ${MACHINES_GENERATED_FILE}
cat ${MACHINEDEPLOYMENT_GENERATED_FILE} >> ${MACHINES_GENERATED_FILE}
echo "---" >> ${MACHINES_GENERATED_FILE}
echo "Generated ${MACHINES_GENERATED_FILE}"

# Generate Cluster API provider components file.
kustomize build "github.com/kubernetes-sigs/cluster-api//config/default/?ref=master" --reorder=none > "${COMPONENTS_CLUSTER_API_GENERATED_FILE}"
echo "Generated ${COMPONENTS_CLUSTER_API_GENERATED_FILE}"

# Generate Kubeadm Bootstrap Provider components file.
kustomize build "github.com/kubernetes-sigs/cluster-api-bootstrap-provider-kubeadm//config/default/?ref=master" --reorder=none > "${COMPONENTS_KUBEADM_GENERATED_FILE}"
echo "Generated ${COMPONENTS_KUBEADM_GENERATED_FILE}"

# Generate OpenStack Infrastructure Provider components file.
kustomize build "${SOURCE_DIR}/../config/default" --reorder=none | envsubst > "${COMPONENTS_OPENSTACK_GENERATED_FILE}"
echo "Generated ${COMPONENTS_OPENSTACK_GENERATED_FILE}"

# Generate OpenStack Infrastructure Provider cloud-secrets file.
kustomize build "${SOURCE_DIR}/clouds-secrets" --reorder=none | envsubst > "${COMPONENTS_OPENSTACK_CLOUDS_SECRETS_GENERATED_FILE}"
echo "Generated ${COMPONENTS_OPENSTACK_CLOUDS_SECRETS_GENERATED_FILE}"
echo "WARNING: ${COMPONENTS_OPENSTACK_CLOUDS_SECRETS_GENERATED_FILE} includes OpenStack credentials"

# Generate a single provider components file.
kustomize build "${SOURCE_DIR}/provider-components"| envsubst > "${PROVIDER_COMPONENTS_GENERATED_FILE}"
echo "Generated ${PROVIDER_COMPONENTS_GENERATED_FILE}"
echo "WARNING: ${PROVIDER_COMPONENTS_GENERATED_FILE} includes OpenStack credentials"
