package utils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitLogger 初始化全局 Logger
// env: "dev" 或 "prod"，根据 config.Config 传入
func InitLogger(env string) {
	var config zap.Config

	if env == "dev" {
		// 开发模式：控制台输出，人类可读格式，Debug 级别
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		// 生产模式：JSON 输出，Info 级别
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// 可以在这里配置输出路径，例如 config.OutputPaths = []string{"stdout", "./logs/app.log"}

	logger, err := config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	// 替换全局的 zap logger，这样在其他地方可以直接用 zap.L() 或 zap.S()
	zap.ReplaceGlobals(logger)
}
