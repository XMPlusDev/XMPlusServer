package node

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"

	"github.com/xmplusdev/xmray/api"
)

func OutboundBuilder(config *Config, nodeInfo *api.NodeInfo, tag string) (*core.OutboundHandlerConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if nodeInfo == nil {
		return nil, fmt.Errorf("nodeInfo is nil")
	}

	outboundDetourConfig := &conf.OutboundDetourConfig{}
	outboundDetourConfig.Protocol = "freedom"
	outboundDetourConfig.Tag = tag

	if nodeInfo.SendThroughIP != "" {
		outboundDetourConfig.SendThrough = &nodeInfo.SendThroughIP
	}

	domainStrategy := "Asis"
	if config.EnableDNS {
		if config.DNSStrategy != "" {
			domainStrategy = config.DNSStrategy
		}
	}

	proxySetting := &conf.FreedomConfig{
		DomainStrategy: domainStrategy,
	}

	for _, fr := range nodeInfo.FinalRules {
		rule := buildFinalRule(fr)
		if rule != nil {
			proxySetting.FinalRules = append(proxySetting.FinalRules, rule)
		}
	}

	settingBytes, err := json.Marshal(proxySetting)
	if err != nil {
		return nil, fmt.Errorf("marshal proxy %s config failed: %s", nodeInfo.NodeType, err)
	}
	setting := json.RawMessage(settingBytes)
	outboundDetourConfig.Settings = &setting

	if nodeInfo.SocketSettings != nil && nodeInfo.SocketSettings.Enabled {
		outboundDetourConfig.StreamSetting = buildSocketOnlyStream(nodeInfo.SocketSettings)
	}

	return outboundDetourConfig.Build()
}

func buildSocketOnlyStream(s *api.SocketSettings) *conf.StreamConfig {
	return &conf.StreamConfig{
		SocketSettings: buildSocketConfig(s, false),
	}
}

func BlackholeOutboundBuilder(tag string) (*core.OutboundHandlerConfig, error) {
	outboundDetourConfig := &conf.OutboundDetourConfig{}
	outboundDetourConfig.Protocol = "blackhole"
	outboundDetourConfig.Tag = fmt.Sprintf("%s_blackhole", tag)

	blackholeSetting := &conf.BlackholeConfig{
		Response: json.RawMessage(`{"type":"http"}`),
	}

	settingBytes, err := json.Marshal(blackholeSetting)
	if err != nil {
		return nil, fmt.Errorf("marshal blackhole config failed: %s", err)
	}
	setting := json.RawMessage(settingBytes)
	outboundDetourConfig.Settings = &setting

	return outboundDetourConfig.Build()
}

func OutboundRelayBuilder(nodeInfo *api.RelayNodeInfo, tag string, subscription *api.SubscriptionInfo, Passwd string) (*core.OutboundHandlerConfig, error) {
	if nodeInfo == nil {
		return nil, fmt.Errorf("nodeInfo is nil")
	}
	if subscription == nil {
		return nil, fmt.Errorf("subscription is nil")
	}

	outboundDetourConfig := &conf.OutboundDetourConfig{}

	var (
		protocol      string
		streamSetting *conf.StreamConfig
	)

	var proxySetting any

	switch nodeInfo.NodeType {
	case "vless":
		protocol = "vless"
		vUser, err := vlessUser(tag, nodeInfo, subscription)
		if err != nil {
			return nil, fmt.Errorf("Marshal Vless User config failed: %s", err)
		}
		proxySetting = struct {
			Vnext []*conf.VLessOutboundVnext `json:"vnext"`
		}{
			Vnext: []*conf.VLessOutboundVnext{{
				Address: &conf.Address{Address: net.ParseAddress(nodeInfo.Address)},
				Port:    uint16(nodeInfo.ListeningPort),
				Users:   []json.RawMessage{vUser},
			}},
		}
	case "vmess":
		protocol = "vmess"
		userVmess, err := vmessUser(tag, subscription)
		if err != nil {
			return nil, fmt.Errorf("Marshal Vmess User config failed: %s", err)
		}
		proxySetting = struct {
			Receivers []*conf.VMessOutboundTarget `json:"vnext"`
		}{
			Receivers: []*conf.VMessOutboundTarget{{
				Address: &conf.Address{Address: net.ParseAddress(nodeInfo.Address)},
				Port:    uint16(nodeInfo.ListeningPort),
				Users:   []json.RawMessage{userVmess},
			}},
		}
	case "trojan":
		protocol = "trojan"
		proxySetting = struct {
			Servers []*conf.TrojanServerTarget `json:"servers"`
		}{
			Servers: []*conf.TrojanServerTarget{{
				Address:  &conf.Address{Address: net.ParseAddress(nodeInfo.Address)},
				Port:     uint16(nodeInfo.ListeningPort),
				Password: subscription.Passwd,
				Email:    fmt.Sprintf("%s_%s", tag, subscription.Email),
				Level:    0,
				Flow:     "",
			}},
		}
	case "shadowsocks":
		protocol = "shadowsocks"
		proxySetting = struct {
			Servers []*conf.ShadowsocksServerTarget `json:"servers"`
		}{
			Servers: []*conf.ShadowsocksServerTarget{{
				Address:  &conf.Address{Address: net.ParseAddress(nodeInfo.Address)},
				Port:     uint16(nodeInfo.ListeningPort),
				Password: Passwd,
				Email:    fmt.Sprintf("%s_%s", tag, subscription.Email),
				Level:    0,
				Cipher:   nodeInfo.Cipher,
				UoT:      true,
			}},
		}
	case "hysteria":
		protocol = "hysteria"
		proxySetting = struct {
			*conf.HysteriaClientConfig
		}{
			&conf.HysteriaClientConfig{
				Address: &conf.Address{Address: net.ParseAddress(nodeInfo.Address)},
				Port:    uint16(nodeInfo.ListeningPort),
				Version: nodeInfo.HysteriaSettings.Version,
			},
		}
	default:
		return nil, fmt.Errorf("Unsupported Relay Node Type: %s", nodeInfo.NodeType)
	}

	settingBytes, err := json.Marshal(proxySetting)
	if err != nil {
		return nil, fmt.Errorf("marshal proxy %s config failed: %s", nodeInfo.NodeType, err)
	}
	setting := json.RawMessage(settingBytes)

	outboundDetourConfig.Protocol = protocol
	outboundDetourConfig.Settings = &setting

	streamSetting = new(conf.StreamConfig)
	transportProtocol := conf.TransportProtocol(nodeInfo.NetworkType)
	networkType, err := transportProtocol.Build()
	if err != nil {
		return nil, fmt.Errorf("convert TransportProtocol failed: %s", err)
	}

	streamSetting.Network = &transportProtocol

	switch networkType {
	case "tcp", "raw":
		tcpSetting := &conf.TCPConfig{}
		if nodeInfo.RawSettings != nil {
			tcpSetting.HeaderConfig = nodeInfo.RawSettings.Header
			tcpSetting.AcceptProxyProtocol = nodeInfo.RawSettings.AcceptProxyProtocol
		}
		streamSetting.TCPSettings = tcpSetting
	case "websocket", "ws":
		wsSettings := &conf.WebSocketConfig{}
		if nodeInfo.WsSettings != nil {
			wsSettings.Path = nodeInfo.WsSettings.Path
			wsSettings.Host = nodeInfo.WsSettings.Host
			wsSettings.HeartbeatPeriod = nodeInfo.WsSettings.HeartbeatPeriod
			wsSettings.AcceptProxyProtocol = nodeInfo.WsSettings.AcceptProxyProtocol
		}
		streamSetting.WSSettings = wsSettings
	case "httpupgrade":
		httpupgradeSettings := &conf.HttpUpgradeConfig{}
		if nodeInfo.HttpSettings != nil {
			httpupgradeSettings.AcceptProxyProtocol = nodeInfo.HttpSettings.AcceptProxyProtocol
			httpupgradeSettings.Host = nodeInfo.HttpSettings.Host
			httpupgradeSettings.Path = nodeInfo.HttpSettings.Path
		}
		streamSetting.HTTPUPGRADESettings = httpupgradeSettings
	case "grpc":
		grpcSettings := &conf.GRPCConfig{}
		if nodeInfo.GrpcSettings != nil {
			grpcSettings.ServiceName = nodeInfo.GrpcSettings.ServiceName
			grpcSettings.Authority = nodeInfo.GrpcSettings.Authority
			grpcSettings.InitialWindowsSize = nodeInfo.GrpcSettings.WindowsSize
			grpcSettings.UserAgent = nodeInfo.GrpcSettings.UserAgent
			grpcSettings.IdleTimeout = nodeInfo.GrpcSettings.IdleTimeout
			grpcSettings.HealthCheckTimeout = nodeInfo.GrpcSettings.HealthCheckTimeout
			grpcSettings.PermitWithoutStream = nodeInfo.GrpcSettings.PermitWithoutStream
		}
		streamSetting.GRPCSettings = grpcSettings
	case "mkcp", "kcp":
		kcpSettings := &conf.KCPConfig{}
		if nodeInfo.KcpSettings != nil {
			kcpSettings.Mtu = &nodeInfo.KcpSettings.Mtu
		}
		streamSetting.KCPSettings = kcpSettings
	case "hysteria", "hysteria2":
		hysteriaSettings := &conf.HysteriaConfig{}
		if nodeInfo.HysteriaSettings != nil {
			hysteriaSettings.Version = nodeInfo.HysteriaSettings.Version
			hysteriaSettings.Auth = subscription.Passwd
		}
		streamSetting.HysteriaSettings = hysteriaSettings
	case "xhttp", "splithttp":
		xhttpSettings := &conf.SplitHTTPConfig{}
		if nodeInfo.XhttpSettings != nil {
			xhttpSettings.Host = nodeInfo.XhttpSettings.Host
			xhttpSettings.Path = nodeInfo.XhttpSettings.Path
			xhttpSettings.Mode = nodeInfo.XhttpSettings.Mode

			if nodeInfo.XhttpSettings.XPaddingObfsMode {
				type xhttpExtra struct {
					XPaddingObfsMode  bool   `json:"xPaddingObfsMode"`
					XPaddingMethod    string `json:"xPaddingMethod,omitempty"`
					XPaddingPlacement string `json:"xPaddingPlacement,omitempty"`
					XPaddingKey       string `json:"xPaddingKey,omitempty"`
					XPaddingHeader    string `json:"xPaddingHeader,omitempty"`
					UplinkHTTPMethod  string `json:"uplinkHTTPMethod,omitempty"`
					SessionPlacement  string `json:"sessionPlacement,omitempty"`
					SessionKey        string `json:"sessionKey,omitempty"`
					SeqPlacement      string `json:"seqPlacement,omitempty"`
					SeqKey            string `json:"seqKey,omitempty"`
				}
				extra := xhttpExtra{
					XPaddingObfsMode:  true,
					XPaddingMethod:    nodeInfo.XhttpSettings.XPaddingMethod,
					XPaddingPlacement: nodeInfo.XhttpSettings.XPaddingPlacement,
					XPaddingKey:       nodeInfo.XhttpSettings.XPaddingKey,
					XPaddingHeader:    nodeInfo.XhttpSettings.XPaddingHeader,
					UplinkHTTPMethod:  nodeInfo.XhttpSettings.UplinkHTTPMethod,
					SessionPlacement:  nodeInfo.XhttpSettings.SessionPlacement,
					SessionKey:        nodeInfo.XhttpSettings.SessionKey,
					SeqPlacement:      nodeInfo.XhttpSettings.SeqPlacement,
					SeqKey:            nodeInfo.XhttpSettings.SeqKey,
				}
				extraBytes, err := json.Marshal(extra)
				if err != nil {
					return nil, fmt.Errorf("marshal xhttp extra config failed: %w", err)
				}
				xhttpSettings.Extra = json.RawMessage(extraBytes)
			}
		}
		streamSetting.XHTTPSettings = xhttpSettings
	}

	if nodeInfo.MaskSettings != nil && nodeInfo.MaskSettings.Enabled {
		finalMaskSettings := &conf.FinalMask{}

		for _, entry := range nodeInfo.MaskSettings.UDP {
			udpMask := conf.Mask{Type: entry.Type}
			if entry.Settings != nil {
				udpMask.Settings = entry.Settings
			}
			finalMaskSettings.Udp = append(finalMaskSettings.Udp, udpMask)
		}

		for _, entry := range nodeInfo.MaskSettings.TCP {
			tcpMask := conf.Mask{Type: entry.Type}
			if entry.Settings != nil {
				tcpMask.Settings = entry.Settings
			}
			finalMaskSettings.Tcp = append(finalMaskSettings.Tcp, tcpMask)
		}

		if nodeInfo.MaskSettings.QuicParams != nil && nodeInfo.MaskSettings.EnabledQuic {
			finalMaskSettings.QuicParams = buildQuicParams(nodeInfo.MaskSettings.QuicParams)
		}

		streamSetting.FinalMask = finalMaskSettings
	}

	if nodeInfo.SocketSettings != nil && nodeInfo.SocketSettings.Enabled {
		streamSetting.SocketSettings = buildSocketConfig(nodeInfo.SocketSettings, false)
	}

	if nodeInfo.SecurityType == "tls" && nodeInfo.TlsSettings != nil {
		streamSetting.Security = "tls"
		streamSetting.TLSSettings = &conf.TLSConfig{
			ServerName:           nodeInfo.TlsSettings.ServerName,
			Fingerprint:          nodeInfo.TlsSettings.FingerPrint,
			VerifyPeerCertByName: nodeInfo.TlsSettings.VerifyPeerCertByName,
			ECHConfigList:        nodeInfo.TlsSettings.ECHConfigList,
			PinnedPeerCertSha256: nodeInfo.TlsSettings.PinnedPeerCertSha256,
		}
	}

	if nodeInfo.SecurityType == "reality" && nodeInfo.RealitySettings != nil {
		streamSetting.Security = "reality"
		streamSetting.REALITYSettings = &conf.REALITYConfig{
			Show:          nodeInfo.RealitySettings.Show,
			ServerName:    nodeInfo.RealitySettings.ServerName,
			PublicKey:     nodeInfo.RealitySettings.PublicKey,
			Fingerprint:   nodeInfo.RealitySettings.Fingerprint,
			ShortId:       nodeInfo.RealitySettings.ShortId,
			SpiderX:       nodeInfo.RealitySettings.SpiderX,
			Mldsa65Verify: nodeInfo.RealitySettings.Mldsa65Verify,
		}
	}

	outboundDetourConfig.Tag = fmt.Sprintf("%s_%d", tag, subscription.Id)
	if nodeInfo.SendThroughIP != "" {
		outboundDetourConfig.SendThrough = &nodeInfo.SendThroughIP
	}
	outboundDetourConfig.StreamSetting = streamSetting

	return outboundDetourConfig.Build()
}

func buildFinalRule(fr api.FinalRuleSettings) *conf.FreedomFinalRuleConfig {
	if fr.Action == "" {
		return nil
	}
	rule := &conf.FreedomFinalRuleConfig{
		Action: fr.Action,
	}

	if fr.Network != "" {
		nl := conf.NetworkList{}
		for _, n := range strings.Split(fr.Network, ",") {
			n = strings.TrimSpace(n)
			if n != "" {
				nl = append(nl, conf.Network(n))
			}
		}
		rule.Network = &nl
	}

	if fr.Port != "" {
		pl := &conf.PortList{}
		if err := json.Unmarshal([]byte(`"`+fr.Port+`"`), pl); err == nil {
			rule.Port = pl
		}
	}

	if len(fr.IP) > 0 {
		sl := conf.StringList(fr.IP)
		rule.IP = &sl
	}

	if fr.BlockDelay != "" {
		from, to, err := conf.ParseRangeString(fr.BlockDelay)
		if err == nil {
			r := conf.Int32Range{}
			r.Left = int32(from)
			r.Right = int32(to)
			if r.Left > r.Right {
				r.From, r.To = r.Right, r.Left
			} else {
				r.From, r.To = r.Left, r.Right
			}
			rule.BlockDelay = &r
		}
	}

	return rule
}

func vmessUser(tag string, subscription *api.SubscriptionInfo) (json.RawMessage, error) {
	if subscription == nil {
		return nil, fmt.Errorf("subscription is nil")
	}
	account := struct {
		Level    int    `json:"level"`
		Email    string `json:"email"`
		ID       string `json:"id"`
		Security string `json:"security"`
	}{
		Level:    0,
		Email:    fmt.Sprintf("%s_%s", tag, subscription.Email),
		ID:       subscription.Passwd,
		Security: "auto",
	}
	return json.Marshal(&account)
}

func vlessUser(tag string, nodeInfo *api.RelayNodeInfo, subscription *api.SubscriptionInfo) (json.RawMessage, error) {
	if nodeInfo == nil {
		return nil, fmt.Errorf("nodeInfo is nil")
	}
	if subscription == nil {
		return nil, fmt.Errorf("subscription is nil")
	}
	account := struct {
		Level      int    `json:"level"`
		Email      string `json:"email"`
		Id         string `json:"id"`
		Flow       string `json:"flow"`
		Encryption string `json:"encryption"`
	}{
		Level:      0,
		Email:      fmt.Sprintf("%s_%s", tag, subscription.Email),
		Id:         subscription.Passwd,
		Flow:       nodeInfo.Flow,
		Encryption: nodeInfo.Encryption,
	}
	return json.Marshal(&account)
}