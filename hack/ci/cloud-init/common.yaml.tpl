#cloud-config
runcmd:
- sysctl -p /etc/sysctl.d/devstack.conf
- /root/devstack.sh
bootcmd:
# Start a lightweight resource sampler as early as possible, as a separate
# systemd unit so that it keeps running after cloud-init finishes. It records
# load, CPU steal, pressure stall information and memory every 10s, and the
# time taken by a fixed CPU and disk workload every minute. It also measures
# how long it takes to write to the syslog, the journal and the console, and
# records what journald and rsyslogd are blocked on. This allows us to tell
# slow or stalled VMs apart from problems in devstack itself. The output is
# collected by hack/ci/create_devstack.sh.
- |
  cat > /usr/local/bin/capo-sampler.sh <<'EOF'
  #!/bin/sh
  # Run a command every 10s in the background, and log how long it took.
  # The line is stamped with the start time. A blocked command only blocks its
  # own loop, so the stall itself is visible as a missing or very slow sample.
  timed() {
    name=$1
    shift
    while true; do
      t0=$(date -u +%Y-%m-%dT%H:%M:%SZ)
      s=$(date +%s%N)
      "$@" >/dev/null 2>&1
      e=$(date +%s%N)
      echo "$t0 $name-ms: $(((e-s)/1000000))"
      sleep 10
    done
  }
  # What is a process blocked on? Kernel stack needs root, which we are.
  procstate() {
    pid=$(pidof "$1" | cut -d' ' -f1)
    [ -n "$pid" ] || return 0
    echo "$ts $1: $(ps -o stat=,wchan:32= -p "$pid" | tr -s ' ' ' ')"
    stack=$(sed -e 's/^\[<[0-9a-f]*>\] //' -e 's/+.*//' "/proc/$pid/stack" 2>/dev/null | tr '\n' ' ')
    echo "$ts $1-stack: $stack"
  }
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) kernel: $(uname -r)"
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) cmdline: $(cat /proc/cmdline)"
  timed logger logger -t capo-probe probe &
  timed journal-stream systemd-cat -t capo-probe echo probe &
  timed console sh -c 'echo capo-probe > /dev/console' &
  i=0
  while true; do
    ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    echo "$ts loadavg: $(cat /proc/loadavg)"
    echo "$ts stat: $(head -n 1 /proc/stat)"
    echo "$ts meminfo: $(grep -E '^(MemAvailable|SwapFree|Dirty|Writeback):' /proc/meminfo | tr -s ' \n' ' ')"
    for r in cpu io memory; do
      echo "$ts psi-$r: $(head -n 1 /proc/pressure/$r 2>/dev/null)"
    done
    procstate systemd-journald
    procstate rsyslogd
    # Unix sockets with data queued, i.e. someone is not reading
    echo "$ts unix-queues: $(ss -xanp 2>/dev/null | awk '$2!="LISTEN" && ($3+0>0 || $4+0>0)' | head -n 10 | cut -c1-160 | tr -s ' \n' ' ')"
    # Processes in uninterruptible sleep
    echo "$ts d-state: $(ps -eo pid=,stat=,wchan:24=,comm= | awk '$2 ~ /^D/' | head -n 15 | tr -s ' \n' ' ')"
    if [ $((i % 6)) -eq 0 ]; then
      s=$(date +%s%N)
      n=0
      while [ $n -lt 100000 ]; do n=$((n+1)); done
      e=$(date +%s%N)
      echo "$ts cpu-probe-ms: $(((e-s)/1000000))"
      s=$(date +%s%N)
      dd if=/dev/zero of=/var/tmp/capo-sampler.probe bs=1M count=32 oflag=direct conv=fsync 2>/dev/null
      e=$(date +%s%N)
      rm -f /var/tmp/capo-sampler.probe
      echo "$ts disk-probe-ms: $(((e-s)/1000000))"
    fi
    i=$((i+1))
    sleep 10
  done
  EOF
  chmod +x /usr/local/bin/capo-sampler.sh
  # bootcmd runs in the cloud-init network stage, which is ordered before
  # sshd.service and sysinit.target. A transient unit with default dependencies
  # starts after basic.target, so waiting for it here would deadlock boot and
  # sshd would never start. Hence --no-block and DefaultDependencies=no.
  systemd-run --no-block --unit=capo-sampler --collect --property=DefaultDependencies=no \
    /bin/sh -c '/usr/local/bin/capo-sampler.sh >> /var/log/capo-sampler.log 2>&1'
final_message: "The system is finally up, after $UPTIME seconds"
users:
- name: cloud
  lock_passwd: true
  sudo: ALL=(ALL) NOPASSWD:ALL
  ssh_authorized_keys:
  - ${SSH_PUBLIC_KEY}
# Infrastructure packages required:
#   python3 - required by sshuttle
#   git - required to obtain devstack
#   jq - required by devstack-common.sh
packages:
- python3
- git
- jq
package_upgrade: true
write_files:
- path: /etc/sysctl.d/devstack.conf
  permissions: "0644"
  content: |
    net.ipv4.ip_forward=1
    net.ipv4.conf.default.rp_filter=0
    net.ipv4.conf.all.rp_filter=0
- path: /tmp/devstack-common.sh
  permissions: "0644"
  content: |
    # ensure nested virtualization
    function ensure_kvm {
      sudo modprobe kvm-intel
      if [ ! -c /dev/kvm ]; then
          echo /dev/kvm is not present
          exit 1
      fi
    }

    function run_devstack {
      su - stack -c "TERM=vt100 /opt/stack/devstack/stack.sh"
    }
