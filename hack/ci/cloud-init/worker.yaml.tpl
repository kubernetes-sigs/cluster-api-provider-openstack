- path: /tmp/local.conf
  permissions: 0644
  content: |
    [[local|localrc]]
    GIT_BASE=https://github.com
    HOST_IP=${HOST_IP}
    SERVICE_TIMEOUT=240

    # Enable Logging
    LOGFILE=/opt/stack/logs/stack.sh.log
    VERBOSE=True
    LOG_COLOR=True

    DATABASE_PASSWORD=secretdatabase
    RABBIT_PASSWORD=secretrabbit
    ADMIN_PASSWORD=secretadmin
    SERVICE_PASSWORD=secretservice
    SERVICE_TOKEN=111222333444

    SERVICE_HOST=${CONTROLLER_IP}
    RABBIT_HOST=$SERVICE_HOST
    GLANCE_HOSTPORT=$SERVICE_HOST:9292

    # Required to generate DB URL for c-vol
    DATABASE_TYPE=mysql
    DATABASE_HOST=$SERVICE_HOST

    # Neutron
    enable_plugin neutron https://github.com/openstack/neutron stable/${OPENSTACK_RELEASE}

    # Nova
    ENABLED_SERVICES=n-cpu,placement-client,c-vol,neutron-agent

    # Additional services
    ENABLED_SERVICES+=${OPENSTACK_ADDITIONAL_SERVICES}

    [[post-config|$NOVA_CONF]]
    [DEFAULT]
    cpu_allocation_ratio = 2.0

    [[post-config|$CINDER_CONF]]
    [DEFAULT]
    storage_availability_zone = ${SECONDARY_AZ}

    [[post-config|/$NEUTRON_CORE_PLUGIN_CONF]]
    [ml2]
    path_mtu = ${MTU}
- path: /root/devstack.sh
  permissions: 0755
  content: |
    #!/bin/bash

    set -x -o errexit -o nounset -o pipefail

    source /tmp/devstack-common.sh

    ensure_kvm

    # from https://raw.githubusercontent.com/openstack/octavia/master/devstack/contrib/new-octavia-devstack.sh
    git clone -b stable/${OPENSTACK_RELEASE} https://github.com/openstack/devstack.git /tmp/devstack
    cp /tmp/local.conf /tmp/devstack/

    # Create the stack user
    HOST_IP=${HOST_IP} /tmp/devstack/tools/create-stack-user.sh

    # Move everything into place (/opt/stack is the $HOME folder of the stack user)
    mv /tmp/devstack /opt/stack/
    chown -R stack:stack /opt/stack/devstack/

    run_devstack
