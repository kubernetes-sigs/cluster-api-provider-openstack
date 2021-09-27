- content: |
    #!/bin/bash

    set -o errexit -o nounset -o pipefail

    source /tmp/devstack-common-kvm.sh
    source /tmp/devstack-common-resize-disks.sh

    # Run stack.sh and post-install config
    source /tmp/devstack-common-install.sh
  path: /root/devstack.sh
  permissions: 0755
