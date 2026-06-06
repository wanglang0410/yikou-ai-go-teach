package api

import (
	"time"
	"yikou-ai-go-teach/internal/dal/model"
	"yikou-ai-go-teach/pkg/response"
)

type YiKouChatHistoryQueryRequest struct {
	Id             int64     `json:"id"`
	AppId          int64     `json:"appId"`
	Message        string    `json:"message"`
	MessageType    string    `json:"messageType"`
	UserId         int64     `json:"userId"`
	LastCreateTime time.Time `json:"lastCreateTime"`
}

type YiKouChatHistoryQueryResponse response.BaseResponse[response.PageResponse[*model.ChatHistory]]
