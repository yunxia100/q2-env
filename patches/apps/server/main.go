package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"

	"regexp"
	"runtime/debug"
	"strconv"
	"time"
	"ymlink-q2/apps/server/ctrler"
	"ymlink-q2/apps/server/self"
	"ymlink-q2/define"
	"ymlink-q2/ini"
	"ymlink-q2/model"
	"ymlink-q2/plugin"
	"ymlink-q2/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	// 初始化日志
	InitLogger()

	utils.SetMen(define.SYSTEM_MEM_LIMIT)
	utils.SetCpu(define.SYSTEM_CPU_LIMIT)
	utils.SetLog(define.SYSTEM_LOG_LEVEL)

	utils.Setup(ini.MONGO1(nil))
	utils.Setup(ini.INFLUX1(nil))
	utils.Setup(ini.FRIENDB1(nil))
	utils.Setup(ini.IP2REGION1(nil))

	utils.Setup(self.Mobiles.Loading)
	utils.Setup(self.Users.Loading)
	utils.Setup(self.Custservices.Loading)
	utils.Setup(self.Proxys.Loading)
	utils.Setup(self.ProxyExtractors.Loading)
	utils.Setup(self.RobotLabels.Loading)
	utils.Setup(self.RobotBatchs.Loading)
	utils.Setup(self.Robots.Loading)
	utils.Setup(self.Webrobots.Loading)
	utils.Setup(self.RobotMaterials.Loading)
	utils.Setup(self.TaskGreetRules.Loading)
	utils.Setup(self.TaskGreetWords.Loading)
	utils.Setup(self.TaskQzoneRemarkRules.Loading)
	utils.Setup(self.TaskUsedbs.Loading)
	utils.Setup(self.TaskQzonedbs.Loading)
	utils.Setup(self.TaskMaterialdbs.Loading)
	utils.Setup(self.TaskRealinfodbs.Loading)
	utils.Setup(self.Tasks.Loading)
	utils.Setup(self.QQGroupTasks.Loading)
	utils.Setup(self.QQGroupMaterials.Loading)

	utils.Setup(ctrler.Mobile.Start)
	utils.Setup(ctrler.ProxyExtractor.Start)
	utils.Setup(ctrler.RobotBatch.Start)
	utils.Setup(ctrler.RobotFriend.Start)
	utils.Setup(ctrler.RobotDrive.Start)
	utils.Setup(ctrler.Robot.Start)
	utils.Setup(ctrler.Task.Start)
	utils.Setup(ctrler.QQGroup.Start)

	utils.Setup(ini.HTTPSERVER1(func(API *gin.RouterGroup) {

		API.Use(ctrler.User.Auth())

		DEBUG := API.Group("/debug")
		{
			DEBUG.GET("/proxy", debug_proxy)
			DEBUG.GET("/ip", debug_ip)
			DEBUG.GET("/table_statistic", ctrler.Robot.TableStatistic)
			DEBUG.GET("/robot_error", debug_robot_error)
		}

		Mobile := API.Group("/mobile")
		{
			Mobile.GET("/fetch", ctrler.Mobile.Fetch)
			Mobile.GET("/register", ctrler.Mobile.Register)
			Mobile.GET("/keepalive", ctrler.Mobile.Keepalive)
		}

		USER := API.Group("/user")
		{
			USER.POST("/create", ctrler.User.Create)
			USER.DELETE("/delete", ctrler.User.Delete)
			USER.POST("/update", ctrler.User.Update)
			USER.GET("/fetch", ctrler.User.Fetch)
			USER.GET("/status", ctrler.User.Status)
			USER.GET("/disabled_robot", ctrler.User.DisabledRobot)
			USER.GET("/environment", ctrler.User.Environment)
			USER.GET("/progress", ctrler.User.Progress)
			// not auth
			USER.POST("/signin", ctrler.User.Singin)
		}

		CUSTSERVICE := API.Group("/custservice")
		{
			CUSTSERVICE.POST("/create", ctrler.Custservice.Create)
			CUSTSERVICE.DELETE("/delete", ctrler.Custservice.Delete)
			CUSTSERVICE.POST("/update", ctrler.Custservice.Update)
			CUSTSERVICE.GET("/fetch", ctrler.Custservice.Fetch)

			API := CUSTSERVICE.Group("/api")
			{
				API.Use(ctrler.Custservice.Auth())

				API.GET("/message", ctrler.Custservice.Message)
				API.GET("/read_message", ctrler.Custservice.ReadMessage)
				API.POST("/send_message", ctrler.Custservice.SendMessage)
				API.GET("/message_history", ctrler.Custservice.MessageHistory)
				API.GET("friend_notices", ctrler.Custservice.FriendNotices)
				API.POST("friend_pass", ctrler.Custservice.FriendPass)
				// not auth
				API.POST("/signin", ctrler.Custservice.Singin)
			}
		}

		PROXY := API.Group("/proxy")
		{
			PROXY.POST("/create", ctrler.Proxy.Create)
			PROXY.DELETE("/delete", ctrler.Proxy.Delete)
			PROXY.DELETE("/recover", ctrler.Proxy.Recover)
			PROXY.POST("/update", ctrler.Proxy.Update)
			PROXY.GET("/fetch", ctrler.Proxy.Fetch)
			PROXY.GET("/get_region", ctrler.Proxy.GetRegion)
			PROXY.GET("/status", ctrler.Proxy.Status)

			EXTRACTOR := PROXY.Group("/extractor")
			{
				EXTRACTOR.POST("/create", ctrler.ProxyExtractor.Create)
				EXTRACTOR.GET("/fetch", ctrler.ProxyExtractor.Fetch)

				// [PATCH] 诊断接口 - 查看proxy extractor缓存槽位状态
				EXTRACTOR.GET("/debug_status", func(ctx *gin.Context) {
					var result []map[string]interface{}
					for _, pe := range self.ProxyExtractors {
						slots := []map[string]interface{}{}
						if pe.Config.Ipzan != nil {
							for i, p := range pe.Config.Ipzan.Cache.Proxys {
								if p != nil {
									slots = append(slots, map[string]interface{}{
										"index":    i,
										"ip":       p.Config.Ip,
										"port":     p.Config.Port,
										"username": p.Config.Username,
										"disabled": p.Disabled,
										"expired":  p.Config.Expired,
									})
								}
							}
						}
						result = append(result, map[string]interface{}{
							"id":        pe.Id.Hex(),
							"name":      pe.Name,
							"brand":     pe.Brand,
							"isrun":     pe.Cache.Isrun,
							"slots":     slots,
							"slot_count": len(slots),
						})
					}
					plugin.HttpSuccess(ctx, result)
				})
			}
		}

		WEBROBOT := API.Group("/webrobot")
		{
			WEBROBOT.GET("/fetch", ctrler.Webrobot.Fetch)
			WEBROBOT.DELETE("/delete", ctrler.Webrobot.Delete)
			// not auth
			WEBROBOT.POST("/create", ctrler.Webrobot.Create)
		}

		ROBOT := API.Group("/robot")
		{
			ROBOT.GET("/test", ctrler.Robot.Test)
			ROBOT.GET("/fetch", ctrler.Robot.Fetch)
			ROBOT.GET("/fetch_by_filter", ctrler.Robot.FetchByFilter)
			ROBOT.POST("/status", ctrler.Robot.Status)
			ROBOT.GET("/copy", ctrler.Robot.Copy)
			ROBOT.GET("/drive", ctrler.Robot.Drive)
			ROBOT.GET("/message", ctrler.Robot.Message)
			ROBOT.GET("/read_message", ctrler.Robot.ReadMessage)
			ROBOT.POST("/send_message", ctrler.Robot.SendMessage)
			ROBOT.GET("/message_history", ctrler.Robot.MessageHistory)
			ROBOT.GET("/statistic", ctrler.Robot.Statistic)
			ROBOT.GET("/reload_pack", ctrler.Robot.ReloadPack)
			ROBOT.POST("/reload_status", ctrler.Robot.ReloadStatus)
			ROBOT.POST("/update_drive", ctrler.Robot.UpdateDrive)
			ROBOT.POST("/update_account", ctrler.Robot.UpdateAccount)
			ROBOT.POST("/update_label", ctrler.Robot.UpdateLabel)
			ROBOT.POST("/set_ban", ctrler.Robot.SetBan)
			ROBOT.POST("/update_temp_batch", ctrler.Robot.UpdateTempBatch)
			ROBOT.POST("/update_stop", ctrler.Robot.UpdateStop)
			ROBOT.POST("/update_software_version", ctrler.Robot.UpdateSoftwareVersion)
			ROBOT.GET("/scan_login", ctrler.Robot.ScanLogin)
			ROBOT.POST("/update_daily", ctrler.Robot.UpdateDaily)
			ROBOT.POST("/update_custservice", ctrler.Robot.UpdateCustservice)
			ROBOT.GET("/work", ctrler.Robot.Work)
			ROBOT.POST("/create_work", ctrler.Robot.CreateWork)
			ROBOT.DELETE("/clear_work", ctrler.Robot.ClearWork)
			ROBOT.GET("/reset_work", ctrler.Robot.ResetWork)
			ROBOT.POST("/delete_devicelist", ctrler.Robot.DeleteDevicelist)
			ROBOT.DELETE("/delete", ctrler.Robot.Delete)
			ROBOT.GET("/assistant_login_scan_qrcode", ctrler.Robot.AssistantLoginScanQrcode)
			// not auth
			ROBOT.POST("/create", ctrler.Robot.Create)
			ROBOT.POST("/import_pack", ctrler.Robot.ImportPack)
			ROBOT.GET("/login", ctrler.Robot.Login)
			ROBOT.GET("/register", ctrler.Robot.Register)
			ROBOT.GET("/register_clear", ctrler.Robot.RegisterClear)
			ROBOT.GET("/login_clear", ctrler.Robot.LoginClear)
			ROBOT.GET("/delete_by_batch", ctrler.Robot.DeleteByBatch)
			ROBOT.GET("/proxy_reset", ctrler.Robot.ProxyReset)
			ROBOT.GET("/login_assistant_qrcode", ctrler.Robot.LoginAssistantQrcode)
			ROBOT.GET("/set_password", ctrler.Robot.SetPassword)

			// [PATCH] 获取机器人的群列表
			ROBOT.GET("/group_list", func(ctx *gin.Context) {
				robotIdStr := ctx.Query("robot_id")
				if robotIdStr == "" {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请提供 robot_id", nil)
					return
				}
				robotObjId, err := primitive.ObjectIDFromHex(robotIdStr)
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "robot_id 格式错误", nil)
					return
				}
				robot := self.Robots.Existed(robotObjId)
				if robot == nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人不存在", nil)
					return
				}
				result, err := robot.GetGroupList()
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "获取群列表失败: "+err.Error(), nil)
					return
				}
				plugin.HttpSuccess(ctx, result)
			})

			// [PATCH] 获取某个群的成员列表
			ROBOT.GET("/group_members", func(ctx *gin.Context) {
				robotIdStr := ctx.Query("robot_id")
				groupUidStr := ctx.Query("group_uid")
				groupUinStr := ctx.Query("group_uin")
				if robotIdStr == "" || groupUidStr == "" || groupUinStr == "" {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请提供 robot_id, group_uid, group_uin", nil)
					return
				}
				robotObjId, err := primitive.ObjectIDFromHex(robotIdStr)
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "robot_id 格式错误", nil)
					return
				}
				robot := self.Robots.Existed(robotObjId)
				if robot == nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人不存在", nil)
					return
				}
				groupUid, _ := strconv.Atoi(groupUidStr)
				groupUin, _ := strconv.Atoi(groupUinStr)
				result, err := robot.GetGroupMenber(groupUid, groupUin, nil)
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "获取群成员失败: "+err.Error(), nil)
					return
				}
				plugin.HttpSuccess(ctx, result)
			})

			// [PATCH] 单次申请入群 - 选择机器人 + 输入群号 + 验证语 → 发送入群申请
			ROBOT.POST("/join_group", func(ctx *gin.Context) {
				var req struct {
					RobotId   string `json:"robot_id" form:"robot_id"`
					GroupCode string `json:"group_code" form:"group_code"`
					Hello     string `json:"hello" form:"hello"`
				}
				if err := ctx.ShouldBind(&req); err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "参数错误", err.Error())
					return
				}
				req.GroupCode = strings.TrimSpace(req.GroupCode)
				if req.RobotId == "" || req.GroupCode == "" {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请填写机器人ID和群号", nil)
					return
				}
				robotObjId, err := primitive.ObjectIDFromHex(req.RobotId)
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人ID格式错误", nil)
					return
				}
				robot := self.Robots.Existed(robotObjId)
				if robot == nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人不存在", nil)
					return
				}

				groupUid, _ := strconv.Atoi(req.GroupCode)
				if groupUid == 0 {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "群号格式错误，请输入纯数字", nil)
					return
				}

				// 搜索群获取 JoinGroupAuth（驱动必须有 authKey 才能真正入群）
				joinGroupAuth := ""
				groupName := ""
				groupAllow := 0 // 0=需审核, 1=免验证直接入群
				groupFound := false
				searchResult, searchErr := robot.SearchGroup(req.GroupCode, nil)
				if searchErr == nil && searchResult.ResultCode == 0 && searchResult.ItemGroups != nil {
					groupCodeStr := strconv.Itoa(groupUid)
					for _, groups := range *searchResult.ItemGroups {
						for _, item := range groups.ResultItems {
							itemId := strings.TrimSpace(item.ResultId)
							if itemId == groupCodeStr || itemId == req.GroupCode {
								var ext model.RobotSearchResultExtensionGroup
								utils.InterfaceToStruct(item.Extension, &ext)
								joinGroupAuth = ext.JoinGroupAuth
								groupAllow = ext.Allow
								groupName = item.Name
								groupFound = true
								break
							}
						}
						if groupFound {
							break
						}
					}
				}

				// 驱动要求 joinGroupAuth 不能为空才能真正执行入群
				// 若搜索未拿到 authKey，直接报错，避免驱动静默失败（返回 result=0 但实际未加入）
				if joinGroupAuth == "" {
					errMsg := "无法获取群入群凭证(authKey)，请确认群号正确或稍后重试"
					if searchErr != nil {
						errMsg = "搜索群信息失败: " + searchErr.Error()
					} else if !groupFound {
						errMsg = fmt.Sprintf("未找到群 %s，请确认群号正确", req.GroupCode)
					}
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, errMsg, nil)
					return
				}

				// 根据是否有验证语选择入群方式
				var enterResult model.RobotEnterGroupResult
				if req.Hello != "" {
					// 有验证语：带验证语申请入群
					authUrl := "https://qm.qq.com/join?authKey=" + joinGroupAuth
					enterResult, err = robot.EnterGroupSendHello(groupUid, authUrl, req.Hello, nil)
				} else {
					// 无验证语：直接以 search 方式申请入群
					enterResult, err = robot.EnterGroup(groupUid, "search", "", joinGroupAuth, nil)
				}
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "申请入群失败: "+err.Error(), nil)
					return
				}
				if enterResult.Result != 0 {
					hint := enterResult.ErrorString
					if hint == "" {
						switch enterResult.Result {
						case 1:
							hint = "已在群内或申请已发送，请等待审核"
						case 3:
							hint = "机器人无权申请该群"
						default:
							hint = fmt.Sprintf("错误码 %d/%d", enterResult.Result, enterResult.ErrorCode)
						}
					}
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "入群失败："+hint, nil)
					return
				}

				// 根据群的 Allow 字段给出准确提示
				var msg string
				if groupAllow == 1 {
					if groupName != "" {
						msg = fmt.Sprintf("入群申请已提交「%s」，免验证群稍后将自动加入", groupName)
					} else {
						msg = "入群申请已提交，免验证群稍后将自动加入"
					}
				} else {
					if groupName != "" {
						msg = fmt.Sprintf("申请已发送，等待群「%s」管理员审核", groupName)
					} else {
						msg = "申请已发送，等待管理员审核"
					}
				}
				plugin.HttpSuccess(ctx, map[string]interface{}{
					"message":    msg,
					"group_code": req.GroupCode,
					"group_name": groupName,
					"allow":      groupAllow,
				})
			})

			// [PATCH] 群发消息 - 选择机器人 + 选择群 + 输入文本 → 发送群消息
			ROBOT.POST("/send_group_msg", func(ctx *gin.Context) {
				var req struct {
					RobotId   string `json:"robot_id" form:"robot_id"`
					GroupCode int    `json:"group_code" form:"group_code"`
					Text      string `json:"text" form:"text"`
				}
				if err := ctx.ShouldBind(&req); err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "参数错误", err.Error())
					return
				}
				if req.RobotId == "" || req.GroupCode == 0 || req.Text == "" {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请填写机器人ID、群号和消息内容", nil)
					return
				}
				robotObjId, err := primitive.ObjectIDFromHex(req.RobotId)
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人ID格式错误", nil)
					return
				}
				robot := self.Robots.Existed(robotObjId)
				if robot == nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人不存在", nil)
					return
				}

				sendResult, err := robot.SendGroupMsgText(robot.Kernel.UserLoginData.Uin, req.GroupCode, req.Text)
				if err != nil && sendResult.SendTime == 0 {
					plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "发送失败: "+err.Error(), nil)
					return
				}
				if sendResult.Result != 0 {
					errMsg := "发送失败"
					switch sendResult.Result {
					case 120:
						errMsg = "发送失败: 机器人在该群被禁言或受限，请检查群设置（如新成员禁言）"
					case 121:
						errMsg = "发送失败: 被群管理员禁言"
					default:
						errMsg = fmt.Sprintf("发送失败: 协议错误码 %d", sendResult.Result)
					}
					if sendResult.ErrMsg != "" {
						errMsg += " (" + sendResult.ErrMsg + ")"
					}
					plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, errMsg, map[string]interface{}{
						"result":    sendResult.Result,
						"send_time": sendResult.SendTime,
					})
					return
				}
				plugin.HttpSuccess(ctx, map[string]interface{}{
					"message":   "发送成功",
					"send_time": sendResult.SendTime,
					"result":    sendResult.Result,
				})
			})

			// [PATCH] 获取机器人好友列表
			ROBOT.GET("/friend_list", func(ctx *gin.Context) {
				robotIdStr := ctx.Query("robot_id")
				if robotIdStr == "" {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请提供 robot_id", nil)
					return
				}
				robotObjId, err := primitive.ObjectIDFromHex(robotIdStr)
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "robot_id 格式错误", nil)
					return
				}
				robot := self.Robots.Existed(robotObjId)
				if robot == nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人不存在", nil)
					return
				}
				result, err := robot.GetFrineds()
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "获取好友列表失败: "+err.Error(), nil)
					return
				}
				plugin.HttpSuccess(ctx, result)
			})

			// [PATCH] 邀请好友入群 - 支持已有群直接拉人 或 搜索新群先入群再拉人
			ROBOT.POST("/invite_to_group", func(ctx *gin.Context) {
				var req struct {
					RobotId    string `json:"robot_id"`
					GroupCode  int    `json:"group_code"`
					FriendUins []int  `json:"friend_uins"`
					Msg        string `json:"msg"`
					NeedJoin   bool   `json:"need_join"`
					Hello      string `json:"hello"`
				}
				if err := ctx.BindJSON(&req); err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "参数格式错误", err.Error())
					return
				}
				if req.RobotId == "" || req.GroupCode == 0 || len(req.FriendUins) == 0 {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请填写机器人ID、群号和好友QQ号", nil)
					return
				}
				robotObjId, err := primitive.ObjectIDFromHex(req.RobotId)
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人ID格式错误", nil)
					return
				}
				robot := self.Robots.Existed(robotObjId)
				if robot == nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人不存在", nil)
					return
				}

				// 如果需要先入群
				joinMsg := ""
				if req.NeedJoin {
					groupCodeStr := strconv.Itoa(req.GroupCode)
					searchResult, err := robot.SearchGroup(groupCodeStr, nil)
					if err != nil {
						plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "搜索群失败: "+err.Error(), nil)
						return
					}
					if searchResult.ResultCode != 0 {
						plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "搜索群失败: "+searchResult.ErrorMsg, nil)
						return
					}
					joinGroupAuth := ""
					if searchResult.ItemGroups != nil {
						for _, groups := range *searchResult.ItemGroups {
							for _, item := range groups.ResultItems {
								if item.ResultId == groupCodeStr {
									var ext model.RobotSearchResultExtensionGroup
									utils.InterfaceToStruct(item.Extension, &ext)
									joinGroupAuth = ext.JoinGroupAuth
									break
								}
							}
							if joinGroupAuth != "" {
								break
							}
						}
					}
					if joinGroupAuth == "" {
						plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "搜索不到此群，请检查群号", nil)
						return
					}
					var enterResult model.RobotEnterGroupResult
					if req.Hello != "" {
						authUrl := "https://qm.qq.com/join?authKey=" + joinGroupAuth
						enterResult, err = robot.EnterGroupSendHello(req.GroupCode, authUrl, req.Hello, nil)
					} else {
						enterResult, err = robot.EnterGroup(req.GroupCode, "search", "", joinGroupAuth, nil)
					}
					if err != nil {
						plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "入群失败: "+err.Error(), nil)
						return
					}
					if enterResult.Result != 0 {
						plugin.HttpDefault(ctx, plugin.REQUEST_BAD,
							fmt.Sprintf("入群失败！错误码: %d/%d, 提示: %s", enterResult.Result, enterResult.ErrorCode, enterResult.ErrorString), nil)
						return
					}
					joinMsg = "机器人已成功入群，"
					// 入群后等待1秒让服务端同步
					time.Sleep(1 * time.Second)
				}

				pullResult, err := robot.PullGroup(req.GroupCode, req.FriendUins, req.Msg, nil)
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, joinMsg+"邀请好友失败: "+err.Error(), nil)
					return
				}

				plugin.HttpSuccess(ctx, map[string]interface{}{
					"message":    joinMsg + fmt.Sprintf("成功邀请 %d 位好友入群", len(req.FriendUins)),
					"group_code": pullResult.GroupCode,
				})
			})

			LABEL := ROBOT.Group("/label")
			{
				LABEL.GET("/fetch", ctrler.RobotLabel.Fetch)
				LABEL.POST("/create", ctrler.RobotLabel.Create)
				LABEL.POST("/update", ctrler.RobotLabel.Update)
				LABEL.DELETE("/delete", ctrler.RobotLabel.Delete)
			}

			BATCH := ROBOT.Group("/batch")
			{
				BATCH.POST("/create", ctrler.RobotBatch.Create)
				BATCH.GET("/fetch", ctrler.RobotBatch.Fetch)
				BATCH.POST("/update_key", ctrler.RobotBatch.UpdateKey)
				BATCH.DELETE("/delete", ctrler.RobotBatch.Delete)
				// not auth
				BATCH.GET("/info", ctrler.RobotBatch.Info)
				BATCH.GET("/status", ctrler.RobotBatch.Status)

				// [PATCH] 批量账密提交
				// 支持格式:
				//   账号----密码              → 创建新设备 + 登录
				//   账号----密码----objid     → 复用已有设备（避免风控）
				// objid: 底层驱动的5位设备ID，用于重部署后恢复已在线的账号
				BATCH.POST("/account_submit", func(ctx *gin.Context) {
					var req struct {
						Key   string   `json:"key"`
						Lines []string `json:"lines"`
					}
					if err := ctx.BindJSON(&req); err != nil {
						plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "参数格式错误", err.Error())
						return
					}
					if req.Key == "" {
						plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "key不能为空", nil)
						return
					}

					robot_batch := self.RobotBatchs.ExistedByKey(req.Key)
					if robot_batch == nil {
						plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "生成器不存在", nil)
						return
					}
					if robot_batch.Mode != define.ROBOT_BATCH_MODE_ACCOUNT {
						plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "该生成器不是账密模式", nil)
						return
					}

					type SubmitResult struct {
						Line    string `json:"line"`
						Uid     int    `json:"uid"`
						Success bool   `json:"success"`
						Msg     string `json:"msg"`
						Objid   string `json:"objid,omitempty"`
						Reused  bool   `json:"reused,omitempty"`
					}
					var results []SubmitResult

					for _, line := range req.Lines {
						line = strings.TrimSpace(line)
						if line == "" {
							continue
						}

						// 用正则按4个以上连续短横线分割
						sepRe := regexp.MustCompile(`-{4,}`)
						parts := sepRe.Split(line, -1)
						if len(parts) < 2 || len(parts) > 3 {
							results = append(results, SubmitResult{Line: line, Success: false, Msg: "格式错误，需要 账号----密码 或 账号----密码----objid"})
							continue
						}

						uidStr := strings.TrimSpace(parts[0])
						password := strings.TrimSpace(parts[1])
						bindObjid := ""
						if len(parts) == 3 {
							bindObjid = strings.TrimSpace(parts[2])
						}

						uid, err := strconv.Atoi(uidStr)
						if err != nil || uid == 0 {
							results = append(results, SubmitResult{Line: line, Success: false, Msg: "账号格式错误"})
							continue
						}

						if password == "" {
							results = append(results, SubmitResult{Line: line, Uid: uid, Success: false, Msg: "密码不能为空"})
							continue
						}

						// 检查QQ号是否已存在（内存中）
						if existRobot := self.Robots.ExistedByUid(uid); existRobot != nil {
							// 如果提供了新的objid，更新已有记录的设备绑定
							if bindObjid != "" && len(bindObjid) == 5 && existRobot.Kernel.Objid != bindObjid {
								existRobot.Kernel.Objid = bindObjid
								existRobot.Submit.Password = password
								existRobot.Cache.Offline = ""
								// 标记为已登录成功，复用底层驱动的活跃会话
								loginStatus := &model.RobotStatusLogin{}
								loginStatus.Time = time.Now().UnixMilli()
								loginStatus.Code = define.ROBOT_LOGIN_STATUS_SUCC
								existRobot.Status.Login = loginStatus
								if existRobot.Status.RenewOnline == nil {
									existRobot.Status.RenewOnline = &model.RobotStatusCurrent{}
									existRobot.Status.RenewOnline.Time = time.Now().Unix()
									existRobot.Status.RenewOnline.Timer = time.Now().Unix() + define.INTERVAL_RENEW_ONLINE
								}
								if existRobot.Status.RenewSecretKey == nil {
									existRobot.Status.RenewSecretKey = &model.RobotStatusCurrent{}
									existRobot.Status.RenewSecretKey.Time = time.Now().Unix()
									existRobot.Status.RenewSecretKey.Timer = time.Now().Unix() + define.INTERVAL_RENEW_SECRET_KEY
								}
								existRobot.Stop = false
								results = append(results, SubmitResult{Line: line, Uid: uid, Success: true, Objid: bindObjid, Reused: true, Msg: "已更新设备绑定（使用现有会话）"})
							} else {
								results = append(results, SubmitResult{Line: line, Uid: uid, Success: false, Objid: existRobot.Kernel.Objid, Msg: "账号已存在"})
							}
							continue
						}

						var table_robot model.Robot
						table_robot.BatchId = robot_batch.Id
						table_robot.LabelIds = robot_batch.LabelIds

						// 分配驱动服务器URL
						table_robot.DriveUrl = define.DRIVE_GET(define.DRIVE_MAPPING_ITEM{
							Hardware:        robot_batch.Device.Hardware,
							Software:        robot_batch.Device.Software,
							SoftwareVersion: robot_batch.Device.SoftwareVersion,
						})

						reused := false
						if bindObjid != "" && len(bindObjid) == 5 {
							// ====== 复用已有设备（避免风控）======
							// 跳过 CreateKernel，直接使用已有的 objid
							table_robot.Kernel.Objid = bindObjid
							table_robot.Kernel.Hardware = robot_batch.Device.Hardware
							table_robot.Kernel.SoftWare = robot_batch.Device.Software
							table_robot.Kernel.Version = robot_batch.Device.SoftwareVersion

							reused = true
						} else {
							// ====== 创建新设备 ======
							if err := table_robot.CreateKernel(
								robot_batch.Device.Hardware,
								robot_batch.Device.Software,
								robot_batch.Device.SoftwareVersion,
							); err != nil {
								results = append(results, SubmitResult{Line: line, Uid: uid, Success: false, Msg: "创建设备失败: " + err.Error()})
								continue
							}
						}

						// 设置提交信息
						table_robot.Submit.Uid = uid
						table_robot.Submit.Password = password
						table_robot.Status.Register = &model.RobotStatusCurrent{}

						// 如果是复用设备，标记为已登录成功
						// 设备在底层驱动仍有活跃会话，无需重新 pwdLogin（否则会触发滑块）
						// 同时正确设置 RenewSecretKey / RenewOnline 定时器，让后台 handler 正常续期
						if reused {
							table_robot.Kernel.UserLoginData.Uin = uid

							loginStatus := &model.RobotStatusLogin{}
							loginStatus.Time = time.Now().UnixMilli()
							loginStatus.Code = define.ROBOT_LOGIN_STATUS_SUCC
							table_robot.Status.Login = loginStatus

							table_robot.Status.RenewSecretKey = &model.RobotStatusCurrent{}
							table_robot.Status.RenewSecretKey.Time = time.Now().Unix()
							table_robot.Status.RenewSecretKey.Timer = time.Now().Unix() + define.INTERVAL_RENEW_SECRET_KEY

							table_robot.Status.RenewOnline = &model.RobotStatusCurrent{}
							table_robot.Status.RenewOnline.Time = time.Now().Unix()
							table_robot.Status.RenewOnline.Timer = time.Now().Unix() + define.INTERVAL_RENEW_ONLINE

							table_robot.Stop = false
						}

						// 分配代理（安全调用：代理列表为空时 GetRandom 内部会 panic，用 recover 保护）
						var proxySetErr error
						func() {
							defer func() { recover() }()
							if table_proxy := self.Proxys.GetRandom(robot_batch.UserId, nil, "", true); table_proxy != nil {
								if err := table_robot.SetProxy(table_proxy); err != nil {
									proxySetErr = err
									return
								}
								table_robot.ProxyId = table_proxy.Id
							}
						}()
						if proxySetErr != nil {
							results = append(results, SubmitResult{Line: line, Uid: uid, Success: false, Msg: "设置代理失败: " + proxySetErr.Error()})
							continue
						}

						// 保存到数据库
						if err := table_robot.CreateTable(robot_batch.UserId); err != nil {
							results = append(results, SubmitResult{Line: line, Uid: uid, Success: false, Msg: "保存失败: " + err.Error()})
							continue
						}

						self.Robots.Create(&table_robot)
						robot_batch.Cache.RobotUpdateTime = time.Now().UnixMilli()

						msg := "提交成功"
						if reused {
							msg = "提交成功（复用已有设备，使用现有会话）"
						}
						results = append(results, SubmitResult{Line: line, Uid: uid, Success: true, Objid: table_robot.Kernel.Objid, Reused: reused, Msg: msg})
					}

					// 更新用户统计信息
					ctrler.User.Updating(&robot_batch.UserId, nil, define.MONGO_COLLECTION_ROBOT)

					plugin.HttpSuccess(ctx, results)
				})
			}

			// [PATCH] 强制设置机器人已登录状态（复用已有底层驱动会话）
			// POST /api/robot/force_login?robot_id=xxx&objid=yyy
			ROBOT.POST("/force_login", func(ctx *gin.Context) {
				robotIdStr := ctx.Query("robot_id")
				objid := ctx.Query("objid")
				if robotIdStr == "" || objid == "" {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请提供 robot_id 和 objid", nil)
					return
				}
				robotObjId, err := primitive.ObjectIDFromHex(robotIdStr)
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "robot_id 格式错误", nil)
					return
				}
				robot := self.Robots.Existed(robotObjId)
				if robot == nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人不存在", nil)
					return
				}
				// 更新 objid 并标记已登录
				robot.Kernel.Objid = objid
				robot.Cache.Offline = ""
				loginStatus := &model.RobotStatusLogin{}
				loginStatus.Time = time.Now().UnixMilli()
				loginStatus.Code = define.ROBOT_LOGIN_STATUS_SUCC
				robot.Status.Login = loginStatus
				if robot.Status.RenewSecretKey == nil {
					robot.Status.RenewSecretKey = &model.RobotStatusCurrent{}
					robot.Status.RenewSecretKey.Time = time.Now().Unix()
					robot.Status.RenewSecretKey.Timer = time.Now().Unix() + define.INTERVAL_RENEW_SECRET_KEY
				}
				if robot.Status.RenewOnline == nil {
					robot.Status.RenewOnline = &model.RobotStatusCurrent{}
					robot.Status.RenewOnline.Time = time.Now().Unix()
					robot.Status.RenewOnline.Timer = time.Now().Unix() + define.INTERVAL_RENEW_ONLINE
				}
				robot.Stop = false
				plugin.HttpSuccess(ctx, map[string]string{"objid": objid, "msg": "已强制设置登录成功"})
			})

			// [PATCH] 获取好友请求列表（陌生人加好友）
			ROBOT.GET("/friend_notices", func(ctx *gin.Context) {
				robotIdStr := ctx.Query("robot_id")
				if robotIdStr == "" {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请提供 robot_id", nil)
					return
				}
				robotObjId, err := primitive.ObjectIDFromHex(robotIdStr)
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "robot_id 格式错误", nil)
					return
				}
				robot := self.Robots.Existed(robotObjId)
				if robot == nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人不存在", nil)
					return
				}
				result, err := robot.FriendNotices()
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "获取好友请求失败: "+err.Error(), nil)
					return
				}
				// 转换为前端友好的格式
				notices := []map[string]interface{}{}
				for _, msg := range result.FriendMsgs {
					if msg.Msg == nil {
						continue
					}
					notices = append(notices, map[string]interface{}{
						"msg_type":     msg.MsgType,
						"msg_seq":      msg.MsgSeq,
						"msg_time":     msg.MsgTime,
						"req_uin":      msg.ReqUin,
						"nick":         msg.Msg.ReqUinNick,
						"gender":       msg.Msg.ReqUinGender,
						"age":          msg.Msg.ReqUinAge,
						"src_id":       msg.Msg.SrcId,
						"sub_src_id":   msg.Msg.SubSrcId,
						"msg_title":    msg.Msg.MsgTitle,
						"msg_additional": msg.Msg.MsgAdditional,
						"msg_source":   msg.Msg.MsgSource,
						"msg_detail":   msg.Msg.MsgDetail,
					})
				}
				plugin.HttpSuccess(ctx, notices)
			})

			// [PATCH] 通过好友请求
			ROBOT.POST("/friend_pass", func(ctx *gin.Context) {
				var req struct {
					RobotId  string `json:"robot_id" form:"robot_id"`
					ReqUin   int64  `json:"req_uin" form:"req_uin"`
					SrcId    int64  `json:"src_id" form:"src_id"`
					SubSrcId int64  `json:"sub_src_id" form:"sub_src_id"`
				}
				if err := ctx.ShouldBind(&req); err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "参数错误", err.Error())
					return
				}
				if req.RobotId == "" {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "请提供 robot_id", nil)
					return
				}
				robotObjId, err := primitive.ObjectIDFromHex(req.RobotId)
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "robot_id 格式错误", nil)
					return
				}
				robot := self.Robots.Existed(robotObjId)
				if robot == nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD, "机器人不存在", nil)
					return
				}
				result, err := robot.FriendPass(
					strconv.FormatInt(req.ReqUin, 10),
					strconv.FormatInt(req.SrcId, 10),
					strconv.FormatInt(req.SubSrcId, 10),
				)
				if err != nil {
					plugin.HttpDefault(ctx, plugin.REQUEST_SERVER_ERROR, "通过好友失败: "+err.Error(), nil)
					return
				}
				if result.Head.Result == -1 {
					plugin.HttpDefault(ctx, plugin.REQUEST_BAD,
						fmt.Sprintf("通过好友失败: %s", result.Head.MsgFail), nil)
					return
				}
				plugin.HttpSuccess(ctx, map[string]interface{}{
					"message": "已通过好友请求",
				})
			})

			FRIEND := ROBOT.Group("/friend")
			{
				FRIEND.POST("/fetch", ctrler.RobotFriend.Fetch)
				FRIEND.POST("/fetch_value", ctrler.RobotFriend.FetchValue)
				FRIEND.GET("/export", ctrler.RobotFriend.Export)
			}

			MATERIAL := ROBOT.Group("/material")
			{
				MATERIAL.GET("/fetch", ctrler.RobotMaterial.Fetch)
				MATERIAL.POST("/upload", ctrler.RobotMaterial.Upload)
				MATERIAL.GET("/get_grouplink", ctrler.RobotMaterial.GetGroupLink)
				MATERIAL.DELETE("/delete", ctrler.RobotMaterial.Delete)
			}

			MESSAGE := ROBOT.Group("/message")
			{
				MESSAGE.POST("/image", ctrler.Robot.MessageImage)
				MESSAGE.POST("/file", ctrler.Robot.MessageFile)
				MESSAGE.POST("/file_notonline", ctrler.Robot.MessageFileNotonline)
			}
		}

		TASK := API.Group("/task")
		{
			TASK.GET("/fetch", ctrler.Task.Fetch)
			TASK.POST("/create", ctrler.Task.Create)
			TASK.POST("/update", ctrler.Task.Update)
			TASK.POST("/switch", ctrler.Task.Switch)
			TASK.GET("/cache", ctrler.Task.Cache)
			TASK.GET("/log", ctrler.Task.Log)
			TASK.DELETE("/delete", ctrler.Task.Delete)

			CONFIG := TASK.Group("/config")
			{
				QZONEGREET := CONFIG.Group("/qzonegreet")
				{
					QZONEGREET.GET("/thread", ctrler.Task.QzonegreetThread)
					QZONEGREET.GET("/value", ctrler.Task.QzonegreetValue)
				}

				MATERIALGREET := CONFIG.Group("/materialgreet")
				{
					MATERIALGREET.GET("/thread", ctrler.Task.MaterialgreetThread)
					MATERIALGREET.GET("/value", ctrler.Task.MaterialgreetValue)
					MATERIALGREET.GET("/friend", ctrler.Task.MaterialgreetFriend)
					MATERIALGREET.GET("/group", ctrler.Task.MaterialgreetGroup)
					MATERIALGREET.POST("/recover", ctrler.Task.MaterialgreetRecover)
					MATERIALGREET.GET("/create_group", ctrler.Task.MaterialgreetCreateGroup)
					MATERIALGREET.POST("/setting_group", ctrler.Task.MaterialgreetSettingGroup)
					MATERIALGREET.GET("/update_group", ctrler.Task.MaterialgreetUpdateGroup)

				}

				GROUPCHATPULL := CONFIG.Group("/groupchatpull")
				{
					GROUPCHATPULL.GET("/robotdb_value", ctrler.Task.GroupchatpullRobotdbValue)
					GROUPCHATPULL.GET("/groups", ctrler.Task.GroupchatpullGroups)
					GROUPCHATPULL.GET("/create_group", ctrler.Task.GroupchatpullCreateGroup)
					GROUPCHATPULL.POST("/setting_group", ctrler.Task.GroupchatpullSettingGroup)
					GROUPCHATPULL.POST("/import", ctrler.Task.GroupchatpullImportFans)
					GROUPCHATPULL.POST("/setting_robot", ctrler.Task.GroupchatpullSettingRobot)
					GROUPCHATPULL.POST("/setting_fans", ctrler.Task.GroupchatpullSettingFans)
					GROUPCHATPULL.GET("/export_table", ctrler.Task.GroupchatpullExportTable)
				}

				QZONEREMARK := CONFIG.Group("/qzoneremark")
				{
					QZONEREMARK.GET("/thread", ctrler.Task.QzoneremarkThread)
					QZONEREMARK.GET("/value", ctrler.Task.QzoneremarkValue)
				}

				QZONEVISITOR := CONFIG.Group("/qzonevisitor")
				{
					QZONEVISITOR.GET("/thread", ctrler.Task.QzonevisitorThread)
					QZONEVISITOR.GET("/value", ctrler.Task.QzonevisitorValue)
				}

				QSHOWRECOMMEND := CONFIG.Group("/qshowrecommend")
				{
					QSHOWRECOMMEND.GET("/thread", ctrler.Task.QshowrecommendThread)
					QSHOWRECOMMEND.GET("/value", ctrler.Task.QshowrecommendValue)
				}

				GROUPMENBER := CONFIG.Group("/groupmenber")
				{
					GROUPMENBER.GET("/group", ctrler.Task.GroupmenberGroup)
				}
			}

			GREET := TASK.Group("/greet")
			{
				RULE := GREET.Group("/rule")
				{
					RULE.GET("/fetch", ctrler.TaskGreetRule.Fetch)
					RULE.POST("/create", ctrler.TaskGreetRule.Create)
					RULE.POST("/update", ctrler.TaskGreetRule.Update)
				}

				WORD := GREET.Group("/word")
				{
					WORD.GET("/fetch", ctrler.TaskGreetWord.Fetch)
					WORD.POST("/create", ctrler.TaskGreetWord.Create)
					WORD.POST("/update", ctrler.TaskGreetWord.Update)
				}
			}

			QZONE := TASK.Group("/qzone")
			{
				REMARK := QZONE.Group("remark")
				{
					RULE := REMARK.Group("/rule")
					{
						RULE.GET("/fetch", ctrler.TaskQzoneRemarkRule.Fetch)
						RULE.POST("/create", ctrler.TaskQzoneRemarkRule.Create)
						RULE.POST("/update", ctrler.TaskQzoneRemarkRule.Update)
					}
				}
			}

			USEDB := TASK.Group("/usedb")
			{
				USEDB.GET("/fetch", ctrler.Usedb.Fetch)
				USEDB.POST("/create", ctrler.Usedb.Create)
				USEDB.DELETE("/delete", ctrler.Usedb.Delete)
			}

			QZONEDB := TASK.Group("/qzonedb")
			{
				QZONEDB.GET("/fetch", ctrler.Qzonedb.Fetch)
				QZONEDB.POST("/create", ctrler.Qzonedb.Create)
				QZONEDB.DELETE("/delete", ctrler.Qzonedb.Delete)
				QZONEDB.POST("/create_null", ctrler.Qzonedb.CreateNull)
			}

			MATERIALDB := TASK.Group("/materialdb")
			{
				MATERIALDB.GET("/fetch", ctrler.Materialdb.Fetch)
				MATERIALDB.POST("/create", ctrler.Materialdb.Create)
				MATERIALDB.POST("/create_null", ctrler.Materialdb.CreateNull)
				MATERIALDB.GET("/export", ctrler.Materialdb.Export)
				MATERIALDB.DELETE("/delete", ctrler.Materialdb.Delete)
			}

			REALINFODB := TASK.Group("/realinfodb")
			{
				REALINFODB.GET("/fetch", ctrler.Realinfodb.Fetch)
				REALINFODB.POST("/create", ctrler.Realinfodb.Create)
				REALINFODB.DELETE("/delete", ctrler.Realinfodb.Delete)
			}
		}

		QQGroup := API.Group("/qqgroup")
		{
			Task := QQGroup.Group("/task")
			{
				Task.GET("/fetch", ctrler.QQGroupTask.Fetch)
				Task.POST("/create", ctrler.QQGroupTask.Create)
				Task.POST("/update", ctrler.QQGroupTask.Update)
				Task.POST("/enable", ctrler.QQGroupTask.Enable)
				Task.DELETE("/delete", ctrler.QQGroupTask.Delete)
			}
			Material := QQGroup.Group("/material")
			{
				Material.GET("/fetch", ctrler.QQGroupTaterial.Fetch)
				Material.POST("/create", ctrler.QQGroupTaterial.Create)
				Material.POST("/update", ctrler.QQGroupTaterial.Update)
				Material.DELETE("/delete", ctrler.QQGroupTaterial.Delete)
			}
		}

		// 风控 risk management
		risk := API.Group("risk")
		{
			risk.GET("/fetch", ctrler.RiskManagement.Fetch)
			risk.POST("create", ctrler.RiskManagement.Create)
			risk.POST("/update", ctrler.RiskManagement.Update)
			risk.DELETE("/delete", ctrler.RiskManagement.Delete)
		}
	}))
}

// ============================================================
// [PATCH] 日志增强：文件输出 + Gin 请求日志 + Panic 恢复
// ============================================================

func InitLogger() {
	// 创建日志目录
	os.MkdirAll("./log", 0755)

	// 日志文件
	logFile, err := os.OpenFile("./log/ymlink-q2.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("[PATCH] Failed to open log file:", err)
	}

	logrus.SetReportCaller(true)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.DateTime,
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			filename := fmt.Sprintf("%s:%d", path.Base(frame.File), frame.Line)
			return frame.Function, filename
		},
	})

	// [PATCH] 同时输出到控制台和日志文件
	if logFile != nil {
		logrus.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}
}

func main() {
	logrus.WithFields(logrus.Fields{"host": define.HTTP_SERVER1_URL, "logLevel": define.SYSTEM_LOG_LEVEL}).Info("ymlink-q2 running..")

	utils.Loop(time.Hour, func(now time.Time) {
		debug.FreeOSMemory()
	})

	utils.Loop(time.Second, func(now time.Time) {
		system_info()
		// runtime.GC()
	})

	// DEBUG
	// end_timer := 0
	// utils.Loop(time.Second, func(now time.Time) {
	// 	if end_timer++; end_timer > 60*60*3 {
	// 		os.Exit(88)
	// 	}
	// })

	utils.Setup(debug_quest)

	if define.SYSTEM_MODE == define.SYSTEM_MODE_TEST {
		test_handle()
	}

	utils.Exit(func() {
		ini.Friendb1.DisConnect()
	})
}
