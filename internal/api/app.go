package api

import (
	"yikou-ai-go-teach/internal/dal/model"
	"yikou-ai-go-teach/internal/dal/vo"
	"yikou-ai-go-teach/pkg/request"
	"yikou-ai-go-teach/pkg/response"
)

type YiKouAppAddRequest struct {
	InitPrompt string `json:"initPrompt"`
}

type YiKouAppAddResponse response.BaseResponse[string]

type YiKouAppUpdateRequest struct {
	request.DeleteRequest
	AppName string `json:"appName"`
}

type YiKouAppUpdateResponse response.BaseResponse[bool]

type YiKouAppDeleteResponse response.BaseResponse[bool]

type YiKouAppGetResponse response.BaseResponse[model.App]

type YiKouAppGetVoResponse response.BaseResponse[vo.AppVo]

type YiKouAppMyListRequest struct {
	request.PageRequest
	AppName string `json:"appName"`
}

type YiKouAppMyListResponse response.BaseResponse[response.PageResponse[vo.AppVo]]

type YiKouAppFeaturedListRequest struct {
	request.PageRequest
	AppName     string `json:"appName"`
	CodeGenType string `json:"codeGenType"`
	InitPrompt  string `json:"initPrompt"`
	Priority    int32  `json:"priority"`
}

type YiKouAppFeaturedListResponse response.BaseResponse[response.PageResponse[vo.AppVo]]

type YiKouAppAdminUpdateRequest struct {
	Id       string `json:"id"`
	AppName  string `json:"appName"`
	Cover    string `json:"cover"`
	Priority int32  `json:"priority"`
}

type YiKouAppAdminUpdateResponse response.BaseResponse[bool]

type YiKouAppAdminDeleteResponse response.BaseResponse[bool]

type YiKouAppAdminGetResponse response.BaseResponse[vo.AppVo]

type YiKouAppAdminListRequest struct {
	request.PageRequest
	ID           string `json:"id"`
	AppName      string `json:"appName"`
	Cover        string `json:"cover"`
	InitPrompt   string `json:"initPrompt"`
	CodeGenType  string `json:"codeGenType"`
	DeployKey    string `json:"deployKey"`
	DeployedTime string `json:"deployedTime"`
	Priority     int32  `json:"priority"`
	UserID       int64  `json:"userId"`
}

type YiKouAppAdminListResponse response.BaseResponse[response.PageResponse[model.App]]
