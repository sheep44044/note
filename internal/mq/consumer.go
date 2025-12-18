package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"note/internal/ai"
	"note/internal/cache"
	"note/internal/models"

	"gorm.io/gorm"
)

// Consumer 结构体用于持有 DB 连接等依赖
type Consumer struct {
	db     *gorm.DB
	cache  *cache.RedisCache
	rabbit *RabbitMQ
	ai     *ai.AIService
}

// NewConsumer 初始化消费者管理器
func NewConsumer(db *gorm.DB, cache *cache.RedisCache, rabbit *RabbitMQ, ai *ai.AIService) *Consumer {
	return &Consumer{
		db:     db,
		cache:  cache,
		rabbit: rabbit,
		ai:     ai,
	}
}

// Start 启动所有消费者监听
func (c *Consumer) Start() {
	// 启动点赞消费者
	go c.consumeFavorite()
	go c.consumeReaction()
	go c.ConsumeHistory()
	go c.consumeFeedPush()
	go c.consumeAITasks()
	// 这里可以启动其他消费者...
}

func (c *Consumer) consumeFavorite() {
	msgs, err := c.rabbit.Consume("favorite_queue")
	if err != nil {
		slog.Error("Failed to start favorite consumer", "error", err)
		return
	}

	slog.Info("Waiting for favorite messages...")

	for d := range msgs {
		var msg models.FavoriteMsg
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			slog.Error("Failed to unmarshal msg", "error", err)
			continue // 格式错误直接丢弃
		}

		if msg.Action == "add" {
			fav := models.Favorite{UserID: msg.UserID, NoteID: msg.NoteID}
			if err := c.db.Create(&fav).Error; err != nil {
				// 只有非重复键错误才当做异常处理
				if !errors.Is(err, gorm.ErrDuplicatedKey) {
					slog.Error("Failed to insert favorite", "error", err)
					//  在这里可以做重试逻辑，或者手动 Nack
				}
			} else {
				// 更新计数（+1）
				c.db.Model(&models.Note{}).Where("id = ?", msg.NoteID).
					Update("favorite_count", gorm.Expr("favorite_count + 1"))
				slog.Info("Favorite added", "user_id", msg.UserID, "note_id", msg.NoteID)
			}
		} else if msg.Action == "remove" {
			// 处理取消收藏
			if err := c.db.Where("user_id = ? AND note_id = ?", msg.UserID, msg.NoteID).
				Delete(&models.Favorite{}).Error; err == nil {

				c.db.Model(&models.Note{}).Where("id = ?", msg.NoteID).
					Update("favorite_count", gorm.Expr("GREATEST(favorite_count - 1, 0)"))
				slog.Info("Favorite removed", "user_id", msg.UserID, "note_id", msg.NoteID)
			}
		}
	}
}

func (c *Consumer) consumeReaction() {
	msgs, err := c.rabbit.Consume("react_queue")
	if err != nil {
		slog.Error("Failed to start react consumer", "error", err)
		return
	}

	slog.Info("Waiting for react messages...")
	for d := range msgs {
		var msg models.ReactionMsg
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			slog.Error("Failed to unmarshal msg", "error", err)
			continue
		}
		if msg.Action == "add" {
			reaction := models.Reaction{
				UserID: msg.UserID,
				NoteID: msg.NoteID,
				Emoji:  msg.Emoji,
			}
			if err := c.db.Create(&reaction).Error; err != nil {
				if !errors.Is(err, gorm.ErrDuplicatedKey) {
					slog.Error("Failed to insert reaction", "error", err)
				}
			}
		}
	}
}

func (c *Consumer) ConsumeHistory() {
	msgs, err := c.rabbit.Consume("history_queue")
	if err != nil {
		slog.Error("Failed to start react consumer", "error", err)
		return
	}

	slog.Info("Waiting for react messages...")
	for d := range msgs {
		var msg models.HistoryMsg
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			slog.Error("Failed to unmarshal msg", "error", err)
			continue
		}

		c.db.Where("user_id = ? AND note_id = ?", msg.UserID, msg.NoteID).Delete(&models.History{})

		history := models.History{
			UserID: msg.UserID,
			NoteID: msg.NoteID,
		}

		if err := c.db.Create(&history).Error; err != nil {
			slog.Error("Failed to insert history", "error", err)
		}
	}
}

func (c *Consumer) consumeFeedPush() {
	msgs, err := c.rabbit.Consume("feed_queue")
	if err != nil {
		slog.Error("Feed consumer start failed", "err", err)
		return
	}

	// 预定义一个最大 Feed 长度，防止 Redis 无限膨胀
	const MaxFeedLength = 500

	for d := range msgs {
		var msg models.FeedMsg
		json.Unmarshal(d.Body, &msg)

		// 1. 查粉丝 (这一步如果粉丝量巨大，在生产环境通常需要分页查，这里先演示一次性查)
		var fanIDs []uint
		// 从 user_follows 表查谁关注了 AuthorID
		c.db.Model(&models.UserFollow{}).
			Where("followed_id = ?", msg.AuthorID).
			Pluck("follower_id", &fanIDs)

		if len(fanIDs) == 0 {
			continue
		}

		// 2. 批量推送到粉丝的 Redis List
		// 使用 Pipeline 可以在一次网络请求中发送多条命令，极大提升性能
		ctx := context.Background()
		pipe := c.cache.Pipeline() // 假设你的 Consumer 结构体里存了 Redis Client

		for _, fanID := range fanIDs {
			key := fmt.Sprintf("timeline:user:%d", fanID)

			// LPUSH: 把最新笔记 ID 塞到列表头部
			pipe.LPush(ctx, key, msg.NoteID)

			// LTRIM: 保持列表只有最新的 500 条
			pipe.LTrim(ctx, key, 0, MaxFeedLength-1)
		}

		// 执行管道命令
		_, err := pipe.Exec(ctx)
		if err != nil {
			slog.Error("Feed push pipeline failed", "err", err)
			// 这里可以选择 Nack 重试，或者记录日志人工处理
		} else {
			slog.Info("Feed pushed to fans", "author_id", msg.AuthorID, "fan_count", len(fanIDs))
		}
	}
}

func (c *Consumer) consumeAITasks() {
	msgs, err := c.rabbit.Consume("ai_queue")
	if err != nil {
		slog.Error("Failed to start AI consumer", "error", err)
		return
	}

	slog.Info("Waiting for AI tasks...")

	for d := range msgs {
		var msg models.AITaskMsg
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			continue
		}

		slog.Info("Processing AI task", "note_id", msg.NoteID, "task", msg.Task)

		// 1. 查数据库获取笔记内容
		var note models.Note
		if err := c.db.First(&note, msg.NoteID).Error; err != nil {
			slog.Error("Note not found", "id", msg.NoteID)
			continue
		}

		// 2. 根据任务类型调用 AI
		var updateMap = make(map[string]interface{})

		if msg.Task == "generate_title" {
			// 这里调用你在 Service 层写好的方法
			newTitle, err := c.ai.GenerateTitle(note.Content)
			if err == nil && newTitle != "" {
				updateMap["title"] = newTitle
			}
		} else if msg.Task == "generate_summary" {
			summary, err := c.ai.GenerateSummary(note.Content)
			if err == nil && summary != "" {
				updateMap["summary"] = summary
			}
		}

		// 3. 更新数据库
		if len(updateMap) > 0 {
			if err := c.db.Model(&note).Updates(updateMap).Error; err != nil {
				slog.Error("Failed to update note with AI result", "err", err)
			} else {
				slog.Info("AI Update success", "note_id", note.ID)

				cacheKey := fmt.Sprintf("note:%d", note.ID)
				c.cache.Del(context.Background(), cacheKey)
			}
		}
	}
}
