package handler

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"yikou-ai-go-teach/pkg/response"
)

type PingResponse response.BaseResponse[string]

// Ping
// @Summary   测试接口
// @Description 根据名字返回问候语
// @Accept    json
// @Produce   json
// @Success   200 {object} PingResponse
// @Router    /api/ping [get]
func Ping(ctx context.Context, c *app.RequestContext) {
	c.JSON(consts.StatusOK, response.NewSuccessResponse[string]("pong"))
}
