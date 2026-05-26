package agent

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"testing"
	"yikou-ai-go-teach/config"
	"yikou-ai-go-teach/internal/ai/llm"
	"yikou-ai-go-teach/pkg/enum"
)

func TestCodeGenAgent_GenerateHtmlCode(t *testing.T) {
	config.SetEnvFlag("local")
	// 解析命令行参数
	initConfig := config.InitConfig()
	chatModel := llm.NewChatModel(initConfig)
	codeGenAgent := NewCodeGenAgent(chatModel, enum.HtmlCodeGen)
	code, err := codeGenAgent.GenerateHtmlCode(context.Background(), "做个mysql学习知识图")
	if err != nil {
		return
	}
	assert.NotNil(t, code)
}

func TestCodeGenAgent_GenerateMultiFileCode(t *testing.T) {
	initConfig := config.InitConfig()
	chatModel := llm.NewChatModel(initConfig)
	codeGenAgent := NewCodeGenAgent(chatModel, enum.MultiFileGen)
	code, err := codeGenAgent.GenerateMultiFileCode(context.Background(), "做个留言版")
	if err != nil {
		return
	}
	assert.NotNil(t, code)
}
