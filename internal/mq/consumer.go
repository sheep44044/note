package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"note/internal/ai"
	"note/internal/cache"
	"note/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	go c.consumeFavorite()
	go c.consumeReaction()
	go c.ConsumeHistory()
	go c.consumeFeedPush()
	go c.consumeAITasks()
}

func (c *Consumer) consumeFavorite() {
	msgs, err := c.rabbit.Consume("favorite_queue")
	if err != nil {
		zap.L().Error("Failed to start favorite consumer", zap.Error(err))
		return
	}

	zap.L().Info("Waiting for favorite messages...")

	for d := range msgs {
		var msg models.FavoriteMsg
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			zap.L().Error("Failed to unmarshal msg", zap.Error(err))
			continue
		}

		err := c.db.Transaction(func(tx *gorm.DB) error {
			if msg.Action == "add" {
				fav := models.Favorite{UserID: msg.UserID, NoteID: msg.NoteID}

				if err := tx.Create(&fav).Error; err != nil {
					// 如果是重复收藏，直接返回 nil (忽略错误)，不更新计数
					if errors.Is(err, gorm.ErrDuplicatedKey) {
						return nil
					}
					return err // 其他错误抛出
				}

				return tx.Model(&models.Note{}).Where("id = ?", msg.NoteID).
					Update("favorite_count", gorm.Expr("favorite_count + 1")).Error

			} else if msg.Action == "remove" {
				result := tx.Where("user_id = ? AND note_id = ?", msg.UserID, msg.NoteID).
					Delete(&models.Favorite{})

				if result.Error != nil {
					return result.Error
				}

				if result.RowsAffected > 0 {
					return tx.Model(&models.Note{}).Where("id = ?", msg.NoteID).
						Update("favorite_count", gorm.Expr("GREATEST(favorite_count - 1, 0)")).Error
				}
				return nil
			}
			return nil
		})

		if err != nil {
			zap.L().Error("Process favorite msg failed",
				zap.String("action", msg.Action),
				zap.Uint("uid", msg.UserID),
				zap.Error(err),
			)
		} else {
			zap.L().Info("Favorite processed",
				zap.String("action", msg.Action),
				zap.Uint("uid", msg.UserID),
				zap.Uint("nid", msg.NoteID),
			)
		}
	}
}

func (c *Consumer) consumeReaction() {
	msgs, err := c.rabbit.Consume("react_queue")
	if err != nil {
		zap.L().Error("Failed to start react consumer", zap.Error(err))
		return
	}
	zap.L().Info("Waiting for react messages...")

	for d := range msgs {
		var msg models.ReactionMsg
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			zap.L().Error("Failed to unmarshal msg", zap.Error(err))
			continue
		}

		if msg.Action == "toggle" {
			c.handleToggleReaction(msg)
		}
	}
}

func (c *Consumer) handleToggleReaction(msg models.ReactionMsg) {
	err := c.db.Transaction(func(tx *gorm.DB) error {

		result := tx.Where("user_id = ? AND note_id = ? AND emoji = ?", msg.UserID, msg.NoteID, msg.Emoji).
			Delete(&models.Reaction{})

		if result.Error != nil {
			return result.Error
		}

		// 定义变更量：1 (新增) 或 -1 (取消)
		delta := 0
		if result.RowsAffected > 0 {
			delta = -1 // 之前有点赞，现在取消
			zap.L().Info("Reaction removed", zap.Uint("uid", msg.UserID), zap.String("emoji", msg.Emoji))
		} else {
			// 之前没点赞，现在新增
			reaction := models.Reaction{
				UserID: msg.UserID,
				NoteID: msg.NoteID,
				Emoji:  msg.Emoji,
			}
			if err := tx.Create(&reaction).Error; err != nil {
				// 处理并发重复点击
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return nil
				}
				return err
			}
			delta = 1
			zap.L().Info("Reaction added", zap.Uint("uid", msg.UserID), zap.String("emoji", msg.Emoji))
		}

		var note models.Note
		// 锁定这行记录 (Select For Update)
		// 这样在事务提交前，其他 Consumer 无法修改这条 Note，保证计数准确
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&note, msg.NoteID).Error; err != nil {
			return err
		}

		if note.ReactionCounts == nil {
			note.ReactionCounts = make(map[string]int)
		}

		currentCount := note.ReactionCounts[msg.Emoji]
		newCount := currentCount + delta

		if newCount < 0 {
			newCount = 0
		}

		if newCount == 0 {
			delete(note.ReactionCounts, msg.Emoji)
		} else {
			note.ReactionCounts[msg.Emoji] = newCount
		}

		// 注意：必须使用 Select("ReactionCounts") 或者 Save，确保 GORM 知道要更新这个字段
		// 使用 UpdateColumn 可以避免更新 UpdatedAt 时间戳（如果你不想让点赞刷新修改时间）
		if err := tx.Model(&note).Update("reaction_counts", note.ReactionCounts).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		zap.L().Error("Handle reaction failed", zap.Error(err))
	}
}

func (c *Consumer) ConsumeHistory() {
	msgs, err := c.rabbit.Consume("history_queue")
	if err != nil {
		zap.L().Error("Failed to start react consumer", zap.Error(err))
		return
	}

	zap.L().Info("Waiting for react messages...")
	for d := range msgs {
		var msg models.HistoryMsg
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			zap.L().Error("Failed to unmarshal msg", zap.Error(err))
			continue
		}

		history := models.History{
			UserID: msg.UserID,
			NoteID: msg.NoteID,
		}

		err := c.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "note_id"}}, // 指定冲突列
			DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),      // 冲突时更新时间
		}).Create(&history).Error

		if err != nil {
			zap.L().Error("Failed to upsert history", zap.Error(err))
		}
	}
}

func (c *Consumer) consumeFeedPush() {
	msgs, err := c.rabbit.Consume("feed_queue")
	if err != nil {
		zap.L().Error("Feed consumer start failed", zap.Error(err))
		return
	}

	const MaxFeedLength = 500

	for d := range msgs {
		var msg models.FeedMsg
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			zap.L().Error("Failed to unmarshal msg", zap.Error(err))
			continue
		}

		var fanIDs []uint
		c.db.Model(&models.UserFollow{}).
			Where("followed_id = ?", msg.AuthorID).
			Pluck("follower_id", &fanIDs)

		if len(fanIDs) == 0 {
			continue
		}

		ctx := context.Background()
		pipe := c.cache.Pipeline()

		for _, fanID := range fanIDs {
			key := fmt.Sprintf("timeline:user:%d", fanID)

			pipe.LPush(ctx, key, msg.NoteID)
			pipe.LTrim(ctx, key, 0, MaxFeedLength-1)
		}

		_, err := pipe.Exec(ctx)
		if err != nil {
			zap.L().Error("Feed push pipeline failed", zap.Error(err))
		} else {
			zap.L().Info("Feed pushed to fans", zap.Uint("author_id", msg.AuthorID), zap.Int("fan_count", len(fanIDs)))
		}
	}
}

func (c *Consumer) consumeAITasks() {
	msgs, err := c.rabbit.Consume("ai_queue")
	if err != nil {
		zap.L().Error("Failed to start AI consumer", zap.Error(err))
		return
	}

	zap.L().Info("Waiting for AI tasks...")

	for d := range msgs {
		var msg models.AITaskMsg
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			continue
		}

		zap.L().Info("Processing AI task", zap.Uint("note_id", msg.NoteID), zap.String("task", msg.Task))

		var note models.Note
		if err := c.db.First(&note, msg.NoteID).Error; err != nil {
			zap.L().Error("Note not found", zap.Uint("id", msg.NoteID))
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

		if len(updateMap) > 0 {
			if err := c.db.Model(&note).Updates(updateMap).Error; err != nil {
				zap.L().Error("Failed to update note with AI result", zap.Error(err))
			} else {
				zap.L().Info("AI Update success", zap.Uint("note_id", note.ID))

				cacheKey := fmt.Sprintf("note:%d", note.ID)
				_ = c.cache.Del(context.Background(), cacheKey)
			}
		}
	}
}
