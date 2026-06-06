package agent

import (
	"context"
	"github.com/bytedance/gopkg/util/logger"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"testing"
	"yikou-ai-go-teach/config"
	"yikou-ai-go-teach/internal/ai/llm"
	"yikou-ai-go-teach/internal/dal"
	"yikou-ai-go-teach/internal/logic"
	"yikou-ai-go-teach/pkg/enum"
)

func TestCodeGenAgent_GenerateHtmlCode(t *testing.T) {
	config.SetEnvFlag("local")
	// 解析命令行参数
	initConfig := config.InitConfig()
	chatModel := llm.NewChatModel(initConfig)
	codeGenAgent := NewCodeGenAgent(chatModel, enum.HtmlCodeGen, nil)
	code, err := codeGenAgent.GenerateHtmlCode(context.Background(), "做个mysql学习知识图")
	if err != nil {
		return
	}
	assert.NotNil(t, code)
}

func TestCodeGenAgent_GenerateMultiFileCode(t *testing.T) {
	config.SetEnvFlag("local")
	initConfig := config.InitConfig()
	chatModel := llm.NewChatModel(initConfig)
	db := dal.InitDB(initConfig)
	redisClient := dal.InitRedis(initConfig)
	chatHistoryService := logic.NewChatHistoryService(db)
	genAgentFactory := NewCodeGenAgentFactory(chatModel, redisClient, chatHistoryService)
	codeGenAgent, err := genAgentFactory.GetCodeGenAgent(1, enum.MultiFileGen)
	if err != nil {
		logger.Errorf("%v", err)
		return
	}
	code, err := codeGenAgent.GenerateMultiFileCode(context.Background(), "做个留言版")
	if err != nil {
		logger.Errorf("%v", err)
		return
	}
	assert.NotNil(t, code)
}
