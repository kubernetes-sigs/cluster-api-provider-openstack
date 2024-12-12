- path: /tmp/local.conf
  permissions: "0644"
  content: |
    [[local|localrc]]
    GIT_BASE=https://github.com
    HOST_IP=${HOST_IP}
    SERVICE_TIMEOUT=240
    FLOATING_RANGE=${FLOATING_RANGE}

    # Enable Logging
    LOGFILE=/opt/stack/logs/stack.sh.log
    VERBOSE=True
    LOG_COLOR=True

    # Host tuning
    ENABLE_SYSCTL_MEM_TUNING="True"
    ENABLE_SYSCTL_NET_TUNING="True"
    ENABLE_ZSWAP="True"

    # Octavia
    enable_plugin octavia https://github.com/openstack/octavia stable/${OPENSTACK_RELEASE}
    enable_plugin octavia-dashboard https://github.com/openstack/octavia-dashboard stable/${OPENSTACK_RELEASE}

    DATABASE_PASSWORD=secretdatabase
    RABBIT_PASSWORD=secretrabbit
    ADMIN_PASSWORD=secretadmin
    SERVICE_PASSWORD=secretservice
    SERVICE_TOKEN=111222333444

    # Pre-requisite
    ENABLED_SERVICES=key,rabbit,mysql
    # Nova
    ENABLED_SERVICES+=,n-api,n-cpu,n-cond,n-sch,n-novnc,n-api-meta
    # Placement service needed for Nova
    ENABLED_SERVICES+=,placement-api,placement-client
    # Glance
    ENABLED_SERVICES+=,g-api

    # Neutron
    enable_plugin neutron https://github.com/openstack/neutron stable/${OPENSTACK_RELEASE}
    ENABLED_SERVICES+=,q-svc,neutron-trunk,ovn-controller,ovs-vswitchd,ovn-northd,ovsdb-server,q-ovn-metadata-agent
    
    DISABLED_SERVICES=q-agt,q-dhcp,q-l3,q-meta,q-metering
    PUBLIC_BRIDGE_MTU=${MTU}
    ENABLE_CHASSIS_AS_GW="True"
    OVN_DBS_LOG_LEVEL="dbg"
    Q_ML2_PLUGIN_MECHANISM_DRIVERS="ovn,logger"
    OVN_L3_CREATE_PUBLIC_NETWORK="True"
    Q_AGENT="ovn"

    # WORKAROUND:
    # 	https://github.com/kubernetes-sigs/cluster-api-provider-openstack/issues/2320
    # 	OVN built from source using LTS versions. Should be removed once OVS is more stable without the pin.
    # 	https://opendev.org/openstack/neutron/src/commit/83de306105f9329e24c97c4af6c3886de20e7d70/zuul.d/tempest-multinode.yaml#L603-L604
    OVN_BUILD_FROM_SOURCE=True
    OVN_BRANCH=branch-24.03
    OVS_BRANCH=branch-3.3

    # Octavia
    ENABLED_SERVICES+=,octavia,o-api,o-cw,o-hm,o-hk,o-da

    # Horizon (enable for manual tests)
    # ENABLED_SERVICES+=,horizon

    # Cinder
    ENABLED_SERVICES+=,c-sch,c-api,c-vol
    VOLUME_BACKING_FILE_SIZE=100G

    # Additional services
    ENABLED_SERVICES+=${OPENSTACK_ADDITIONAL_SERVICES}
    DISABLED_SERVICES+=${OPENSTACK_DISABLED_SERVICES}

    # Don't download default images, just our test images
    DOWNLOAD_DEFAULT_IMAGES=False
    # Increase the total image size limit
    GLANCE_LIMIT_IMAGE_SIZE_TOTAL=20000
    # We upload the Amphora image so it doesn't have to be build
    # Upload the images so we don't have to upload them from Prow
    # NOTE: If you get issues when changing/adding images, check if the limits
    # are sufficient and change the variable above if needed.
    # https://docs.openstack.org/glance/latest/admin/quotas.html
    IMAGE_URLS="https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/amphora/2022-12-05/amphora-x64-haproxy.qcow2,"
    IMAGE_URLS+="https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/cirros/2022-12-05/cirros-0.6.1-x86_64-disk.img,"
    IMAGE_URLS+="https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/ubuntu/2023-09-29/ubuntu-2204-kube-v1.27.2.img,"
    IMAGE_URLS+="https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/ubuntu/2024-01-10/ubuntu-2204-kube-v1.28.5.img,"
    IMAGE_URLS+="https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/flatcar/flatcar-stable-3815.2.0-kube-v1.28.5.img,"
    IMAGE_URLS+="https://stable.release.flatcar-linux.net/amd64-usr/current/flatcar_production_openstack_image.img"

    [[post-config|$NOVA_CONF]]
    [DEFAULT]
    # On GCE's n2-standard-16 an allocation ratio of 2.0 gives us 32 vCPUS,
    # which is enough to run any 2 test clusters concurrently.
    cpu_allocation_ratio = 2.0

    # We ensure that the controller has capacity to run all workloads, and that
    # all workloads run on the controller unless explicitly scheduled to the
    # worker. This prevents non-deterministic failures of multi-AZ tests due to
    # capacity on the worker.
    default_schedule_zone = ${PRIMARY_AZ}

    [scheduler]
    # query_placement_for_availability_zone is the default from Xena
    query_placement_for_availability_zone = True

    [workarounds]
    # FIXME(stephenfin): This is temporary while we get to the bottom of
    # https://bugs.launchpad.net/nova/+bug/2091114 It should not be kept after
    # we bump to 2025.1
    disable_deep_image_inspection = True

    [[post-config|$CINDER_CONF]]
    [DEFAULT]
    storage_availability_zone = ${PRIMARY_AZ}

    [[post-config|$NEUTRON_CONF]]
    [DEFAULT]
    global_physnet_mtu = ${MTU}
    service_plugins = trunk,router

    # The following are required for OVN to set default DNS when a subnet is
    # created without specifying DNS servers.
    # Not specifying these will result in the default DNS servers being set to
    # 127.0.0.53 which might be problematic in some environments.
    [[post-config|/$Q_PLUGIN_CONF_FILE]]
    [ovn]
    dns_servers = ${OPENSTACK_DNS_NAMESERVERS}
- path: /tmp/register-worker.sh
  permissions: "0755"
  content: |
    #!/bin/bash

    source /opt/stack/devstack/openrc admin admin

    # Wait until the worker shows up as a second compute service
    while [ $(openstack compute service list --service nova-compute -f value | wc -l) -lt 2 ]
    do
      sleep 60
    done

    nova-manage cell_v2 discover_hosts

    # Look for hypervisors other than the current host and add them to a
    # secondary AZ
    if ! openstack aggregate show ${SECONDARY_AZ} > /dev/null 2>&1; then
      openstack aggregate create --zone ${SECONDARY_AZ} ${SECONDARY_AZ}
    fi

    for hypervisor in $(openstack hypervisor list -f value -c "Hypervisor Hostname" 2>/dev/null | grep -v $(hostname)); do
      openstack aggregate add host ${SECONDARY_AZ} ${hypervisor}
    done
- path: /etc/systemd/system/register-worker.service
  permissions: "0644"
  content: |
    [Unit]
    Description=Register devstack worker node

    [Service]
    Type=oneshot
    User=stack
    ExecStart=/tmp/register-worker.sh
    Environment=TERM=ansi

    [Install]
    WantedBy=multi-user.target
- path: /tmp/devstack-post.sh
  permissions: "0755"
  content: |
    #!/bin/bash

    set -o -x

    # Add environment variables for auth/endpoints
    echo 'source /opt/stack/devstack/openrc admin admin' >> /opt/stack/.bashrc

    source /opt/stack/devstack/openrc admin admin

    # Add the controller to its own host aggregate and availability zone
    aggregateid=$(openstack aggregate create --zone "${PRIMARY_AZ}" "${PRIMARY_AZ}" -f value -c id)
    for host in $(openstack compute service list --service nova-compute -f value -c Host)
    do
        openstack aggregate add host "$aggregateid" "$host"
    done

    # Create the volume type
    VOLUME_TYPE_NAME="test-volume-type"
    if openstack volume type create --description "Test volume type" --public "${VOLUME_TYPE_NAME}" &> /dev/null; then
        echo "Volume type '${VOLUME_TYPE_NAME}' created successfully."
    else
        echo "Error: Failed to create volume type '${VOLUME_TYPE_NAME}'."
    fi

    # the flavors are created in a way that we can execute at least 2 e2e tests in parallel (overall we have 32 vCPUs)
    openstack flavor delete m1.tiny
    openstack flavor create --ram 512 --disk 1 --ephemeral 1 --vcpus 1 --public --id 1 m1.tiny --property hw_rng:allowed='True'
    openstack flavor delete m1.small
    openstack flavor create --ram 4192 --disk 20 --ephemeral 5 --vcpus 2 --public --id 2 m1.small --property hw_rng:allowed='True'
    openstack flavor delete m1.medium
    openstack flavor create --ram 6144 --disk 20 --ephemeral 5 --vcpus 2 --public --id 3 m1.medium --property hw_rng:allowed='True'
    # Create an additional flavor for the e2e tests that will be used by the e2e bastion tests
    openstack flavor create --ram 512 --disk 1 --ephemeral 1 --vcpus 1 --public --id 10 m1.tiny.alt --property hw_rng:allowed='True'

    # Adjust the CPU quota
    openstack quota set --cores 32 demo
    openstack quota set --secgroups 200 demo
    openstack quota set --secgroup-rules 1000 demo
    openstack quota set --secgroups 100 admin
    openstack quota set --secgroup-rules 1000 admin
- path: /root/devstack.sh
  permissions: "0755"
  content: |
    #!/bin/bash

    set -o -x -o errexit -o nounset -o pipefail

    source /tmp/devstack-common.sh

    ensure_kvm

    # from https://raw.githubusercontent.com/openstack/octavia/master/devstack/contrib/new-octavia-devstack.sh
    git clone -b stable/${OPENSTACK_RELEASE} https://github.com/openstack/devstack.git /tmp/devstack
    cp /tmp/local.conf /tmp/devstack/

    # Create the stack user
    HOST_IP=${HOST_IP} /tmp/devstack/tools/create-stack-user.sh
    chmod 0755 /opt/stack

    # Move everything into place (/opt/stack is the $HOME folder of the stack user)
    mv /tmp/devstack /opt/stack/
    chown -R stack:stack /opt/stack/devstack/

    run_devstack

    # Run post-configuration as stack user
    su - stack -c /tmp/devstack-post.sh

    # When using ML2/OVS all public traffic will be routed via the L3 agent,
    # which is only running on the controller
    INTERFACE=$(ip -j addr show | jq -re 'map(select(.addr_info | map(.local == "${HOST_IP}") | any)) | first | .ifname')
    sudo iptables -t nat -I POSTROUTING -o ${INTERFACE} -s ${FLOATING_RANGE} -j MASQUERADE
    sudo iptables -I FORWARD -s ${FLOATING_RANGE} -j ACCEPT

    # Start polling for the worker node
    # We defined the register-worker unit above
    systemctl daemon-reload
    systemctl start --no-block register-worker
