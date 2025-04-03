#!/bin/bash
set -e
GO_VERSION="go1.23.2.linux-amd64.tar.gz"
IMG=erfan272758/eifa-replica-operator:v11
# golang
wget https://go.dev/dl/$GO_VERSION
rm -rf /usr/local/go
tar -C /usr/local -xzf $GO_VERSION

# clone project
git clone https://github.com/erfan-272758/eifa-replica-operator.git
cd eifa-replica-operator
export IMG=${IMG}
make deploy
