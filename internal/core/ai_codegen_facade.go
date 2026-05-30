package core

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/schema"
	"io"
	"strings"
	"yikou-ai-go-teach/internal/ai"
	"yikou-ai-go-teach/internal/core/parser"
	"yikou-ai-go-teach/internal/core/saver"
	"yikou-ai-go-teach/pkg/enum"

	"github.com/bytedance/gopkg/util/logger"
)

type YiKouAiCodegenFacade struct {
	codegenService        ai.IYiKouAiCodegenService
	codeParserExecutor    *parser.CodeParserExecutor
	codeFileSaverExecutor *saver.CodeFileSaverExecutor
}

func NewYiKouAiCodegenFacade(codegenService ai.IYiKouAiCodegenService,
	codeParserExecutor *parser.CodeParserExecutor,
	codeFileSaverExecutor *saver.CodeFileSaverExecutor) *YiKouAiCodegenFacade {
	return &YiKouAiCodegenFacade{
		codegenService:        codegenService,
		codeParserExecutor:    codeParserExecutor,
		codeFileSaverExecutor: codeFileSaverExecutor,
	}
}

func (y *YiKouAiCodegenFacade) GenHtmlCodeAndSave(ctx context.Context, userMessage string) error {
	resp, err := y.codegenService.GenerateHtmlCode(ctx, userMessage)
	if err != nil {
		return err
	}
	dirPath, err := saver.SaveHtmlCode(*resp)
	if err != nil {
		return err
	}
	logger.Info("HTML代码已保存到目录: %s", dirPath)
	return nil
}

func (y *YiKouAiCodegenFacade) GenMultiFileCodeAndSave(ctx context.Context, userMessage string) error {
	resp, err := y.codegenService.GenerateMultiFileCode(ctx, userMessage)
	if err != nil {
		return err
	}
	dirPath, err := saver.SaveMultiFileCode(*resp)
	if err != nil {
		return err
	}
	logger.Info("多文件代码已保存到目录: %s", dirPath)
	return nil
}

func (y *YiKouAiCodegenFacade) GenCodeAndSave(ctx context.Context, userMessage string, typeStr enum.CodeGenTypeEnum) error {
	switch typeStr {
	case enum.HtmlCodeGen:
		return y.GenHtmlCodeAndSave(ctx, userMessage)
	case enum.MultiFileGen:
		return y.GenMultiFileCodeAndSave(ctx, userMessage)
	default:
		return fmt.Errorf("不支持的代码生成类型: %s", typeStr)
	}
}

func (y *YiKouAiCodegenFacade) GenHtmlCodeStreamAndSave(ctx context.Context, userMessage string) error {
	streamResp, err := y.codegenService.GenerateHtmlCodeStream(ctx, userMessage)
	if err != nil {
		return err
	}
	defer streamResp.Close()

	var builder strings.Builder
	for {
		chunk, err := streamResp.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		builder.WriteString(chunk.Content)
	}
	result := parser.ParseHtmlCode(builder.String())
	dirPath, err := saver.SaveHtmlCode(*result)
	if err != nil {
		return err
	}
	logger.Info("HTML代码已保存到目录: %s", dirPath)
	return nil
}

func (y *YiKouAiCodegenFacade) GenMultiFileCodeStreamAndSave(ctx context.Context, userMessage string) error {
	streamResp, err := y.codegenService.GenerateMultiFileCodeStream(ctx, userMessage)
	if err != nil {
		return err
	}
	defer streamResp.Close()

	var builder strings.Builder
	for {
		chunk, err := streamResp.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		builder.WriteString(chunk.Content)
	}
	result := parser.ParseMultiFileCode(builder.String())
	dirPath, err := saver.SaveMultiFileCode(*result)
	if err != nil {
		return err
	}
	logger.Info("多文件代码已保存到目录: %s", dirPath)
	return nil
}

func (y *YiKouAiCodegenFacade) processCodeStream(respStream *schema.StreamReader[*schema.Message], typeStr enum.CodeGenTypeEnum, appId int64) (*schema.StreamReader[*schema.Message], error) {
	// 先复制流，一个用于处理，一个返回给上游
	streams := respStream.Copy(2)
	processingStream := streams[0]
	returnStream := streams[1]

	// 在 goroutine 中处理流数据，不阻塞返回
	go func() {
		var builder strings.Builder
		defer processingStream.Close()

		for {
			chunk, err := processingStream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return
			}
			builder.WriteString(chunk.Content)
		}

		parsedResp, err := y.codeParserExecutor.ExecuteParser(builder.String(), typeStr)
		if err != nil {
			return
		}
		dirPath, err := y.codeFileSaverExecutor.ExecuteSaver(parsedResp, typeStr, appId)
		if err != nil {
			return
		}
		logger.Info("代码已保存到目录: %s", dirPath)
	}()

	return returnStream, nil
}

func (y *YiKouAiCodegenFacade) GenCodeStreamAndSave(ctx context.Context, userMessage string, typeStr enum.CodeGenTypeEnum, appId int64) (*schema.StreamReader[*schema.Message], error) {
	switch typeStr {
	case enum.HtmlCodeGen:
		streamResp, err := y.codegenService.GenerateHtmlCodeStream(ctx, userMessage)
		if err != nil {
			return nil, err
		}
		return y.processCodeStream(streamResp, typeStr, appId)
	case enum.MultiFileGen:
		streamResp, err := y.codegenService.GenerateMultiFileCodeStream(ctx, userMessage)
		if err != nil {
			return nil, err
		}
		return y.processCodeStream(streamResp, typeStr, appId)
	default:
		return nil, fmt.Errorf("不支持的代码生成类型: %s", typeStr)
	}
}
