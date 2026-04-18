package ctrler

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"ymlink-q2/apps/server/self"
	"ymlink-q2/define"
	"ymlink-q2/ini"
	"ymlink-q2/model"
	"ymlink-q2/plugin"
	"ymlink-q2/utils"
)

func (ctrler *ctrler_robot) run_init() {

	ctrler.run_handles = []func(*model.Robot) (bool, error){

		ctrler.kernel_handle,

		func(robot *model.Robot) (bool, error) {
			return robot.Stop, nil
		},

		ctrler.proxy_check,
		ctrler.renew_secret_key_handle,
		ctrler.renew_online_handle,
		ctrler.info_handle,
		ctrler.profile_handle,
		ctrler.photo_wall_handle,
		ctrler.friends_handle,
		ctrler.applist_handle,
		ctrler.auth_applist_handle,
		ctrler.devicelist_handle,
		ctrler.qzone_permission_handle,
		ctrler.summarycard_view_handle,
		ctrler.signature_history_handle,
		ctrler.qzone_main_handle,
		ctrler.ban_handle,
		// ctrler.dailyclockin,
		// ctrler.dailyapplist,
		ctrler.system_message,
		ctrler.group_list,
		ctrler.work_handle,
	}
}

func (ctrler *ctrler_robot) running(user *model.User) {

	for _, robot := range self.Robots {

		if robot.UserId != user.Id || robot.Deleted || robot.Cache.Isrun || time.Now().UnixMilli() < robot.Cache.Runtime {
			continue
		}

		if !ctrler.login_check(robot) {
			continue
		}

	WAIT:
		for idx, number := 0, 0; idx < len(self.Robots); idx++ {

			if self.Robots[idx].UserId != user.Id || !self.Robots[idx].Cache.Isrun {
				continue
			}

			if number++; number < define.SYSTEM_ROBOT_THREAD_LIMIT {
				continue
			}

			time.Sleep(100 * time.Millisecond)
			goto WAIT
		}

		robot.Step("start")

		robot.Cache.Isrun = true

		go func(robot *model.Robot) (done bool, err error) {

			defer func() {
				if info := recover(); info != nil {
					robot.Cache.Error = fmt.Sprint(utils.PrintDateTime(time.Now().Unix()), "异常恢复:", info)
				}

				if err != nil {

					if ctrler.proxy_is_disabled(robot, err) {
						robot.Cache.Runtime = time.Now().UnixMilli() + 10_000

					} else if strings.Contains(err.Error(), "身份验证失败") {
						robot.Cache.Offline = err.Error()

					} else {
						robot.Cache.Runtime = time.Now().UnixMilli() + 5000
					}

				} else if done {
					robot.Cache.Runtime = time.Now().UnixMilli() + 2000

				} else {
					robot.Cache.Runtime = time.Now().UnixMilli() + 1000
				}

				robot.Cache.Isrun = false
			}()

			for _, handle := range ctrler.run_handles {

				done, err = handle(robot)

				if done || err != nil {
					break
				}
			}

			return

		}(robot)
	}
}

func (ctrler *ctrler_robot) proxy_check(robot *model.Robot) (done bool, err error) {

	defer func() {
		if err != nil {
			robot.Cache.ProxyChangeTimer = time.Now().Unix() + 30
		}
	}()

	robot.Step("proxy_check")

	if time.Now().Unix() < robot.Cache.ProxyChangeTimer {
		done = true
		return
	}

	var (
		proxy     *model.Proxy
		new_proxy *model.Proxy
	)

	if proxy = self.Proxys.Existed(robot.ProxyId); proxy == nil {
		goto CHANGE
	}

	if robot.Status.Proxy.Code == define.ROBOT_STATUS_PROXY_DISABLED {
		goto CHANGE
	}

	if proxy.Disabled || time.Now().Unix() > proxy.Config.Expired {

		robot.Status.Proxy.Code = define.ROBOT_STATUS_PROXY_DISABLED

		robot.Update(plugin.Bson{
			"status.proxy": robot.Status.Proxy,
		})

		goto CHANGE
	}

	if total := self.Robots.ProxyUseTotal(robot.UserId, proxy.Id); total > proxy.Limit {
		goto CHANGE
	}

	return

CHANGE:
	if proxy != nil {
		robot.AddProxyAddress(proxy.Config.Address())
	}

	if new_proxy = self.Proxys.GetRandom(robot.UserId, robot.Cache.ProxyAddresss, robot.Submit.Province, true); new_proxy != nil {
		goto UPDATE
	}

	err = fmt.Errorf("代理已过期，无可用代理。")
	return

UPDATE:

	if err = robot.SetProxy(new_proxy); err != nil {
		err = fmt.Errorf("设置代理异常: %s", err.Error())
		return
	}

	if err = robot.Update(plugin.Bson{"proxy_id": new_proxy.Id}); err != nil {
		err = fmt.Errorf("切换代理异常: %s", err.Error())
		return
	}

	if proxy != nil {

		failed_number := proxy.Status.FailedNumber + 1
		failed_value := robot.Status.Proxy.Value
		failed_time := time.Now().Unix()

		proxy.Update(plugin.Bson{
			"status.failed_number": failed_number,
			"status.failed_value":  failed_value,
			"status.failed_time":   failed_time,
		})

		proxy.Status.FailedNumber = failed_number
		proxy.Status.FailedValue = failed_value
		proxy.Status.FailedTime = failed_time
	}

	robot.Status.Proxy.Code = define.ROBOT_STATUS_PROXY_CHANGING

	robot.ProxyId = new_proxy.Id

	robot.Status.RenewOnline.Timer = time.Now().Unix()

	robot.Update(plugin.Bson{
		"proxy_id":            new_proxy.Id,
		"status.proxy":        robot.Status.Proxy,
		"status.renew_online": robot.Status.RenewOnline,
	})

	return
}

func (ctrler *ctrler_robot) login_check(robot *model.Robot) (result bool) {

	if robot.Submit.Uid == 0 && robot.Kernel.UserLoginData.Uin != 0 {
		robot.Submit.Uid = robot.Kernel.UserLoginData.Uin

		robot.Update(plugin.Bson{
			"submit.uid": robot.Submit.Uid,
		})
	}

	if robot.Status.Login == nil {

		// [PATCH] 若机器人已有有效驱动会话(objid)，服务重启后自动恢复登录态，
		// 防止 status.login 丢失导致消息轮询被永久阻断
		if robot.Kernel.Objid != "" {
			login := &model.RobotStatusLogin{}
			login.Code = define.ROBOT_LOGIN_STATUS_SUCC
			login.Time = time.Now().Unix()
			robot.Status.Login = login
			robot.Update(plugin.Bson{"status.login": login})
			result = true
			return
		}

		if robot.Status.RenewSecretKey != nil || robot.Status.RenewOnline != nil ||
			robot.Status.Info != nil || robot.Status.Profile != nil || robot.Status.Friends != nil {

			status := model.RobotStatus{
				Statistic: robot.Status.Statistic,
			}

			robot.Update(plugin.Bson{
				"status": status,
			})

			robot.Status = status
		}

		return
	}

	if robot.Status.Login.Code == define.ROBOT_LOGIN_STATUS_SUCC {

		if robot.Cache.Offline != "" {

			robot.Update(plugin.Bson{
				"status.login": robot.Status.Login,
			})

			robot.Status.Login.Code = define.ROBOT_LOGIN_STATUS_DISABLED
			robot.Status.Login.Time = time.Now().Unix()
			robot.Status.Login.Value = robot.Cache.Offline

			robot.Cache.Offline = ""

			return
		}

		return true
	}

	if robot.Status.RenewOnline != nil || robot.Status.RenewSecretKey != nil {

		robot.Update(plugin.Bson{
			"status.renew_secret_key": nil,
			"status.renew_online":     nil,
		})

		robot.Status.RenewSecretKey = nil
		robot.Status.RenewOnline = nil
	}

	return
}

func (ctrler *ctrler_robot) proxy_is_disabled(robot *model.Robot, err error) bool {

	if err == nil {
		return false
	}

	err_str := err.Error()

	switch {

	case !strings.Contains(err_str, "socks5"):

		return false

	case strings.Contains(err_str, "username/password authentication failed"):

		if proxy := self.Proxys.Existed(robot.ProxyId); proxy != nil {

			proxy.Update(plugin.Bson{"disabled": true})

			proxy.Disabled = true
		}

		proxy_status := model.RobotStatusCurrent{}
		proxy_status.Time = time.Now().Unix()
		proxy_status.Code = define.ROBOT_STATUS_PROXY_DISABLED
		proxy_status.Value = err_str

		robot.Update(plugin.Bson{
			"status.proxy": proxy_status,
		})

		robot.Status.Proxy = proxy_status

	case robot.Status.Proxy.Code != define.ROBOT_STATUS_PROXY_SUCC:

		proxy_status := model.RobotStatusCurrent{}
		proxy_status.Code = define.ROBOT_STATUS_PROXY_SUCC

		robot.Update(plugin.Bson{
			"status.proxy": proxy_status,
		})

		robot.Status.Proxy = proxy_status
	}

	return true
}

func (ctrler *ctrler_robot) proxy_changed(robot *model.Robot) {

	if robot.Status.Proxy.Code == define.ROBOT_STATUS_PROXY_SUCC {
		return
	}

	proxy_status := model.RobotStatusCurrent{}
	proxy_status.Time = time.Now().Unix()
	proxy_status.Code = define.ROBOT_STATUS_PROXY_SUCC
	proxy_status.Value = ""

	robot.Update(plugin.Bson{
		"status.proxy": proxy_status,
	})

	robot.Status.Proxy = proxy_status
}

func (ctrler *ctrler_robot) kernel_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("kernel_handle")

	var status *model.RobotStatusCurrent

	if status = robot.Status.Kernel; status == nil {
		status = &model.RobotStatusCurrent{}

	} else if status.Code == 0 {
		return
	}

	var result []*model.RobotKernelPlus

	result, err = model.RobotKernelExport(robot.DriveUrl, []string{robot.Kernel.Objid})

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 1*60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()
		goto END

	case len(result) == 0:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 1*60
		// status.Time = time.Now().Unix()
		status.Value = "导出失败"
		goto END

	case result[0].Objid != robot.Kernel.Objid:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 1*60
		// status.Time = time.Now().Unix()
		status.Value = fmt.Sprintf("导出异常，目标：%s，但得到：%s！", robot.Kernel.Objid, result[0].Objid)
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Time = time.Now().Unix()
		status.Value = ""

		robot.Kernel.Hardware = result[0].Hardware
		robot.Kernel.ObjSetupTime = result[0].ObjSetupTime
		robot.Kernel.SoftWare = result[0].SoftWare
		robot.Kernel.Version = result[0].Version

		robot.Kernel.Device.Name = result[0].Device.Name

		robot.Kernel.UserLoginData.Uis = result[0].UserLoginData.Uis
		robot.Kernel.UserLoginData.Uin = result[0].UserLoginData.Uin
		robot.Kernel.UserLoginData.Password = result[0].UserLoginData.Password

		robot.Kernel.LoginTime = result[0].LoginTime

		robot.Update(plugin.Bson{
			"kernel": result[0],
		})
	}

END:

	robot.Update(plugin.Bson{
		"status.kernel": status,
	})

	if robot.Status.Kernel == nil {
		robot.Status.Kernel = status
	}

	return
}

func (ctrler *ctrler_robot) renew_secret_key_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("renew_secret_key_handle")

	var status *model.RobotStatusCurrent

	if status = robot.Status.RenewSecretKey; status == nil {
		status = &model.RobotStatusCurrent{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var result model.RobotRenewResult

	result, err = robot.RenewSecretKey()

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		if status.Time == 0 {
			status.Time = time.Now().Unix()
		}
		status.Value = err.Error()

	case status.Code == define.ROBOT_LOGIN_STATUS_CONNECTING || status.Code == define.ROBOT_LOGIN_STATUS_CONNECTING2 || status.Code == define.ROBOT_LOGIN_STATUS_CON_TIMEOUT:

		status.Timer = time.Now().Unix() + 10

		return

	case result.Code == 0:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_RENEW_SECRET_KEY + utils.Random(0, 3*24*60*60)
		status.Time = time.Now().Unix()
		status.Value = ""

		ctrler.proxy_changed(robot)

	default:

		login_status := model.RobotStatusLogin{}

		login_status.Time = time.Now().Unix()
		login_status.Code = result.Code
		login_status.Value = result.GetMsg()

		robot.Update(plugin.Bson{
			"status.login": login_status,
		})

		robot.Status.Login = &login_status

		return
	}

	robot.Update(plugin.Bson{
		"status.renew_secret_key": status,
	})

	if robot.Status.RenewSecretKey == nil {
		robot.Status.RenewSecretKey = status
	}

	return
}

func (ctrler *ctrler_robot) renew_online_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("renew_online_handle")

	var status *model.RobotStatusCurrent

	if status = robot.Status.RenewOnline; status == nil {
		robot.Onlining()
		status = &model.RobotStatusCurrent{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	{
		done = true

		var tryed bool

		var result model.RobotRenewOnlineResult

	HANDLE:

		result, err = robot.RenewOnline()

		switch {

		case err == nil:

			status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
			status.Timer = time.Now().Unix() + define.INTERVAL_RENEW_ONLINE + utils.Random(0, 3*60)
			status.Time = time.Now().Unix()
			status.Value = utils.ToJson3(result)

			ctrler.proxy_changed(robot)

		case strings.Contains(err.Error(), "SsoSendRawData") && !tryed:

			tryed = true

			if robot.Submit.Uid == 3868719484 {

				if user := self.Users.Existed(robot.UserId); user != nil {
					fmt.Printf("%s renew_online_handle: %10s  %15s  %s\n", robot.Id.Hex(), user.Username, robot.Kernel.UserLoginData.Uis, err.Error())
				}
			}

			robot.Onlining()

			goto HANDLE

		case strings.Contains(err.Error(), "登录"):

			login_status := model.RobotStatusLogin{}

			login_status.Time = time.Now().Unix()
			login_status.Code = define.ROBOT_LOGIN_STATUS_LOSE
			login_status.Value = err.Error()

			robot.Update(plugin.Bson{
				"status.login": login_status,
			})

			robot.Status.Login = &login_status

			return

		default:

			status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
			status.Time = time.Now().Unix()
			status.Timer = time.Now().Unix() + 60
			status.Value = err.Error()
		}
	}

	robot.Update(plugin.Bson{
		"status.renew_online": status,
	})

	if robot.Status.RenewOnline == nil {
		robot.Status.RenewOnline = status
	}

	return
}

func (ctrler *ctrler_robot) info_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("info_handle")

	var status *model.RobotStatusInfo

	if status = robot.Status.Info; status == nil {
		status = &model.RobotStatusInfo{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var result model.SummaryCardResult

	result, err = robot.GetSummaryCard(robot.Kernel.UserLoginData.Uin, nil)

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_INFO + utils.Random(0, 3*60*60)
		status.Time = time.Now().Unix()

		status.Nick = result.Nick

		status.Age = result.Age
		status.Sex = result.Sex
		status.Level = result.Level

		status.Province = result.Province
		status.City = result.City

		status.Value = ""
	}

END:

	robot.Update(plugin.Bson{
		"status.info": status,
	})

	if robot.Status.Info == nil {
		robot.Status.Info = status
	}

	return
}

func (ctrler *ctrler_robot) profile_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("profile_handle")

	var status *model.RobotStatusCurrent

	if status = robot.Status.Profile; status == nil {
		status = &model.RobotStatusCurrent{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var result model.RobotProfileResult

	result, err = robot.GetProfile([]int{robot.Kernel.UserLoginData.Uin})

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()
		goto END

	case len(result.MsgUserdata) == 0:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = "len(MsgUserdata) = 0"
		goto END

	case len(result.MsgUserdata[0].MsgData.ValuesIn) == 0:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = "len(valueIn) = 0"
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_PROFILE + utils.Random(0, 3*24*60*60)
		status.Time = time.Now().Unix()
		status.Value = result.MsgUserdata[0].MsgData.ValuesIn[0].Value.Url + "140"
	}

END:

	robot.Update(plugin.Bson{
		"status.profile": status,
	})

	if robot.Status.Profile == nil {
		robot.Status.Profile = status
	}

	return
}

func (ctrler *ctrler_robot) photo_wall_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("photo_wall_handle")

	var status *model.RobotStatusPhotoWall

	if status = robot.Status.PhotoWall; status == nil {
		status = &model.RobotStatusPhotoWall{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var result model.RobotPhotoWallResult

	result, err = robot.GetPhotoWall()

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_PROFILE + utils.Random(0, 3*24*60*60)
		status.Time = time.Now().Unix()
		status.Result = result
	}

END:

	robot.Update(plugin.Bson{
		"status.photo_wall": status,
	})

	if robot.Status.PhotoWall == nil {
		robot.Status.PhotoWall = status
	}

	return
}

func (ctrler *ctrler_robot) friends_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("friends_handle")

	var status *model.RobotStatusFriends

	if status = robot.Status.Friends; status == nil {
		status = &model.RobotStatusFriends{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	friends_old := map[int]plugin.FriendbValue{}

	friends_total := 0

	var result model.RobotFriendsResult

	result, err = robot.GetFrineds()

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 10*60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_FRIENDS + utils.Random(0, 60*60)
		status.Time = time.Now().Unix()
		status.Value = ""
	}

	ini.Friendb1.Foreach(plugin.FriendbFilter{RobotUid: &robot.Kernel.UserLoginData.Uin}, func(stop *bool, value *plugin.FriendbValue) {

		friends_old[value.Index] = *value
	})

	for idx := 0; idx < len(result.FriendInfo); idx++ {

		if result.FriendInfo[idx].FriendUin == 66600000 ||
			result.FriendInfo[idx].FriendUin == 2854203763 || result.FriendInfo[idx].FriendUin == 2854196306 {
			result.FriendInfo = append(result.FriendInfo[:idx], result.FriendInfo[idx+1:]...)
			break
		}
	}

	for _, robot := range self.Robots {

		for idx := 0; idx < len(result.FriendInfo); idx++ {

			if result.FriendInfo[idx].FriendUin == robot.Submit.Uid {
				result.FriendInfo = append(result.FriendInfo[:idx], result.FriendInfo[idx+1:]...)
				break
			}
		}
	}

	for _, friend_old := range friends_old {

		var friend_now *model.RobotFriendItemResult

		for _, item := range result.FriendInfo {

			if item.FriendUin == friend_old.Uid {
				friend_now = item
				break
			}
		}

		// --- 原本有（已通过），现在有 --- 更新信息
		if friend_old.Time != 0 && friend_now != nil {

			ini.Friendb1.Foreach(plugin.FriendbFilter{Index: &friend_old.Index}, func(stop *bool, value *plugin.FriendbValue) {

				value.Name = friend_now.Nick
				value.Attributes[2] = friend_now.BothFlag == 1
				value.Attributes[3] = strings.Contains(friend_now.TermDesc, "在线")

				for _, message := range robot.Messages {
					if message.Uid == value.Uid {
						message.Nick = value.Name
					}
				}
			})
		}

		// --- 原本有（未通过），现在有 --- 突然通过
		if friend_old.Time == 0 && friend_now != nil {

			ini.Friendb1.Foreach(plugin.FriendbFilter{Index: &friend_old.Index}, func(stop *bool, value *plugin.FriendbValue) {

				value.Name = friend_now.Nick
				value.Time = time.Now().Unix()
				value.Attributes[2] = friend_now.BothFlag == 1
				value.Attributes[3] = strings.Contains(friend_now.TermDesc, "在线")

				value.Time = time.Now().Unix()

				for _, message := range robot.Messages {
					if message.Uid == value.Uid {
						message.Nick = value.Name
					}
				}
			})
		}

		// --- 原本有（已通过），现在无 --- 删除
		if friend_old.Time != 0 && friend_now == nil {

			ini.Friendb1.Foreach(plugin.FriendbFilter{Index: &friend_old.Index}, func(stop *bool, value *plugin.FriendbValue) {

				// value.Clear()
				value.Attributes[0] = true
			})
		}

		// --- 原本有（未通过），现在无 --- 依然未通过
	}

	for _, friend_now := range result.FriendInfo {

		friends_total++

		exist := false

		for _, friend_old := range friends_old {

			if friend_now.FriendUin == friend_old.Uid {
				exist = true
				break
			}
		}

		// --- 原本无，现在有 --- 自带好友
		if !exist {

			ini.Friendb1.Foreach(plugin.FriendbFilter{Wlock: true, Blank: true}, func(stop *bool, value *plugin.FriendbValue) {

				var user *model.User

				if user = self.Users.Existed(robot.UserId); user == nil {
					return
				}

				value.CreateTime = time.Now().Unix()

				value.Status = plugin.FRIENDB_STATUS_USED
				value.Uid = friend_now.FriendUin
				value.RobotUid = robot.Kernel.UserLoginData.Uin
				value.Name = friend_now.Nick
				value.Attributes[1] = false
				value.Attributes[2] = friend_now.BothFlag == 1
				value.Attributes[3] = strings.Contains(friend_now.TermDesc, "在线")
				value.UserMark = user.Mark

				value.Time = time.Now().Unix()

				for _, message := range robot.Messages {
					if message.Uid == value.Uid {
						message.Nick = value.Name
					}
				}

				*stop = true
			})
		}

		// --- 原本无，现在无 --- 自带好友
	}

	status.Total = friends_total

END:

	robot.Update(plugin.Bson{
		"status.friends": status,
	})

	if robot.Status.Friends == nil {
		robot.Status.Friends = status
	}

	return
}

func (ctrler *ctrler_robot) applist_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("applist_handle")

	var status *model.RobotStatusApplist

	if status = robot.Status.Applist; status == nil {
		status = &model.RobotStatusApplist{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var result model.RobotApplistResult

	result, err = robot.GetApplist(robot.Kernel.UserLoginData.Uin, 10, nil)

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_APPLIST + utils.Random(0, 3*60*60)
		status.Time = time.Now().Unix()
		status.Result = result
	}

END:

	robot.Update(plugin.Bson{
		"status.applist": status,
	})

	if robot.Status.Applist == nil {
		robot.Status.Applist = status
	}

	return
}

func (ctrler *ctrler_robot) auth_applist_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("auth_applist_handle")

	var status *model.RobotStatusAuthApplist

	if status = robot.Status.AuthApplist; status == nil {
		status = &model.RobotStatusAuthApplist{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var result model.RobotAuthApplist

	result, err = robot.GetAuthAppList()

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()
		goto END

	case result.Wording != "ok":

		status.Code = define.ROBOT_STATUS_SYSTEM_FAIL
		status.Timer = time.Now().Unix() + define.INTERVAL_APPLIST + 60*60
		status.Value = result.Wording
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_APPLIST + utils.Random(0, 3*60*60)
		status.Time = time.Now().Unix()
		status.Result = result.GetAuthAppListRsp.AppInfos
	}

END:

	robot.Update(plugin.Bson{
		"status.auth_applist": status,
	})

	if robot.Status.AuthApplist == nil {
		robot.Status.AuthApplist = status
	}

	return
}

func (ctrler *ctrler_robot) devicelist_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("devicelist_handle")

	var status *model.RobotStatusDevicelist

	if status = robot.Status.Devicelist; status == nil {
		status = &model.RobotStatusDevicelist{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var result model.RobotDevicelistResult

	result, err = robot.GetDeviceList(nil)

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_DEVICELIST + utils.Random(0, 3*60*60)
		status.Time = time.Now().Unix()
		status.Result = result
	}

END:

	robot.Update(plugin.Bson{
		"status.devicelist": status,
	})

	if robot.Status.Devicelist == nil {
		robot.Status.Devicelist = status
	}

	return
}

func (ctrler *ctrler_robot) qzone_permission_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("qzone_permission_handle")

	var status *model.RobotStatusQzonePermission

	if status = robot.Status.QzonePermission; status == nil {
		status = &model.RobotStatusQzonePermission{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var result model.RobotQzonePermissionResult

	result, err = robot.GetQzonePermission()

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_QZONE_PERMISSION + utils.Random(0, 3*60*60)
		status.Time = time.Now().Unix()

		status.RightVal = result.Rightval
	}

END:

	robot.Update(plugin.Bson{
		"status.qzone_permission": status,
	})

	if robot.Status.QzonePermission == nil {
		robot.Status.QzonePermission = status
	}

	return
}

func (ctrler *ctrler_robot) summarycard_view_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("summarycard_view_handle")

	var status *model.RobotStatusSummarycardView

	if status = robot.Status.SummarycardView; status == nil {
		status = &model.RobotStatusSummarycardView{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var result model.RobotSummarycardViewResult

	result, err = robot.GetSummarycardView()

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_QZONE_PERMISSION + utils.Random(0, 3*60*60)
		status.Time = time.Now().Unix()

		status.Result = result
	}

END:

	robot.Update(plugin.Bson{
		"status.summarycard_view": status,
	})

	if robot.Status.SummarycardView == nil {
		robot.Status.SummarycardView = status
	}

	return
}

func (ctrler *ctrler_robot) signature_history_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("signature_history_handle")

	var status *model.RobotStatusSignatureHistory

	if status = robot.Status.SignatureHistory; status == nil {
		status = &model.RobotStatusSignatureHistory{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var result model.RobotSignatureHistoryResult

	result, err = robot.GetSignatureHistory(robot.Kernel.UserLoginData.Uin, 10)

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_QZONE_PERMISSION + utils.Random(0, 3*60*60)
		status.Time = time.Now().Unix()

		status.Result = result
	}

END:

	robot.Update(plugin.Bson{
		"status.signature_history": status,
	})

	if robot.Status.SignatureHistory == nil {
		robot.Status.SignatureHistory = status
	}

	return
}

func (ctrler *ctrler_robot) qzone_main_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("qzone_main")

	var status *model.RobotStatusQzoneMain

	if status = robot.Status.QzoneMain; status == nil {
		status = &model.RobotStatusQzoneMain{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var result model.RobotQzoneMainResult

	result, err = robot.GetQzoneMain(robot.Kernel.UserLoginData.Uin, nil)

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_QZONE_MAIN + utils.Random(0, 30*60)
		status.Time = time.Now().Unix()

		status.TodayNum = result.Visit.TodayNum
		status.TotalNum = result.Visit.TotalNum
		status.LastTime = result.Visit.LastTime
	}

END:

	robot.Update(plugin.Bson{
		"status.qzone_main": status,
	})

	if robot.Status.QzoneMain == nil {
		robot.Status.QzoneMain = status
	}

	return
}

func (ctrler *ctrler_robot) ban_handle(robot *model.Robot) (done bool, err error) {

	robot.Step("ban_handle")

	var status *model.RobotStatusBan

	if status = robot.Status.Ban; status == nil {
		status = &model.RobotStatusBan{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	// ---

	var (
		codeStr       string
		encryptedData string
		iv            string
		miniskey      string
		ban           string
	)

	if result, _err := robot.GetCode(); _err != nil {
		err = _err
		goto RESULT

	} else {
		codeStr = result.Code
	}

	time.Sleep(time.Millisecond * 100)

	if result, _err := robot.GetUserInfo(); _err != nil {
		err = _err
		goto RESULT

	} else {
		encryptedData = result.EncryptedData
		iv = result.Iv
	}

	time.Sleep(time.Millisecond * 100)

	if result, _err := robot.GetMiniKey(codeStr, encryptedData, iv); _err != nil {
		err = _err
		goto RESULT

	} else {

		for _, item := range result.Resultinfo.List {
			if item.Miniskey != "" {
				miniskey = item.Miniskey
				break
			}
		}
	}

	time.Sleep(time.Millisecond * 100)

	if result, _err := robot.GetBan(miniskey); _err != nil {
		err = _err
		goto RESULT

	} else {

		for _, item := range result.Resultinfo.List {
			ban = item.Isfullsocialban
		}
	}

	// ---

RESULT:

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60
		// status.Time = time.Now().Unix()
		status.Value = err.Error()

		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_DAILY + 60*60*utils.Random(1, 3)
		status.Time = time.Now().Unix()
		status.Result = ban
		status.Miniskey = miniskey
	}

END:

	robot.Update(plugin.Bson{
		"status.ban": status,
	})

	if robot.Status.Ban == nil {
		robot.Status.Ban = status
	}

	return
}

func (ctrler *ctrler_robot) dailyclockin(robot *model.Robot) (done bool, err error) {

	robot.Step("dailyclockin")

	var status *model.RobotStatusDailyclockin

	if status = robot.Status.Dailyclockin; status == nil {
		status = &model.RobotStatusDailyclockin{}
	}

	if !status.Switch || time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var result model.RobotDailyclockinResult

	result, err = robot.Dailyclockin()

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60*60
		status.Value = err.Error()
		goto END

	case result.Ret != 0:

		status.Code = define.ROBOT_STATUS_SYSTEM_FAIL
		status.Timer = time.Now().Unix() + define.INTERVAL_DAILY + 60*60*utils.Random(1, 3)
		status.Value = fmt.Sprintf("[%d]%s", result.Ret, result.Msg)
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_DAILY + 60*60*utils.Random(1, 3)
		status.Time = time.Now().Unix()
	}

END:

	robot.Update(plugin.Bson{
		"status.dailyclockin": status,
	})

	if robot.Status.Dailyclockin == nil {
		robot.Status.Dailyclockin = status
	}

	return
}

func (ctrler *ctrler_robot) dailyapplist(robot *model.Robot) (done bool, err error) {

	robot.Step("dailyapplist")

	var status *model.RobotStatusDailyapplist

	if status = robot.Status.Dailyapplist; status == nil {
		status = &model.RobotStatusDailyapplist{}
	}

	if !status.Switch || time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var (
		result model.RobotSendApplistTextResult

		material   *model.RobotMaterial
		file_bytes []byte
	)

	if material = self.RobotMaterials.Existed(robot.UserId, status.MaterialName); material == nil || len(material.FileNames) == 0 {
		err = fmt.Errorf("image material not existed")

	} else if err = material.Download(material.FileNames[utils.Random(0, len(material.FileNames)-1)], 0, &file_bytes); err != nil {

	} else {
		result, err = robot.PushApplistText(string(file_bytes))
	}

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60*10
		status.Value = err.Error()
		goto END

	case result.Ret != 0:

		status.Code = define.ROBOT_STATUS_SYSTEM_FAIL
		status.Timer = time.Now().Unix() + define.INTERVAL_DAILY + 60*60*utils.Random(1, 3)
		status.Value = fmt.Sprintf("[%d]%s", result.Ret, result.Tid)
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_DAILY + 60*60*utils.Random(1, 3)
		status.Time = time.Now().Unix()

		robot.Status.Applist = nil
	}

END:

	robot.Update(plugin.Bson{
		"status.dailyapplist": status,
	})

	if robot.Status.Dailyapplist == nil {
		robot.Status.Dailyapplist = status
	}

	return
}

func (ctrler *ctrler_robot) system_message(robot *model.Robot) (done bool, err error) {

	robot.Step("system_message")

	var status *model.RobotStatusSystemMessage

	if status = robot.Status.SystemMessage; status == nil {
		status = &model.RobotStatusSystemMessage{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var (
		result model.RobotSystemMessageResult
	)

	result, err = robot.GetSystemMessage()

	switch {

	case err != nil:

		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60*10
		status.Value = err.Error()
		goto END

	default:

		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_FRIENDS + utils.Random(0, 60*60)
		status.Time = time.Now().Unix()

		status.Result = result
	}

END:
	robot.Update(plugin.Bson{
		"status.system_message": status,
	})

	if robot.Status.SystemMessage == nil {
		robot.Status.SystemMessage = status
	}

	return
}

// 更新群信息
func (ctrler *ctrler_robot) group_list(robot *model.Robot) (done bool, err error) {
	robot.Step("updateGroupInfo")

	var status *model.RobotStatusGroupList
	if status = robot.Status.GroupList; status == nil {
		status = &model.RobotStatusGroupList{}
	}

	if time.Now().Unix() < status.Timer {
		return
	}

	done = true

	var (
		result model.RobotGroupListResult
	)

	result, err = robot.GetGroupList()
	if err != nil {
		status.Code = define.ROBOT_STATUS_SYSTEM_ERROR
		status.Timer = time.Now().Unix() + 60*10
		status.Value = err.Error()

	} else {
		status.Code = define.ROBOT_STATUS_SYSTEM_SUCC
		status.Timer = time.Now().Unix() + define.INTERVAL_FRIENDS + utils.Random(0, 60*60)
		status.Time = time.Now().Unix()
		status.Result = result

		for k, v := range result.TroopList {
			// 获取群成员
			members, err := robot.GetGroupMenber(v.GroupCode, v.GroupUin, nil)
			if err != nil {
				slog.Error("update group_list", "msg", "call GetGroupMenber failed")
				continue
			}
			result.TroopList[k].Members = &members
		}
	}

	robot.Update(plugin.Bson{
		"status.group_list": status,
	})

	if robot.Status.SystemMessage == nil {
		robot.Status.GroupList = status
	}

	return

}

func work_set_summarycard(robot *model.Robot, work *model.RobotWork) (isrun int) {

	robot.Step("work_set_summarycard")

	setting := work.SetSummarycard
	if setting == nil {
		return
	}

	work.Status = model.ROBOT_WORK_STATUS_DOING

	var (
		material   *model.RobotMaterial
		file_bytes []byte
	)

	if material = self.RobotMaterials.Existed(robot.UserId, setting.Nick.MaterialName); material == nil {
		work.Msg = "material not existed"
		goto FAIL
	}

	if err := material.Download(setting.Nick.FileName, 0, &file_bytes); err != nil {
		work.Msg = err.Error()
		goto FAIL
	}

	if err := robot.SetSummarycard(string(file_bytes), setting.Sex, setting.Age); err != nil {
		work.Msg = err.Error()
		goto FAIL
	}

	robot.Update(plugin.Bson{
		"status.info": nil,
	})

	robot.Status.Info = nil

	work.Status = model.ROBOT_WORK_STATUS_SUCC
	work.EndTime = time.Now().Unix()
	return 1

FAIL:
	work.Status = model.ROBOT_WORK_STATUS_FAIL
	work.EndTime = time.Now().Unix()

	return 1
}

func work_set_profile(robot *model.Robot, work *model.RobotWork) (isrun int) {

	robot.Step("work_set_profile")

	setting := work.SetProfile
	if setting == nil {
		return
	}

	work.Status = model.ROBOT_WORK_STATUS_DOING

	var (
		material   *model.RobotMaterial
		file_bytes []byte
	)

	if material = self.RobotMaterials.Existed(robot.UserId, setting.MaterialName); material == nil {
		work.Msg = "material not existed"
		goto FAIL
	}

	if err := material.Download(setting.FileName, 1, &file_bytes); err != nil {
		work.Msg = err.Error()
		goto FAIL
	}

	if res, err := robot.SetProfile(file_bytes); err != nil {

		work.Msg = err.Error()
		goto FAIL

	} else if res.ErrorCode != 0 {
		work.Msg = utils.ToJson2(res)
		goto FAIL
	}

	robot.Update(plugin.Bson{
		"status.profile": nil,
	})

	robot.Status.Profile = nil

	work.Status = model.ROBOT_WORK_STATUS_SUCC
	work.EndTime = time.Now().Unix()
	return 1

FAIL:

	work.Status = model.ROBOT_WORK_STATUS_FAIL
	work.EndTime = time.Now().Unix()
	return 1
}

func work_push_image_multiple(robot *model.Robot, work *model.RobotWork) (isrun int) {

	robot.Step("work_push_image_multiple")

	setting := work.PushImageMultiple
	if setting == nil {
		return
	}

	work.Status = model.ROBOT_WORK_STATUS_DOING

	var (
		material   *model.RobotMaterial
		file_bytes []byte
	)

	if err := robot.ClearApplistImageMultipleCache(); err != nil {
		work.Msg = err.Error()
		goto FAIL
	}

	for index, image := range setting.Images {

		if material = self.RobotMaterials.Existed(robot.UserId, image.MaterialName); material == nil {
			work.Msg = "image material not existed"
			goto FAIL
		}

		if err := material.Download(image.FileName, 0, &file_bytes); err != nil {
			work.Msg = err.Error()
			goto FAIL
		}

		if res, err := robot.PushApplistImageMultipleCache(file_bytes); err != nil {
			work.Msg = err.Error()
			goto FAIL

		} else if res.PictureNumber != index+1 {
			work.Msg = utils.ToJson2(res)
			goto FAIL
		}
	}

	file_bytes = []byte{}

	if setting.Text != nil {

		if material = self.RobotMaterials.Existed(robot.UserId, setting.Text.MaterialName); material == nil {
			work.Msg = "text material not existed"
			goto FAIL
		}

		if err := material.Download(setting.Text.FileName, 0, &file_bytes); err != nil {
			work.Msg = err.Error()
			goto FAIL
		}
	}

	if res, err := robot.PushApplistImageMultiple(string(file_bytes)); err != nil {
		work.Msg = err.Error()
		goto FAIL

	} else if res.BusiNessDataRsp != nil {
		work.Msg = utils.ToJson2(res)
		goto FAIL
	}

	time.Sleep(time.Second)

	robot.Status.Applist = nil

	robot.Update(plugin.Bson{
		"status.applist": robot.Status.Applist,
	})

	work.Status = model.ROBOT_WORK_STATUS_SUCC
	work.EndTime = time.Now().Unix()
	return 1

FAIL:

	time.Sleep(time.Second)

	robot.Status.Applist = nil

	robot.Update(plugin.Bson{
		"status.applist": robot.Status.Applist,
	})

	work.Status = model.ROBOT_WORK_STATUS_FAIL
	work.EndTime = time.Now().Unix()
	return 1
}

func work_delete_applist(robot *model.Robot, work *model.RobotWork) (isrun int) {

	robot.Step("work_delete_applist")

	setting := work.DeleteApplist
	if setting == nil {
		return
	}

	work.Status = model.ROBOT_WORK_STATUS_DOING

	if robot.Status.Applist != nil && robot.Status.Applist.Result.AllApplistData != nil {

		for index := 0; index < setting.Number; index++ {

			if index > len(*robot.Status.Applist.Result.AllApplistData)-1 {
				continue
			}

			app := (*robot.Status.Applist.Result.AllApplistData)[index]

			if res, err := robot.DeleteApplist(app.CellComm.Appid, app.CellId.CellId, app.CellComm.ClientKey); err != nil {
				work.Msg = err.Error()
				goto FAIL

			} else if res.Ret != 0 {
				work.Msg = utils.ToJson3(res)
				goto FAIL
			}
		}
	}

	robot.Update(plugin.Bson{
		"status.applist": nil,
	})

	robot.Status.Applist = nil

	work.Status = model.ROBOT_WORK_STATUS_SUCC
	work.EndTime = time.Now().Unix()
	return 1

FAIL:
	work.Status = model.ROBOT_WORK_STATUS_FAIL
	work.EndTime = time.Now().Unix()
	return 1
}

func work_clean_photo_wall(robot *model.Robot, work *model.RobotWork) (isrun int) {

	robot.Step("work_clean_photo_wall")

	setting := work.CleanPhotoWall
	if setting == nil {
		return
	}

	work.Status = model.ROBOT_WORK_STATUS_DOING

	if result, err := robot.CleanPhotoWall(); err != nil {
		work.Msg = err.Error()
		goto FAIL

	} else if result.Ret != 0 {
		work.Msg = fmt.Sprintf("[%d]%s", result.Ret, result.Msg)
		goto FAIL
	}

	robot.Update(plugin.Bson{
		"status.photo_wall": nil,
	})

	robot.Status.PhotoWall = nil

	work.Status = model.ROBOT_WORK_STATUS_SUCC
	work.EndTime = time.Now().Unix()
	return 1

FAIL:
	work.Status = model.ROBOT_WORK_STATUS_FAIL
	work.EndTime = time.Now().Unix()
	return 1
}

func work_set_qzone_permission(robot *model.Robot, work *model.RobotWork) (isrun int) {

	robot.Step("work_set_qzone_permission")

	setting := work.SetQzonePermission
	if setting == nil {
		return
	}

	work.Status = model.ROBOT_WORK_STATUS_DOING

	if result, err := robot.SetQzonePermission(setting.RightVal); err != nil {
		work.Msg = err.Error()
		goto FAIL

	} else if result.Ret != 0 {
		work.Msg = fmt.Sprintf("[%d]%s", result.Ret, result.Msg)
		goto FAIL
	}

	robot.Update(plugin.Bson{
		"status.set_qzone_permission": nil,
	})

	robot.Status.QzonePermission = nil

	work.Status = model.ROBOT_WORK_STATUS_SUCC
	work.EndTime = time.Now().Unix()
	return 1

FAIL:
	work.Status = model.ROBOT_WORK_STATUS_FAIL
	work.EndTime = time.Now().Unix()

	return 1
}

func work_send_message(robot *model.Robot, work *model.RobotWork) (isrun int) {

	robot.Step("work_send_message")

	setting := work.SendMessage
	if setting == nil {
		return
	}

	if setting.Timer != 0 && time.Now().Unix() <= setting.Timer {
		return
	}

	work.Status = model.ROBOT_WORK_STATUS_DOING

	var (
		material   *model.RobotMaterial
		file_bytes []byte

		result model.RobotSendMsgResult
		err    error
	)

	for _, message := range setting.List {

		if message.Finished {
			continue
		}

		if material = self.RobotMaterials.Existed(robot.UserId, message.MaterialName); material == nil {
			work.Msg = "material not existed"
			goto FAIL
		}

		if err := material.Download(message.FileName, 0, &file_bytes); err != nil {
			work.Msg = err.Error()
			goto FAIL
		}

		switch message.Mode {

		case define.ROBOT_MATERIAL_MODE_TEXT:
			result, err = robot.SendMsgText(work.SendMessage.FriendUid, string(file_bytes))

		case define.ROBOT_MATERIAL_MODE_IMAGE:
			result, err = robot.SendMsgImage(work.SendMessage.FriendUid, 0, 0, file_bytes)

		case define.ROBOT_MATERIAL_MODE_AUDIO:
			result, err = robot.SendMsgVoice(work.SendMessage.FriendUid, file_bytes)

		case define.ROBOT_MATERIAL_MODE_VIDEO:
			result, err = robot.SendMsgVideo(work.SendMessage.FriendUid, 0, 0, file_bytes)

		case define.ROBOT_MATERIAL_MODE_GROUPLINK:
			result, err = robot.SendMsgGroupLink(work.SendMessage.FriendUid, string(file_bytes))
		}

		if err != nil {
			work.Msg = err.Error()
			goto FAIL

		} else if result.Result != 0 {
			work.Msg = fmt.Sprintf("[%d] %s", result.Result, result.ErrMsg)
			goto FAIL
		}

		switch message.Mode {

		case define.ROBOT_MATERIAL_MODE_TEXT:
		case define.ROBOT_MATERIAL_MODE_IMAGE:
		case define.ROBOT_MATERIAL_MODE_AUDIO:
		case define.ROBOT_MATERIAL_MODE_VIDEO:

		case define.ROBOT_MATERIAL_MODE_GROUPLINK:

			ini.Friendb1.Foreach(plugin.FriendbFilter{
				Index: &work.SendMessage.FriendIndex, RobotUid: &robot.Kernel.UserLoginData.Uin,
			}, func(stop *bool, value *plugin.FriendbValue) {
				value.Attributes[6] = true
				*stop = true
			})
		}

		message.Finished = true

		setting.Timer = time.Now().Unix() + message.Interval

		break
	}

	for _, message := range setting.List {

		if !message.Finished {
			work.Status = model.ROBOT_WORK_STATUS_WAIT
			return 1
		}
	}

	work.Status = model.ROBOT_WORK_STATUS_SUCC
	work.EndTime = time.Now().Unix()
	return 1

FAIL:
	work.Status = model.ROBOT_WORK_STATUS_FAIL
	work.EndTime = time.Now().Unix()

	return 1
}

func work_set_signature(robot *model.Robot, work *model.RobotWork) (isrun int) {

	robot.Step("work_set_signature")

	setting := work.SetSignature
	if setting == nil {
		return
	}

	work.Status = model.ROBOT_WORK_STATUS_DOING

	var (
		material   *model.RobotMaterial
		file_bytes []byte
	)

	if material = self.RobotMaterials.Existed(robot.UserId, setting.MaterialName); material == nil {
		work.Msg = "material not existed"
		goto FAIL
	}

	if err := material.Download(setting.FileName, 1, &file_bytes); err != nil {
		work.Msg = err.Error()
		goto FAIL
	}

	if res, err := robot.SetSignature(string(file_bytes)); err != nil {
		work.Msg = err.Error()
		goto FAIL

	} else if res.Cmd != 0 {
		work.Msg = utils.ToJson2(res)
		goto FAIL
	}

	robot.Update(plugin.Bson{
		"status.signature_history": nil,
	})

	robot.Status.SignatureHistory = nil

	work.Status = model.ROBOT_WORK_STATUS_SUCC
	work.EndTime = time.Now().Unix()
	return 1

FAIL:
	work.Status = model.ROBOT_WORK_STATUS_FAIL
	work.EndTime = time.Now().Unix()

	return 1
}

func work_set_summarycard_view(robot *model.Robot, work *model.RobotWork) (isrun int) {

	robot.Step("work_set_summarycard_view")

	setting := work.SetSummarycardView
	if setting == nil {
		return
	}

	work.Status = model.ROBOT_WORK_STATUS_DOING

	if _, err := robot.SetSummarycardView(setting.ProfileMembershipAndRank); err != nil {
		work.Msg = err.Error()
		goto FAIL
	}

	robot.Update(plugin.Bson{
		"status.summarycard_view": nil,
	})

	robot.Status.SummarycardView = nil

	work.Status = model.ROBOT_WORK_STATUS_SUCC
	work.EndTime = time.Now().Unix()
	return 1

FAIL:
	work.Status = model.ROBOT_WORK_STATUS_FAIL
	work.EndTime = time.Now().Unix()

	return 1
}

func work_set_signature_sync_mood(robot *model.Robot, work *model.RobotWork) (isrun int) {

	robot.Step("work_set_signature_sync_mood")

	setting := work.SetSignatureSyncMood
	if setting == nil {
		return
	}

	work.Status = model.ROBOT_WORK_STATUS_DOING

	if res, err := robot.SetSignatureSyncMood(setting.Value); err != nil {
		work.Msg = err.Error()
		goto FAIL

	} else if res.Uin != robot.Kernel.UserLoginData.Uin {
		work.Msg = utils.ToJson2(res)
		goto FAIL
	}

	robot.Update(plugin.Bson{
		"status.signature_sync_mood": &setting.Value,
	})

	robot.Status.SignatureSyncMood = &setting.Value

	work.Status = model.ROBOT_WORK_STATUS_SUCC
	work.EndTime = time.Now().Unix()
	return 1

FAIL:
	work.Status = model.ROBOT_WORK_STATUS_FAIL
	work.EndTime = time.Now().Unix()

	return 1
}

func work_set_photo_wall(robot *model.Robot, work *model.RobotWork) (isrun int) {

	robot.Step("work_set_photo_wall")

	setting := work.SetPhotoWall
	if setting == nil {
		return
	}

	work.Status = model.ROBOT_WORK_STATUS_DOING

	var (
		material   *model.RobotMaterial
		file_bytes []byte
	)

	if material = self.RobotMaterials.Existed(robot.UserId, setting.MaterialName); material == nil {
		work.Msg = "material not existed"
		goto FAIL
	}

	if err := material.Download(setting.FileName, 1, &file_bytes); err != nil {
		work.Msg = err.Error()
		goto FAIL
	}

	if res, err := robot.SetPhotoWall(file_bytes); err != nil {
		work.Msg = err.Error()
		goto FAIL

	} else if res.Code != 0 {
		work.Msg = utils.ToJson2(res)
		goto FAIL

	} else {

		robot.Update(plugin.Bson{
			"status.photo_wall": nil,
		})

		robot.Status.PhotoWall = nil
	}

	work.Status = model.ROBOT_WORK_STATUS_SUCC
	work.EndTime = time.Now().Unix()
	return 1

FAIL:
	work.Status = model.ROBOT_WORK_STATUS_FAIL
	work.EndTime = time.Now().Unix()

	return 1
}

func (ctrler *ctrler_robot) work_handle(robot *model.Robot) (bool, error) {

	for index, work := range robot.Works {

		if work == nil {
			continue
		}

		if work.Status != model.ROBOT_WORK_STATUS_WAIT {
			continue
		}

		is_run := 0

		is_run += work_set_summarycard(robot, work)
		is_run += work_set_profile(robot, work)
		is_run += work_push_image_multiple(robot, work)
		is_run += work_delete_applist(robot, work)
		is_run += work_set_qzone_permission(robot, work)
		is_run += work_send_message(robot, work)
		is_run += work_set_signature(robot, work)
		is_run += work_set_summarycard_view(robot, work)
		is_run += work_set_signature_sync_mood(robot, work)
		is_run += work_set_photo_wall(robot, work)
		is_run += work_clean_photo_wall(robot, work)

		if is_run > 0 {
			robot.Cache.Mutex.Do(true, func() {
				robot.Update(plugin.Bson{"works." + strconv.Itoa(index): work})
			})
		}

		return is_run > 0, nil
	}

	return false, nil
}
