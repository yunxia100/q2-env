package ctrler

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"strconv"
	"strings"
	"time"
	"ymlink-q2/apps/server/self"
	"ymlink-q2/define"
	"ymlink-q2/ini"
	"ymlink-q2/model"
	"ymlink-q2/plugin"
	"ymlink-q2/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (ctrler *ctrler_custservice) Fetch(ctx *gin.Context) {

	var (
		user_info *model.User
	)

	if user_info = BindUserInfo(ctx); user_info == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户信息提取异常，请重试", nil)
		return
	}

	plugin.HttpSuccess(ctx, self.Custservices.Fetch(user_info.Id))
}

func (ctrler *ctrler_custservice) Create(ctx *gin.Context) {

	var (
		user_info         *model.User
		table_custservice model.Custservice
	)

	if user_info = BindUserInfo(ctx); user_info == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户信息提取异常，请重试", nil)
		return
	}

	if err := ctx.Bind(&table_custservice); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "格式异常", err.Error())
		return
	}

	if len(table_custservice.Username) < 5 {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户名长度至少为五位", nil)
		return
	}

	if strings.Contains(table_custservice.Username, " ") {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户名不能存在空格", nil)
		return
	}

	if len(table_custservice.Password) < 6 {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "密码长度至少六位", nil)
		return
	}

	if custservice := self.Custservices.ExistedByUsername(user_info.Id, table_custservice.Username); custservice != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户名已被使用", nil)
		return
	}

	if err := table_custservice.Create(user_info.Id); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "请求失败", err.Error())
		return
	}

	self.Custservices.Create(&table_custservice)

	User.Updating(&user_info.Id, &user_info.PcId, define.MONGO_COLLECTION_CUSTSERVICE)

	plugin.HttpSuccess(ctx, nil)
}

func (ctrler *ctrler_custservice) Delete(ctx *gin.Context) {
	var (
		user_info *model.User

		custservice_id = plugin.ObjectId(ctx.Query("id"))

		custservice *model.Custservice
	)

	if user_info = BindUserInfo(ctx); user_info == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户信息提取异常，请重试", nil)
		return
	}

	if custservice = self.Custservices.Existed(custservice_id); custservice == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "不存在", nil)
		return
	}

	if custservice.UserId != user_info.Id {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "无权限", nil)
		return
	}

	if err := custservice.Delete(); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "请求失败", err.Error())
		return
	}

	self.Proxys.Delete(custservice.Id)

	User.Updating(&user_info.Id, &user_info.PcId, define.MONGO_COLLECTION_CUSTSERVICE)

	plugin.HttpSuccess(ctx, nil)
}

func (ctrler *ctrler_custservice) Update(ctx *gin.Context) {

	var (
		user_info *model.User

		custservice       *model.Custservice
		table_custservice model.Custservice
	)

	if user_info = BindUserInfo(ctx); user_info == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户信息提取异常，请重试", nil)
		return
	}

	if err := ctx.Bind(&table_custservice); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "格式异常", err.Error())
		return
	}

	if custservice = self.Custservices.Existed(table_custservice.Id); custservice == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "客服不存在", nil)
		return
	}

	if custservice.UserId != user_info.Id {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "无权限", nil)
		return
	}

	if err := custservice.Update(plugin.Bson{
		"username": table_custservice.Username,
		"password": table_custservice.Password,
		"remark":   table_custservice.Remark,
	}); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "请求失败", err.Error())
		return
	}

	custservice.Username = table_custservice.Username
	custservice.Password = table_custservice.Password
	custservice.Remark = table_custservice.Remark

	User.Updating(&user_info.Id, &user_info.PcId, define.MONGO_COLLECTION_CUSTSERVICE)

	plugin.HttpSuccess(ctx, nil)
}

func (ctrler *ctrler_custservice) Singin(ctx *gin.Context) {

	var (
		user_info *model.User

		table_custservice model.Custservice
		custservice_info  *model.Custservice
	)

	if err := ctx.Bind(&table_custservice); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "格式异常", err.Error())
		return
	}

	if user_info = self.Users.Existed(table_custservice.UserId); user_info == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "无效地址", nil)
		return
	}

	if custservice_info = self.Custservices.ExistedByUsername(user_info.Id, table_custservice.Username); custservice_info == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户名不存在", nil)
		return
	}

	if custservice_info.Password != table_custservice.Password {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "密码错误", nil)
		return
	}

	if token, err := utils.TokenRelease(&define.HTTP_SERVER1_AUTH_PASSWORD, 0, custservice_info.Id.Hex()+" "+plugin.NewObjectID().Hex()+" "+custservice_info.Password); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "请求失败", err.Error())
		return

	} else {
		plugin.HttpSuccess(ctx, plugin.Bson{
			"username": custservice_info.Username,
			"token":    token,
			"id":       custservice_info.Id,
		})
	}
}

func (ctrler *ctrler_custservice) Message(ctx *gin.Context) {

	var (
		custservice_info *model.Custservice

		uids_str       = ctx.Query("uids_str")
		read_time, err = strconv.ParseInt(ctx.Query("read_time"), 10, 64)
		robot_uid      = utils.StringToInt(ctx.Query("robot_uid"))
	)

	if custservice_info = BindCustserviceInfo(ctx); custservice_info == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户信息提取异常，请重试", nil)
		return
	}

	if err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "格式异常", err.Error())
		return
	}

	plugin.HttpSuccess(ctx, self.Robots.Message(&custservice_info.UserId, &custservice_info.Id, &uids_str, &read_time, robot_uid, nil))
}

func (ctrler *ctrler_custservice) ReadMessage(ctx *gin.Context) {

	var (
		robot_id, _   = primitive.ObjectIDFromHex(ctx.Query("robot_id"))
		friend_uid, _ = strconv.Atoi(ctx.Query("friend_uid"))

		custservice_info *model.Custservice
		robot            *model.Robot
	)

	if custservice_info = BindCustserviceInfo(ctx); custservice_info == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户信息提取异常，请重试", nil)
		return
	}

	if robot = self.Robots.Existed(robot_id); robot == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人不存在", nil)
		return
	}

	if robot.CustserviceId == nil || *robot.CustserviceId != custservice_info.Id {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "无权限", nil)
		return
	}

	for index, message := range robot.Messages {

		if message.Uid == friend_uid {

			if err := robot.Update(plugin.Bson{
				"messages." + strconv.Itoa(index) + ".unread": 0,
			}); err != nil {
				plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "请求失败", err.Error())
				return
			}

			message.Unread = 0

			break
		}
	}

	plugin.HttpSuccess(ctx, nil)
}

func (ctrler *ctrler_custservice) MessageHistory(ctx *gin.Context) {

	var (
		custservice_info *model.Custservice
		filter           model.RobotMessageFilter

		storage = model.RobotMessageHistoryStorage{}

		count int64
	)

	if custservice_info = BindCustserviceInfo(ctx); custservice_info == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户信息提取异常，请重试", nil)
		return
	}

	if err := ctx.BindQuery(&filter); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "格式异常", err.Error())
		return
	}

	// [PATCH] 直接从 InfluxDB 读取，跳过 MessageCheckUpdate 检查
	if count, err := storage.Read(ini.Influx1, custservice_info.UserId, &filter); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "请求失败", err.Error())
		return
	} else {
		_ = count
	}

	plugin.HttpSuccess(ctx, plugin.Bson{
		"count":   count,
		"storage": &storage,
	})
}

func (ctrler *ctrler_custservice) SendMessage(ctx *gin.Context) {

	var (
		robot_id, _   = primitive.ObjectIDFromHex(ctx.Query("robot_id"))
		friend_uid, _ = strconv.Atoi(ctx.Query("friend_uid"))
		mode          = ctx.Query("mode")

		custservice_info *model.Custservice
		robot            *model.Robot

		file       multipart.File
		file_bytes []byte
		err        error

		result model.RobotSendMsgResult
	)

	if custservice_info = BindCustserviceInfo(ctx); custservice_info == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户信息提取异常，请重试", nil)
		return
	}

	if robot = self.Robots.Existed(robot_id); robot == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人不存在", nil)
		return
	}

	if robot.CustserviceId == nil || *robot.CustserviceId != custservice_info.Id {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "无权限", nil)
		return
	}

	if file, _, err = ctx.Request.FormFile("file"); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "格式异常", err.Error())
		return
	}
	defer file.Close()

	if file_bytes, err = io.ReadAll(file); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "格式异常", err.Error())
		return
	}

	switch mode {

	case define.ROBOT_MATERIAL_MODE_TEXT:
		{
			// [PATCH] 自有风控 - 拦截
			if self.RiskManagement.Contains(string(file_bytes)) {
				logrus.WithFields(logrus.Fields{"CustserviceId": robot.CustserviceId}).Warn("客服发送的消息触发自有风控")
				// 拦截计数（返回更新后的文档）
				var updatedDoc bson.M
				ini.Mongo1.Database.Collection(define.MONGO_COLLECTION_CUSTSERVICE).FindOneAndUpdate(
					context.Background(),
					bson.M{"_id": robot.CustserviceId},
					bson.M{"$inc": bson.M{"risk_block_count": 1}, "$set": bson.M{"risk_block_last": time.Now().Unix()}},
					options.FindOneAndUpdate().SetReturnDocument(options.After),
				).Decode(&updatedDoc)
				blockCount := int64(0)
				if cnt, ok := updatedDoc["risk_block_count"]; ok {
					blockCount = cast.ToInt64(cnt)
				}
				msg := fmt.Sprintf("\"%s\"触发了风控 拦截+%d", string(file_bytes), blockCount)
				plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, msg, nil)
				return
			}

			// [PATCH] 易盾文本风控 - 拦截（服务异常时放行）
			resultResp := plugin.CheckText(string(file_bytes))
			if resultResp.Code == 200 && resultResp.Data.Suggestion > 0 {
				logrus.WithFields(logrus.Fields{"result": resultResp, "CustserviceId": robot.CustserviceId}).Warn("客服发送的消息包含风控敏感词，已拦截")
				// 拦截计数（返回更新后的文档）
				var updatedDoc bson.M
				ini.Mongo1.Database.Collection(define.MONGO_COLLECTION_CUSTSERVICE).FindOneAndUpdate(
					context.Background(),
					bson.M{"_id": robot.CustserviceId},
					bson.M{"$inc": bson.M{"risk_block_count": 1}, "$set": bson.M{"risk_block_last": time.Now().Unix()}},
					options.FindOneAndUpdate().SetReturnDocument(options.After),
				).Decode(&updatedDoc)
				blockCount := int64(0)
				if cnt, ok := updatedDoc["risk_block_count"]; ok {
					blockCount = cast.ToInt64(cnt)
				}
				hitWord := resultResp.Data.HitKeyword
				if hitWord == "" {
					hitWord = string(file_bytes)
				}
				msg := fmt.Sprintf("\"%s\"触发了三方风控 拦截+%d", hitWord, blockCount)
				plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, msg, nil)
				return
			} else if resultResp.Code != 200 {
				logrus.WithFields(logrus.Fields{"result": resultResp}).Warn("易盾文本风控服务异常，放行消息")
			}

			result, err = robot.SendMsgText(friend_uid, string(file_bytes))
		}

	case define.ROBOT_MATERIAL_MODE_IMAGE:
		{
			// [PATCH] 易盾图片风控 - 仅记录日志，不拦截
			resultResp := plugin.CheckImage(string(file_bytes), robot.CustserviceId.Hex())
			if resultResp.Code == 200 && resultResp.Data.Suggestion > 0 {
				logrus.WithFields(logrus.Fields{"result": resultResp, "CustserviceId": robot.CustserviceId}).Warn("客服发送的图片风控异常（仅记录）")
			} else if resultResp.Code != 200 {
				logrus.WithFields(logrus.Fields{"result": resultResp}).Warn("易盾图片风控服务异常（仅记录）")
			}

			// [PATCH] 从图片 bytes 读取实际宽高
			imgWidth, imgHeight := 0, 0
			if imgCfg, _, decErr := image.DecodeConfig(bytes.NewReader(file_bytes)); decErr == nil {
				imgWidth = imgCfg.Width
				imgHeight = imgCfg.Height
			} else {
				logrus.WithFields(logrus.Fields{"err": decErr}).Warn("无法解析图片宽高，使用默认值0")
			}
			result, err = robot.SendMsgImage(friend_uid, imgWidth, imgHeight, file_bytes)
		}

	case define.ROBOT_MATERIAL_MODE_AUDIO:
		result, err = robot.SendMsgVoice(friend_uid, file_bytes)

	case define.ROBOT_MATERIAL_MODE_VIDEO:
		result, err = robot.SendMsgVideo(friend_uid, 0, 0, file_bytes)

	case define.ROBOT_MATERIAL_MODE_GROUPLINK:
		result, err = robot.SendMsgGroupLink(friend_uid, string(file_bytes))
	}

	if err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "发送失败", err.Error())
		return
	} else if result.Result != 0 {
		plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "发送失败", fmt.Sprintf("[%d] %s", result.Result, result.ErrMsg))
		return
	}

	if robot.Status.Message.Time != 0 {
		robot.Status.Message.Timer = time.Now().Unix() - 1
	}

	// data := map[string]any{"sendTime": result.SendTime, "suggestion": result.Suggestion}

	// [PATCH] 发出消息写入 InfluxDB，客服聊天界面显示发送的消息
	if robot.Submit.Uid != 0 {
		outStorage := model.RobotMessageHistoryStorage{}
		outStorage.AddPoint([]*model.RobotMessageValue{
			{
				Channel: model.ROBOT_MESSAGE_CHANNEL.PERSION,
				Type:    mode,
				From:    robot.Submit.Uid,
				To:      friend_uid,
				Time:    time.Now().UnixMilli(),
				Data:    string(file_bytes),
			},
		})
		go outStorage.Write(ini.Influx1, custservice_info.UserId)
	}

	plugin.HttpDefault(ctx, plugin.REQUEST_SUCCESS, "发送成功", result.SendTime)
}

func (ctrler *ctrler_custservice) FriendNotices(ctx *gin.Context) {
	var (
		exist           bool
		robotId         string
		oRobotId        primitive.ObjectID
		custserviceInfo *model.Custservice
		robot           *model.Robot
		result          *model.RobotFriendNoticesResult
		err             error
	)

	if custserviceInfo = BindCustserviceInfo(ctx); custserviceInfo == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户信息提取异常，请重试", nil)
		return
	}

	if robotId, exist = ctx.GetQuery("robot_id"); !exist {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人ID参数必须上传", nil)
		return
	}

	if oRobotId, err = primitive.ObjectIDFromHex(robotId); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "上传的机器人ID参数格式错误", nil)
		return
	}

	robot = self.Robots.Existed(oRobotId)
	if robot == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人信息不存在", nil)
		return
	}

	if result, err = robot.FriendNotices(); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, err.Error(), nil)
		return
	}

	response := &model.RobotFriendNoticesResponse{
		LatestFriendSeq:    result.LatestFriendSeq,
		LatestGroupSeq:     result.LatestGroupSeq,
		FollowingFriendSeq: result.FollowingFriendSeq,
		FriendNotices:      make([]*model.FriendNotice, 0),
		MsgDisplay:         result.MsgDisplay,
		Over:               result.Over,
	}

	for _, msg := range result.FriendMsgs {
		if msg.Msg == nil {
			continue
		}

		response.FriendNotices = append(response.FriendNotices, &model.FriendNotice{
			MsgType: msg.MsgType,
			MsgSeq:  msg.MsgSeq,
			MsgTime: msg.MsgTime,
			ReqUin:  msg.ReqUin,
			Msg: &model.Msg{
				SubType:       msg.Msg.SubType,
				MsgTitle:      msg.Msg.MsgTitle,
				MsgDescribe:   msg.Msg.MsgDecided,
				MsgAdditional: msg.Msg.MsgAdditional,
				MsgSource:     msg.Msg.MsgSource,
				MsgDecided:    msg.Msg.MsgDecided,
				SrcId:         msg.Msg.SrcId,
				SubSrcId:      msg.Msg.SubSrcId,
				Relation:      msg.Msg.Relation,
				ReqUinFaceid:  msg.Msg.ReqUinFaceid,
				ReqUinNick:    msg.Msg.ReqUinNick,
				MsgDetail:     msg.Msg.MsgDetail,
				ReqUinGender:  msg.Msg.ReqUinGender,
				ReqUinAge:     msg.Msg.ReqUinAge,
			},
		})
	}

	plugin.HttpSuccess(ctx, response)

	return
}

func (ctrler *ctrler_custservice) FriendPass(ctx *gin.Context) {
	var (
		custserviceInfo *model.Custservice
		request         model.RobotFriendPassRequest
		robot           *model.Robot
		oRobotId        primitive.ObjectID
		result          *model.RobotFriendPassResult
		err             error
	)

	if custserviceInfo = BindCustserviceInfo(ctx); custserviceInfo == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "用户信息提取异常，请重试", nil)
		return
	}

	if err = ctx.ShouldBind(&request); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请求参数错误", nil)
		return
	}

	if oRobotId, err = primitive.ObjectIDFromHex(request.RobotId); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请求机器人ID参数格式错误", nil)
		return
	}

	robot = self.Robots.Existed(oRobotId)
	if robot == nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人信息不存在", nil)
		return
	}

	if result, err = robot.FriendPass(cast.ToString(request.ReqUin), cast.ToString(request.SrcId), cast.ToString(request.SubSrcId)); err != nil {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, err.Error(), nil)
		return
	}

	if result.Head.Result == -1 {
		plugin.HttpDefault(ctx, plugin.REQUEST_BAD, fmt.Sprintf("通过好友响应错误,code:%d,message:%s", result.Head.Result, result.Head.MsgFail), nil)
		return
	}

	plugin.HttpSuccess(ctx, plugin.Bson{})

	return
}
