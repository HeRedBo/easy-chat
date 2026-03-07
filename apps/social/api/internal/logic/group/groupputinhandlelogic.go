// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"

	"github.com/HeRedBo/easy-chat/apps/social/api/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/social/api/internal/types"
	"github.com/HeRedBo/easy-chat/apps/social/rpc/socialclient"
	"github.com/HeRedBo/easy-chat/pkg/constants"
	"github.com/HeRedBo/easy-chat/pkg/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type GroupPutInHandleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 申请进群处理
func NewGroupPutInHandleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupPutInHandleLogic {
	return &GroupPutInHandleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GroupPutInHandleLogic) GroupPutInHandle(req *types.GroupPutInHandleRep) (resp *types.GroupPutInHandleResp, err error) {
	// todo: add your logic here and delete this line
	_, err = l.svcCtx.Social.GroupPutInHandle(l.ctx, &socialclient.GroupPutInHandleReq{
		GroupReqId:   req.GroupReqId,
		GroupId:      req.GroupId,
		HandleUid:    ctxdata.GetUid(l.ctx),
		HandleResult: req.HandleResult,
	})

	if constants.HandlerResult(req.HandleResult) != constants.PassHandlerResult {
		return
	}

	// TODO: 通过后的业务
	return
}
