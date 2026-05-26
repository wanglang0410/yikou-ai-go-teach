package parser

import (
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"testing"
)

func TestParseHtmlCode(t *testing.T) {
	codeContent := `随便写一段描述：
` + "```html\n" + `<!DOCTYPE html>
<html>
<head>
    <title>测试页面</title>
</head>
<body>
    <h1>Hello World!</h1>
</body>
</html>
` + "```\n" + `
随便写一段描述`

	result := ParseHtmlCode(codeContent)
	assert.NotNil(t, result)
}

func TestParseMultiFileCode(t *testing.T) {
	codeContent := `创建一个完整的网页：
` + "```html\n" + `<!DOCTYPE html>
<html>
<head>
    <title>多文件示例</title>
    <link rel="stylesheet" href="style.css">
</head>
<body>
    <h1>欢迎使用</h1>
    <script src="script.js"></script>
</body>
</html>
` + "```\n" + `
` + "```css\n" + `h1 {
    color: blue;
    text-align: center;
}
` + "```\n" + `
` + "```js\n" + `console.log('页面加载完成');
` + "```\n" + `
文件创建完成！`

	result := ParseMultiFileCode(codeContent)
	assert.NotNil(t, result)
}
