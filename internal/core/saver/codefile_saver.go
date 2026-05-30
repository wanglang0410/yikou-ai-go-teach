package saver

import (
	"fmt"
	"github.com/sony/sonyflake"
	"os"
	"path/filepath"
	"strconv"
	"yikou-ai-go-teach/internal/ai/aimodel"
	"yikou-ai-go-teach/pkg/enum"
	"yikou-ai-go-teach/pkg/myfile"
)

// buildUniqueDir 构建唯一的目录名
// 目录名格式: {代码生成类型}_{唯一ID}
func buildUniqueDir(typeStr enum.CodeGenTypeEnum) (string, error) {
	// 生成雪花id
	var sf = sonyflake.NewSonyflake(sonyflake.Settings{
		MachineID: func() (uint16, error) { return 1, nil },
	})
	id, err := sf.NextID()
	if err != nil {
		return "", err
	}
	// 构建唯一目录名
	uniqueDirName := fmt.Sprintf("%s_%s", typeStr, strconv.FormatUint(id, 20))
	fileSaveDir, err := myfile.GetCodeOutputRoot()
	dirPath := filepath.Join(fileSaveDir, uniqueDirName)
	// 创建目录
	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return "", err
	}
	return dirPath, nil
}

// writeToFile 将内容写入文件并保存
func writeToFile(dirPath string, fileName string, content string) error {
	filePath := filepath.Join(dirPath, fileName)
	err := os.WriteFile(filePath, []byte(content), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

// SaveHtmlCode 保存 HTML 代码文件
func SaveHtmlCode(response aimodel.HtmlCodeResponse) (string, error) {
	dirPath, err := buildUniqueDir(enum.HtmlCodeGen)
	if err != nil {
		return "", err
	}
	fileName := "index.html"
	return dirPath, writeToFile(dirPath, fileName, response.HtmlCode)
}

// SaveMultiFileCode 保存多文件代码文件
func SaveMultiFileCode(response aimodel.MultiFileCodeResponse) (string, error) {
	dirPath, err := buildUniqueDir(enum.MultiFileGen)
	if err != nil {
		return "", err
	}
	// 保存 HTML 文件
	err = writeToFile(dirPath, "index.html", response.HtmlCode)
	if err != nil {
		return "", err
	}
	// 保存 JS 文件
	err = writeToFile(dirPath, "script.js", response.JsCode)
	if err != nil {
		return "", err
	}
	// 保存 CSS 文件
	err = writeToFile(dirPath, "style.css", response.CssCode)
	if err != nil {
		return "", err
	}
	return dirPath, nil
}

type CodeFileSaver[T any] interface {
	getCodeType() enum.CodeGenTypeEnum
	saveFiles(response T, baseDir string) error
	validateInput(response T) error
}

type CodeFileSaverTemplate[T any] struct {
	CodeFileSaver[T]
}

func (d *CodeFileSaverTemplate[T]) saveCode(response T, appId int64) (string, error) {
	err := d.validateInput(response)
	if err != nil {
		return "", err
	}
	dirPath, err := d.buildUniqueDir(appId)
	if err != nil {
		return "", err
	}
	return dirPath, d.saveFiles(response, dirPath)
}

// buildUniqueDir 构建唯一的目录名
// 目录名格式: {代码生成类型}_{唯一ID}
func (d *CodeFileSaverTemplate[T]) buildUniqueDir(appId int64) (string, error) {
	if appId == 0 {
		return "", fmt.Errorf("应用id不能为空")
	}
	//构建唯一目录名
	fileSaveDir, err := myfile.GetCodeOutputRoot()
	uniqueDirName := fmt.Sprintf("%s_%s", d.getCodeType(), strconv.FormatUint(uint64(appId), 20))
	dirPath := filepath.Join(fileSaveDir, uniqueDirName)
	// 创建目录
	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return "", err
	}
	return dirPath, nil
}

// writeToFile 将内容写入文件并保存
func (d *CodeFileSaverTemplate[T]) writeToFile(dirPath string, fileName string, content string) error {
	filePath := filepath.Join(dirPath, fileName)
	err := os.WriteFile(filePath, []byte(content), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

type HtmlCodeFileSaverTemplate struct {
	CodeFileSaverTemplate[*aimodel.HtmlCodeResponse]
}

func NewHtmlCodeFileSaverTemplate() *HtmlCodeFileSaverTemplate {
	t := &HtmlCodeFileSaverTemplate{}
	t.CodeFileSaverTemplate.CodeFileSaver = t
	return t
}

func (h *HtmlCodeFileSaverTemplate) getCodeType() enum.CodeGenTypeEnum {
	return enum.HtmlCodeGen
}

func (h *HtmlCodeFileSaverTemplate) saveFiles(response *aimodel.HtmlCodeResponse, baseDir string) error {
	fileName := "index.html"
	return h.writeToFile(baseDir, fileName, response.HtmlCode)
}

func (h *HtmlCodeFileSaverTemplate) validateInput(response *aimodel.HtmlCodeResponse) error {
	if response == nil {
		return fmt.Errorf("代码结果为空")
	}
	if response.HtmlCode == "" {
		return fmt.Errorf("HTML 代码为空")
	}
	return nil
}

type MultiFileCodeFileSaverTemplate struct {
	CodeFileSaverTemplate[*aimodel.MultiFileCodeResponse]
}

func NewMultiFileCodeFileSaverTemplate() *MultiFileCodeFileSaverTemplate {
	t := &MultiFileCodeFileSaverTemplate{}
	t.CodeFileSaverTemplate.CodeFileSaver = t
	return t
}

func (m *MultiFileCodeFileSaverTemplate) getCodeType() enum.CodeGenTypeEnum {
	return enum.MultiFileGen
}

func (m *MultiFileCodeFileSaverTemplate) saveFiles(response *aimodel.MultiFileCodeResponse, baseDir string) error {
	// 保存 HTML 文件
	err := m.writeToFile(baseDir, "index.html", response.HtmlCode)
	if err != nil {
		return err
	}
	// 保存 JS 文件
	err = m.writeToFile(baseDir, "script.js", response.JsCode)
	if err != nil {
		return err
	}
	// 保存 CSS 文件
	err = m.writeToFile(baseDir, "style.css", response.CssCode)
	if err != nil {
		return err
	}
	return nil
}

func (m *MultiFileCodeFileSaverTemplate) validateInput(response *aimodel.MultiFileCodeResponse) error {
	if response == nil {
		return fmt.Errorf("代码结果为空")
	}
	if response.HtmlCode == "" {
		return fmt.Errorf("HTML 代码为空")
	}
	if response.JsCode == "" {
		return fmt.Errorf("JS 代码为空")
	}
	if response.CssCode == "" {
		return fmt.Errorf("CSS 代码为空")
	}
	return nil
}

type CodeFileSaverExecutor struct {
	htmlCodeFileSaver      *HtmlCodeFileSaverTemplate
	multiFileCodeFileSaver *MultiFileCodeFileSaverTemplate
}

func NewCodeFileSaverExecutor() *CodeFileSaverExecutor {
	return &CodeFileSaverExecutor{
		htmlCodeFileSaver:      NewHtmlCodeFileSaverTemplate(),
		multiFileCodeFileSaver: NewMultiFileCodeFileSaverTemplate(),
	}
}

func (e *CodeFileSaverExecutor) ExecuteSaver(content interface{}, saveType enum.CodeGenTypeEnum, appId int64) (string, error) {
	switch saveType {
	case enum.HtmlCodeGen:
		return e.htmlCodeFileSaver.saveCode(content.(*aimodel.HtmlCodeResponse), appId)
	case enum.MultiFileGen:
		return e.multiFileCodeFileSaver.saveCode(content.(*aimodel.MultiFileCodeResponse), appId)
	default:
		return "", fmt.Errorf("不支持的代码文件类型: %s", saveType)
	}
}
