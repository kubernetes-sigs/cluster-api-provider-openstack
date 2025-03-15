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

# retry $1 times with $2 sleep in between
function retry {
    attempt=0
    max_attempts=${1}
    interval=${2}
    shift; shift
    until [[ "$attempt" -ge "$max_attempts" ]] ; do
        attempt=$((attempt+1))
        set +e
        eval "$*" && return || echo "failed $attempt times: $*"
        set -e
        sleep "$interval"
    done
    echo "error: reached max attempts at retry($*)"
    return 1
}

function wait_for_ssh {
    local ip=$1 && shift

    retry 10 30 "$(get_ssh_cmd) ${ip} -- true"
}

function get_ssh_cmd {
    echo "ssh -l cloud $(get_ssh_common_args)"
}

function get_ssh_common_args {
    local private_key_file=$(get_ssh_private_key_file)
    if [ -z "$private_key_file" ]; then
        # If there's no private key file use the public key instead
        # This allows us to specify a private key which is held only on a
        # hardware device and therefore has no key file
        private_key_file=$(get_ssh_public_key_file)
    fi

    echo "-i ${private_key_file} " \
         "-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o IdentitiesOnly=yes -o PasswordAuthentication=no "
}
