package logic

import (
	"context"
	"fmt"
	"github.com/bytedance/gopkg/util/logger"
	"github.com/cloudwego/eino/schema"
	"strings"
	"time"
	"yikou-ai-go-teach/internal/api"
	"yikou-ai-go-teach/internal/dal/model"
	"yikou-ai-go-teach/internal/dal/query"
	"yikou-ai-go-teach/internal/dal/vo"
	"yikou-ai-go-teach/internal/store"
	"yikou-ai-go-teach/pkg/enum"
	"yikou-ai-go-teach/pkg/errorutil"
	"yikou-ai-go-teach/pkg/response"
	"yikou-ai-go-teach/pkg/snowflake"

	"gorm.io/gorm"
)

func NewChatHistoryService(db *gorm.DB) *ChatHistoryService {
	return &ChatHistoryService{
		db: db,
	}
}

type ChatHistoryService struct {
	db *gorm.DB
}

func (s *ChatHistoryService) LoadChatHistoryToMemory(ctx context.Context, appId int64, memoryStore store.MemoryStore, maxCount int) (int, error) {
	historySummary, err := query.Use(s.db).ChatHistory.
		Where(query.ChatHistory.AppID.Eq(appId), query.ChatHistory.MessageType.Eq(string(enum.SummaryMessageType))).
		Order(query.ChatHistory.CreateTime.Desc()).
		Limit(maxCount).
		Find()
	var historyList []*model.ChatHistory
	if len(historySummary) == 0 || err != nil {
		historyList, err = query.Use(s.db).ChatHistory.
			Where(query.ChatHistory.AppID.Eq(appId)).
			Order(query.ChatHistory.CreateTime.Desc()).
			Limit(maxCount).
			Find()
		if err != nil {
			return 0, err
		}
		historyList = historyList[1:]
	} else {
		historyList = historySummary
	}

	err = memoryStore.ClearMessages(ctx)
	if err != nil {
		return 0, err
	}
	loadedCount := 0
	for i := len(historyList) - 1; i >= 0; i-- {
		history := historyList[i]
		if history.MessageType == string(enum.UserMessageType) {
			err = memoryStore.AppendMessage(ctx, schema.UserMessage(history.Message))
			if err != nil {
				return loadedCount, err
			}
			loadedCount++
		} else if history.MessageType == string(enum.AIMessageType) {
			err = memoryStore.AppendMessage(ctx, schema.AssistantMessage(history.Message, nil))
			if err != nil {
				return loadedCount, err
			}
			loadedCount++
		} else if history.MessageType == string(enum.SummaryMessageType) {
			err = memoryStore.AppendMessage(ctx, schema.SystemMessage(history.Message))
			if err != nil {
				return loadedCount, err
			}
			loadedCount++
		}
	}

	return loadedCount, nil
}

func (s *ChatHistoryService) ListAppChatHistoryByPage(ctx context.Context,
	appId int64, pageSize int32, lastCreateTime time.Time, loginUser *vo.UserVo) (*response.PageResponse[*model.ChatHistory], error) {
	if appId == 0 || appId < 0 || pageSize <= 0 || pageSize > 50 {
		return nil, errorutil.ParamsError
	}
	if loginUser == nil {
		return nil, errorutil.NotLoginError
	}
	// 校验用户角色是否为管理员或者应用创建者
	app, err := query.Use(s.db).App.Where(query.App.ID.Eq(appId)).First()
	if err != nil {
		return nil, err
	}
	if app.UserID != loginUser.ID && loginUser.UserRole != string(enum.AdminRole) {
		return nil, errorutil.NotAuthError
	}

	// 构建查询条件
	chatHistoryQuery := query.Use(s.db).ChatHistory.Where(query.ChatHistory.AppID.Eq(appId),
		query.ChatHistory.MessageType.Neq(string(enum.SummaryMessageType)))

	// 处理时间过滤
	if !lastCreateTime.IsZero() {
		chatHistoryQuery = chatHistoryQuery.Where(query.ChatHistory.CreateTime.Lt(lastCreateTime))
	}

	// 查询总记录数
	totalRow, err := chatHistoryQuery.Count()
	if err != nil {
		return nil, err
	}

	// 计算总页数
	totalPage := 0
	if totalRow > 0 {
		totalPage = int((totalRow + int64(pageSize) - 1) / int64(pageSize))
	}

	// 分页查询应用的聊天记录
	chatHistoryList, err := chatHistoryQuery.
		Order(query.ChatHistory.CreateTime.Desc()).
		Limit(int(pageSize)).
		Find()
	if err != nil {
		return nil, err
	}

	return &response.PageResponse[*model.ChatHistory]{
		Records:            chatHistoryList,
		PageNum:            1,
		PageSize:           int(pageSize),
		TotalPage:          totalPage,
		TotalRow:           int(totalRow),
		OptimizeCountQuery: true,
	}, nil
}

func (s *ChatHistoryService) DeleteByAppId(ctx context.Context, appId int64) error {
	if appId == 0 || appId < 0 {
		return errorutil.ParamsError.WithMessage("应用ID不能为空")
	}
	_, err := query.Use(s.db).ChatHistory.Where(query.ChatHistory.AppID.Eq(appId)).Delete()
	if err != nil {
		return err
	}
	return nil
}

func (s *ChatHistoryService) AddChatMessage(ctx context.Context, appId int64,
	message string, messageType enum.ChatHistoryMessageTypeEnum, userId int64) error {
	// 校验参数
	if appId <= 0 || messageType == "" || userId <= 0 || message == "" {
		return errorutil.ParamsError
	}

	// 计算轮次
	lastMessage, err := query.Use(s.db).ChatHistory.
		Where(query.ChatHistory.AppID.Eq(appId)).
		Order(query.ChatHistory.CreateTime.Desc()).
		First()

	var turnNumber int32
	if err != nil {
		turnNumber = 0
	} else {
		turnNumber = lastMessage.TurnNumber
	}

	// 如果当前是用户消息，开启新的一轮
	if messageType == enum.UserMessageType {
		turnNumber += 1
	}

	chatMessageId, err := snowflake.GenerateSnowFlakeId()
	if err != nil {
		return err
	}
	err = query.Use(s.db).ChatHistory.Create(&model.ChatHistory{
		ID:          chatMessageId,
		AppID:       appId,
		Message:     message,
		MessageType: string(messageType),
		UserID:      userId,
		TurnNumber:  turnNumber,
	})
	if err != nil {
		return err
	}

	// 当对话轮次达到20轮时，生成总结
	if turnNumber >= 20 && messageType == enum.AIMessageType {
		go s.generateSummary(context.Background(), appId, userId)
	}

	return nil
}

// generateSummary 生成对话总结
func (s *ChatHistoryService) generateSummary(ctx context.Context, appId int64, userId int64) {
	// 获取历史对话记录
	historyList, err := query.Use(s.db).ChatHistory.
		Where(query.ChatHistory.AppID.Eq(appId)).
		Order(query.ChatHistory.CreateTime.Asc()).
		Find()
	if err != nil {
		logger.Errorf("获取历史对话失败: %v\n", err)
		return
	}

	// 构建对话历史字符串
	var chatHistoryBuilder strings.Builder
	for _, history := range historyList {
		if history.MessageType == string(enum.UserMessageType) {
			chatHistoryBuilder.WriteString(fmt.Sprintf("用户: %s\n", history.Message))
		} else if history.MessageType == string(enum.AIMessageType) {
			chatHistoryBuilder.WriteString(fmt.Sprintf("AI: %s\n", history.Message))
		}
	}

	err = s.AddChatMessage(ctx, appId, chatHistoryBuilder.String(), enum.SummaryMessageType, userId)
	if err != nil {
		logger.Errorf("对话总结保存失败: %v\n", err)
	}
}

func (s *ChatHistoryService) ListAllChatHistoryByPageForAdmin(ctx context.Context, pageNum int32, pageSize int32, queryRequest *api.YiKouChatHistoryQueryRequest) (*response.PageResponse[*model.ChatHistory], error) {
	// 校验参数
	if pageNum <= 0 || pageSize <= 0 || pageSize > 50 {
		return nil, errorutil.ParamsError
	}
	if queryRequest == nil {
		return nil, errorutil.ParamsError
	}

	// 构建查询条件
	chatHistoryQuery := query.Use(s.db).ChatHistory.Where(query.ChatHistory.ID.IsNotNull())

	// 应用查询条件
	if queryRequest.Id > 0 {
		chatHistoryQuery = chatHistoryQuery.Where(query.ChatHistory.ID.Eq(queryRequest.Id))
	}
	if queryRequest.AppId > 0 {
		chatHistoryQuery = chatHistoryQuery.Where(query.ChatHistory.AppID.Eq(queryRequest.AppId))
	}
	if queryRequest.UserId > 0 {
		chatHistoryQuery = chatHistoryQuery.Where(query.ChatHistory.UserID.Eq(queryRequest.UserId))
	}
	if queryRequest.MessageType != "" {
		chatHistoryQuery = chatHistoryQuery.Where(query.ChatHistory.MessageType.Eq(queryRequest.MessageType))
	}
	if queryRequest.Message != "" {
		chatHistoryQuery = chatHistoryQuery.Where(query.ChatHistory.Message.Like("%" + queryRequest.Message + "%"))
	}
	if !queryRequest.LastCreateTime.IsZero() {
		chatHistoryQuery = chatHistoryQuery.Where(query.ChatHistory.CreateTime.Lt(queryRequest.LastCreateTime))
	}

	// 查询总记录数
	totalRow, err := chatHistoryQuery.Count()
	if err != nil {
		return nil, err
	}

	// 计算总页数
	totalPage := 0
	if totalRow > 0 {
		totalPage = int((totalRow + int64(pageSize) - 1) / int64(pageSize))
	}

	// 计算偏移量
	offset := int((pageNum - 1) * pageSize)

	// 分页查询
	chatHistoryList, err := chatHistoryQuery.
		Order(query.ChatHistory.CreateTime.Desc()).
		Limit(int(pageSize)).
		Offset(offset).
		Find()
	if err != nil {
		return nil, err
	}

	return &response.PageResponse[*model.ChatHistory]{
		Records:            chatHistoryList,
		PageNum:            int(pageNum),
		PageSize:           int(pageSize),
		TotalPage:          totalPage,
		TotalRow:           int(totalRow),
		OptimizeCountQuery: true,
	}, nil
}
