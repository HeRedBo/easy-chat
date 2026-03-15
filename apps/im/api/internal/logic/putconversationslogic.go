// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/HeRedBo/easy-chat/apps/im/api/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/api/internal/types"
	"github.com/HeRedBo/easy-chat/apps/im/rpc/imclient"
	"github.com/HeRedBo/easy-chat/pkg/ctxdata"
	"github.com/jinzhu/copier"

	"github.com/zeromicro/go-zero/core/logx"
)

type PutConversationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新会话
func NewPutConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PutConversationsLogic {
	return &PutConversationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PutConversationsLogic) PutConversations(req *types.PutConversationsReq) (resp *types.PutConversationsResp, err error) {
	// todo: add your logic here and delete this line

	uid := ctxdata.GetUid(l.ctx)

	var conversationList map[string]*imclient.Conversation
	err = copier.Copy(&conversationList, req.ConversationList)
	if err != nil {
		return nil, err
	}

	data, err := l.svcCtx.PutConversations(l.ctx, &imclient.PutConversationsReq{
		UserId:           uid,
		ConversationList: conversationList,
	})
	var res types.PutConversationsResp
	err = copier.Copy(&res, data)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
