package ctrler

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"
	"ymlink-q2/apps/server/self"
	"ymlink-q2/define"
	"ymlink-q2/model"
	"ymlink-q2/plugin"
	pb "ymlink-q2/tencent/protobuf_tars/protobuf/gen"
	"ymlink-q2/utils"

	"google.golang.org/protobuf/proto"
)

func (ctrler *ctrler_robot) message_init() {

	ctrler.message_handles = []func(*model.Robot) (bool, error){

		ctrler.messsage_handle,
		ctrler.messsage_info_handle,
	}
}

func (ctrler *ctrler_robot) messagging() {

	for _, robot := range self.Robots {

		if robot.Deleted || robot.Cache.Ismessage || time.Now().UnixMilli() < robot.Cache.Messagetime {
			continue
		}

		if robot.Stop {
			continue
		}

		if robot.Status.Kernel == nil || robot.Status.Kernel.Code != define.ROBOT_STATUS_SYSTEM_SUCC {
			continue
		}

		if robot.Status.Login == nil || robot.Status.Login.Code != define.ROBOT_STATUS_SYSTEM_SUCC {
			continue
		}

		if robot.Status.Proxy.Code != define.ROBOT_STATUS_SYSTEM_SUCC {
			continue
		}

	WAIT:
		for idx, number := 0, 0; idx < len(self.Robots); idx++ {

			if !self.Robots[idx].Cache.Ismessage {
				continue
			}

			if number++; number >= define.SYSTEM_ROBOT_MESSAGE_LIMIT {
				time.Sleep(100 * time.Millisecond)
				goto WAIT
			}
		}

		robot.Cache.Ismessage = true

		go func(robot *model.Robot) (done bool, err error) {

			defer func() {

				if info := recover(); info != nil {
					robot.Cache.Error = fmt.Sprint(utils.PrintDateTime(time.Now().Unix()), "异常恢复:", info)
				}

				if err != nil {
					robot.Cache.Messagetime = time.Now().UnixMilli() + 1000
				} else if done {
					robot.Cache.Messagetime = time.Now().UnixMilli() + 100
				} else {
					robot.Cache.Messagetime = time.Now().UnixMilli() + 100
				}

				robot.Cache.Ismessage = false
			}()

			for _, handle := range ctrler.message_handles {

				if done, err = handle(robot); done || err != nil {
					return
				}
			}

			return

		}(robot)
	}
}

func (ctrler *ctrler_robot) messsage_handle(robot *model.Robot) (done bool, err error) {

	status := &robot.Status.Message

	// [PATCH] 若 status.Time 是未来时间戳（服务重启后从 MongoDB 加载了新机器人数据），
	// 立即重置，防止 time.Now()-status.Time 为负数导致 fallback 永远不触发
	if status.Time > time.Now().Unix()+60 {
		slog.Debug("Robot", "step", "message_time_reset", "uid", robot.Kernel.UserLoginData.Uin, "old_time", status.Time)
		status.Time = time.Now().Unix() - 60
		status.Timer = status.Time
		robot.Update(plugin.Bson{"status.message": status})
	}

	timer := status.Timer

	if status.Time != 0 && status.Time == timer {
		// [PATCH] Periodic fallback: force sync every 30 seconds even without WebSocket push
		if time.Now().Unix()-status.Time > 30 {
			slog.Debug("Robot", "step", "message_fallback_sync", "uid", robot.Kernel.UserLoginData.Uin, "time", status.Time, "timer", status.Timer)
			status.Timer = time.Now().Unix() - 1
			timer = status.Timer
		} else {
			return
		}
	}

	done = true

	var (
		result          pb.GetMsgResp
		update_value    = plugin.Bson{}
		exited_new      bool
		end_time        int64
		storage_message *ctrler_robot_storage_message
	)

	ctrler.storage_mutex.Do(false, func() {
		storage_message = ctrler.storage_message[robot.UserId]
	})

	if storage_message == nil {
		return
	}

	result, err = robot.GetMessage(status.Record)

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Value = err.Error()

	case result.GetResult() != 0:

		status.Code = int(*result.Result)
		status.Value = fmt.Sprintf("[%d]:%s", result.GetResult(), result.GetErrMsg())

	default:

		for _, item := range result.Msg {

			values := []*model.RobotMessageValue{}

			for _, msg := range item.Msg {

				if end_time == 0 || int64(msg.Head.GetMsgTime()) > end_time {
					end_time = int64(msg.Head.GetMsgTime())
				}

				if msg.Head.GetFromUin() > 1_0000000000000 {
					continue
				}

				value := model.RobotMessageValue{
					Channel: model.ROBOT_MESSAGE_CHANNEL.PERSION,
					Type:    define.ROBOT_MESSAGE_TYPE_UNKNOW,
					From:    int(msg.Head.GetFromUin()),
					To:      int(msg.Head.GetToUin()),
					Time:    int64(msg.Head.GetMsgTime())*1000 + int64(msg.Head.GetMsgSeq())%1000,
					Data:    "null",
				}

				if value.From == value.To {
					continue
				}

				// if !robot.CheckMessage(int(item.GetPeerUin()), &value) {
				// 	continue
				// }

				switch msg.GetHead().GetMsgType() {

				case 529: // 离线文件

					file_extra := new(pb.FileExtra)

					if err := proto.Unmarshal(msg.GetBody().GetMsgContent(), file_extra); err != nil {
						value.Type = define.ROBOT_MESSAGE_TYPE_ERROR
						value.Data = err.Error()
					}

					value.Type = define.ROBOT_MESSAGE_TYPE_FILE_NOTONLINE
					value.Data = utils.ToJson2(file_extra.GetFile())
				}

				for _, elem := range msg.Body.RichText.GetElems() {

					switch {

					case elem.GetText() != nil:

						value.Type = define.ROBOT_MESSAGE_TYPE_TEXT
						value.Data = elem.GetText().GetStr()

					case elem.GetLightApp() != nil:

						value.Type = define.ROBOT_MESSAGE_TYPE_LINK
						value.Data = string(elem.GetLightApp().GetData())

					case elem.GetNotOnlineImage() != nil:

						value.Type = define.ROBOT_MESSAGE_TYPE_IMAGE_NOTONLINE
						value.Data = utils.ToJson2(elem.GetNotOnlineImage())

					case elem.GetCommonElem() != nil:

						common_elem := elem.GetCommonElem()

						switch common_elem.GetServiceType() {

						case 48:

							msg_info := new(pb.EleMsgInfo)

							if err := proto.Unmarshal(common_elem.GetPbElem(), msg_info); err != nil {
								value.Type = define.ROBOT_MESSAGE_TYPE_ERROR
								value.Data = err.Error()
							}

							switch common_elem.GetBusinessType() {
							case 10:
								value.Type = define.ROBOT_MESSAGE_TYPE_IMAGE
							case 20:
								value.Type = define.ROBOT_MESSAGE_TYPE_IMAGES
							case 11:
								value.Type = define.ROBOT_MESSAGE_TYPE_VIDEO
							case 12:
								value.Type = define.ROBOT_MESSAGE_TYPE_AUDIO
							default:
								value.Type = define.ROBOT_MESSAGE_TYPE_FILE
							}

							value.Data = utils.ToJson2(msg_info)
						}
					}
				}

				values = append(values, &value)
			}

			if len(values) == 0 {
				continue
			}

			if index, message := robot.SetMessage(int(item.GetPeerUin()), values); message == nil {
				continue

			} else if index != -1 {
				update_value["messages."+strconv.Itoa(index)] = message

			} else {
				robot.Push(plugin.Bson{"messages": message})
				exited_new = true
			}

			storage_message.mutex.Do(true, func() {
				storage_message.history.AddPoint(values)
			})
		}

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Value = ""

		if end_time != 0 {

			if status.Record >= end_time {
				status.Record += 1
			} else {
				status.Record = end_time + 3
			}
		}

		if *result.SyncFlag == 2 {

			if status.Record == 0 {
				status.Record = time.Now().Unix()
			}

			if timer == status.Timer {

				if status.Timer == 0 {
					status.Timer = status.Record * 1000
				}

				status.Time = status.Timer

			} else {
				status.Time = timer
			}
		}
	}

	update_value["status.message"] = status

	robot.Update(update_value)

	if exited_new {
		_, err = ctrler.messsage_info_handle(robot)
	}

	return
}

func (ctrler *ctrler_robot) messsage_info_handle(robot *model.Robot) (done bool, err error) {

	var (
		profile_uids = []int{}
		nick_uids    = []int{}
	)

	for _, message := range robot.Messages {

		if message.Profile == "" {
			profile_uids = append(profile_uids, message.Uid)
		}

		if message.Nick == "" {
			nick_uids = append(nick_uids, message.Uid)
		}
	}

	if len(profile_uids) == 0 && len(nick_uids) == 0 {
		return
	}

	done = true

	if len(profile_uids) > 0 {

		var result model.RobotProfileResult

		if result, err = robot.GetProfile(profile_uids); err != nil {
			return
		}

		for _, user_data := range result.MsgUserdata {

			for index, message := range robot.Messages {

				if user_data.FUin == message.Uid {

					profile := " "

					for _, data := range user_data.MsgData.ValuesIn {
						profile = data.Value.Url + "140"
					}

					message.Profile = profile

					robot.Update(plugin.Bson{
						"messages." + strconv.Itoa(index) + ".profile": message.Profile,
					})
				}
			}
		}
	}

	for _, nick_uid := range nick_uids {

		var result model.RobotPersonalInfoResult

		if result, err = robot.GetPersonalInfo(nick_uid); err != nil {
			return
		}

		for index, message := range robot.Messages {

			if message.Uid == nick_uid {

				message.Nick = result.Nick

				robot.Update(plugin.Bson{
					"messages." + strconv.Itoa(index) + ".nick": message.Nick,
				})
			}
		}
	}

	return
}
