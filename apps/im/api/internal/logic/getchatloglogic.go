// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/HeRedBo/easy-chat/apps/im/api/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/api/internal/types"
	"github.com/HeRedBo/easy-chat/apps/im/rpc/imclient"
	"github.com/jinzhu/copier"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetChatLogLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 根据用户获取聊天记录
func NewGetChatLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetChatLogLogic {
	return &GetChatLogLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetChatLogLogic) GetChatLog(req *types.ChatLogReq) (resp *types.ChatLogResp, err error) {
	// todo: add your logic here and delete this line
	data, err := l.svcCtx.GetChatLog(l.ctx, &imclient.GetChatLogReq{
		ConversationId: req.ConversationId,
		StartSendTime:  req.StartSendTime,
		EndSendTime:    req.EndSendTime,
		Count:          req.Count,
	})

	if err != nil {
		return nil, err
	}

	var res types.ChatLogResp
	copier.Copy(&res, data)
	return &res, nil
}
