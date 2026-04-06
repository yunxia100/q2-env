package ctrler

import (
	"fmt"
	"strconv"
	"ymlink-q2/apps/server/self"
	"ymlink-q2/model"
	"ymlink-q2/plugin"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GET /api/robot/friend_notices?robot_id=xxx
func (ctrler *ctrler_robot) FriendNotices(ctx *gin.Context) {
	robotId := ctx.Query("robot_id")
	if robotId == "" {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人ID参数必须上传", nil)
		return
	}

	oRobotId, err := primitive.ObjectIDFromHex(robotId)
	if err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "上传的机器人ID参数格式错误", nil)
		return
	}

	robot := self.Robots.Existed(oRobotId)
	if robot == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人信息不存在", nil)
		return
	}

	result, err := robot.PatchFriendNotices()
	if err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, err.Error(), nil)
		return
	}

	// 转换为前端所需格式
	items := make([]*model.PatchFriendNoticeItem, 0)
	for _, msg := range result.FriendMsgs {
		if msg.Msg == nil {
			continue
		}
		items = append(items, &model.PatchFriendNoticeItem{
			MsgType:       msg.MsgType,
			MsgSeq:        msg.MsgSeq,
			MsgTime:       msg.MsgTime,
			ReqUin:        msg.ReqUin,
			Nick:          msg.Msg.ReqUinNick,
			Gender:        msg.Msg.ReqUinGender,
			Age:           msg.Msg.ReqUinAge,
			SrcId:         msg.Msg.SrcId,
			SubSrcId:      msg.Msg.SubSrcId,
			MsgTitle:      msg.Msg.MsgTitle,
			MsgAdditional: msg.Msg.MsgAdditional,
			MsgSource:     msg.Msg.MsgSource,
			MsgDetail:     msg.Msg.MsgDetail,
		})
	}

	plugin.HttpSuccess(ctx, items)
}

// POST /api/robot/friend_pass
func (ctrler *ctrler_robot) FriendPass(ctx *gin.Context) {
	var request model.PatchFriendPassRequest
	if err := ctx.ShouldBind(&request); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请求参数错误", nil)
		return
	}

	if request.RobotId == "" {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人ID参数必须上传", nil)
		return
	}

	oRobotId, err := primitive.ObjectIDFromHex(request.RobotId)
	if err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请求机器人ID参数格式错误", nil)
		return
	}

	robot := self.Robots.Existed(oRobotId)
	if robot == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人信息不存在", nil)
		return
	}

	result, err := robot.PatchFriendPass(
		strconv.FormatInt(request.ReqUin, 10),
		strconv.FormatInt(request.SrcId, 10),
		strconv.FormatInt(request.SubSrcId, 10),
	)
	if err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, err.Error(), nil)
		return
	}

	if result.Head.Result == -1 {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, fmt.Sprintf("通过好友响应错误,code:%d,message:%s", result.Head.Result, result.Head.MsgFail), nil)
		return
	}

	plugin.HttpSuccess(ctx, plugin.Bson{})
}
