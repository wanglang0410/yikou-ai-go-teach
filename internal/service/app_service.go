package service

import (
	"context"
	"github.com/cloudwego/eino/schema"
	"yikou-ai-go-teach/internal/api"
	"yikou-ai-go-teach/internal/dal/model"
	"yikou-ai-go-teach/internal/dal/vo"
	"yikou-ai-go-teach/pkg/response"
)

type IAppService interface {
	ChatToGenCode(ctx context.Context, appId int64, message string, loginUser *vo.UserVo) (*schema.StreamReader[*schema.Message], error)
	AddApp(ctx context.Context, req *api.YiKouAppAddRequest, userId int64) (int64, error)
	UpdateApp(ctx context.Context, req *api.YiKouAppUpdateRequest, userId int64) (bool, error)
	DeleteApp(ctx context.Context, id int64, userId int64) (bool, error)
	GetApp(ctx context.Context, id int64, userId int64) (*model.App, error)
	GetAppVo(ctx context.Context, id int64, userId int64) (vo.AppVo, error)
	GetAppVoList(ctx context.Context, appList []*model.App) ([]vo.AppVo, error)
	ListMyApp(ctx context.Context, req *api.YiKouAppMyListRequest, userId int64) (*response.PageResponse[vo.AppVo], error)
	ListGoodApp(ctx context.Context, req *api.YiKouAppFeaturedListRequest) (*response.PageResponse[vo.AppVo], error)
	AdminUpdateApp(ctx context.Context, req *api.YiKouAppAdminUpdateRequest) (bool, error)
	AdminDeleteApp(ctx context.Context, id int64) (bool, error)
	AdminGetAppVo(ctx context.Context, id int64) (vo.AppVo, error)
	AdminListApp(ctx context.Context, req *api.YiKouAppAdminListRequest) (*response.PageResponse[*model.App], error)
}
