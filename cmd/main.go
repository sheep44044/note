package main

import (
	"context"
	"log"
	"log/slog"
	"note/config"
	"note/internal/ai"
	"note/internal/cache"
	"note/internal/middleware"
	"note/internal/models"
	"note/internal/mq"
	"note/internal/note"
	"note/internal/storage"
	"note/internal/tag"
	"note/internal/user"
	"note/internal/vector"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	dsn := cfg.DBUser + ":" + cfg.DBPassword + "@tcp(" + cfg.DBHost + ":" + cfg.DBPort + ")/" +
		cfg.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	// 初始化Redis
	rdb, err := cache.New(cfg)
	if err != nil {
		slog.Warn("Redis connection failed, continuing without Redis", "error", err)
	} else {
		slog.Info("Redis connected successfully")
	}

	rabbit, err := mq.New(cfg)
	if err != nil {
		// 如果 MQ 是必须的，这里应该 panic；如果是可选的，可以 log warn
		slog.Warn("RabbitMQ connection failed", "error", err)
	} else {
		slog.Info("RabbitMQ connected successfully")
		defer rabbit.Close() // 程序退出时关闭
	}

	qdrant := vector.NewQdrantService(cfg.QdrantHost, cfg.QdrantPort, "notes_collection", cfg.QdrantAPIKey)

	aiService := ai.NewAIService(cfg)

	consumer := mq.NewConsumer(db, rdb, rabbit, aiService)
	consumer.Start()

	minioSvc := storage.NewFileStorage(
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
		log.Fatal(err)
	}
	// 程序退出时关闭 Tracer，把剩下的数据发出去
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	// 迁移所有模型
	err = db.AutoMigrate(&models.User{}, &models.Note{}, &models.Tag{}, &models.Favorite{}, &models.Reaction{}, &models.UserFollow{})
	if err != nil {
		panic("failed to migrate database: " + err.Error())
	}

	r := gin.Default()
	r.Use(middleware.LoggerMiddleware())

	// 公开路由：用户注册/登录
	userHandler := user.NewUserHandler(db, cfg, rdb)
	r.POST("/register", userHandler.Register)
	r.POST("/login", userHandler.Login)

	// 鉴权路由
	auth := r.Group("/")
	auth.Use(middleware.JWTAuthMiddleware(cfg))
	{
		users := auth.Group("/users")
		{
			users.POST("/logout", userHandler.Logout)
			users.POST("/change-password", userHandler.ModifyPassword)

			users.GET("/me", userHandler.PersonalPage)
			users.PUT("/me", userHandler.UpdateMyProfile)

			users.POST("/:id/follow", userHandler.FollowUser)
			users.DELETE("/:id/follow", userHandler.UnfollowUser)
		}

		noteHandler := note.NewNoteHandler(db, rdb, rabbit, aiService, qdrant, minioSvc)
		notes := auth.Group("/notes")
		{
			notes.GET("", noteHandler.GetNotes)
			notes.GET("/search", noteHandler.SearchNotes)
			notes.GET("/:id", middleware.NoteOwnerMiddleware(db), noteHandler.GetNote)
			notes.POST("", noteHandler.CreateNote)
			notes.PUT("/:id", middleware.NoteOwnerMiddleware(db), noteHandler.UpdateNote)
			notes.DELETE("/:id", middleware.NoteOwnerMiddleware(db), noteHandler.DeleteNote)

			notes.GET("/recent", noteHandler.GetRecentNotes)

			notes.PATCH("/:id/pin", noteHandler.TogglePin)
			notes.POST("/:id/favorite", noteHandler.FavoriteNote)
			notes.DELETE("/:id/unfavorite", noteHandler.UnfavoriteNote)
			notes.GET("/favorites", noteHandler.ListMyFavorites)

			notes.GET("/community", noteHandler.ListPublicNotes)
			notes.GET("/follow", noteHandler.GetFollowingFeed)
		}

		tagHandler := tag.NewNoteTag(db, rdb)
		tags := auth.Group("/tags")
		{
			tags.GET("", tagHandler.GetTags)
			tags.GET("/:id", tagHandler.GetTag)
			tags.POST("", tagHandler.CreateTag)
			tags.PUT("/:id", tagHandler.UpdateTag)
			tags.DELETE("/:id", tagHandler.DeleteTag)
		}
	}

	addr := ":" + cfg.ServerPort
	slog.Info("server starting", "addr", addr)
	r.Run(addr)
}
