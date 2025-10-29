- path: /tmp/local.conf
  permissions: "0644"
  content: |
    [[local|localrc]]
    GIT_BASE=https://github.com
    HOST_IP=${HOST_IP}
    SERVICE_TIMEOUT=240

    # Enable Logging
    LOGFILE=/opt/stack/logs/stack.sh.log
    VERBOSE=True
    LOG_COLOR=False

    # Host tuning
    ENABLE_SYSCTL_MEM_TUNING="True"
    ENABLE_SYSCTL_NET_TUNING="True"
    ENABLE_ZSWAP="True"

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

    # Nova
    ENABLED_SERVICES=n-cpu,placement-client,c-vol
    VOLUME_BACKING_FILE_SIZE=100G

    # Neutron
    enable_plugin neutron https://github.com/openstack/neutron stable/${OPENSTACK_RELEASE}
    ENABLED_SERVICES+=,ovn-controller,ovs-vswitchd,ovsdb-server,q-fake,q-ovn-metadata-agent
    DISABLED_SERVICES=q-svc,horizon,ovn-northd,q-agt,q-dhcp,q-l3,q-meta,q-metering,q-vpn
    PUBLIC_BRIDGE_MTU=${MTU}
    ENABLE_CHASSIS_AS_GW="False"
    OVN_DBS_LOG_LEVEL="dbg"
    Q_ML2_PLUGIN_MECHANISM_DRIVERS="ovn"
    Q_AGENT="ovn"

    # WORKAROUND:
    # 	https://github.com/kubernetes-sigs/cluster-api-provider-openstack/issues/2320
    # 	OVN built from source using LTS versions. Should be removed once OVS is more stable without the pin.
    # 	https://opendev.org/openstack/neutron/src/commit/83de306105f9329e24c97c4af6c3886de20e7d70/zuul.d/tempest-multinode.yaml#L603-L604
    OVN_BUILD_FROM_SOURCE=True
    OVN_BRANCH=branch-24.03
    OVS_BRANCH=branch-3.3

    # Additional services
    ENABLED_SERVICES+=${OPENSTACK_ADDITIONAL_SERVICES}
    DISABLED_SERVICES+=${OPENSTACK_DISABLED_SERVICES}

    [[post-config|$NOVA_CONF]]
    [DEFAULT]
    cpu_allocation_ratio = 2.0

    [workarounds]
    # FIXME(stephenfin): This is temporary while we get to the bottom of
    # https://bugs.launchpad.net/nova/+bug/2091114 It should not be kept after
    # we bump to 2025.1
    disable_deep_image_inspection = True

    [[post-config|$CINDER_CONF]]
    [DEFAULT]
    storage_availability_zone = ${SECONDARY_AZ}

    [[post-config|$NEUTRON_CONF]]
    [DEFAULT]
    global_physnet_mtu = ${MTU}
- path: /root/devstack.sh
  permissions: "0755"
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
