package main

import (
	"note/config"
	"note/internal/middleware"
	"note/internal/models"
	"note/internal/note"
	"note/internal/svc"
	"note/internal/tag"
	"note/internal/user"
	"note/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	utils.InitLogger("dev")

	// 确保程序退出时刷新缓冲区的日志
	defer func() {
		_ = zap.L().Sync()
	}()

	zap.L().Info("Logger initialized successfully")

	svcCtx := svc.NewServiceContext(cfg)
	defer svcCtx.Close()

	// 启动消费者
	if svcCtx.Consumer != nil {
		svcCtx.Consumer.Start()
	}

	// 迁移所有模型
	err = svcCtx.DB.AutoMigrate(&models.User{}, &models.Note{}, &models.Tag{}, &models.Favorite{}, &models.Reaction{}, &models.UserFollow{}, &models.History{})
	if err != nil {
		zap.L().Panic("failed to migrate database", zap.Error(err))
	}

	r := gin.Default()
	r.Use(middleware.LoggerMiddleware())

	// 公开路由：用户注册/登录
	userHandler := user.NewUserHandler(svcCtx)
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
			users.GET("/:id/following", userHandler.GetFollowingList)
			users.GET("/:id/followers", userHandler.GetFollowersList)
		}

		noteHandler := note.NewNoteHandler(svcCtx)
		notes := auth.Group("/notes")
		{
			notes.GET("", noteHandler.GetNotes)
			notes.GET("/:id", noteHandler.GetNote)
			notes.POST("", noteHandler.CreateNote)
			notes.PUT("/:id", middleware.NoteOwnerMiddleware(svcCtx.DB), noteHandler.UpdateNote)
			notes.DELETE("/:id", middleware.NoteOwnerMiddleware(svcCtx.DB), noteHandler.DeleteNote)

			notes.POST("/images", noteHandler.UploadImage)
			notes.GET("/search", noteHandler.SearchNotes)
			notes.GET("/smartsearch", noteHandler.SmartSearch)

			notes.GET("/recent", noteHandler.GetRecentNotes)

			notes.PATCH("/:id/pin", middleware.NoteOwnerMiddleware(svcCtx.DB), noteHandler.TogglePin)
			notes.POST("/:id/favorite", noteHandler.FavoriteNote)
			notes.DELETE("/:id/unfavorite", noteHandler.UnfavoriteNote)
			notes.GET("/favorites", noteHandler.ListMyFavorites)

			notes.GET("/community", noteHandler.ListPublicNotes)
			notes.GET("/follow", noteHandler.GetFollowingFeed)
		}

		tagHandler := tag.NewNoteTag(svcCtx)
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
	zap.L().Info("server starting", zap.String("addr", addr))

	err = r.Run(addr)

	if err != nil {
		zap.L().Fatal("server failed to start", zap.Error(err))
	}
}
