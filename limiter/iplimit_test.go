package limiter

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

const testTag = "node_a"

// newTestLimiter returns an InboundInfo wired to an in-process Redis.
func newTestLimiter(t *testing.T, tag string) (*InboundInfo, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	// No retries: the server is in-process, so a failure is a real one and
	// should surface immediately rather than after a backoff.
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() { client.Close() })

	info := &InboundInfo{
		Tag:              tag,
		SubscriptionInfo: new(sync.Map),
		BucketHub:        new(sync.Map),
	}
	info.GlobalIPLimit.config = &RedisConfig{Enable: true, Timeout: 5}
	info.GlobalIPLimit.client = client
	info.GlobalIPLimit.expiry = 60 * time.Second
	return info, mr
}

// hkeys returns a hash's field names, or nil when the key does not exist.
func hkeys(t *testing.T, mr *miniredis.Miniredis, key string) []string {
	t.Helper()
	keys, err := mr.HKeys(key)
	if err != nil {
		return nil
	}
	return keys
}

// A zero Timeout used to be passed straight to context.WithTimeout, producing a
// context that was already expired — so every Redis command failed instantly
// with "context deadline exceeded" without Redis ever being contacted.
func TestRedisTimeoutNeverZero(t *testing.T) {
	for _, cfg := range []*RedisConfig{nil, {}, {Timeout: 0}, {Timeout: -1}} {
		got := cfg.timeout()
		if got != defaultRedisTimeout {
			t.Errorf("timeout() = %v for %+v, want the %v default", got, cfg, defaultRedisTimeout)
		}
		ctx, cancel := context.WithTimeout(context.Background(), got)
		err := ctx.Err()
		cancel()
		if err != nil {
			t.Errorf("a context built from %+v was born expired: %v", cfg, err)
		}
	}
	if got := (&RedisConfig{Timeout: 3}).timeout(); got != 3*time.Second {
		t.Errorf("timeout() = %v, want 3s when configured", got)
	}
}

func TestRedisPoolSizeDefault(t *testing.T) {
	for _, cfg := range []*RedisConfig{nil, {}, {PoolSize: 0}, {PoolSize: -5}} {
		if got := cfg.poolSize(); got != defaultRedisPoolSize {
			t.Errorf("poolSize() = %d for %+v, want %d", got, cfg, defaultRedisPoolSize)
		}
	}
	if got := (&RedisConfig{PoolSize: 128}).poolSize(); got != 128 {
		t.Errorf("poolSize() = %d, want 128 when configured", got)
	}
}

// The hash key must not collide with the old format's key, which held a plain
// string — a hash command against it would fail with WRONGTYPE.
func TestIPKeyIsNamespaced(t *testing.T) {
	const subscription = "subscription_1@xmplus.subscription"
	if key := ipKey(subscription); key == subscription || !strings.HasPrefix(key, ipKeyPrefix) {
		t.Errorf("key = %q, want a distinct key carrying the %q prefix", key, ipKeyPrefix)
	}
}

// A first connection is admitted and recorded.
func TestIPLimitAdmitsAndRecords(t *testing.T) {
	info, mr := newTestLimiter(t, testTag)

	if checkLimit(info, testTag+"_user@x", 7, "192.0.2.1", 2, testTag) {
		t.Fatal("the first connection was rejected")
	}

	key := ipKey("user@x")
	if got := mr.HGet(key, ipField("192.0.2.1", testTag)); got != "7" {
		t.Errorf("stored UID %q, want %q", got, "7")
	}
	if ttl := mr.TTL(key); ttl != 60*time.Second {
		t.Errorf("TTL = %v, want 60s", ttl)
	}
}

// Reconnecting from a known address is admitted and does not add a field —
// this is the path that used to rewrite the whole map on every connection.
func TestIPLimitRepeatConnectionAddsNothing(t *testing.T) {
	info, mr := newTestLimiter(t, testTag)
	const email = testTag + "_user@x"

	for range 25 {
		if checkLimit(info, email, 7, "192.0.2.1", 1, testTag) {
			t.Fatal("a repeat connection from a known address was rejected")
		}
	}
	if n := len(hkeys(t, mr, ipKey("user@x"))); n != 1 {
		t.Errorf("hash holds %d fields after 25 connections, want 1", n)
	}
}

// The limit counts distinct addresses.
func TestIPLimitRejectsBeyondLimit(t *testing.T) {
	info, _ := newTestLimiter(t, testTag)
	const email = testTag + "_user@x"

	if checkLimit(info, email, 7, "192.0.2.1", 2, testTag) {
		t.Fatal("address 1 rejected")
	}
	if checkLimit(info, email, 7, "192.0.2.2", 2, testTag) {
		t.Fatal("address 2 rejected while under the limit")
	}
	if !checkLimit(info, email, 7, "192.0.2.3", 2, testTag) {
		t.Error("address 3 admitted despite a limit of 2")
	}
	// The rejected address must not have been recorded.
	if !checkLimit(info, email, 7, "192.0.2.3", 2, testTag) {
		t.Error("a rejected address was recorded and later admitted")
	}
}

// An address already counted under another node is admitted even at the limit:
// the limit is on distinct addresses, not on connections.
func TestIPLimitSameAddressAcrossNodes(t *testing.T) {
	info, mr := newTestLimiter(t, testTag)
	const email = testTag + "_user@x"

	if checkLimit(info, email, 7, "192.0.2.1", 1, testTag) {
		t.Fatal("first address rejected")
	}
	if checkLimit(info, email, 7, "192.0.2.1", 1, "node_b") {
		t.Error("the same address on a second node was rejected")
	}
	if !checkLimit(info, email, 7, "192.0.2.9", 1, "node_b") {
		t.Error("a second distinct address was admitted despite a limit of 1")
	}
	if fields := hkeys(t, mr, ipKey("user@x")); len(fields) != 2 {
		t.Errorf("hash holds %d fields, want 2 (one per node)", len(fields))
	}
}

// A limit of zero means unlimited.
func TestIPLimitZeroIsUnlimited(t *testing.T) {
	info, mr := newTestLimiter(t, testTag)
	const email = testTag + "_user@x"

	for i := range 40 {
		if checkLimit(info, email, 7, fmt.Sprintf("192.0.2.%d", i), 0, testTag) {
			t.Fatalf("address %d rejected with no limit set", i)
		}
	}
	if n := len(hkeys(t, mr, ipKey("user@x"))); n != 40 {
		t.Errorf("recorded %d addresses, want 40", n)
	}
}

// The reason this was rewritten: concurrent connections for one subscription
// used to each read the whole map, add their address and write it back, so all
// but the last were erased. Every admitted address must survive.
func TestIPLimitConcurrentNoLostUpdates(t *testing.T) {
	info, mr := newTestLimiter(t, testTag)
	const email = testTag + "_user@x"
	const addresses = 50

	var wg sync.WaitGroup
	admitted := make([]bool, addresses)
	for i := range addresses {
		wg.Add(1)
		go func() {
			defer wg.Done()
			admitted[i] = !checkLimit(info, email, 7, fmt.Sprintf("192.0.2.%d", i), 0, testTag)
		}()
	}
	wg.Wait()

	want := 0
	for _, ok := range admitted {
		if ok {
			want++
		}
	}
	if got := len(hkeys(t, mr, ipKey("user@x"))); got != want {
		t.Errorf("%d addresses were admitted but only %d survived — updates were lost", want, got)
	}
}

// Under a limit, concurrent connections must admit exactly the limit, never
// more. The old check-then-write let several racers pass the count together.
func TestIPLimitConcurrentRespectsLimit(t *testing.T) {
	info, mr := newTestLimiter(t, testTag)
	const email = testTag + "_user@x"
	const limit = 3

	var wg sync.WaitGroup
	var mu sync.Mutex
	admitted := 0
	for i := range 40 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !checkLimit(info, email, 7, fmt.Sprintf("192.0.2.%d", i), limit, testTag) {
				mu.Lock()
				admitted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if admitted != limit {
		t.Errorf("admitted %d addresses, want exactly %d", admitted, limit)
	}
	if got := len(hkeys(t, mr, ipKey("user@x"))); got != limit {
		t.Errorf("recorded %d addresses, want %d", got, limit)
	}
}

// GetOnlineIPs reports this node's addresses and clears only those, leaving
// another node's entries for that node to report.
func TestGetOnlineIPsClearsOnlyOwnTag(t *testing.T) {
	info, mr := newTestLimiter(t, testTag)
	const email = testTag + "_user@x"
	info.SubscriptionInfo.Store(email, SubscriptionInfo{Id: 7})

	checkLimit(info, email, 7, "192.0.2.1", 0, testTag)
	checkLimit(info, email, 7, "192.0.2.2", 0, testTag)
	checkLimit(info, email, 7, "192.0.2.3", 0, "node_b")

	l := &Limiter{InboundInfo: new(sync.Map)}
	l.InboundInfo.Store(testTag, info)

	online, err := l.GetOnlineIPs(testTag)
	if err != nil {
		t.Fatalf("GetOnlineIPs: %v", err)
	}
	if len(*online) != 2 {
		t.Fatalf("reported %d addresses, want 2 from this node", len(*online))
	}
	for _, o := range *online {
		if o.Id != 7 {
			t.Errorf("reported UID %d, want 7", o.Id)
		}
		if o.IP != "192.0.2.1" && o.IP != "192.0.2.2" {
			t.Errorf("reported unexpected address %q", o.IP)
		}
	}

	remaining := hkeys(t, mr, ipKey("user@x"))
	if len(remaining) != 1 || remaining[0] != ipField("192.0.2.3", "node_b") {
		t.Errorf("remaining fields = %v, want only the other node's entry", remaining)
	}
}

// IPv6 literals contain colons; the field separator and suffix trimming must
// survive them intact.
func TestIPLimitIPv6RoundTripThroughRedis(t *testing.T) {
	info, _ := newTestLimiter(t, testTag)
	const email = testTag + "_user@x"
	const addr = "2001:db8:85a3::8a2e:370:7334"
	info.SubscriptionInfo.Store(email, SubscriptionInfo{Id: 7})

	if checkLimit(info, email, 7, addr, 0, testTag) {
		t.Fatal("IPv6 address rejected")
	}

	l := &Limiter{InboundInfo: new(sync.Map)}
	l.InboundInfo.Store(testTag, info)
	online, err := l.GetOnlineIPs(testTag)
	if err != nil {
		t.Fatalf("GetOnlineIPs: %v", err)
	}
	if len(*online) != 1 || (*online)[0].IP != addr {
		t.Errorf("reported %v, want the IPv6 address intact", *online)
	}
}

// A Redis failure must not lock users out.
func TestIPLimitFailsOpen(t *testing.T) {
	info, mr := newTestLimiter(t, testTag)
	mr.Close() // every command from here on errors

	if checkLimit(info, testTag+"_user@x", 7, "192.0.2.1", 1, testTag) {
		t.Error("a connection was rejected because Redis was unreachable")
	}
}
