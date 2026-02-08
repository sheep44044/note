package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv string `mapstructure:"APP_ENV"`

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

	VolcEngineKey     string `mapstructure:"VOLC_ENGINE_KEY"`
	VolcEngineBaseURL string `mapstructure:"VOLC_ENGINE_BASE_URL"`
	VolcChatModelID   string `mapstructure:"VOLC_CHAT_MODEL_ID"`
	VolcEmbedModelID  string `mapstructure:"VOLC_EMBED_MODEL_ID"`

	MinioEndpoint  string `mapstructure:"MINIO_ENDPOINT"`
	MinioPublicURL string `mapstructure:"MINIO_PUBLIC_URL"`
	MinioAccessKey string `mapstructure:"MINIO_ACCESS_KEY"`
	MinioSecretKey string `mapstructure:"MINIO_SECRET_KEY"`
	MinioBucket    string `mapstructure:"MINIO_BUCKET"`
	MinioUseSSL    bool   `mapstructure:"MINIO_USE_SSL"`

	JaegerEndpoint string `mapstructure:"JAEGER_ENDPOINT"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	setDefaults(v)

	// 允许环境变量覆盖 (例如在 Docker/K8s 中通过 ENV 注入)
	// 这一步非常关键：它允许系统环境变量覆盖 .env 文件
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			fmt.Println("Warning: .env file not found, using defaults and system env")
		}
	} else {
		fmt.Printf("Using config file: %s\n", v.ConfigFileUsed())
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("APP_ENV", "dev") // 默认为开发环境

	v.SetDefault("DB_USER", "root")
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", "3306")

	v.SetDefault("SERVER_PORT", "8080")
	v.SetDefault("JWT_EXPIRATION_TIME", "24h")

	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", "6379")
	v.SetDefault("REDIS_DB", 0)

	v.SetDefault("QDRANT_HOST", "localhost")
	v.SetDefault("QDRANT_PORT", 6334)

	v.SetDefault("VOLC_ENGINE_BASE_URL", "https://ark.cn-beijing.volces.com/api/v3")

	v.SetDefault("MINIO_ENDPOINT", "localhost:9000")
	v.SetDefault("MINIO_PUBLIC_URL", "http://localhost:9000")
	v.SetDefault("MINIO_BUCKET", "notes-images")
	v.SetDefault("MINIO_USE_SSL", false)

	v.SetDefault("JAEGER_ENDPOINT", "http://localhost:14268/api/traces")
}
