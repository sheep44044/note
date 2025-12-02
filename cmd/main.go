package main

import (
	"log/slog"
	"note/config"
	"note/internal/middleware"
	"note/internal/models"
	"note/internal/note"
	"note/internal/redis1"
	"note/internal/tag"
	"note/internal/user"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// 初始化Redis
	if err := redis1.Init(cfg); err != nil {
		slog.Warn("Redis connection failed, continuing without Redis", "error", err)
	} else {
		slog.Info("Redis connected successfully")
	}

	dsn := cfg.DBUser + ":" + cfg.DBPassword + "@tcp(" + cfg.DBHost + ":" + cfg.DBPort + ")/" +
		cfg.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	// 迁移所有模型
	err = db.AutoMigrate(&models.User{}, &models.Note{}, &models.Tag{})
	if err != nil {
		panic("failed to migrate database: " + err.Error())
	}

	r := gin.Default()
	r.Use(middleware.LoggerMiddleware())

	// 公开路由：用户注册/登录
	userHandler := user.NewUserHandler(db, cfg, redis1.Rdb)
	r.POST("/register", userHandler.Register)
	r.POST("/login", userHandler.Login)

	// 鉴权路由
	auth := r.Group("/")
	auth.Use(middleware.JWTAuthMiddleware(cfg))
	{
		auth.POST("/logout", userHandler.Logout)
		auth.POST("/user/change-password", userHandler.ModifyPassword)

		noteHandler := note.NewNoteHandler(db)
		notes := auth.Group("/notes")
		{
			notes.GET("", noteHandler.GetNotes)
			notes.GET("/:id", noteHandler.GetNote)
			notes.POST("", noteHandler.CreateNote)
			notes.PUT("/:id", noteHandler.UpdateNote)
			notes.DELETE("/:id", noteHandler.DeleteNote)

			notes.GET("/recent", noteHandler.GetRecentNotes)
		}

		tagHandler := tag.NewNoteTag(db)
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
