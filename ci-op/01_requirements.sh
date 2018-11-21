#!/bin/bash

source common.sh

sudo apt-get update
sudo apt-get install -y git docker.io build-essential
sudo gpasswd -a ubuntu docker
sudo chmod 777 /var/run/docker.sock

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update
sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl

cd $HOME

curl -O https://storage.googleapis.com/golang/go1.11.2.linux-amd64.tar.gz
mkdir go-bin
tar xvzf go1.11.2.linux-amd64.tar.gz -C go-bin
sudo cp -f go-bin/go/bin/* /usr/bin
sudo cp -rf go-bin/go /usr/local

curl -Lo minikube https://storage.googleapis.com/minikube/releases/v0.30.0/minikube-linux-amd64 && chmod +x minikube && sudo cp minikube /usr/local/bin/ && rm minikube


mkdir -p $GOPATH/src/sigs.k8s.io

# We shouldn't need to clone CAPO
# git clone https://github.com/kubernetes-sigs/cluster-api-provider-openstack $CAPO_SRC

cd $CAPO_SRC
make depend
make clusterctl
