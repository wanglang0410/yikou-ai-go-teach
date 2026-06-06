package service

import (
	"context"
	"time"
	"yikou-ai-go-teach/internal/api"
	"yikou-ai-go-teach/internal/dal/model"
	"yikou-ai-go-teach/internal/dal/vo"
	"yikou-ai-go-teach/internal/store"
	"yikou-ai-go-teach/pkg/enum"
	"yikou-ai-go-teach/pkg/response"
)

type IChatHistoryService interface {
	AddChatMessage(ctx context.Context, appId int64, message string, messageType enum.ChatHistoryMessageTypeEnum, userId int64) error
	DeleteByAppId(ctx context.Context, appId int64) error
	ListAppChatHistoryByPage(ctx context.Context, appId int64, pageSize int32, lastCreateTime time.Time, loginUser *vo.UserVo) (*response.PageResponse[*model.ChatHistory], error)
	ListAllChatHistoryByPageForAdmin(ctx context.Context, pageNum int32, pageSize int32, queryRequest *api.YiKouChatHistoryQueryRequest) (*response.PageResponse[*model.ChatHistory], error)
	LoadChatHistoryToMemory(ctx context.Context, appId int64, memoryStore store.MemoryStore, maxCount int) (int, error)
}
