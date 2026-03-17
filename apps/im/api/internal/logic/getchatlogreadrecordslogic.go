// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/HeRedBo/easy-chat/apps/im/api/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/api/internal/types"
	"github.com/HeRedBo/easy-chat/apps/im/rpc/im"
	"github.com/HeRedBo/easy-chat/apps/social/rpc/social"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/user"
	"github.com/HeRedBo/easy-chat/pkg/bitmap"
	"github.com/HeRedBo/easy-chat/pkg/constants"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetChatLogReadRecordsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 验证消息已读记录查询
func NewGetChatLogReadRecordsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetChatLogReadRecordsLogic {
	return &GetChatLogReadRecordsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetChatLogReadRecordsLogic) GetChatLogReadRecords(req *types.GetChatLogReadRecordsReq) (resp *types.GetChatLogReadRecordsResp, err error) {
	// todo: add your logic here and delete this line
	chatlogs, err := l.svcCtx.Im.GetChatLog(l.ctx, &im.GetChatLogReq{
		MsgId: req.MsgId,
	})

	if err != nil || len(chatlogs.List) == 0 {
		return nil, err
	}

	var (
		chatlog = chatlogs.List[0]
		reads   = []string{chatlog.SendId}
		unReads []string
		ids     []string
	)
	// 分别设置已读未读
	switch constants.ChatType(chatlog.ChatType) {
	case constants.SingleChatType:
		if len(chatlog.ReadRecords) == 0 || chatlog.ReadRecords[0] == 0 {
			unReads = []string{chatlog.RecvId}
		} else {
			reads = append(reads, chatlog.RecvId)
		}
		ids = []string{chatlog.RecvId, chatlog.SendId}
	case constants.GroupChatType:
		groupUsers, err := l.svcCtx.Social.GroupUsers(l.ctx, &social.GroupUsersReq{
			GroupId: chatlog.RecvId,
		})
		if err != nil {
			return nil, err
		}
		bitmaps := bitmap.Load(chatlog.ReadRecords)
		for _, member := range groupUsers.List {
			ids = append(ids, member.UserId)
			if member.UserId == chatlog.SendId {
				continue
			}
			if bitmaps.IsSet(member.UserId) {
				reads = append(reads, member.UserId)
			} else {
				unReads = append(unReads, member.UserId)
			}
		}
	}

	userEntities, err := l.svcCtx.User.FindUser(l.ctx, &user.FindUserReq{
		Ids: ids,
	})

	if err != nil {
		return nil, err
	}

	userEntitySet := make(map[string]*user.UserEntity, len(userEntities.User))
	for _, entity := range userEntities.User {
		userEntitySet[entity.Id] = entity
	}

	// 设置手机号码
	for i, read := range reads {
		if entity, ok := userEntitySet[read]; ok {
			reads[i] = entity.Phone
		}
	}

	for i, unread := range unReads {
		if entity, ok := userEntitySet[unread]; ok {
			unReads[i] = entity.Phone
		}
	}

	return &types.GetChatLogReadRecordsResp{
		Reads:   reads,
		UnReads: unReads,
	}, nil

}
