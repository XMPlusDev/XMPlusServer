package limiter

type RedisConfig struct {
	Enable   bool   `mapstructure:"Enable"`
	Network  string `mapstructure:"Network"`
	Addr     string `mapstructure:"Addr"`
	Username string `mapstructure:"Username"`
	Password string `mapstructure:"Password"`
	DB       int    `mapstructure:"DB"`
	Timeout  int    `mapstructure:"Timeout"`
}
