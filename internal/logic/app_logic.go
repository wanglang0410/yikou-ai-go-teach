package logic

import (
	"context"
	"github.com/cloudwego/eino/schema"
	"strconv"
	"yikou-ai-go-teach/internal/api"
	"yikou-ai-go-teach/internal/core"
	"yikou-ai-go-teach/internal/dal/model"
	"yikou-ai-go-teach/internal/dal/query"
	"yikou-ai-go-teach/internal/dal/vo"
	"yikou-ai-go-teach/internal/service"
	"yikou-ai-go-teach/pkg/enum"
	"yikou-ai-go-teach/pkg/errorutil"
	"yikou-ai-go-teach/pkg/response"
	"yikou-ai-go-teach/pkg/snowflake"

	"github.com/bytedance/gopkg/util/logger"
	"gorm.io/gorm"
)

func NewAppService(
	aiCodeGenFacade *core.YiKouAiCodegenFacade,
	userService service.IUserService,
	chatHistoryService service.IChatHistoryService,
	db *gorm.DB,
) *AppService {
	return &AppService{
		aiCodeGenFacade:    aiCodeGenFacade,
		userService:        userService,
		chatHistoryService: chatHistoryService,
		db:                 db,
	}
}

type AppService struct {
	aiCodeGenFacade    *core.YiKouAiCodegenFacade
	userService        service.IUserService
	chatHistoryService service.IChatHistoryService
	db                 *gorm.DB
}

func (s *AppService) ChatToGenCode(ctx context.Context, appId int64, message string, loginUser *vo.UserVo) (*schema.StreamReader[*schema.Message], error) {
	// 1. 校验参数
	if message == "" {
		return nil, errorutil.ParamsError.WithMessage("消息不能为空")
	}
	if appId == 0 || appId < 0 {
		return nil, errorutil.ParamsError.WithMessage("应用ID不能为空")
	}
	// 2. 校验应用是否存在
	app, err := query.Use(s.db).App.Where(query.App.ID.Eq(appId), query.App.IsDelete.Eq(0)).First()
	if err != nil {
		return nil, err
	}
	// 3. 校验用户是否有权限使用该应用
	if app.UserID != loginUser.ID {
		return nil, errorutil.NotAuthError.WithMessage("无权使用该应用")
	}
	// 4. 获取代码生成类型
	if enum.CodeGenTypeTextMap[enum.CodeGenTypeEnum(app.CodeGenType)] == "" {
		return nil, errorutil.ParamsError.WithMessage("应用代码生成类型不支持")
	}
	// 5. 将用户消息保存到对话记录
	err = s.chatHistoryService.AddChatMessage(ctx, appId, message, enum.UserMessageType, loginUser.ID)
	if err != nil {
		logger.Errorf("保存对话历史失败: %v\n", err)
	}
	// 6. 调用代码生成服务
	return s.aiCodeGenFacade.GenCodeStreamAndSave(ctx, message, enum.CodeGenTypeEnum(app.CodeGenType), appId)
}

func (s *AppService) AddApp(ctx context.Context, req *api.YiKouAppAddRequest, userId int64) (int64, error) {
	if req.InitPrompt == "" {
		return 0, errorutil.ParamsError.WithMessage("初始化prompt不能为空")
	}

	appName := req.InitPrompt
	count := 0
	for i := range appName {
		if count >= 12 {
			appName = appName[:i]
			break
		}
		count++
	}

	appId, err := snowflake.GenerateSnowFlakeId()
	if err != nil {
		return 0, err
	}

	newApp := &model.App{
		ID:          appId,
		AppName:     appName,
		InitPrompt:  req.InitPrompt,
		UserID:      userId,
		CodeGenType: string(enum.HtmlCodeGen),
		Priority:    0,
	}
	err = query.Use(s.db).App.
		Select(query.App.ID, query.App.AppName, query.App.InitPrompt, query.App.UserID, query.App.Priority, query.App.CodeGenType).
		Create(newApp)
	if err != nil {
		return 0, err
	}

	logger.Infof("应用创建成功，ID: %d, 类型: %s", appId, enum.HtmlCodeGen)
	return newApp.ID, nil
}

func (s *AppService) UpdateApp(ctx context.Context, req *api.YiKouAppUpdateRequest, userId int64) (bool, error) {
	if req.Id == 0 {
		return false, errorutil.ParamsError.WithMessage("应用ID不能为空")
	}

	app, err := query.Use(s.db).App.Where(query.App.ID.Eq(int64(req.Id))).First()
	if err != nil {
		return false, err
	}

	if app.UserID != userId {
		return false, errorutil.ParamsError.WithMessage("无权修改该应用")
	}

	updateMap := make(map[string]interface{})
	if req.AppName != "" {
		updateMap["appName"] = req.AppName
	}

	_, err = query.Use(s.db).App.Where(query.App.ID.Eq(int64(req.Id))).Updates(updateMap)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *AppService) DeleteApp(ctx context.Context, id int64, userId int64) (bool, error) {
	app, err := query.Use(s.db).App.Where(query.App.ID.Eq(id)).First()
	if err != nil {
		return false, err
	}

	if app.UserID != userId {
		return false, errorutil.ParamsError.WithMessage("无权删除该应用")
	}

	// 逻辑删除应用
	_, err = query.Use(s.db).App.Where(query.App.ID.Eq(id)).Update(query.App.IsDelete, 1)
	if err != nil {
		return false, err
	}

	err = s.chatHistoryService.DeleteByAppId(ctx, id)
	if err != nil {
		logger.Errorf("对话历史删除失败: %v\n", err)
	}
	return true, nil
}

func (s *AppService) GetApp(ctx context.Context, id int64, userId int64) (*model.App, error) {
	app, err := query.Use(s.db).App.Where(query.App.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	if app.UserID != userId {
		return nil, errorutil.ParamsError.WithMessage("无权查看该应用")
	}
	return app, nil
}

func (s *AppService) GetAppVo(ctx context.Context, id int64, userId int64) (vo.AppVo, error) {
	app, err := s.GetApp(ctx, id, userId)
	if err != nil {
		return vo.AppVo{}, err
	}

	// 获取用户信息
	userVo, err := s.userService.GetUserVo(ctx, app.UserID)
	if err != nil {
		return vo.AppVo{}, err
	}

	appVo := vo.AppVo{
		ID:           app.ID,
		AppName:      app.AppName,
		Cover:        app.Cover,
		InitPrompt:   app.InitPrompt,
		CodeGenType:  app.CodeGenType,
		DeployKey:    app.DeployKey,
		DeployedTime: app.DeployedTime,
		Priority:     app.Priority,
		UserID:       app.UserID,
		User:         userVo,
		CreateTime:   app.CreateTime,
		UpdateTime:   app.UpdateTime,
	}
	return appVo, nil
}

func (s *AppService) GetAppVoList(ctx context.Context, appList []*model.App) ([]vo.AppVo, error) {
	// 批量获取用户信息（去重）
	userIdSet := make(map[int64]bool)
	for _, app := range appList {
		userIdSet[app.UserID] = true
	}

	// 转换为切片
	userIdList := make([]int64, 0, len(userIdSet))
	for userId := range userIdSet {
		userIdList = append(userIdList, userId)
	}

	// 获取所有用户信息
	userList, err := query.Use(s.db).User.Where(query.User.ID.In(userIdList...)).Find()
	if err != nil {
		return nil, err
	}
	userVoMap := make(map[int64]vo.UserVo)
	for _, dbUser := range userList {
		userVo, err := s.userService.GetUserVo(ctx, dbUser.ID)
		if err != nil {
			return nil, err
		}
		userVoMap[dbUser.ID] = userVo
	}

	// 转换为AppVo列表
	var appVoList []vo.AppVo
	for _, app := range appList {
		appVo := vo.AppVo{
			ID:           app.ID,
			AppName:      app.AppName,
			Cover:        app.Cover,
			InitPrompt:   app.InitPrompt,
			CodeGenType:  app.CodeGenType,
			DeployKey:    app.DeployKey,
			DeployedTime: app.DeployedTime,
			Priority:     app.Priority,
			UserID:       app.UserID,
			User:         userVoMap[app.UserID],
			CreateTime:   app.CreateTime,
			UpdateTime:   app.UpdateTime,
		}
		appVoList = append(appVoList, appVo)
	}

	return appVoList, nil
}

func (s *AppService) ListMyApp(ctx context.Context, req *api.YiKouAppMyListRequest, userId int64) (*response.PageResponse[vo.AppVo], error) {
	if req.PageNum <= 0 {
		req.PageNum = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 20 {
		req.PageSize = 20
	}

	queryBuilder := query.Use(s.db).App.Where(query.App.IsDelete.Eq(0), query.App.UserID.Eq(userId))

	if req.AppName != "" {
		queryBuilder = queryBuilder.Where(query.App.AppName.Like("%" + req.AppName + "%"))
	}

	totalCount, err := queryBuilder.Count()
	if err != nil {
		return nil, err
	}

	totalPage := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))
	offset := (req.PageNum - 1) * req.PageSize

	if req.SortField != "" {
		if orderExpr, ok := query.App.GetFieldByName(req.SortField); ok {
			if req.SortOrder == "desc" {
				queryBuilder = queryBuilder.Order(orderExpr.Desc())
			} else {
				queryBuilder = queryBuilder.Order(orderExpr)
			}
		} else {
			queryBuilder = queryBuilder.Order(query.App.CreateTime.Desc())
		}
	} else {
		queryBuilder = queryBuilder.Order(query.App.CreateTime.Desc())
	}

	appList, err := queryBuilder.Offset(offset).Limit(req.PageSize).Find()
	if err != nil {
		return nil, err
	}

	// 转换为AppVo列表
	appVoList, err := s.GetAppVoList(ctx, appList)
	if err != nil {
		return nil, err
	}

	// 构建分页响应
	pageResponse := &response.PageResponse[vo.AppVo]{
		Records:            appVoList,
		PageNum:            req.PageNum,
		PageSize:           req.PageSize,
		TotalPage:          totalPage,
		TotalRow:           int(totalCount),
		OptimizeCountQuery: false,
	}

	return pageResponse, nil
}

func (s *AppService) ListGoodApp(ctx context.Context, req *api.YiKouAppFeaturedListRequest) (*response.PageResponse[vo.AppVo], error) {
	if req.PageNum <= 0 {
		req.PageNum = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 20 {
		req.PageSize = 20
	}

	queryBuilder := query.Use(s.db).App.Where(query.App.IsDelete.Eq(0), query.App.Priority.Gt(0))

	if req.AppName != "" {
		queryBuilder = queryBuilder.Where(query.App.AppName.Like("%" + req.AppName + "%"))
	}
	if req.CodeGenType != "" {
		queryBuilder = queryBuilder.Where(query.App.CodeGenType.Eq(req.CodeGenType))
	}
	if req.InitPrompt != "" {
		queryBuilder = queryBuilder.Where(query.App.InitPrompt.Like("%" + req.InitPrompt + "%"))
	}
	if req.Priority != 0 {
		queryBuilder = queryBuilder.Where(query.App.Priority.Eq(req.Priority))
	}

	totalCount, err := queryBuilder.Count()
	if err != nil {
		return nil, err
	}

	totalPage := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))
	offset := (req.PageNum - 1) * req.PageSize

	if req.SortField != "" {
		if orderExpr, ok := query.App.GetFieldByName(req.SortField); ok {
			if req.SortOrder == "desc" {
				queryBuilder = queryBuilder.Order(orderExpr.Desc())
			} else {
				queryBuilder = queryBuilder.Order(orderExpr)
			}
		} else {
			queryBuilder = queryBuilder.Order(query.App.Priority.Desc(), query.App.CreateTime.Desc())
		}
	} else {
		queryBuilder = queryBuilder.Order(query.App.Priority.Desc(), query.App.CreateTime.Desc())
	}

	appList, err := queryBuilder.Offset(offset).Limit(req.PageSize).Find()
	if err != nil {
		return nil, err
	}

	appVoList, err := s.GetAppVoList(ctx, appList)
	if err != nil {
		return nil, err
	}

	pageResponse := &response.PageResponse[vo.AppVo]{
		Records:   appVoList,
		PageNum:   req.PageNum,
		PageSize:  req.PageSize,
		TotalPage: totalPage,
		TotalRow:  int(totalCount),
	}

	return pageResponse, nil
}

func (s *AppService) AdminUpdateApp(ctx context.Context, req *api.YiKouAppAdminUpdateRequest) (bool, error) {
	if req.Id == "" {
		return false, errorutil.ParamsError.WithMessage("应用ID不能为空")
	}
	appId, err := strconv.Atoi(req.Id)
	if err != nil {
		return false, err
	}
	_, err = query.Use(s.db).App.Where(query.App.ID.Eq(int64(appId))).First()
	if err != nil {
		return false, err
	}

	updateMap := make(map[string]interface{})
	if req.AppName != "" {
		updateMap["appName"] = req.AppName
	}
	if req.Cover != "" {
		updateMap["cover"] = req.Cover
	}
	updateMap["priority"] = req.Priority

	_, err = query.Use(s.db).App.Where(query.App.ID.Eq(int64(appId))).Updates(updateMap)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *AppService) AdminDeleteApp(ctx context.Context, id int64) (bool, error) {
	_, err := query.Use(s.db).App.Where(query.App.ID.Eq(id)).Update(query.App.IsDelete, 1)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *AppService) AdminGetAppVo(ctx context.Context, id int64) (vo.AppVo, error) {
	app, err := query.Use(s.db).App.Where(query.App.ID.Eq(id)).First()
	if err != nil {
		return vo.AppVo{}, err
	}

	// 获取用户信息
	userVo, err := s.userService.GetUserVo(ctx, app.UserID)
	if err != nil {
		return vo.AppVo{}, err
	}

	appVo := vo.AppVo{
		ID:           app.ID,
		AppName:      app.AppName,
		Cover:        app.Cover,
		InitPrompt:   app.InitPrompt,
		CodeGenType:  app.CodeGenType,
		DeployKey:    app.DeployKey,
		DeployedTime: app.DeployedTime,
		Priority:     app.Priority,
		UserID:       app.UserID,
		User:         userVo,
		CreateTime:   app.CreateTime,
		UpdateTime:   app.UpdateTime,
	}
	return appVo, nil
}

func (s *AppService) AdminListApp(ctx context.Context, req *api.YiKouAppAdminListRequest) (*response.PageResponse[*model.App], error) {
	if req.PageNum <= 0 {
		req.PageNum = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	queryBuilder := query.Use(s.db).App.Where(query.App.IsDelete.Eq(0))

	if req.ID != "" {
		appId, err := strconv.Atoi(req.ID)
		if err != nil {
			return nil, err
		}
		queryBuilder = queryBuilder.Where(query.App.ID.Eq(int64(appId)))
	}
	if req.AppName != "" {
		queryBuilder = queryBuilder.Where(query.App.AppName.Like("%" + req.AppName + "%"))
	}
	if req.Cover != "" {
		queryBuilder = queryBuilder.Where(query.App.Cover.Like("%" + req.Cover + "%"))
	}
	if req.InitPrompt != "" {
		queryBuilder = queryBuilder.Where(query.App.InitPrompt.Like("%" + req.InitPrompt + "%"))
	}
	if req.CodeGenType != "" {
		queryBuilder = queryBuilder.Where(query.App.CodeGenType.Eq(req.CodeGenType))
	}
	if req.DeployKey != "" {
		queryBuilder = queryBuilder.Where(query.App.DeployKey.Like("%" + req.DeployKey + "%"))
	}
	if req.Priority != 0 {
		queryBuilder = queryBuilder.Where(query.App.Priority.Eq(req.Priority))
	}
	if req.UserID != 0 {
		queryBuilder = queryBuilder.Where(query.App.UserID.Eq(req.UserID))
	}

	totalCount, err := queryBuilder.Count()
	if err != nil {
		return nil, err
	}

	totalPage := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))
	offset := (req.PageNum - 1) * req.PageSize

	if req.SortField != "" {
		if orderExpr, ok := query.App.GetFieldByName(req.SortField); ok {
			if req.SortOrder == "desc" {
				queryBuilder = queryBuilder.Order(orderExpr.Desc())
			} else {
				queryBuilder = queryBuilder.Order(orderExpr)
			}
		} else {
			queryBuilder = queryBuilder.Order(query.App.CreateTime.Desc())
		}
	} else {
		queryBuilder = queryBuilder.Order(query.App.CreateTime.Desc())
	}

	appList, err := queryBuilder.Offset(offset).Limit(req.PageSize).Find()
	if err != nil {
		return nil, err
	}

	pageResponse := &response.PageResponse[*model.App]{
		Records:            appList,
		PageNum:            req.PageNum,
		PageSize:           req.PageSize,
		TotalPage:          totalPage,
		TotalRow:           int(totalCount),
		OptimizeCountQuery: false,
	}

	return pageResponse, nil
}
