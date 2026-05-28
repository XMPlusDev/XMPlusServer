package dispatcher

import (
	"context"
	"fmt"
	logger "log"
	netModule "net"
	"reflect"
	"strings"
	"sync"
	"unsafe"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features"
	"github.com/xtls/xray-core/features/inbound"
	"github.com/xtls/xray-core/features/routing"
	"github.com/xtls/xray-core/proxy"
	"github.com/xtls/xray-core/transport"

	"github.com/xmplusdev/xmray/api"
	"github.com/xmplusdev/xmray/helper/counter"
	"github.com/xmplusdev/xmray/helper/limiter"

	"golang.org/x/time/rate"
)

func maskIP(ipStr string, keepSegments int) string {
	ip := netModule.ParseIP(ipStr)
	if ip == nil {
		return ipStr
	}
	if ip.To4() != nil {
		parts := strings.Split(ipStr, ".")
		if len(parts) != 4 {
			return ipStr
		}
		for i := keepSegments; i < 4; i++ {
			parts[i] = "*"
		}
		return strings.Join(parts, ".")
	}
	fullIP := ip.String()
	parts := strings.Split(fullIP, ":")
	for i := keepSegments; i < len(parts); i++ {
		parts[i] = "*"
	}
	return strings.Join(parts, ":")
}

type sessionInfo struct {
	tag   string
	email string
	ip    string
}

func RegisterOn(server *core.Instance, lim *limiter.Limiter) (*LimitingDispatcher, error) {
	raw := server.GetFeature(routing.DispatcherType())
	if raw == nil {
		return nil, errors.New("no dispatcher feature found in instance")
	}
	inner, ok := raw.(routing.Dispatcher)
	if !ok {
		return nil, errors.New("dispatcher feature does not implement routing.Dispatcher")
	}

	ibm, _ := server.GetFeature(inbound.ManagerType()).(inbound.Manager)

	ld := &LimitingDispatcher{
		inner:   inner,
		limiter: lim,
		ibm:     ibm,
	}

	if err := replaceFeature(server, routing.DispatcherType(), ld); err != nil {
		return nil, err
	}

	return ld, nil
}

func replaceFeature(server *core.Instance, targetType interface{}, limitingDispatcher *LimitingDispatcher) error {
	rv := reflect.ValueOf(server).Elem()
	featField := rv.FieldByName("features")
	if !featField.IsValid() {
		return fmt.Errorf("core.Instance has no 'features' field")
	}
	writable := reflect.NewAt(featField.Type(), unsafe.Pointer(featField.UnsafeAddr())).Elem()
	slice := writable.Interface().([]features.Feature)
	for i, f := range slice {
		if f.Type() == targetType {
			slice[i] = limitingDispatcher
			writable.Set(reflect.ValueOf(slice))
			return nil
		}
	}
	return fmt.Errorf("no feature with type %v found", targetType)
}

type LimitingDispatcher struct {
	inner   routing.Dispatcher
	limiter *limiter.Limiter
	ibm     inbound.Manager
	counters sync.Map
}

func (ld *LimitingDispatcher) Type() interface{} { return routing.DispatcherType() }

func (ld *LimitingDispatcher) Start() error {
	if s, ok := ld.inner.(common.Runnable); ok {
		return s.Start()
	}
	return nil
}

func (ld *LimitingDispatcher) Close() error { return common.Close(ld.inner) }

func (ld *LimitingDispatcher) isUserValidInInbound(ctx context.Context, user *protocol.MemoryUser, inboundTag string) bool {
	if user == nil || len(user.Email) == 0 {
		return false
	}
	if ld.ibm == nil {
		return true
	}
	handler, err := ld.ibm.GetHandler(ctx, inboundTag)
	if err != nil {
		return false
	}
	userManager, ok := handler.(proxy.UserManager)
	if !ok {
		return true
	}
	return userManager.GetUser(ctx, user.Email) != nil
}

func resolveUserLimits(sessionInbound *session.Inbound) *sessionInfo {
	if sessionInbound == nil || sessionInbound.User == nil {
		return nil
	}
	user := sessionInbound.User
	if len(user.Email) == 0 {
		return nil
	}
	return &sessionInfo{
		tag:   sessionInbound.Tag,
		email: user.Email,
		ip:    sessionInbound.Source.Address.IP().String(),
	}
}

type sessionContext struct {
	inbound   *session.Inbound
	info      *sessionInfo
	user      *protocol.MemoryUser
	bucket    *rate.Limiter
	hasBucket bool
}

func (ld *LimitingDispatcher) resolveSession(ctx context.Context, link *transport.Link) (*sessionContext, error) {
	sessionInbound := session.InboundFromContext(ctx)
	if sessionInbound == nil {
		return nil, nil
	}

	sessionInbound.CanSpliceCopy = 3

	info := resolveUserLimits(sessionInbound)
	if info == nil {
		return nil, nil
	}

	user := sessionInbound.User

	if !ld.isUserValidInInbound(ctx, user, info.tag) {
		parts := strings.Split(info.email, "|")
		logger.Printf("Subscription (ID:%s) deleted. Closing connection", parts[len(parts)-1])
		if sessionInbound.Conn != nil {
			sessionInbound.Conn.Close()
		}
		common.Close(link.Writer)
		common.Interrupt(link.Reader)
		return nil, errors.New("closing connection for: ", info.email)
	}

	bucket, isSpeedLimited, reject := ld.limiter.GetLimiter(
		info.tag,
		info.email,
		info.ip,
	)

	if reject {
		parts := strings.Split(info.email, "|")
		logger.Printf("Subscription (ID:%s) ip/traffic limit exceeded. Connection from %s aborted",
			parts[len(parts)-1], maskIP(info.ip, 2))
		common.Close(link.Writer)
		common.Interrupt(link.Reader)
		return nil, errors.New("subscription limit exceeded for: ", info.email)
	}

	return &sessionContext{
		inbound:   sessionInbound,
		info:      info,
		user:      user,
		bucket:    bucket,
		hasBucket: isSpeedLimited && bucket != nil,
	}, nil
}

func (ld *LimitingDispatcher) getOrCreateCounter(tag string) *counter.TrafficCounter {
	if v, ok := ld.counters.Load(tag); ok {
		return v.(*counter.TrafficCounter)
	}
	tc := counter.NewTrafficCounter()
	if v, loaded := ld.counters.LoadOrStore(tag, tc); loaded {
		return v.(*counter.TrafficCounter)
	}
	return tc
}

func (ld *LimitingDispatcher) getLink(ctx context.Context, link *transport.Link) error {
	sc, err := ld.resolveSession(ctx, link)
	if err != nil || sc == nil {
		return err
	}
	
	tc := ld.getOrCreateCounter(sc.info.tag)
	storage := tc.GetCounter(sc.info.email)
	
	link.Writer = &counter.StatWriter{
		Writer:  link.Writer,
		Storage: storage,
	}

	if sc.hasBucket {
		link.Writer = ld.limiter.RateWriter(link.Writer, sc.bucket)
	}

	return nil
}

func (ld *LimitingDispatcher) wrapLink(ctx context.Context, link *transport.Link) (*transport.Link, error) {
	twr := &buf.TimeoutWrapperReader{Reader: link.Reader}
	link.Reader = twr

	sc, err := ld.resolveSession(ctx, link)
	if err != nil || sc == nil {
		return link, err
	}

	tc := ld.getOrCreateCounter(sc.info.tag)
	storage := tc.GetCounter(sc.info.email)
	
	link.Writer = &counter.StatWriter{
		Writer:  link.Writer,
		Storage: storage,
	}

	link.Reader = &counter.StatReader{
		Reader:  twr,
		Storage: storage,
	}

	if sc.hasBucket {
		link.Writer = ld.limiter.RateWriter(link.Writer, sc.bucket)
		link.Reader = ld.limiter.RateTimeoutReader(twr, sc.bucket)
	}

	return link, nil
}

func (ld *LimitingDispatcher) Dispatch(ctx context.Context, dest net.Destination) (*transport.Link, error) {
	link, err := ld.inner.Dispatch(ctx, dest)
	if err != nil {
		return nil, err
	}
	if err := ld.getLink(ctx, link); err != nil {
		return nil, err
	}
	return link, nil
}

func (ld *LimitingDispatcher) DispatchLink(ctx context.Context, dest net.Destination, link *transport.Link) error {
	wrapped, err := ld.wrapLink(ctx, link)
	if err != nil {
		return err
	}
	return ld.inner.DispatchLink(ctx, dest, wrapped)
}

func (ld *LimitingDispatcher) AddInboundLimiter(tag string, expiry int, nodeSpeedLimit uint64, subscriptionList *[]api.SubscriptionInfo, redisConfig *limiter.RedisConfig) error {
	return ld.limiter.AddInboundLimiter(tag, expiry, nodeSpeedLimit, subscriptionList, redisConfig)
}

func (ld *LimitingDispatcher) UpdateInboundLimiter(tag string, updatedSubscriptionList *[]api.SubscriptionInfo) error {
	return ld.limiter.UpdateInboundLimiter(tag, updatedSubscriptionList)
}

func (ld *LimitingDispatcher) DeleteInboundLimiter(tag string) error {
	return ld.limiter.DeleteInboundLimiter(tag)
}

func (ld *LimitingDispatcher) DeleteSubscriptionBuckets(tag string, emails []string) {
	ld.limiter.DeleteSubscriptionBuckets(tag, emails)
}

func (ld *LimitingDispatcher) GetOnlineIPs(tag string) (*[]api.OnlineIP, error) {
	return ld.limiter.GetOnlineIPs(tag)
}

func (ld *LimitingDispatcher) DrainDeltas(tag string) *limiter.PendingTraffic {
	tc, ok := ld.GetTrafficCounter(tag)
	if !ok || tc == nil {
		return nil
	}
	return ld.limiter.DrainDeltas(tag, tc)
}

func (ld *LimitingDispatcher) ResetTraffic(pending *limiter.PendingTraffic) {
	if pending == nil {
		return
	}
	ld.limiter.ResetTraffic(pending)
}

func (ld *LimitingDispatcher) GetTrafficCounter(tag string) (*counter.TrafficCounter, bool) {
	v, ok := ld.counters.Load(tag)
	if !ok {
		return nil, false
	}
	return v.(*counter.TrafficCounter), true
}

func (ld *LimitingDispatcher) DeleteTrafficCounter(tag string) {
	ld.counters.Delete(tag)
}