// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/HeRedBo/easy-chat/apps/im/api/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/api/internal/types"
	"github.com/HeRedBo/easy-chat/apps/im/rpc/im"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetUpUserConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 建立会话
func NewSetUpUserConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetUpUserConversationLogic {
	return &SetUpUserConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetUpUserConversationLogic) SetUpUserConversation(req *types.SetUpUserConversationReq) (resp *types.SetUpUserConversationResp, err error) {
	// todo: add your logic here and delete this line
	_, err = l.svcCtx.SetUpUserConversation(l.ctx, &im.SetUpUserConversationReq{
		SendId:   req.SendId,
		RecvId:   req.RecvId,
		ChatType: req.ChatType,
	})

	if err != nil {
		return nil, err
	}
	return &types.SetUpUserConversationResp{}, nil
}
