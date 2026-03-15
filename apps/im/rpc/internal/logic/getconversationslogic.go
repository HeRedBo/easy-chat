package logic

import (
	"context"

	"github.com/HeRedBo/easy-chat/apps/im/immodels"
	"github.com/HeRedBo/easy-chat/apps/im/rpc/im"
	"github.com/HeRedBo/easy-chat/apps/im/rpc/internal/svc"
	"github.com/HeRedBo/easy-chat/pkg/xerr"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetConversationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationsLogic {
	return &GetConversationsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取会话
func (l *GetConversationsLogic) GetConversations(in *im.GetConversationsReq) (*im.GetConversationsResp, error) {
	// todo: add your logic here and delete this line
	// 根据用户查询用户会话列表
	data, err := l.svcCtx.ConversationsModel.FindByUserId(l.ctx, in.UserId)
	if err != nil {
		if err == immodels.ErrNotFound {
			return &im.GetConversationsResp{}, nil
		}
		return nil, errors.Wrapf(xerr.NewDBErr(), "ConversationsModel.FindByUserId err %v, req %v", err, in.UserId)
	}
	var res im.GetConversationsResp
	copier.Copy(&res, &data)
	// 根据会话列表，查询具体的会话
	ids := make([]string, 0, len(data.ConversationList))
	for _, conversation := range data.ConversationList {
		ids = append(ids, conversation.ConversationId)
	}

	// 统计会话的消息情况
	list, err := l.svcCtx.ConversationModel.ListByConversationIds(l.ctx, ids)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewDBErr(), "list conversation by err %v, req %v", err, in.UserId)
	}

	// 计算是否存在未读消息
	for _, conversation := range list {
		if _, ok := res.ConversationList[conversation.ConversationId]; !ok {
			continue
		}
		// 用户读取的消息量
		total := res.ConversationList[conversation.ConversationId].Total
		if total < int32(conversation.Total) {
			// 有新的消息
			res.ConversationList[conversation.ConversationId].Total = int32(conversation.Total)
			// 有多少是未读
			res.ConversationList[conversation.ConversationId].ToRead = int32(conversation.Total) - total
			// 更改当前会话为显示状态
			res.ConversationList[conversation.ConversationId].IsShow = true
		}
	}
	return &res, nil
}
