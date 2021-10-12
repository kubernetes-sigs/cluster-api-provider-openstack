- path: /tmp/local.conf
  permissions: 0644
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

    # Neutron
    enable_plugin neutron https://github.com/openstack/neutron stable/${OPENSTACK_RELEASE}

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
    ENABLED_SERVICES+=,n-api,n-obj,n-cpu,n-cond,n-sch,n-novnc,n-api-meta
    # Placement service needed for Nova
    ENABLED_SERVICES+=,placement-api,placement-client
    # Glance
    ENABLED_SERVICES+=,g-api,g-reg

    # Octavia-Neutron
    ENABLED_SERVICES+=,neutron-api,neutron-agent,neutron-dhcp,neutron-l3
    ENABLED_SERVICES+=,neutron-metadata-agent,neutron-qos
    # Octavia
    ENABLED_SERVICES+=,octavia,o-api,o-cw,o-hm,o-hk,o-da

    # Horizon (enable for manual tests)
    # ENABLED_SERVICES+=,horizon

    # Cinder
    ENABLED_SERVICES+=,c-sch,c-api,c-vol

    # Additional services
    ENABLED_SERVICES+=${OPENSTACK_ADDITIONAL_SERVICES}

    LIBVIRT_TYPE=kvm

    # Don't download default images, just our test images
    DOWNLOAD_DEFAULT_IMAGES=False
    # We upload the Amphora image so it doesn't have to be build
    IMAGE_URLS="https://storage.googleapis.com/artifacts.k8s-staging-capi-openstack.appspot.com/test/amphora/2021-03-27/amphora-x64-haproxy.qcow2"

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

    [[post-config|$CINDER_CONF]]
    [DEFAULT]
    storage_availability_zone = ${PRIMARY_AZ}

    [[post-config|/$NEUTRON_CORE_PLUGIN_CONF]]
    [ml2]
    path_mtu = ${MTU}
- path: /tmp/register-worker.sh
  permissions: 0755
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
  permissions: 0644
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
    chmod 0755 /opt/stack

    # Move everything into place (/opt/stack is the $HOME folder of the stack user)
    mv /tmp/devstack /opt/stack/
    chown -R stack:stack /opt/stack/devstack/

    run_devstack
    upload_images

    # When using ML2/OVS all public traffic will be routed via the L3 agent,
    # which is only running on the controller
    INTERFACE=$(ip -j addr show | jq -re 'map(select(.addr_info | map(.local == "${HOST_IP}") | any)) | first | .ifname')
    sudo iptables -t nat -I POSTROUTING -o ${INTERFACE} -s ${FLOATING_RANGE} -j MASQUERADE
    sudo iptables -I FORWARD -s ${FLOATING_RANGE} -j ACCEPT

    # Start polling for the worker node
    # We defined the register-worker unit above
    systemctl daemon-reload
    systemctl start --no-block register-worker
