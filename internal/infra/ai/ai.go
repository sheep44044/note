package ai

import (
	"context"
	"fmt"
	"note/config"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sashabaranov/go-openai"
)

type AIService struct {
	client *openai.Client
	cfg    *config.Config
}

func NewAIService(cfg *config.Config) *AIService {
	// 使用配置文件中的 BaseURL 和 Key 初始化
	aiConfig := openai.DefaultConfig(cfg.VolcEngineKey)
	aiConfig.BaseURL = cfg.VolcEngineBaseURL

	return &AIService{
		client: openai.NewClientWithConfig(aiConfig),
		cfg:    cfg,
	}
}

// GenerateTitle 使用 Chat 模型 (读取 VOLC_CHAT_MODEL_ID)
func (s *AIService) GenerateTitle(content string) (string, error) {
	// 设置 30 秒超时：如果 30 秒没生成完，强制取消，报错返回
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	safeContent := truncateContent(content, 2000)

	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{

			Model: s.cfg.VolcChatModelID,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "你是一个笔记助手，请为以下内容生成一个15字以内的标题，不要包含引号：",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: safeContent,
				},
			},
			Temperature: 0.7,
		},
	)
	if err != nil {
		return "", fmt.Errorf("title generation failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("api returned no choices")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// GenerateSummary 调用 AI 生成摘要
func (s *AIService) GenerateSummary(content string) (string, error) {
	// 设置 30 秒超时：如果 30 秒没生成完，强制取消，报错返回
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	safeContent := truncateContent(content, 2000)

	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: s.cfg.VolcChatModelID,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "请为以下笔记生成一段50字以内的简短摘要。",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: safeContent,
				},
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("summary generation failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("api returned no choices")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// GetEmbedding 使用 Embedding 模型 (读取 VOLC_EMBED_MODEL_ID)
func (s *AIService) GetEmbedding(text string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// 预处理：去除换行符能提升向量质量
	text = strings.ReplaceAll(text, "\n", " ")

	safeText := truncateContent(text, 2000)

	resp, err := s.client.CreateEmbeddings(
		ctx,
		openai.EmbeddingRequest{
			Input: []string{safeText},
			Model: openai.EmbeddingModel(s.cfg.VolcEmbedModelID),
		},
	)

	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("embedding data is empty")
	}

	return resp.Data[0].Embedding, nil
}

func truncateContent(content string, limit int) string {
	if utf8.RuneCountInString(content) <= limit {
		return content
	}
	runes := []rune(content)
	return string(runes[:limit])
}
