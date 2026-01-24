package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBName     string `mapstructure:"DB_NAME"`
	ServerPort string `mapstructure:"SERVER_PORT"`

	JWTSecretKey      string        `mapstructure:"JWT_SECRET_KEY"`
	JWTIssuer         string        `mapstructure:"JWT_ISSUER"`
	JWTExpirationTime time.Duration `mapstructure:"JWT_EXPIRATION_TIME"`

	RedisHost     string `mapstructure:"REDIS_HOST"`
	RedisPort     string `mapstructure:"REDIS_PORT"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`
	RedisDB       int    `mapstructure:"REDIS_DB"`

	MQHost     string `mapstructure:"RABBITMQ_HOST"`
	MQPort     string `mapstructure:"RABBITMQ_PORT"`
	MQUser     string `mapstructure:"RABBITMQ_USER"`
	MQPassword string `mapstructure:"RABBITMQ_PASSWORD"`

	QdrantHost   string `mapstructure:"QDRANT_HOST"`
	QdrantPort   int    `mapstructure:"QDRANT_PORT"`
	QdrantAPIKey string `mapstructure:"QDRANT_API_KEY"`

	VolcEngineKey     string `mapstructure:"VolcEngineKey"`
	VolcEngineBaseURL string `mapstructure:"VolcEngineBaseURL"`
	VolcChatModelID   string `mapstructure:"VolcChatModelID"`
	VolcEmbedModelID  string `mapstructure:"VolcEmbedModelID"`
}

func Load() (*Config, error) {
	v := viper.New()
	setDefaults(v)
	configureViper(v)
	if err := readConfiguration(v); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// 兜底默认值（如果 env 未设置）
	if cfg.JWTSecretKey == "" {
		cfg.JWTSecretKey = "your_fallback_secret_key_change_in_production"
	}
	if cfg.JWTIssuer == "" {
		cfg.JWTIssuer = "note_app"
	}
	if cfg.JWTExpirationTime == 0 {
		cfg.JWTExpirationTime = time.Hour * 24
	}

	// Redis默认值
	if cfg.RedisHost == "" {
		cfg.RedisHost = "localhost"
	}
	if cfg.RedisPort == "" {
		cfg.RedisPort = "6379"
	}
	if cfg.RedisDB == 0 {
		cfg.RedisDB = 0
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("DB_USER", "root")
	v.SetDefault("DB_PASSWORD", "root")
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", "3306")
	v.SetDefault("DB_NAME", "notes_db")

	v.SetDefault("SERVER_PORT", "8080")
	v.SetDefault("JWT_SECRET_KEY", "your_fallback_secret_key_change_in_production")
	v.SetDefault("JWT_ISSUER", "note_app")
	v.SetDefault("JWT_EXPIRATION_TIME", "24h")

	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", "6379")
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", "0")

	v.SetDefault("RABBITMQ_HOST", "localhost")
	v.SetDefault("RABBITMQ_PORT", "5672")
	v.SetDefault("RABBITMQ_USER", "admin")
	v.SetDefault("RABBITMQ_PASSWORD", "123456")

	v.SetDefault("QDRANT_HOST", "localhost")
	v.SetDefault("QDRANT_PORT", 6334)
	v.SetDefault("QDRANT_API_KEY", "")

	v.SetDefault("volc.engine.key", "这里填你新生成的Key") // 生产环境务必通过环境变量覆盖
	v.SetDefault("volc.engine.base_url", "https://ark.cn-beijing.volces.com/api/v3")
	v.SetDefault("volc.engine.chat_model_id", "ep-20251219174526-wnm95")
	v.SetDefault("volc.engine.embed_model_id", "ep-20251219175039-rrcf2")
}

func configureViper(v *viper.Viper) {
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

func readConfiguration(v *viper.Viper) error {
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Warning: .env file not found, using defaults and system env")
			return nil
		}
		return fmt.Errorf("config file error: %w", err)
	}
	fmt.Printf("Using config file: %s\n", v.ConfigFileUsed())
	return nil
}
