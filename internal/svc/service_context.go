package svc

import (
	"context"
	"note/config"
	"note/internal/infra/ai"
	"note/internal/infra/cache"
	"note/internal/infra/db"
	mq2 "note/internal/infra/mq"
	"note/internal/infra/storage"
	"note/internal/infra/vector"
	"note/internal/middleware"
	"note/internal/utils"
	"os"
	"time"

	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config *config.Config
	DB     *gorm.DB
	Cache  *cache.RedisCache
	Rabbit *mq2.RabbitMQ
	AI     *ai.AIService
	Qdrant *vector.QdrantService
	Minio  *storage.FileStorage

	// 私有字段，用于存储需要关闭的资源
	tracerProvider *trace.TracerProvider
	Consumer       *mq2.Consumer
}

// NewServiceContext 这里是所有初始化的总入口
func NewServiceContext(cfg *config.Config) *ServiceContext {
	dbConn := db.InitMySQL(cfg)

	rdb, err := cache.New(cfg)
	if err != nil {
		zap.L().Warn("Redis connection failed, continuing without Redis", zap.Error(err))
	} else {
		zap.L().Info("Redis connected successfully")
		utils.RedisClient = rdb
	}

	rabbit, err := mq2.New(cfg)
	if err != nil {
		zap.L().Warn("RabbitMQ connection failed", zap.Error(err))
	}

	qdrant := vector.NewQdrantService(cfg.QdrantHost, cfg.QdrantPort, "notes_collection", cfg.QdrantAPIKey)

	aiService := ai.NewAIService(cfg)

	consumer := mq2.NewConsumer(dbConn, rdb, rabbit, aiService, qdrant)

	minioSvc, _ := storage.NewFileStorage(
		cfg.MinioEndpoint,  // 内部连接用: "minio:9000"
		cfg.MinioPublicURL, // 外部展示用: "http://localhost:9000" (上线改成服务器IP)
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
	)

	jaegerURL := os.Getenv("JAEGER_ENDPOINT")
	if jaegerURL == "" {
		jaegerURL = "http://localhost:14268/api/traces"
	}

	// 初始化 Tracer
	tp, err := middleware.InitTracer("note-service", jaegerURL)
	if err != nil {
		zap.L().Fatal("failed to init tracer", zap.Error(err))
	}

	return &ServiceContext{
		Config:         cfg,
		DB:             dbConn,
		Cache:          rdb,
		Rabbit:         rabbit,
		AI:             aiService,
		Qdrant:         qdrant,
		Minio:          minioSvc,
		Consumer:       consumer,
		tracerProvider: tp,
	}
}

func (s *ServiceContext) Close() {
	// 关闭 Tracer
	if s.tracerProvider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.tracerProvider.Shutdown(ctx); err != nil {
			zap.L().Error("Tracer shutdown error", zap.Error(err))
		}
	}

	// 关闭 RabbitMQ
	if s.Rabbit != nil {
		s.Rabbit.Close()
		zap.L().Info("RabbitMQ closed")
	}
}
