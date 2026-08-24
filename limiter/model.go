package limiter

import "time"

const (
	// defaultRedisTimeout applies when Timeout is unset or zero.
	//
	// It matters more than it looks: the deadline is built with
	// context.WithTimeout, and a zero duration yields a context that is already
	// expired, so every command fails immediately with "context deadline
	// exceeded" before Redis is even contacted.
	defaultRedisTimeout = 10 * time.Second

	// defaultRedisPoolSize is sized for connection rate rather than a fixed
	// guess. The pool was pinned at 10, so once those slots were busy every
	// further command waited for one — and that wait counts against the
	// deadline.
	defaultRedisPoolSize = 64
)

type RedisConfig struct {
	Enable   bool   `mapstructure:"Enable"`
	Network  string `mapstructure:"Network"`
	Addr     string `mapstructure:"Addr"`
	Username string `mapstructure:"Username"`
	Password string `mapstructure:"Password"`
	DB       int    `mapstructure:"DB"`
	Timeout  int    `mapstructure:"Timeout"`  // seconds per command; 0 uses defaultRedisTimeout
	PoolSize int    `mapstructure:"PoolSize"` // connections; 0 uses defaultRedisPoolSize
}

// timeout returns the deadline for a single Redis command.
func (c *RedisConfig) timeout() time.Duration {
	if c == nil || c.Timeout <= 0 {
		return defaultRedisTimeout
	}
	return time.Duration(c.Timeout) * time.Second
}

// poolSize returns the Redis connection pool size.
func (c *RedisConfig) poolSize() int {
	if c == nil || c.PoolSize <= 0 {
		return defaultRedisPoolSize
	}
	return c.PoolSize
}
