package limiter

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/marshaler"
	"github.com/eko/gocache/lib/v4/store"
	redisStore "github.com/eko/gocache/store/redis/v4"
	"github.com/redis/go-redis/v9"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/stats"
	"golang.org/x/time/rate"

	"github.com/xmplusdev/xmray/api"
	"github.com/xmplusdev/xmray/counter"
)

type SubscriptionInfo struct {
	Id         int
	SpeedLimit uint64
	IPLimit    int
}

type IPData struct {
	UID   int
	Tag   string
	Email string
}

type InboundInfo struct {
	Tag              string
	NodeSpeedLimit   uint64
	IgnoreIPs        []string
	SubscriptionInfo *sync.Map // key: email → SubscriptionInfo
	BucketHub        *sync.Map // key: email → *rate.Limiter
	GlobalIPLimit    struct {
		config         *RedisConfig
		globalOnlineIP *marshaler.Marshaler
	}
}

type Limiter struct {
	server      *core.Instance
	InboundInfo *sync.Map
	stm         stats.Manager
	redisConfig *RedisConfig
	redisClient *redis.Client
}

func New(server *core.Instance, redisConfig *RedisConfig) *Limiter {
	l := &Limiter{
		server:      server,
		InboundInfo: new(sync.Map),
		stm:         server.GetFeature(stats.ManagerType()).(stats.Manager),
		redisConfig: redisConfig,
	}
	if redisConfig != nil && redisConfig.Enable {
		l.redisClient = redis.NewClient(&redis.Options{
			Network:     redisConfig.Network,
			Addr:        redisConfig.Addr,
			Username:    redisConfig.Username,
			Password:    redisConfig.Password,
			DB:          redisConfig.DB,
			DialTimeout: time.Duration(redisConfig.Timeout) * time.Second,
			PoolSize:    10,
		})
	}
	return l
}

// Close shuts down the shared Redis client. Call this when the instance stops.
func (l *Limiter) Close() {
	if l.redisClient != nil {
		if err := l.redisClient.Close(); err != nil {
			log.Printf("[Limiter] error closing Redis client: %v", err)
		}
	}
}

func (l *Limiter) AddInboundLimiter(tag string, expiry int, nodeSpeedLimit uint64, ignoreIPs []string, subscriptionList *[]api.SubscriptionInfo) error {
	inboundInfo := &InboundInfo{
		Tag:            tag,
		NodeSpeedLimit: nodeSpeedLimit,
		IgnoreIPs:      ignoreIPs,
		BucketHub:      new(sync.Map),
	}

	if l.redisClient != nil {
		inboundInfo.GlobalIPLimit.config = l.redisConfig
		rs := redisStore.NewRedis(l.redisClient, store.WithExpiration(time.Duration(expiry)*time.Second))
		inboundInfo.GlobalIPLimit.globalOnlineIP = marshaler.New(cache.New[any](rs))
	} else {
		log.Printf("[Limiter] Redis disabled for tag %s — IP limiting unavailable", tag)
	}

	subscriptionMap := new(sync.Map)
	for _, u := range *subscriptionList {
		key := fmt.Sprintf("%s_%s", tag, u.Email)
		subscriptionMap.Store(key, SubscriptionInfo{
			Id:         u.Id,
			SpeedLimit: u.SpeedLimit,
			IPLimit:    u.IPLimit,
		})
	}
	inboundInfo.SubscriptionInfo = subscriptionMap
	l.InboundInfo.Store(tag, inboundInfo)
	return nil
}

func (l *Limiter) UpdateInboundLimiter(tag string, updatedServiceList *[]api.SubscriptionInfo) error {
	value, ok := l.InboundInfo.Load(tag)
	if !ok {
		return fmt.Errorf("no limiter found for tag %s", tag)
	}
	inboundInfo := value.(*InboundInfo)

	if inboundInfo.GlobalIPLimit.config == nil || !inboundInfo.GlobalIPLimit.config.Enable {
		log.Printf("[Limiter] Redis disabled for tag %s — IP limiting unavailable", tag)
	}

	for _, u := range *updatedServiceList {
		key := fmt.Sprintf("%s_%s", tag, u.Email)
		inboundInfo.SubscriptionInfo.Store(key, SubscriptionInfo{
			Id:         u.Id,
			SpeedLimit: u.SpeedLimit,
			IPLimit:    u.IPLimit,
		})

		limit := determineRate(inboundInfo.NodeSpeedLimit, u.SpeedLimit)
		if limit > 0 {
			if bucket, ok := inboundInfo.BucketHub.Load(key); ok {
				lim := bucket.(*rate.Limiter)
				lim.SetLimit(rate.Limit(limit))
				lim.SetBurst(int(limit))
			}
		} else {
			inboundInfo.BucketHub.Delete(key)
		}
	}
	return nil
}

// UpdateNodeInfo refreshes node-level limiter settings (speed limit and
// ignore-IP list) in-place, without recreating subscription state or Redis
// wiring. Used when the node info changes but the inbound tag stays the same.
func (l *Limiter) UpdateNodeInfo(tag string, nodeSpeedLimit uint64, ignoreIPs []string) error {
	value, ok := l.InboundInfo.Load(tag)
	if !ok {
		return fmt.Errorf("no limiter found for tag %s", tag)
	}
	inboundInfo := value.(*InboundInfo)
	inboundInfo.NodeSpeedLimit = nodeSpeedLimit
	inboundInfo.IgnoreIPs = ignoreIPs
	return nil
}

func (l *Limiter) DeleteInboundLimiter(tag string) error {
	l.InboundInfo.Delete(tag)
	return nil
}

func (l *Limiter) DeleteSubscriptionBuckets(tag string, emails []string) {
	if value, ok := l.InboundInfo.Load(tag); ok {
		inboundInfo := value.(*InboundInfo)
		for _, email := range emails {
			inboundInfo.BucketHub.Delete(email)
			inboundInfo.SubscriptionInfo.Delete(email)
		}
	}
}

func (l *Limiter) GetOnlineIPs(tag string) (*[]api.OnlineIP, error) {
	value, ok := l.InboundInfo.Load(tag)
	if !ok {
		return nil, fmt.Errorf("no such limiter: %s found", tag)
	}

	var onlineIP []api.OnlineIP
	inboundInfo := value.(*InboundInfo)

	if inboundInfo.GlobalIPLimit.config != nil && inboundInfo.GlobalIPLimit.config.Enable {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(inboundInfo.GlobalIPLimit.config.Timeout)*time.Second)
		defer cancel()

		inboundInfo.BucketHub.Range(func(key, value interface{}) bool {
			email := key.(string)
			if _, ok := inboundInfo.SubscriptionInfo.Load(email); ok {
				uniqueKey := strings.TrimPrefix(email, inboundInfo.Tag+"_")
				v, err := inboundInfo.GlobalIPLimit.globalOnlineIP.Get(ctx, uniqueKey, new(map[string][]IPData))
				if err != nil {
					inboundInfo.BucketHub.Delete(email)
				} else {
					ipMap := v.(*map[string][]IPData)
					emailFound := false
					for _, dataList := range *ipMap {
						for _, data := range dataList {
							if data.Email == email {
								emailFound = true
								break
							}
						}
						if emailFound {
							break
						}
					}
					if !emailFound {
						inboundInfo.BucketHub.Delete(email)
					}
				}
			}
			return true
		})

		inboundInfo.SubscriptionInfo.Range(func(key, value interface{}) bool {
			email := key.(string)
			uniqueKey := strings.TrimPrefix(email, inboundInfo.Tag+"_")
			v, err := inboundInfo.GlobalIPLimit.globalOnlineIP.Get(ctx, uniqueKey, new(map[string][]IPData))
			if err == nil {
				ipMap := v.(*map[string][]IPData)
				modified := false
				for ip, dataList := range *ipMap {
					remaining := dataList[:0]
					for _, data := range dataList {
						if data.Tag == tag {
							onlineIP = append(onlineIP, api.OnlineIP{Id: data.UID, IP: ip})
							modified = true
						} else {
							remaining = append(remaining, data)
						}
					}
					(*ipMap)[ip] = remaining
				}
				if modified {
					go pushIP(inboundInfo, uniqueKey, ipMap)
				}
			}
			return true
		})
	}

	return &onlineIP, nil
}

func (l *Limiter) GetLimiter(tag string, email string, ip string) (limiter *rate.Limiter, isSpeedLimited bool, Reject bool) {
	value, ok := l.InboundInfo.Load(tag)
	if !ok {
		newError("Get Limiter information failed").AtDebug()
		return nil, false, false
	}

	var SpeedLimit uint64
	var ipLimit, uid int

	inboundInfo := value.(*InboundInfo)

	if v, ok := inboundInfo.SubscriptionInfo.Load(email); ok {
		u := v.(SubscriptionInfo)
		uid = u.Id
		SpeedLimit = u.SpeedLimit
		ipLimit = u.IPLimit
	}

	ignored := false
	for _, ignoreip := range inboundInfo.IgnoreIPs {
		if ignoreip == ip {
			ignored = true
			break
		}
	}

	if !ignored && inboundInfo.GlobalIPLimit.config != nil && inboundInfo.GlobalIPLimit.config.Enable {
		if checkLimit(inboundInfo, email, uid, ip, ipLimit, tag) {
			return nil, false, true
		}
	}

	limit := determineRate(inboundInfo.NodeSpeedLimit, SpeedLimit)
	if limit > 0 {
		lim := rate.NewLimiter(rate.Limit(limit), int(limit))
		if v, loaded := inboundInfo.BucketHub.LoadOrStore(email, lim); loaded {
			return v.(*rate.Limiter), true, false
		}
		return lim, true, false
	}
	return nil, false, false
}

type pendingCounter struct {
	storage *counter.TrafficStorage
	up      int64
	down    int64
}

type PendingTraffic struct {
	Result   []api.SubscriptionTraffic
	Counters []pendingCounter
}

func (l *Limiter) DrainDeltas(tag string, tc *counter.TrafficCounter) *PendingTraffic {
	value, ok := l.InboundInfo.Load(tag)
	if !ok {
		return nil
	}
	inboundInfo := value.(*InboundInfo)

	pending := &PendingTraffic{}

	inboundInfo.SubscriptionInfo.Range(func(k, v interface{}) bool {
		email := k.(string)
		sub := v.(SubscriptionInfo)

		up := tc.GetUpCount(email)
		down := tc.GetDownCount(email)
		if up == 0 && down == 0 {
			return true
		}

		pending.Result = append(pending.Result, api.SubscriptionTraffic{
			Id:       sub.Id,
			Upload:   up,
			Download: down,
		})

		if s := tc.GetCounter(email); s != nil {
			pending.Counters = append(pending.Counters, pendingCounter{
				storage: s, up: up, down: down,
			})
		}
		return true
	})

	if len(pending.Result) == 0 {
		return nil
	}
	return pending
}

func (l *Limiter) ResetTraffic(pending *PendingTraffic) {
	if pending == nil {
		return
	}
	for _, pc := range pending.Counters {
		pc.storage.UpCounter.Add(-pc.up)
		pc.storage.DownCounter.Add(-pc.down)
	}
}

func checkLimit(inboundInfo *InboundInfo, email string, uid int, ip string, ipLimit int, tag string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(inboundInfo.GlobalIPLimit.config.Timeout)*time.Second)
	defer cancel()

	uniqueKey := strings.TrimPrefix(email, inboundInfo.Tag+"_")

	v, err := inboundInfo.GlobalIPLimit.globalOnlineIP.Get(ctx, uniqueKey, new(map[string][]IPData))
	if err != nil {
		if _, ok := err.(*store.NotFound); ok {
			go pushIP(inboundInfo, uniqueKey, &map[string][]IPData{ip: {{UID: uid, Tag: tag, Email: email}}})
		} else {
			newError("cache service").Base(err).AtError()
		}
		return false
	}

	ipMap := v.(*map[string][]IPData)
	if dataList, ipExists := (*ipMap)[ip]; ipExists {
		found := false
		for i, data := range dataList {
			if data.UID == uid && data.Tag == tag {
				dataList[i] = IPData{UID: uid, Tag: tag, Email: email}
				found = true
				break
			}
		}
		if !found {
			(*ipMap)[ip] = append(dataList, IPData{UID: uid, Tag: tag, Email: email})
		} else {
			(*ipMap)[ip] = dataList
		}
		go pushIP(inboundInfo, uniqueKey, ipMap)
		return false
	}

	if ipLimit > 0 && len(*ipMap) >= ipLimit {
		return true
	}
	(*ipMap)[ip] = []IPData{{UID: uid, Tag: tag, Email: email}}
	go pushIP(inboundInfo, uniqueKey, ipMap)
	return false
}

func pushIP(inboundInfo *InboundInfo, uniqueKey string, ipMap *map[string][]IPData) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(inboundInfo.GlobalIPLimit.config.Timeout)*time.Second)
	defer cancel()
	if err := inboundInfo.GlobalIPLimit.globalOnlineIP.Set(ctx, uniqueKey, ipMap); err != nil {
		newError("Redis cache service").Base(err).AtError()
	}
}

func determineRate(nodeLimit, subscriptionLimit uint64) uint64 {
	switch {
	case nodeLimit == 0 && subscriptionLimit == 0:
		return 0
	case nodeLimit == 0:
		return subscriptionLimit
	case subscriptionLimit == 0:
		return nodeLimit
	default:
		if nodeLimit < subscriptionLimit {
			return nodeLimit
		}
		return subscriptionLimit
	}
}
