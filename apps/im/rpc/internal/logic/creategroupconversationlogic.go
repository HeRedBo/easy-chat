package logic

import (
	"context"

	"github.com/HeRedBo/easy-chat/apps/im/immodels"
	"github.com/HeRedBo/easy-chat/apps/im/rpc/im"
	"github.com/HeRedBo/easy-chat/apps/im/rpc/internal/svc"
	"github.com/HeRedBo/easy-chat/pkg/constants"
	"github.com/HeRedBo/easy-chat/pkg/xerr"
	"github.com/gookit/goutil/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateGroupConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateGroupConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGroupConversationLogic {
	return &CreateGroupConversationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateGroupConversationLogic) CreateGroupConversation(in *im.CreateGroupConversationReq) (*im.CreateGroupConversationResp, error) {
	// todo: add your logic here and delete this line

	res := &im.CreateGroupConversationResp{}
	_, err := l.svcCtx.ConversationModel.FindOne(l.ctx, in.GroupId)
	if err == nil {
		return res, nil
	}

	if err != immodels.ErrNotFound {
		return nil, errorx.Wrapf(xerr.NewDBErr(), "find conversion err %v,req %v", err, in)
	}

	err = l.svcCtx.ConversationModel.Insert(l.ctx, &immodels.Conversation{
		ConversationId: in.GroupId,
		ChatType:       constants.GroupChatType,
	})
	
	if err != nil {
		return res, errorx.Wrapf(xerr.NewDBErr(), "insert conversation err %v,req %v", err, in)
	}
	return &im.CreateGroupConversationResp{}, nil
}
