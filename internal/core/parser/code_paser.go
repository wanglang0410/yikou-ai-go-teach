package parser

import (
	"fmt"
	"regexp"
	"strings"
	"yikou-ai-go-teach/internal/ai/aimodel"
	"yikou-ai-go-teach/pkg/enum"
)

type Parser[T any] interface {
	Parse(content string) (T, error)
}

type HtmlCodeParser struct{}

func NewHtmlCodeParser() *HtmlCodeParser {
	return &HtmlCodeParser{}
}

func (p *HtmlCodeParser) Parse(content string) (*aimodel.HtmlCodeResponse, error) {
	result := &aimodel.HtmlCodeResponse{}
	matches := htmlCodeRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		result.HtmlCode = strings.TrimSpace(matches[1])
	}
	return result, nil
}

type MultiFileCodeParser struct{}

func NewMultiFileCodeParser() *MultiFileCodeParser {
	return &MultiFileCodeParser{}
}

func (p *MultiFileCodeParser) Parse(content string) (*aimodel.MultiFileCodeResponse, error) {
	result := &aimodel.MultiFileCodeResponse{}
	htmlMatches := htmlCodeRegex.FindStringSubmatch(content)
	if len(htmlMatches) >= 2 {
		result.HtmlCode = strings.TrimSpace(htmlMatches[1])
	}
	cssMatches := cssCodeRegex.FindStringSubmatch(content)
	if len(cssMatches) >= 2 {
		result.CssCode = strings.TrimSpace(cssMatches[1])
	}
	jsMatches := jsCodeRegex.FindStringSubmatch(content)
	if len(jsMatches) >= 3 {
		result.JsCode = strings.TrimSpace(jsMatches[2])
	}
	return result, nil
}

type CodeParserExecutor struct {
	htmlCodeParser      *HtmlCodeParser
	multiFileCodeParser *MultiFileCodeParser
}

func NewCodeParserExecutor() *CodeParserExecutor {
	return &CodeParserExecutor{
		htmlCodeParser:      NewHtmlCodeParser(),
		multiFileCodeParser: NewMultiFileCodeParser(),
	}
}

func (e *CodeParserExecutor) ExecuteParser(content string, parserType enum.CodeGenTypeEnum) (interface{}, error) {
	switch parserType {
	case enum.HtmlCodeGen:
		return e.htmlCodeParser.Parse(content)
	case enum.MultiFileGen:
		return e.multiFileCodeParser.Parse(content)
	default:
		return nil, fmt.Errorf("不支持的解析类型: %s", parserType)
	}
}

var (
	htmlCodeRegex = regexp.MustCompile("(?i)```html\\s*\\n([\\s\\S]*?)```")
	cssCodeRegex  = regexp.MustCompile("(?i)```css\\s*\\n([\\s\\S]*?)```")
	jsCodeRegex   = regexp.MustCompile("(?i)```(?:js|javascript)\\s*\\n([\\s\\S]*?)```")
)

func ParseHtmlCode(codeContent string) *aimodel.HtmlCodeResponse {
	result := &aimodel.HtmlCodeResponse{}

	htmlCode := extractHtmlCode(codeContent)
	if htmlCode != "" {
		result.HtmlCode = strings.TrimSpace(htmlCode)
	} else {
		result.HtmlCode = strings.TrimSpace(codeContent)
	}

	return result
}

func ParseMultiFileCode(codeContent string) *aimodel.MultiFileCodeResponse {
	result := &aimodel.MultiFileCodeResponse{}

	htmlCode := extractCodeByPattern(codeContent, htmlCodeRegex)
	cssCode := extractCodeByPattern(codeContent, cssCodeRegex)
	jsCode := extractCodeByPattern(codeContent, jsCodeRegex)

	if htmlCode != "" {
		result.HtmlCode = strings.TrimSpace(htmlCode)
	}

	if cssCode != "" {
		result.CssCode = strings.TrimSpace(cssCode)
	}

	if jsCode != "" {
		result.JsCode = strings.TrimSpace(jsCode)
	}

	return result
}

func extractHtmlCode(content string) string {
	matches := htmlCodeRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractCodeByPattern(content string, pattern *regexp.Regexp) string {
	matches := pattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
