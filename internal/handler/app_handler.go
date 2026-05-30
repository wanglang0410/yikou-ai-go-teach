package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bytedance/gopkg/util/logger"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
	"io"
	"strconv"
	"strings"
	"yikou-ai-go-teach/internal/api"
	"yikou-ai-go-teach/internal/dal/model"
	"yikou-ai-go-teach/internal/dal/vo"
	"yikou-ai-go-teach/internal/service"
	"yikou-ai-go-teach/pkg/errorutil"
	"yikou-ai-go-teach/pkg/request"
	"yikou-ai-go-teach/pkg/response"
)

type StreamContext struct {
	CancelFunc context.CancelFunc
	Ctx        context.Context
}

type AppHandler struct {
	appService  service.IAppService
	userService service.IUserService
}

func NewAppHandler(
	appService service.IAppService,
	userService service.IUserService,
) *AppHandler {
	return &AppHandler{
		appService:  appService,
		userService: userService,
	}
}

// ChatToGenCode 应用聊天生成代码（流式）
// @Summary 应用聊天生成代码（流式）
// @Description 应用聊天生成代码（流式）
// @Tags 应用模块
// @Accept json
// @Produce json
// @Param appId  query string true "应用ID"
// @Param message query string true "消息"
// @Router /app/chat/gen/code [get]
func (a *AppHandler) ChatToGenCode(ctx context.Context, c *app.RequestContext) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	appIdStr := c.Query("appId")
	w := sse.NewWriter(c)
	lastEventID := sse.GetLastEventID(&c.Request)

	if appIdStr == "" {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](errorutil.ParamsError.WithMessage("应用ID不能为空")))
		return
	}
	message := c.Query("message")
	if message == "" {
		_ = w.WriteEvent(lastEventID, "error", []byte("消息不能为空"))
		_ = w.WriteEvent(lastEventID, "done", []byte{1})
		return
	}
	userVo, err := a.userService.GetLoginUserVo(ctx, c)
	if err != nil {
		_ = w.WriteEvent(lastEventID, "error", []byte(fmt.Sprintf("%v", err)))
		_ = w.WriteEvent(lastEventID, "done", []byte{1})
		return
	}
	appId, err := strconv.ParseInt(appIdStr, 10, 64)
	if err != nil {
		_ = w.WriteEvent(lastEventID, "error", []byte(fmt.Sprintf("%v", err)))
		_ = w.WriteEvent(lastEventID, "done", []byte{1})
		return
	}

	// 获取流数据
	streamResp, err := a.appService.ChatToGenCode(ctx, appId, message, &userVo)
	if err != nil {
		_ = w.WriteEvent(lastEventID, "error", []byte(fmt.Sprintf("%v", err)))
		_ = w.WriteEvent(lastEventID, "done", []byte{1})
		return
	}
	defer streamResp.Close()

	var aiResponseBuilder strings.Builder
	for {
		select {
		case <-ctx.Done():
			logger.Info("连接中断")
			_ = w.WriteEvent(lastEventID, "done", []byte{1})
			return
		default:
		}

		chunk, err := streamResp.Recv()
		if err == io.EOF || errors.Is(err, context.Canceled) {
			break
		}
		if err != nil {
			_ = w.WriteEvent(lastEventID, "error", []byte(fmt.Sprintf("%v", err)))
			_ = w.WriteEvent(lastEventID, "done", []byte{1})
			return
		}
		aiResponseBuilder.WriteString(chunk.Content)

		wrapper := &map[string]string{
			"d": chunk.Content,
		}
		data, err := json.Marshal(wrapper)
		if err != nil {
			logger.Errorf("序列化数据失败: %v\n", err)
			continue
		}

		err = w.WriteEvent(lastEventID, "message", data)
		if err != nil {
			_ = w.WriteEvent(lastEventID, "error", []byte(fmt.Sprintf("%v", err)))
			_ = w.WriteEvent(lastEventID, "done", []byte{1})
			return
		}
	}

	_ = w.WriteEvent(lastEventID, "done", []byte{1})
}

// AddApp 新增应用
// @Summary 新增应用
// @Description 新增应用
// @Tags 应用模块
// @Accept json
// @Produce json
// @Param req body api.YiKouAppAddRequest true "新增应用请求"
// @Success 200 {object} api.YiKouAppAddResponse "应用ID"
// @Router /app/add [post]
func (a *AppHandler) AddApp(ctx context.Context, c *app.RequestContext) {
	req := &api.YiKouAppAddRequest{}
	err := c.BindAndValidate(req)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	userVo, err := a.userService.GetLoginUserVo(ctx, c)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	appId, err := a.appService.AddApp(ctx, req, userVo.ID)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	c.JSON(consts.StatusOK, response.NewSuccessResponse[string](strconv.Itoa(int(appId))))
}

// UpdateApp 更新应用
// @Summary 更新应用
// @Description 更新应用
// @Tags 应用模块
// @Accept json
// @Produce json
// @Param req body api.YiKouAppUpdateRequest true "更新应用请求"
// @Success 200 {object} api.YiKouAppUpdateResponse "更新结果"
// @Router /app/update [post]
func (a *AppHandler) UpdateApp(ctx context.Context, c *app.RequestContext) {
	req := &api.YiKouAppUpdateRequest{}
	err := c.BindAndValidate(req)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	userVo, err := a.userService.GetLoginUserVo(ctx, c)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	success, err := a.appService.UpdateApp(ctx, req, userVo.ID)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	c.JSON(consts.StatusOK, response.NewSuccessResponse[bool](success))
}

// DeleteApp 删除应用
// @Summary 删除应用
// @Description 删除应用
// @Tags 应用模块
// @Accept json
// @Produce json
// @Param req body request.DeleteRequest true "删除应用请求"
// @Success 200 {object} api.YiKouAppDeleteResponse "删除结果"
// @Router /app/delete [post]
func (a *AppHandler) DeleteApp(ctx context.Context, c *app.RequestContext) {
	req := &request.DeleteRequest{}
	err := c.BindAndValidate(req)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	userVo, err := a.userService.GetLoginUserVo(ctx, c)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	success, err := a.appService.DeleteApp(ctx, int64(req.Id), userVo.ID)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	c.JSON(consts.StatusOK, response.NewSuccessResponse[bool](success))
}

// GetAppVo 根据ID获取应用VO
// @Summary 根据ID获取应用VO
// @Description 根据ID获取应用VO
// @Tags 应用模块
// @Accept json
// @Produce json
// @Param id query int true "应用ID"
// @Success 200 {object} api.YiKouAppGetVoResponse "应用VO信息"
// @Router /app/get/vo [get]
func (a *AppHandler) GetAppVo(ctx context.Context, c *app.RequestContext) {
	id := c.Query("id")
	if id == "" {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](errorutil.ParamsError))
		return
	}
	idInt64, _ := strconv.ParseInt(id, 10, 64)
	userVo, err := a.userService.GetLoginUserVo(ctx, c)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	appVo, err := a.appService.GetAppVo(ctx, idInt64, userVo.ID)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	c.JSON(consts.StatusOK, response.NewSuccessResponse[vo.AppVo](appVo))
}

// ListMyApp 分页获取我的应用列表
// @Summary 分页获取我的应用列表
// @Description 分页获取我的应用列表
// @Tags 应用模块
// @Accept json
// @Produce json
// @Param req body api.YiKouAppMyListRequest true "分页查询请求"
// @Success 200 {object} api.YiKouAppMyListResponse "分页应用VO列表"
// @Router /application/list/my [post]
func (a *AppHandler) ListMyApp(ctx context.Context, c *app.RequestContext) {
	req := &api.YiKouAppMyListRequest{}
	err := c.BindAndValidate(req)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	userVo, err := a.userService.GetLoginUserVo(ctx, c)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	pageResponse, err := a.appService.ListMyApp(ctx, req, userVo.ID)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	c.JSON(consts.StatusOK, response.NewSuccessResponse[*response.PageResponse[vo.AppVo]](pageResponse))
}

// ListGoodApp 分页获取精选应用列表
// @Summary 分页获取精选应用列表
// @Description 分页获取精选应用列表
// @Tags 应用模块
// @Accept json
// @Produce json
// @Param req body api.YiKouAppFeaturedListRequest true "分页查询请求"
// @Success 200 {object} api.YiKouAppFeaturedListResponse "分页应用VO列表"
// @Router /app/good/list/page/vo [post]
func (a *AppHandler) ListGoodApp(ctx context.Context, c *app.RequestContext) {
	req := &api.YiKouAppFeaturedListRequest{}
	err := c.BindAndValidate(req)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	pageResponse, err := a.appService.ListGoodApp(ctx, req)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	c.JSON(consts.StatusOK, response.NewSuccessResponse[*response.PageResponse[vo.AppVo]](pageResponse))
}

// AdminUpdateApp 管理员更新应用
// @Summary 管理员更新应用
// @Description 管理员更新应用
// @Tags 应用模块
// @Accept json
// @Produce json
// @Param req body api.YiKouAppAdminUpdateRequest true "更新应用请求"
// @Success 200 {object} api.YiKouAppAdminUpdateResponse "更新结果"
// @Router /app/admin/update [post]
func (a *AppHandler) AdminUpdateApp(ctx context.Context, c *app.RequestContext) {
	req := &api.YiKouAppAdminUpdateRequest{}
	err := c.BindAndValidate(req)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	success, err := a.appService.AdminUpdateApp(ctx, req)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	c.JSON(consts.StatusOK, response.NewSuccessResponse[bool](success))
}

// AdminDeleteApp 管理员删除应用
// @Summary 管理员删除应用
// @Description 管理员删除应用
// @Tags 应用模块
// @Accept json
// @Produce json
// @Param req body request.DeleteRequest true "删除应用请求"
// @Success 200 {object} api.YiKouAppAdminDeleteResponse "删除结果"
// @Router /app/admin/delete [post]
func (a *AppHandler) AdminDeleteApp(ctx context.Context, c *app.RequestContext) {
	req := &request.DeleteRequest{}
	err := c.BindAndValidate(req)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	success, err := a.appService.AdminDeleteApp(ctx, int64(req.Id))
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	c.JSON(consts.StatusOK, response.NewSuccessResponse[bool](success))
}

// AdminGetAppVo 管理员根据ID获取应用VO
// @Summary 管理员根据ID获取应用VO
// @Description 管理员根据ID获取应用VO
// @Tags 应用模块
// @Accept json
// @Produce json
// @Param id query int true "应用ID"
// @Success 200 {object} api.YiKouAppAdminGetResponse "应用VO信息"
// @Router /app/admin/get/vo [get]
func (a *AppHandler) AdminGetAppVo(ctx context.Context, c *app.RequestContext) {
	id := c.Query("id")
	if id == "" {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](errorutil.ParamsError))
		return
	}
	idInt64, _ := strconv.ParseInt(id, 10, 64)
	appVo, err := a.appService.AdminGetAppVo(ctx, idInt64)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	c.JSON(consts.StatusOK, response.NewSuccessResponse[vo.AppVo](appVo))
}

// AdminListApp 管理员分页获取应用列表
// @Summary 管理员分页获取应用列表
// @Description 管理员分页获取应用列表
// @Tags 应用模块
// @Accept json
// @Produce json
// @Param req body api.YiKouAppAdminListRequest true "分页查询请求"
// @Success 200 {object} api.YiKouAppAdminListResponse "分页应用列表"
// @Router /app/admin/list/page/vo [post]
func (a *AppHandler) AdminListApp(ctx context.Context, c *app.RequestContext) {
	req := &api.YiKouAppAdminListRequest{}
	err := c.BindAndValidate(req)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	pageResponse, err := a.appService.AdminListApp(ctx, req)
	if err != nil {
		c.JSON(consts.StatusOK, response.NewErrorResponse[any](err))
		return
	}
	c.JSON(consts.StatusOK, response.NewSuccessResponse[*response.PageResponse[*model.App]](pageResponse))
}
