package node

// stream.go contains shared transport stream helpers used by both inbound.go and outbound.go.

import (
	"fmt"
	"strings"
	"strconv"

	"github.com/xtls/xray-core/infra/conf"
	"github.com/xmplusdev/xmray/api"
)

// applyCommonTransport fills the common transport fields (tcp/ws/httpupgrade/grpc/kcp) of a
// StreamConfig. xhttp and hysteria have inbound-vs-outbound differences so each builder
// handles them separately.
func applyCommonTransport(
	streamSetting *conf.StreamConfig,
	networkType string,
	nodeRaw *api.RawSettings,
	nodeWs *api.WsSettings,
	nodeHttp *api.HttpSettings,
	nodeGrpc *api.GrpcSettings,
	nodeKcp *api.KcpSettings,
) error {
	switch networkType {
	case "tcp", "raw":
		tcpSetting := &conf.TCPConfig{}
		if nodeRaw != nil {
			tcpSetting.AcceptProxyProtocol = nodeRaw.AcceptProxyProtocol
			if nodeRaw.Header != nil {
				tcpSetting.HeaderConfig = nodeRaw.Header
			}
		}
		streamSetting.TCPSettings = tcpSetting

	case "websocket", "ws":
		wsSettings := &conf.WebSocketConfig{}
		if nodeWs != nil {
			wsSettings.Path = nodeWs.Path
			wsSettings.Host = nodeWs.Host
			wsSettings.HeartbeatPeriod = nodeWs.HeartbeatPeriod
			wsSettings.AcceptProxyProtocol = nodeWs.AcceptProxyProtocol
		}
		streamSetting.WSSettings = wsSettings

	case "httpupgrade":
		httpSettings := &conf.HttpUpgradeConfig{}
		if nodeHttp != nil {
			httpSettings.AcceptProxyProtocol = nodeHttp.AcceptProxyProtocol
			httpSettings.Host = nodeHttp.Host
			httpSettings.Path = nodeHttp.Path
		}
		streamSetting.HTTPUPGRADESettings = httpSettings

	case "grpc":
		grpcSettings := &conf.GRPCConfig{}
		if nodeGrpc != nil {
			grpcSettings.ServiceName = nodeGrpc.ServiceName
			grpcSettings.Authority = nodeGrpc.Authority
			grpcSettings.InitialWindowsSize = nodeGrpc.WindowsSize
			grpcSettings.UserAgent = nodeGrpc.UserAgent
			grpcSettings.IdleTimeout = nodeGrpc.IdleTimeout
			grpcSettings.HealthCheckTimeout = nodeGrpc.HealthCheckTimeout
			grpcSettings.PermitWithoutStream = nodeGrpc.PermitWithoutStream
		}
		streamSetting.GRPCSettings = grpcSettings

	case "mkcp", "kcp":
		kcpSettings := &conf.KCPConfig{}
		if nodeKcp != nil {
			kcpSettings.Mtu = &nodeKcp.Mtu
			kcpSettings.Tti = &nodeKcp.Tti
		}
		streamSetting.KCPSettings = kcpSettings

	default:
		return fmt.Errorf("unsupported transport protocol: %v", networkType)
	}
	return nil
}

// applyMaskSettings applies MaskSettings to a StreamConfig. Shared by inbound and outbound.
func applyMaskSettings(streamSetting *conf.StreamConfig, ms *api.MaskSettings) {
	if ms == nil || !ms.Enabled {
		return
	}
	finalMask := &conf.FinalMask{}
	for _, entry := range ms.UDP {
		udpMask := conf.Mask{Type: entry.Type}
		if entry.Settings != nil {
			udpMask.Settings = entry.Settings
		}
		finalMask.Udp = append(finalMask.Udp, udpMask)
	}
	for _, entry := range ms.TCP {
		tcpMask := conf.Mask{Type: entry.Type}
		if entry.Settings != nil {
			tcpMask.Settings = entry.Settings
		}
		finalMask.Tcp = append(finalMask.Tcp, tcpMask)
	}
	if ms.QuicParams != nil && ms.EnabledQuic {
		finalMask.QuicParams = buildQuicParams(ms.QuicParams)
	}
	streamSetting.FinalMask = finalMask
}

func parseInt32Range(s string, defaultA, defaultB int32) (conf.Int32Range, error) {
	if s == "" {
		return conf.Int32Range{From: defaultA, To: defaultB}, nil
	}
	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		if len(parts) != 2 {
			return conf.Int32Range{}, fmt.Errorf("invalid range format: %s", s)
		}
		a, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 32)
		if err != nil {
			return conf.Int32Range{}, fmt.Errorf("invalid range start %q: %w", parts[0], err)
		}
		b, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 32)
		if err != nil {
			return conf.Int32Range{}, fmt.Errorf("invalid range end %q: %w", parts[1], err)
		}
		return conf.Int32Range{From: int32(a), To: int32(b)}, nil
	}
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return conf.Int32Range{}, fmt.Errorf("invalid value %q: %w", s, err)
	}
	return conf.Int32Range{From: int32(v), To: int32(v)}, nil
}
