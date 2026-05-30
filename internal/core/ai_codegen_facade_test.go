package core

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"strings"
	"testing"
	"yikou-ai-go-teach/config"
	"yikou-ai-go-teach/internal/ai/agent"
	"yikou-ai-go-teach/internal/ai/llm"
	"yikou-ai-go-teach/internal/core/parser"
	"yikou-ai-go-teach/internal/core/saver"
	"yikou-ai-go-teach/pkg/enum"
)

func TestYiKouAiCodegenFacade_GenCodeAndSave(t *testing.T) {
	config.SetEnvFlag("local")
	// 解析命令行参数
	initConfig := config.InitConfig()
	chatModel := llm.NewChatModel(initConfig)
	codeGenAgent := agent.NewCodeGenAgent(chatModel, enum.MultiFileGen)
	parserExecutor := parser.NewCodeParserExecutor()
	fileSaverExecutor := saver.NewCodeFileSaverExecutor()
	aiCodegenFacade := NewYiKouAiCodegenFacade(codeGenAgent, parserExecutor, fileSaverExecutor)
	err := aiCodegenFacade.GenCodeAndSave(context.Background(), "帮我生成一个日常记录网站", enum.MultiFileGen)
	if err != nil {
		panic(err)
	}
}

func TestYiKouAiCodegenFacade_GenCodeStreamAndSave(t *testing.T) {
	config.SetEnvFlag("local")
	// 解析命令行参数
	initConfig := config.InitConfig()
	chatModel := llm.NewChatModel(initConfig)
	codeGenAgent := agent.NewCodeGenAgent(chatModel, enum.MultiFileGen)
	parserExecutor := parser.NewCodeParserExecutor()
	fileSaverExecutor := saver.NewCodeFileSaverExecutor()
	aiCodegenFacade := NewYiKouAiCodegenFacade(codeGenAgent, parserExecutor, fileSaverExecutor)
	resp, err := aiCodegenFacade.GenCodeStreamAndSave(context.Background(), "帮我生成一个日常记录网站", enum.MultiFileGen, 1)
	if err != nil {
		panic(err)
	}
	var builder strings.Builder
	for {
		message, err := resp.Recv()
		if err != nil {
			break
		}
		builder.WriteString(message.Content)
	}
	assert.NotNil(t, builder.String())
}
