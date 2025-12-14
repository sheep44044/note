package main

import (
	"log/slog"
	"note/config"
	"note/internal/cache"
	"note/internal/middleware"
	"note/internal/models"
	"note/internal/note"
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
	rdb, err := cache.New(cfg)
	if err != nil {
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

		noteHandler := note.NewNoteHandler(db, rdb)
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
