package agent

import (
	"context"
	"encoding/json"
	"github.com/cloudwego/eino/schema"
	"yikou-ai-go-teach/internal/ai/aimodel"
	"yikou-ai-go-teach/internal/ai/llm"
	"yikou-ai-go-teach/internal/ai/myprompt"
	"yikou-ai-go-teach/internal/store"
	"yikou-ai-go-teach/pkg/enum"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/cloudwego/eino/adk"
)

func NewCodeGenAgent(chatModel ChatModelWrapperAdaptor, codeGenType enum.CodeGenTypeEnum, memoryStore store.MemoryStore) *CodeGenAgent {
	baseAgent := NewBaseAgent(chatModel, memoryStore)
	return &CodeGenAgent{
		BaseAgent: baseAgent,
		agentType: codeGenType,
	}
}

func NewTestCodeGenAgent(chatModel *llm.ChatModelWrapper) *CodeGenAgent {
	baseAgent := NewBaseAgent(chatModel, nil)
	return &CodeGenAgent{
		BaseAgent: baseAgent,
		agentType: enum.HtmlCodeGen,
	}
}

type CodeGenAgent struct {
	*BaseAgent
	agentType enum.CodeGenTypeEnum
}

func (a *CodeGenAgent) getAdkAgent() *adk.ChatModelAgent {
	switch a.agentType {
	case enum.HtmlCodeGen:
		return a.newHtmlFileCodeGenAgent()
	case enum.MultiFileGen:
		return a.newMultiFileCodeGenAgent()
	default:
		return nil
	}
}

func (a *CodeGenAgent) GenerateHtmlCode(ctx context.Context, userMessage string) (*aimodel.HtmlCodeResponse, error) {
	chatTemplate, err := myprompt.NewHtmlChatTemplate()
	if err != nil {
		return nil, err
	}

	adkAgent := a.getAdkAgent()
	message, err := a.Generate(ctx, userMessage+
		`You must answer strictly in the following JSON format:
		{
		  "htmlCode": "your html code here",
		  "description": "description of the code"
		}
		IMPORTANT: You must answer ONLY with a valid JSON object, no markdown, no code blocks, no backticks.
		`,
		chatTemplate, adkAgent)
	if err != nil {
		return nil, err
	}
	var result aimodel.HtmlCodeResponse
	err = json.Unmarshal([]byte(message.Content), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *CodeGenAgent) GenerateMultiFileCode(ctx context.Context, userMessage string) (*aimodel.MultiFileCodeResponse, error) {
	chatTemplate, err := myprompt.NewMultiFileChatTemplate()
	if err != nil {
		return nil, err
	}

	adkAgent := a.getAdkAgent()
	message, err := a.Generate(ctx, userMessage+
		`You must answer strictly in the following JSON format:
		{
		  "htmlCode": "your html code here",
		  "description": "description of the code",
		  "cssCode": "your css code here",
		  "jsCode": "your javascript code here"
		}
		IMPORTANT: You must answer ONLY with a valid JSON object, no markdown, no code blocks, no backticks.
		`,
		chatTemplate, adkAgent)
	if err != nil {
		return nil, err
	}
	var result aimodel.MultiFileCodeResponse
	err = json.Unmarshal([]byte(message.Content), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *CodeGenAgent) GenerateHtmlCodeStream(ctx context.Context, userMessage string) (*schema.StreamReader[*schema.Message], error) {
	chatTemplate, err := myprompt.NewHtmlChatTemplate()
	if err != nil {
		return nil, err
	}
	adkAgent := a.getAdkAgent()
	return a.GenerateStream(ctx, userMessage, chatTemplate, adkAgent)
}

func (a *CodeGenAgent) GenerateMultiFileCodeStream(ctx context.Context, userMessage string) (*schema.StreamReader[*schema.Message], error) {
	chatTemplate, err := myprompt.NewMultiFileChatTemplate()
	if err != nil {
		return nil, err
	}

	adkAgent := a.getAdkAgent()
	return a.GenerateStream(ctx, userMessage, chatTemplate, adkAgent)
}

func (a *CodeGenAgent) newMultiFileCodeGenAgent() *adk.ChatModelAgent {
	if err := myprompt.LoadPrompts(); err != nil {
		logger.Errorf("加载prompts失败: %v", err)
		return nil
	}
	return a.NewAdkAgent(
		"AI 代码生成助手",
		"具有强大的代码生成能力",
		myprompt.GetMultiFilePrompt(),
	)
}

func (a *CodeGenAgent) newHtmlFileCodeGenAgent() *adk.ChatModelAgent {
	if err := myprompt.LoadPrompts(); err != nil {
		logger.Errorf("加载prompts失败: %v", err)
		return nil
	}
	return a.NewAdkAgent(
		"AI 代码生成助手",
		"具有强大的代码生成能力",
		myprompt.GetHtmlPrompt(),
	)
}
