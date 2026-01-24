package ai

import (
	"context"
	"note/config"
	"strings"

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
	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			// 关键：这里使用的是配置里的 Endpoint ID
			Model: s.cfg.VolcChatModelID,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "你是一个笔记助手，请为以下内容生成一个15字以内的标题，不要包含引号：",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: content,
				},
			},
			Temperature: 0.7,
		},
	)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// GenerateSummary 调用 AI 生成摘要
func (s *AIService) GenerateSummary(content string) (string, error) {
	resp, err := s.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: s.cfg.VolcChatModelID,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "请为以下笔记生成一段50字以内的简短摘要。",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: content,
				},
			},
		},
	)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// GetEmbedding 使用 Embedding 模型 (读取 VOLC_EMBED_MODEL_ID)
func (s *AIService) GetEmbedding(text string) ([]float32, error) {
	// 预处理：去除换行符能提升向量质量
	text = strings.ReplaceAll(text, "\n", " ")

	resp, err := s.client.CreateEmbeddings(
		context.Background(),
		openai.EmbeddingRequest{
			Input: []string{text},
			// 关键：这里使用的是配置里的 Endpoint ID
			Model: openai.EmbeddingModel(s.cfg.VolcEmbedModelID),
		},
	)
	if err != nil {
		return nil, err
	}

	return resp.Data[0].Embedding, nil
}
