# XMRay
XRay-core server for NuxtJs version of XMPlus management panel

#### Config directory
```
cd /etc/XMRay
```

### Onclick XMRay backennd Install
```
bash <(curl -Ls https://raw.githubusercontent.com/XMPlusDev/XMRay/script/install.sh)
```

### /etc/XMRay/config.yml
```
Log:
  Level: none 										# Log level: none, error, warning, info, debug 
  AccessPath: 										#/etc/XMRay/access.Log
  ErrorPath: 										#/etc/XMRay/error.log
  DNSLog: false  									# true or false Whether to enable DNS query log
  MaskAddress: half 								# half, full, quater
DnsConfigPath:  								    #/etc/XMRay/dns.json - #https://xtls.github.io/config/dns.html
RouteConfigPath: 									#/etc/XMRay/route.json - #https://xtls.github.io/config/routing.html
OutboundConfigPath: 								#/etc/XMRay/outbound.json - #https://xtls.github.io/config/outbound.html
ConnectionConfig:
  Handshake: 8 
  ConnIdle: 120 
  UplinkOnly: 0 
  DownlinkOnly: 0 
  BufferSize: 64
ReverbConfig:
  - Enable: false             						# Enable websocket to trigger real-time subscription and node updates from panel
    Host: "api.xyz.com:443"   						# Reverb REVERB_HOST:REVERB_PORT  in .env for api /home/XMPlusPanel/.env 
    AppKey:                   						# REVERB_APP_KEY in .env for api /home/XMPlusPanel/.env
    AppSecret:                						# REVERB_APP_SECRET in .env for api /home/XMPlusPanel/.env
    UseTLS: true              						# Set to true if tls enabled for api
Nodes:
  -
    ApiConfig:
      ApiHost: "https://api.xyz.com"
      ApiKey: "123"
      NodeID: 1
      Timeout: 30 
    ControllerConfig:
      EnableDNS: true 								# Use custom DNS config, Please ensure that you set the dns.json well
      DNSStrategy: AsIs 							# AsIs, UseIP, UseIPv4, UseIPv6
      CertConfig:
        Email: author@cert.xyz                    	# Required when Cert Mode is not none
        CertFile: /etc/XMRay/node1.crt  			# Required when Cert Mode is file
        KeyFile: /etc/XMRay/node1.key   			# Required when Cert Mode is file
        Provider: cloudflare                        # Required when Cert Mode is dns
        CertEnv:                                    # Required when Cert Mode is dns
          CLOUDFLARE_EMAIL:                         # Required when Cert Mode is dns
          CLOUDFLARE_API_KEY:                       # Required when Cert Mode is dns
      EnableFallback: false 						# Only support for Trojan and Vlesshttps://xtls.github.io/config/features/fallback.html
      FallBackConfigs:  							# Support multiple fallbacks
        - SNI: 										# TLS SNI(Server Name Indication), Empty for any
          Alpn: 									# Alpn, Empty for any
          Path: 									# HTTP PATH, Empty for any
          Dest: 80 									# Required, Destination of fallback.
          ProxyProtocolVer: 0 						# Send PROXY protocol version, 0 for disable
      RedisConfig:
        Enable: false 								# Enable the global ip limit of a user
        Network: tcp 								# Redis protocol, tcp or unix
        Addr: 127.0.0.1:6379 						# Redis server address, or unix socket path
        Username: 									# Redis username
        Password: 									# Redis password
        DB: 0 										# Redis DB
        Timeout: 10 								# Timeout for redis request
```

## XMPlus Panel Server configuration

### Network Settings

#### TCP
```
{
  "encryption": "none",
  "decryption": "none",
  "flow": "xtls-rprx-vision",
  "cipher": "aes-128-gcm",
  "sniffing": true,
  "listeningIP": "0.0.0.0",
  "listeningPort": "443-443",
  "sendThroughIP": "0.0.0.0",
  "transportProtocol": {
    "type": "raw",
    "settings": {
      "acceptProxyProtocol": false,
      "header": {
        "type": "none"
      }
    }
  }
}
```
#### TCP + HTTP
```
{
  "encryption": "none",
  "decryption": "none",
  "cipher": "aes-128-gcm",
  "sniffing": true,
  "listeningIP": "0.0.0.0",
  "listeningPort": "443-443",
  "sendThroughIP": "0.0.0.0",
  "transportProtocol": {
    "type": "raw",
    "settings": {
      "acceptProxyProtocol": false,
      "header": {
        "type": "http",
        "request": {
          "path": ["/"],
          "headers": {
            "Host": ["www.baidu.com", "www.bing.com"]
          }
        }
      }
    }
  }
}
```
####  WS
```
{
  "encryption": "none",
  "decryption": "none",
  "cipher": "aes-128-gcm",
  "sniffing": true,
  "listeningIP": "0.0.0.0",
  "listeningPort": "443-443",
  "sendThroughIP": "0.0.0.0",
  "transportProtocol": {
    "type": "ws",
    "settings": {
      "acceptProxyProtocol": false,
      "host": "tld.dev",
      "path": "/",
      "heartbeat": 60,
      "custom_host": "tld.dev"
    }
  }
}
```

####  GRPC
```
{
  "encryption": "none",
  "decryption": "none",
  "cipher": "aes-128-gcm",
  "sniffing": true,
  "listeningIP": "0.0.0.0",
  "listeningPort": "443-443",
  "sendThroughIP": "0.0.0.0",
  "acceptProxyProtocol": false,
  "transportProtocol": {
    "type": "grpc",
    "settings": {
      "servicename": "tld",
      "authority": "tld.dev",
      "user_agent": "",
      "initial_windows_size": 0,
      "idle_timeout": 0,
      "health_check_timeout": 0,
      "permit_without_stream": false
    }
  }
}
```

####  HTTPUPGRADE
```
{
  "encryption": "none",
  "decryption": "none",
  "cipher": "aes-128-gcm",
  "sniffing": true,
  "listeningIP": "0.0.0.0",
  "listeningPort": "443-443",
  "sendThroughIP": "0.0.0.0",
  "transportProtocol": {
    "type": "httpupgrade",
    "settings": {
      "acceptProxyProtocol": false,
      "host": "tld.dev",
      "path": "/",
      "custom_host": "tld.dev"
    }
  }
}
```

####  XHTTP

XHTTP is an HTTP/2 and HTTP/3 based transport that splits uplink and downlink traffic into separate HTTP requests. It supports CDN deployments and padding obfuscation to bypass CDN-level traffic detection.

##### Basic configuration

```json
{
  "encryption": "none",
  "decryption": "none",
  "cipher": "aes-128-gcm",
  "sniffing": true,
  "listeningIP": "0.0.0.0",
  "listeningPort": "443",
  "sendThroughIP": "0.0.0.0",
  "transportProtocol": {
    "type": "xhttp",
    "settings": {
      "host": "tld.dev",
      "path": "/",
      "mode": "auto",
      "noSSEHeader": false,
      "scMaxBufferedPosts": 30,
      "scMaxEachPostBytes": 1000000,
      "scStreamUpServerSecs": "20-80",
      "xPaddingBytes": "100-1000"
    }
  }
}
```

##### `mode` values

| Value | Description |
|---|---|
| `auto` | Automatically selects between `packet-up` and `stream-up` based on client capability |
| `packet-up` | Uplink sent as individual HTTP POST requests. Works with most CDNs |
| `stream-up` | Uplink streamed over a single long-lived HTTP connection. Higher performance but requires CDN POST streaming support |
| `stream-one` | Single bidirectional stream. No `downloadSettings` allowed |

##### Core settings

| Field | Type | Default | Description |
|---|---|---|---|
| `host` | string | — | HTTP Host header value |
| `path` | string | `"/"` | HTTP request path |
| `mode` | string | `"auto"` | Transport mode — see table above |
| `noSSEHeader` | bool | `false` | Suppress the `Content-Type: text/event-stream` header on the downlink response |
| `scMaxEachPostBytes` | int | `1000000` | Max bytes per uplink POST request. Only applies in `packet-up` mode |
| `scMaxBufferedPosts` | int | `30` | Max number of POST requests buffered server-side before backpressure |
| `scStreamUpServerSecs` | string | `"20-80"` | Range (seconds) the server keeps a stream-up connection open before rotating. Format: `"min-max"` |
| `xPaddingBytes` | string | `"100-1000"` | Range of random padding bytes added to requests. Format: `"min-max"` |

---

##### Padding obfuscation (CDN detection bypass)

These fields enable obfuscation of the padding pattern to bypass CDN-level traffic fingerprinting (e.g. blocking of the default `x_padding=XXXX...` query parameter pattern).

> **Important:** All obfuscation fields must match exactly between server and client. Fields are set directly in `settings` on the server side. On the client/relay side XMRay automatically wraps them in the required `extra` block.

```json
{
  "transportProtocol": {
    "type": "xhttp",
    "settings": {
      "host": "tld.dev",
      "path": "/",
      "mode": "auto",
      "scStreamUpServerSecs": "20-80",
      "xPaddingBytes": "100-1000",
      "xPaddingObfsMode": true,
      "xPaddingMethod": "tokenish",
      "xPaddingPlacement": "queryInHeader",
      "xPaddingKey": "_dc",
      "xPaddingHeader": "X-Cache",
      "uplinkHTTPMethod": "POST",
      "sessionPlacement": "path",
      "sessionKey": "",
      "seqPlacement": "path",
      "seqKey": ""
    }
  }
}
```

##### Obfuscation fields

| Field | Type | Default | Description |
|---|---|---|---|
| `xPaddingObfsMode` | bool | `false` | Enable padding obfuscation mode. When `true` the padding key, header, and method are customised |
| `xPaddingMethod` | string | `"repeat-x"` | Padding generation method. `"repeat-x"` repeats the character `X`; `"tokenish"` generates a token-like string of mixed characters that resembles a real CDN cache key or auth token |
| `xPaddingPlacement` | string | `"queryInHeader"` | Where the padding value is placed in the request — see placement options below |
| `xPaddingKey` | string | `"x_padding"` | Query parameter name or cookie name used to carry the padding value. Used when `xPaddingPlacement` is `query`, `cookie`, or `queryInHeader` |
| `xPaddingHeader` | string | `"X-Padding"` | HTTP header name used to carry the padding value. Used when `xPaddingPlacement` is `header` or `queryInHeader` |
| `uplinkHTTPMethod` | string | `"POST"` | HTTP method for uplink requests. `"PUT"` and `"PATCH"` are often not blocked by CDNs that block `POST`. `"GET"` is only allowed in `packet-up` mode |
| `sessionPlacement` | string | `"path"` | Where the session ID is placed — see placement options below |
| `sessionKey` | string | auto | Key name for the session ID when `sessionPlacement` is not `path`. Defaults to `x_session` (query/cookie) or `X-Session` (header) |
| `seqPlacement` | string | `"path"` | Where the request sequence number is placed — see placement options below |
| `seqKey` | string | auto | Key name for the sequence number when `seqPlacement` is not `path`. Defaults to `x_seq` (query/cookie) or `X-Seq` (header) |

##### Placement options

| Value | Description |
|---|---|
| `queryInHeader` | Padding key sent as a query parameter **and** the padding value sent in a header. Combines both for maximum CDN compatibility |
| `query` | Sent as a URL query parameter: `?key=value` |
| `cookie` | Sent as an HTTP `Cookie` header: `Cookie: key=value` |
| `header` | Sent as a standalone HTTP header: `Key: value` |
| `path` | (session/seq only) Embedded directly in the URL path |

---

##### Example: CDN obfuscation with custom method and session headers

A configuration that disguises XHTTP traffic as normal CDN cache validation requests:

```json
{
  "transportProtocol": {
    "type": "xhttp",
    "settings": {
      "host": "tld.dev",
      "path": "/assets/bundle",
      "mode": "auto",
      "xPaddingObfsMode": true,
      "xPaddingMethod": "tokenish",
      "xPaddingPlacement": "queryInHeader",
      "xPaddingKey": "_dc",
      "xPaddingHeader": "X-Cache",
      "uplinkHTTPMethod": "PUT",
      "sessionPlacement": "header",
      "sessionKey": "X-Request-ID",
      "seqPlacement": "query",
      "seqKey": "fragment"
    }
  }
}
```

> `uplinkHTTPMethod: "PUT"` requires `mode: "packet-up"` or `mode: "auto"` (auto will use packet-up). It cannot be used with `stream-up` or `stream-one`.

---

##### Recommended key names for obfuscation

Choose names that blend in with real CDN traffic. Some suggestions:

**For `xPaddingKey` (query/cookie):** `_dc`, `cf`, `t`, `ts`, `bust`, `v`, `rev`, `cb`, `cache_key`

**For `xPaddingHeader` (header):** `X-Cache`, `X-CDN-Geo`, `X-Signature`, `X-Request-ID`, `CF-Cache-Status`

**For `sessionKey`:** `X-Request-ID`, `X-Client-ID`, `X-Auth-Token`, `sid`, `token`, `visitor_id`

**For `seqKey`:** `chunk`, `fragment`, `part`, `segment`, `offset`, `range`

---

####  KCP
```
{
  "encryption": "none",
  "decryption": "none",
  "cipher": "aes-128-gcm",
  "sniffing": true,
  "listeningIP": "0.0.0.0",
  "listeningPort": "443-443",
  "sendThroughIP": "0.0.0.0",
  "transportProtocol": {
    "type": "kcp",
    "settings": {
      "mtu": 1350
    }
  }
}
```

####  HYSTERIA
```
{
  "encryption": "none",
  "decryption": "none",
  "sniffing": true,
  "listeningIP": "0.0.0.0",
  "listeningPort": "443-443",
  "sendThroughIP": "0.0.0.0",
  "transportProtocol": {
    "type": "hysteria",
    "settings": {
      "version": 2
    }
  }
}
```

### Mask Settings

[FinalMask](https://xtls.github.io/config/transports/finalmask.html)

`maskSettings` is optional and applies transport-level obfuscation. All three fields (`tcp`, `udp`, `quicParams`) are optional and can be used independently or together.

#### TCP mask types: `header-custom`, `fragment`, `sudoku`
#### UDP mask types: `header-custom`, `header-dns`, `header-dtls`, `header-srtp`, `header-utp`, `header-wechat`, `header-wireguard`, `mkcp-original`, `mkcp-aes128gcm`, `noise`, `salamander`, `sudoku`, `xdns`, `xicmp`

```json
{
  "maskSettings": {
    "tcp": [
      {
        "type": "fragment",
        "settings": {
          "packets": "tlshello",
          "length": { "from": 100, "to": 200 },
          "delay": { "from": 10, "to": 20 },
          "maxSplit": { "from": 0, "to": 0 }
        }
      }
    ],
    "udp": [
      {
        "type": "noise",
        "settings": {
          "reset": { "from": 0, "to": 0 },
          "noise": [
            {
              "type": "str",
              "packet": "GET / HTTP/1.1\r\n",
              "rand": { "from": 0, "to": 0 },
              "delay": { "from": 10, "to": 50 }
            }
          ]
        }
      }
    ],
    "quicParams": {
      "congestion": "bbr",
      "debug": false,
	  "bbrProfile": "standard",
      "brutalUp": "100mbps",
      "brutalDown": "100mbps",
      "udpHop": {
        "ports": ["443,8443"],
        "interval": { "from": 10, "to": 30 }
      },
      "initStreamReceiveWindow": 8388608,
      "maxStreamReceiveWindow": 8388608,
      "initConnectionReceiveWindow": 20971520,
      "maxConnectionReceiveWindow": 20971520,
      "maxIdleTimeout": 30,
      "keepAlivePeriod": 10,
      "disablePathMTUDiscovery": false,
      "maxIncomingStreams": 100
    }
  }
}
```
[quicParams](https://xtls.github.io/config/transports/finalmask.html#quicparams)

`quicParams` fields:

| Field | Type | Description |
|---|---|---|
| `congestion` | string | Congestion control algorithm, e.g. `"bbr"`, `"cubic"` |
| `debug` | bool | Enable debug mode |
| `bbrProfile` | string | Congestion control algorithm, e.g. `"conservative"`, `"standard"`, `"aggressive"` |
| `brutalUp` | string | Upload bandwidth for brutal congestion, e.g. `"100mbps"`, `"1gbps"` |
| `brutalDown` | string | Download bandwidth for brutal congestion |
| `udpHop.ports` | string/arraay | Port list for UDP hopping |
| `udpHop.interval` | object | Hop interval range in seconds `{ "from": N, "to": N }` |
| `initStreamReceiveWindow` | uint64 | Initial stream receive window size (bytes) |
| `maxStreamReceiveWindow` | uint64 | Max stream receive window size (bytes) |
| `initConnectionReceiveWindow` | uint64 | Initial connection receive window size (bytes) |
| `maxConnectionReceiveWindow` | uint64 | Max connection receive window size (bytes) |
| `maxIdleTimeout` | int64 | Max idle timeout in seconds |
| `keepAlivePeriod` | int64 | Keep-alive period in seconds |
| `disablePathMTUDiscovery` | bool | Disable path MTU discovery |
| `maxIncomingStreams` | int64 | Max number of incoming streams |


#### Final Rule Settings (`finalRule`)
[FinalRule](https://xtls.github.io/config/outbounds/freedom.html#finalruleobject)
```
"finalRules" :[
	{
	  "action": "block",
	  "network": "tcp,udp",
	  "port": "53,443",
	  "ip": ["10.0.0.0/8", "2001:db8::/32"],
	  "blockDelay": "30-90"
	}
  ]
```

`finalRules` fields:

| Field | Type | Description |
|---|---|---|
| `action` | string | Action to take when the rule matches. `"allow"` permits the connection, `"block"` drops it |
| `network` | string | Comma-separated network types to match, e.g. `"tcp"`, `"udp"`, `"tcp,udp"` |
| `port` | string | Port or port range to match, e.g. `"53"`, `"443"`, `"8080-9000"`, `"53,443,8080-9000"` |
| `ip` | array | List of IP CIDRs or geo tags to match, e.g. `"10.0.0.0/8"`, `"geoip:cn"` |
| `blockDelay` | string | When `action` is `"block"`, adds a random delay (ms) before dropping, e.g. `"30-90"`. Omit or leave empty for immediate drop |



#### Socket Settings (`socketSettings`)
[Sockopt](https://xtls.github.io/config/transports/sockopt.html)

Socket-level options applied to the underlying TCP/UDP socket. Configured in the panel under `transportSettings.socketSettings`. All fields are optional — omitting a field leaves Xray's default in place.

```json
{
  "socketSettings": {
    "acceptProxyProtocol": false,
    "domainStrategy": "AsIs",
    "tcpFastOpen": false,
    "tcpKeepAliveInterval": 0,
    "tcpKeepAliveIdle": 0,
    "tcpUserTimeout": 0,
    "tcpMaxSeg": 0,
    "tcpWindowClamp": 0,
    "tcpMptcp": false,
    "tcpCongestion": "bbr",
    "interface": "",
    "v6only": false,
    "dialerProxy": "",
    "trustedXForwardedFor": []
  }
}
```

##### Field reference

| Field | Type | Default | Scope | Description |
|---|---|---|---|---|
| `acceptProxyProtocol` | bool | `false` | Inbound | Accept PROXY protocol v1/v2 from an upstream load balancer or reverse proxy (e.g. Nginx, HAProxy). The real client IP is read from the PROXY header. Only valid on TCP-based transports (`tcp`, `ws`, `httpupgrade`). |
| `domainStrategy` | string | `"AsIs"` | Both | DNS resolution strategy for outbound connections. See strategies table below. |
| `tcpFastOpen` | bool \| int | `false` | Both | Enable TCP Fast Open (TFO). `true` uses the OS default queue size. An integer explicitly sets the queue size (e.g. `256`). Reduces latency on reconnect by sending data in the SYN packet. Requires kernel ≥ 3.7 (Linux) or Windows 10 1607+. |
| `tcpKeepAliveInterval` | int | `0` | Both | Interval in seconds between TCP keep-alive probes after the idle period expires. `0` uses the OS default. Keeps long-lived connections alive through NAT/firewall state tables. |
| `tcpKeepAliveIdle` | int | `0` | Both | Seconds of inactivity before the first keep-alive probe is sent. `0` uses the OS default (typically 7200s on Linux). Lower values detect dead connections faster. |
| `tcpUserTimeout` | int | `0` | Both | Milliseconds the kernel waits for unacknowledged data before aborting the connection (`TCP_USER_TIMEOUT`). `0` uses the OS default. Useful for detecting broken connections faster than keep-alive alone. |
| `tcpMaxSeg` | int | `0` | Both | Maximum TCP segment size (MSS) in bytes (`TCP_MAXSEG`). `0` uses the OS default. Reduce below 1460 when using tunnels (e.g. VPN/WireGuard) to avoid fragmentation. |
| `tcpWindowClamp` | int | `0` | Both | Clamp the TCP receive window to this size in bytes (`TCP_WINDOW_CLAMP`). `0` disables clamping. Rarely needed; useful in constrained-bandwidth environments. |
| `tcpMptcp` | bool | `false` | Both | Enable Multipath TCP (MPTCP). Allows a connection to use multiple network paths simultaneously for throughput and resilience. Requires kernel ≥ 5.6 (Linux) with MPTCP compiled in. |
| `tcpCongestion` | string | `""` | Both | TCP congestion control algorithm. Common values: `"bbr"`, `"cubic"`, `"reno"`. Empty string leaves the system default. `bbr` is recommended for high-latency or lossy links. Requires the algorithm to be loaded in the kernel (`modprobe tcp_bbr`). |
| `interface` | string | `""` | Both | Bind the socket to a specific network interface by name (e.g. `"eth0"`, `"wg0"`). Useful for multi-homed servers or policy routing. Empty string binds to the default interface. |
| `v6only` | bool | `false` | Both | When `true`, an IPv6 listening socket (`::`) will not accept IPv4-mapped connections (`IPV6_V6ONLY`). Only relevant when `listeningIP` is `"::"` or `"0.0.0.0"` is absent. |
| `dialerProxy` | string | `""` | Outbound | Tag of another outbound to use as the underlying transport for this connection. Allows chaining outbounds (e.g. route relay traffic through a SOCKS5 outbound). Empty string disables proxying. |
| `trustedXForwardedFor` | string[] | `[]` | Inbound | List of trusted upstream IP CIDRs or addresses whose `X-Forwarded-For` header is trusted for real-client-IP extraction (e.g. `["127.0.0.1", "10.0.0.0/8"]`). Only applies to HTTP-based inbound transports (`ws`, `httpupgrade`, `xhttp`). Empty list disables trusted forwarding. |

##### `domainStrategy` values

| Value | Description |
|---|---|
| `"AsIs"` | Use the domain name as-is; let the OS resolve it. Default. |
| `"UseIP"` | Resolve the domain to IP before connecting, using Xray's internal DNS. |
| `"UseIPv4"` | Resolve and force an IPv4 address. |
| `"UseIPv6"` | Resolve and force an IPv6 address. |
| `"UseIPv4v6"` | Resolve and prefer IPv4, fall back to IPv6. |
| `"UseIPv6v4"` | Resolve and prefer IPv6, fall back to IPv4. |

##### Notes

**`acceptProxyProtocol` vs `trustedXForwardedFor`** — these serve different purposes. `acceptProxyProtocol` reads the real IP from a binary PROXY protocol header injected at the TCP layer (Nginx `proxy_protocol on`). `trustedXForwardedFor` reads it from an HTTP header injected at the application layer (Nginx `proxy_set_header X-Forwarded-For`). Use the one that matches your reverse proxy configuration.

**`tcpKeepAliveInterval` and `tcpKeepAliveIdle`** — both must be set together for keep-alive to behave predictably. Setting only `tcpKeepAliveInterval` without `tcpKeepAliveIdle` may have no effect on some kernels.

**`tcpFastOpen`** — must be enabled on both client and server sides to take effect. Also requires the server OS to have `net.ipv4.tcp_fastopen` set to `3` (`sysctl -w net.ipv4.tcp_fastopen=3`).

**`interface`** — the named interface must exist at the time the node starts. If the interface goes down and comes back up, existing connections are not migrated; new connections will bind correctly on reconnect.

**`dialerProxy`** — the referenced outbound tag must exist in the Xray config. Circular references (A → B → A) will cause a connection loop.

### Security Settings (**** socketSettings, maskSettings and finalRules are optional. You can chose not add to configuration ****)

#### TLS
```
{
  "tlsSettings": {
    "allowInsecure": false,
    "alpn": ["h2", "http/1.1"],
    "certMode": "http",
    "certDomainName": "tld.dev",
    "fragment": "1,40-60,30-50",
    "serverName": "google.com",
    "fingerprint": "chrome",
    "curvePreferences": ["X25519", "X25519MLKEM768"],
    "rejectUnknownSni": false,
    "verifyPeerCertByName": "google.com",
    "pinnedPeerCertSha256": "",
    "echServerKeys": "",
    "echConfigList": ""
  },
  "socketSettings": {
    "acceptProxyProtocol": false,
    "domainStrategy": "AsIs",
    "tcpFastOpen": false,
    "tcpKeepAliveInterval": 0,
    "tcpKeepAliveIdle": 0,
    "tcpUserTimeout": 0,
    "tcpMaxSeg": 0,
    "tcpWindowClamp": 0,
    "tcpMptcp": false,
    "tcpCongestion": "bbr",
    "interface": "",
    "v6only": false,
    "dialerProxy": "",
    "trustedXForwardedFor": []
  },
  "maskSettings": {
    "udp": [
      {
        "type": "salamander",
        "settings": {
          "password": "your-password-here"
        }
      }
    ]
  },
  "finalRules": [
	{
	  "action": "block",
	  "network": "tcp,udp",
	  "port": "53,443",
	  "ip": ["10.0.0.0/8", "2001:db8::/32"],
	  "blockDelay": "30-90"
	}
  ]
}
```
#### REALITY

```
{
  "realitySettings": {
    "target": "www.microsoft.com:443",
    "show": false,
    "shortids": ["6ba85179e30d4fc2"],
    "password": "u2Yirzjxx5R5miuJ-Od8CL4gAiCWj-65WOF2mSVyUz4",
    "privateKey": "sBFSY3OzslfjR2VcSHaQG-6GASrH5YswYyqBR-1m3Vc",
    "fingerprint": "chrome",
    "serverNames": ["www.microsoft.com"],
    "proxyprotocol": 0,
    "mldsa65Seed": "",
    "mldsa65Verify": "",
    "spiderX": "",
    "minClientVer": "",
    "maxClientVer": "",
    "maxTimeDiff": 0
  },
  "socketSettings": {
    "acceptProxyProtocol": false,
    "domainStrategy": "AsIs",
    "tcpFastOpen": false,
    "tcpKeepAliveInterval": 0,
    "tcpKeepAliveIdle": 0,
    "tcpUserTimeout": 0,
    "tcpMaxSeg": 0,
    "tcpWindowClamp": 0,
    "tcpMptcp": false,
    "tcpCongestion": "bbr",
    "interface": "",
    "v6only": false,
    "dialerProxy": "",
    "trustedXForwardedFor": []
  },
  "maskSettings": {
    "udp": [
      {
        "type": "salamander",
        "settings": {
          "password": "your-password-here"
        }
      }
    ]
  },
  "finalRules": [
	{
	  "action": "block",
	  "network": "tcp,udp",
	  "port": "53,443",
	  "ip": ["10.0.0.0/8", "2001:db8::/32"],
	  "blockDelay": "30-90"
	}
  ]
}
```

# XMRay Commands Reference

## Basic Operations

| Command | Description |
|---------|-------------|
| `XMRay` | Show menu (more features) |
| `XMRay start` | Start XMRay |
| `XMRay stop` | Stop XMRay |
| `XMRay restart` | Restart XMRay |
| `XMRay status` | View XMRay status |

## Service Management

| Command | Description |
|---------|-------------|
| `XMRay enable` | Enable XMRay auto-start |
| `XMRay disable` | Disable XMRay auto-start |

## Logging & Configuration

| Command | Description |
|---------|-------------|
| `XMRay log` | View XMRay logs |
| `XMRay config` | Show configuration file content |

## Installation & Updates

| Command | Description |
|---------|-------------|
| `XMRay install` | Install XMRay |
| `XMRay uninstall` | Uninstall XMRay |
| `XMRay update` | Update XMRay |
| `XMRay update vx.x.x` | Update XMRay to specific version |
| `XMRay version` | View XMRay version |

## Key Generation & Utilities

| Command | Description |
|---------|-------------|
| `XMRay warp` | Generate Cloudflare WARP account |
| `XMRay x25519` | Generate key pair for X25519 key exchange (REALITY, VLESS Encryption) |
| `XMRay mldsa65` | Generate key pair for ML-DSA-65 post-quantum signature (REALITY) |
| `XMRay mlkem768` | Generate key pair for ML-KEM-768 post-quantum key exchange (VLESS Encryption) |
| `XMRay vlessenc` | Generate decryption/encryption JSON pair (VLESS Encryption) |
| `XMRay obtain` | Generate SSL/TLS certificate for domain name |
| `XMRay renew` | Renew SSL/TLS certificate for domain name |
| `XMRay ping` | Ping a domain with TLS handshake |
| `XMRay ech` | Generate ECH keys with default or custom server name |
| `XMRay hash` | Calculate hash for specific certificate |
| `XMRay generate` | Generate self-signed TLS certificates for testing and production use |