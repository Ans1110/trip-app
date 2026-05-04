package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	JWT          JWTConfig
	OAuth        OAuthConfig
	Cloudinary   CloudinaryConfig
	External     ExternalConfig
	Notification NotificationConfig
}

type ServerConfig struct {
	Port           int
	Mode           string
	AllowedOrigins []string      `mapstructure:"allowed_origins"`
	RequestTimeout time.Duration `mapstructure:"request_timeout"`
	BodyLimitMB    int           `mapstructure:"body_limit_mb"`
	TrustedProxies []string      `mapstructure:"trusted_proxies"`
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type JWTConfig struct {
	PrivateKeyPath  string        `mapstructure:"private_key_path"`
	PublicKeyPath   string        `mapstructure:"public_key_path"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

type OAuthConfig struct {
	Google   GoogleOAuthConfig
	Facebook FacebookOAuthConfig
	Github   GithubOAuthConfig
}

type GoogleOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

type FacebookOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

type GithubOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

type CloudinaryConfig struct {
	CloudName string `mapstructure:"cloud_name"`
	APIKey    string `mapstructure:"api_key"`
	APISecret string `mapstructure:"api_secret"`
	Folder    string `mapstructure:"folder"`
}

type ExternalConfig struct {
	GoogleMapsAPIKey  string `mapstructure:"google_maps_api_key"`
	OpenWeatherAPIKey string `mapstructure:"open_weather_api_key"`
	OpenAIAPIKey      string `mapstructure:"openai_api_key"`
	GeminiAPIKey      string `mapstructure:"gemini_api_key"`
}

type NotificationConfig struct {
	FCMCredentialPath string `mapstructure:"fcm_credential_path"`
	SMTP              SMTPConfig
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func Load(cfgPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(cfgPath)
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
