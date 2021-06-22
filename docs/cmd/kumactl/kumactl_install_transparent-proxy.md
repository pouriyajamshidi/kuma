## kumactl install transparent-proxy

Install Transparent Proxy pre-requisites on the host

### Synopsis

Install Transparent Proxy by modifying the hosts iptables and /etc/resolv.conf.

Follow the following steps to use the Kuma data plane proxy in Transparent Proxy mode:

 1) create a dedicated user for the Kuma data plane proxy, e.g. 'kuma-dp'
 2) run this command as a 'root' user to modify the host's iptables and /etc/resolv.conf
    - supply the dedicated username with '--kuma-dp-'
    - all changes are easly revertible by issuing 'kumactl uninstall transparent-proxy'
    - by default the SSH port tcp/22 will not be redirected to Envoy, but everything else will.
      Use '--exclude-inbound-ports' to provide a comma separated list of ports that should also be excluded
    - this command also creates a backup copy of the modified resolv.conf under /etc/resolv.conf

 sudo kumactl install transparent-proxy \
          --kuma-dp-user kuma-dp \
          --kuma-cp-ip 10.0.0.1 \
          --exclude-inbound-ports 443

 3) prepare a Dataplane resource yaml like this:

type: Dataplane
mesh: default
name: {{ name }}
networking:
  address: {{ address }}
  inbound:
  - port: {{ port }}
    tags:
      kuma.io/service: demo-client
  transparentProxying:
    redirectPortInbound: 15006
    redirectPortOutbound: 15001

The values in 'transparentProxying' section are the defaults set by this command and if needed be changed by supplying 
'--redirect-inbound-port' and '--redirect-outbound-port' respectively.

 4) the kuma-dp command shall be run with the designated user. 
    - if using systemd to run add 'User=kuma-dp' in the '[Service]' section of the service file
    - leverage 'runuser' similar to (assuming aforementioned yaml):

runuser -u kuma-dp -- \
  /usr/bin/kuma-dp run \
    --cp-address=https://172.19.0.2:5678 \
    --dataplane-token-file=/kuma/token-demo \
    --dataplane-file=/kuma/dpyaml-demo \
    --dataplane-var name=dp-demo \
    --dataplane-var address=172.19.0.4 \
    --dataplane-var port=80  \
    --binary-path /usr/local/bin/envoy

 5) make sure the kuma-cp is running its DNS service on port 53 by setting the environment variable 'KUMA_DNS_SERVER_PORT=53'



```
kumactl install transparent-proxy [flags]
```

### Options

```
      --dry-run                                                                         dry run
      --exclude-inbound-ports string                                                    a comma separated list of inbound ports to exclude from redirect to Envoy
      --exclude-outbound-ports string                                                   a comma separated list of outbound ports to exclude from redirect to Envoy
  -h, --help                                                                            help for transparent-proxy
      --kuma-cp-ip ip                                                                   the IP address of the Kuma CP which exposes the DNS service on port 53. (default 0.0.0.0)
      --kuma-dp-uid string                                                              the UID of the user that will run kuma-dp
      --kuma-dp-user string                                                             the user that will run kuma-dp
      --modify-iptables                                                                 modify the host iptables to redirect the traffic to Envoy (default true)
      --redirect-all-dns-traffic                                                        redirect all DNS requests to a specified port. Implies --redirect-dns.
      --redirect-dns                                                                    redirect all DNS requests to the servers in /etc/resolv.conf to a specified port
      --redirect-dns-port string                                                        the port where the DNS agent is listening (default "15053")
      --redirect-dns-upstream-target-chain string                                       (optional) the iptables chain where the upstream DNS requests should be directed to. It is only applied for IP V4. Use with care. (default "RETURN")
      --redirect-inbound                                                                redirect the inbound traffic to the Envoy. Should be disabled for Gateway data plane proxies. (default true)
      --redirect-inbound-port networking.transparentProxying.redirectPortInbound        inbound port redirected to Envoy, as specified in dataplane's networking.transparentProxying.redirectPortInbound (default "15006")
      --redirect-inbound-port-v6 networking.transparentProxying.redirectPortInboundV6   IPv6 inbound port redirected to Envoy, as specified in dataplane's networking.transparentProxying.redirectPortInboundV6 (default "15010")
      --redirect-outbound-port networking.transparentProxying.redirectPortOutbound      outbound port redirected to Envoy, as specified in dataplane's networking.transparentProxying.redirectPortOutbound (default "15001")
      --skip-resolv-conf /etc/resolv.conf                                               skip modifying the host /etc/resolv.conf
      --store-firewalld                                                                 store the iptables changes with firewalld
      --verbose                                                                         verbose
```

### Options inherited from parent commands

```
      --config-file string   path to the configuration file to use
      --log-level string     log level: one of off|info|debug (default "off")
  -m, --mesh string          mesh to use (default "default")
      --no-config            if set no config file and config directory will be created
```

### SEE ALSO

* [kumactl install](kumactl_install.md)	 - Install various Kuma components.
