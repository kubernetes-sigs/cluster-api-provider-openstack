#!/bin/bash

source common.sh

cd $CAPO_SRC/cmd/clusterctl/examples/openstack/

if [ -f ~/.config/openstack/clouds.yaml ];
then
    ./generate-yaml.sh -f --provider-os $PROVIDER_OS --clouds ~/.config/openstack/clouds.yaml
else
    ./generate-yaml.sh -f --provider-os $PROVIDER_OS
fi

cd $PROVIDER_OS/out
sudo $CAPO_SRC/bin/clusterctl create cluster -v=10 --minikube kubernetes-version=v1.12.1 --minikube vm-driver=none --provider openstack -c cluster.yaml -m machines.yaml -p provider-components.yaml --cleanup-bootstrap-cluster 
