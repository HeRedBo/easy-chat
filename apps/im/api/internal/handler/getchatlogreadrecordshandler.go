// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	"github.com/HeRedBo/easy-chat/apps/im/api/internal/logic"
	"github.com/HeRedBo/easy-chat/apps/im/api/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 验证消息已读记录查询
func GetChatLogReadRecordsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetChatLogReadRecordsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewGetChatLogReadRecordsLogic(r.Context(), svcCtx)
		resp, err := l.GetChatLogReadRecords(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
