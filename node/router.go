package node

import (
	"encoding/json"
	"fmt"

	"github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/infra/conf"
	"github.com/xmplusdev/xmray/api"
)

func RelayRouterBuilder(tag string, relayTag string, subscription *api.SubscriptionInfo) (*router.Config, error) {
	if subscription == nil {
		return nil, fmt.Errorf("subscription is nil")
	}

	routerConfig := &conf.RouterConfig{}
	User := conf.StringList{fmt.Sprintf("%s|%s|%d", tag, subscription.Email, subscription.Id)}
	InboundTag := conf.StringList{tag}

	ruleList := struct {
		RuleTag     string           `json:"ruleTag"`
		Type        string           `json:"type"`
		InboundTag  *conf.StringList `json:"inboundTag"`
		OutboundTag string           `json:"outboundTag"`
		User        *conf.StringList `json:"user"`
	}{
		RuleTag:     fmt.Sprintf("%s_%d", relayTag, subscription.Id),
		Type:        "field",
		InboundTag:  &InboundTag,
		OutboundTag: fmt.Sprintf("%s_%d", relayTag, subscription.Id),
		User:        &User,
	}

	rule, err := json.Marshal(ruleList)
	if err != nil {
		return nil, fmt.Errorf("marshal relay rule config failed: %s", err)
	}

	routerConfig.RuleList = []json.RawMessage{rule}
	return routerConfig.Build()
}

func DefaultRouterBuilder(tag string) (*router.Config, error) {
	routerConfig := &conf.RouterConfig{}
	InboundTag := conf.StringList{tag}

	defaultRule := struct {
		Type        string           `json:"type"`
		RuleTag     string           `json:"ruleTag"`
		InboundTag  *conf.StringList `json:"inboundTag"`
		OutboundTag string           `json:"outboundTag"`
	}{
		Type:        "field",
		RuleTag:     fmt.Sprintf("%s_default", tag),
		InboundTag:  &InboundTag,
		OutboundTag: tag,
	}

	rule, err := json.Marshal(defaultRule)
	if err != nil {
		return nil, fmt.Errorf("marshal default rule config failed: %s", err)
	}

	routerConfig.RuleList = []json.RawMessage{rule}
	return routerConfig.Build()
}

func BlackHoleRouterBuilder(nodeInfo *api.NodeInfo, tag string) (*router.Config, error) {
	if nodeInfo == nil {
		return nil, fmt.Errorf("nodeInfo is nil")
	}

	routerConfig := &conf.RouterConfig{}
	var ruleListJSON []json.RawMessage

	hasBlockingRules := nodeInfo.BlockingRules != nil &&
		((nodeInfo.BlockingRules.Port != "" && nodeInfo.BlockingRules.Port != "0") ||
			len(nodeInfo.BlockingRules.Domain) > 0 ||
			len(nodeInfo.BlockingRules.IP) > 0 ||
			len(nodeInfo.BlockingRules.Protocol) > 0)

	if hasBlockingRules {
		InboundTag := conf.StringList{tag}

		var portRanges []conf.PortRange
		var err error
		if nodeInfo.BlockingRules.Port != "" && nodeInfo.BlockingRules.Port != "0" {
			portRanges, err = parsePortString(nodeInfo.BlockingRules.Port)
			if err != nil {
				return nil, fmt.Errorf("failed to parse port string: %w", err)
			}
		}

		var domain *conf.StringList
		if len(nodeInfo.BlockingRules.Domain) > 0 {
			d := conf.StringList(nodeInfo.BlockingRules.Domain)
			domain = &d
		}
		var ip *conf.StringList
		if len(nodeInfo.BlockingRules.IP) > 0 {
			i := conf.StringList(nodeInfo.BlockingRules.IP)
			ip = &i
		}
		var protocols *conf.StringList
		if len(nodeInfo.BlockingRules.Protocol) > 0 {
			p := conf.StringList(nodeInfo.BlockingRules.Protocol)
			protocols = &p
		}
		var portList *conf.PortList
		if len(portRanges) > 0 {
			portList = &conf.PortList{Range: portRanges}
		}

		blockingRule := struct {
			Type        string           `json:"type"`
			RuleTag     string           `json:"ruleTag"`
			InboundTag  *conf.StringList `json:"inboundTag"`
			OutboundTag string           `json:"outboundTag"`
			Domain      *conf.StringList `json:"domain,omitempty"`
			IP          *conf.StringList `json:"ip,omitempty"`
			Port        *conf.PortList   `json:"port,omitempty"`
			Protocols   *conf.StringList `json:"protocol,omitempty"`
		}{
			Type:        "field",
			RuleTag:     fmt.Sprintf("%s_blackhole", tag),
			InboundTag:  &InboundTag,
			OutboundTag: fmt.Sprintf("%s_blackhole", tag),
			Domain:      domain,
			IP:          ip,
			Protocols:   protocols,
			Port:        portList,
		}

		rule, err := json.Marshal(blockingRule)
		if err != nil {
			return nil, fmt.Errorf("marshal blocking rule config failed: %s", err)
		}
		ruleListJSON = append(ruleListJSON, rule)
	}

	routerConfig.RuleList = ruleListJSON
	return routerConfig.Build()
}
