package node

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	
	"github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/infra/conf"
	"github.com/xmplusdev/xmplus-server/api"
)

func RelayRouterBuilder(tag string, relayTag string, subscription *api.SubscriptionInfo) (*router.Config, error) {
	routerConfig := &conf.RouterConfig{}
	var ruleList any
	
	User := conf.StringList{fmt.Sprintf("%s|%s|%d", tag, subscription.Email, subscription.Id)}
	InboundTag := conf.StringList{tag}
	
	ruleList = struct {
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
		return nil, fmt.Errorf("Marshal Rule list %s config failed: %s", ruleList, err)
	}
		
	RuleList := []json.RawMessage{}
	RuleList = append(RuleList, rule)
	routerConfig.RuleList = RuleList
	return routerConfig.Build()
}

func DefaultRouterBuilder(tag string) (*router.Config, error) {
	routerConfig := &conf.RouterConfig{}
	RuleList := []json.RawMessage{}
	
	InboundTag := conf.StringList{tag}
	
	// Add default rule to route all other traffic to the main outbound
	// IMPORTANT: Only match traffic from this inbound to avoid interfering with relay user-specific rules
	//network := conf.NetworkList([]conf.Network{"tcp", "udp"})
	defaultRule := struct {
		Type        string            `json:"type"`
		RuleTag     string            `json:"ruleTag"`
		InboundTag  *conf.StringList `json:"inboundTag"`
		OutboundTag string            `json:"outboundTag"`
		//Network     *conf.NetworkList `json:"network,omitempty"`
	}{
		Type:        "field",
		RuleTag:     fmt.Sprintf("%s_default", tag),
		InboundTag:  &InboundTag,
		OutboundTag: tag,
		//Network:     &network,
	}
		
	rule, err := json.Marshal(defaultRule)
	if err != nil {
		return nil, fmt.Errorf("Marshal default rule config failed: %s", err)
	}
		
	RuleList = append(RuleList, rule)
	
	routerConfig.RuleList = RuleList
	return routerConfig.Build()
}

func RouterBuilder(nodeInfo *api.NodeInfo, tag string) (*router.Config, error) {
	routerConfig := &conf.RouterConfig{}
	RuleList := []json.RawMessage{}
	
	InboundTag := conf.StringList{tag}
	
	// Only add blocking rule if there are actual blocking rules defined
	hasBlockingRules := false
	if (nodeInfo.BlockingRules.Port != "" && nodeInfo.BlockingRules.Port != "0") ||
	   (nodeInfo.BlockingRules.Domain != nil && len(nodeInfo.BlockingRules.Domain) > 0) ||
	   (nodeInfo.BlockingRules.IP != nil && len(nodeInfo.BlockingRules.IP) > 0) ||
	   (nodeInfo.BlockingRules.Protocol != nil && len(nodeInfo.BlockingRules.Protocol) > 0) {
		hasBlockingRules = true
	}
	
	if hasBlockingRules {
		// Parse port string into PortRange slice
		var portRanges []conf.PortRange
		var err error
		if nodeInfo.BlockingRules.Port != "" && nodeInfo.BlockingRules.Port != "0" {
			portRanges, err = parsePortString(nodeInfo.BlockingRules.Port)
			if err != nil {
				return nil, fmt.Errorf("failed to parse port string: %w", err)
			}
		}
		
		var domain *conf.StringList
		if nodeInfo.BlockingRules.Domain != nil && len(nodeInfo.BlockingRules.Domain) > 0 {
			d := conf.StringList(nodeInfo.BlockingRules.Domain)
			domain = &d
		}

		var ip *conf.StringList
		if nodeInfo.BlockingRules.IP != nil && len(nodeInfo.BlockingRules.IP) > 0 {
			i := conf.StringList(nodeInfo.BlockingRules.IP)
			ip = &i
		}

		var protocols *conf.StringList
		if nodeInfo.BlockingRules.Protocol != nil && len(nodeInfo.BlockingRules.Protocol) > 0 {
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
			return nil, fmt.Errorf("Marshal blocking rule config failed: %s", err)
		}
		
		RuleList = append(RuleList, rule)
	}
	
	routerConfig.RuleList = RuleList
	return routerConfig.Build()
}

// parsePortString parses a port string like "53,443,1000-2000" into PortRange slices
func parsePortString(portStr string) ([]conf.PortRange, error) {
	if portStr == "" {
		return nil, nil
	}
	
	var portRanges []conf.PortRange
	
	// Split by comma
	ports := strings.Split(portStr, ",")
	
	for _, p := range ports {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		
		// Check if it's a range (contains "-")
		if strings.Contains(p, "-") {
			rangeParts := strings.SplitN(p, "-", 2)
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid port range format: %s", p)
			}
			
			fromPort, err := strconv.ParseUint(strings.TrimSpace(rangeParts[0]), 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid port number in range: %s", rangeParts[0])
			}
			
			toPort, err := strconv.ParseUint(strings.TrimSpace(rangeParts[1]), 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid port number in range: %s", rangeParts[1])
			}
			
			portRanges = append(portRanges, conf.PortRange{
				From: uint32(fromPort),
				To:   uint32(toPort),
			})
		} else {
			// Single port
			port, err := strconv.ParseUint(p, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid port number: %s", p)
			}
			
			portRanges = append(portRanges, conf.PortRange{
				From: uint32(port),
				To:   uint32(port),
			})
		}
	}
	
	return portRanges, nil
}