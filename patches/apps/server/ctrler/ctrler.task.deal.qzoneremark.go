package ctrler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"ymlink-q2/apps/server/self"
	"ymlink-q2/define"
	"ymlink-q2/model"
	"ymlink-q2/plugin"
	"ymlink-q2/utils"
)

type ctrler_task_qzoneremark struct {
	object *model.Task
	self   *model.TaskConfigQzoneremark

	user              *model.User
	proxy_extractor   *model.ProxyExtractor
	qzone_remark_rule *model.TaskQzoneRemarkRule
	qzonedb           *model.TaskQzonedb
}

func (ctrler *ctrler_task_qzoneremark) Running(object *model.Task) {

	ctrler.object = object
	fmt.Println("REMARK-DEBUG: Running() started for task", object.Id.Hex())

	for range time.NewTicker(1 * time.Second).C {

		if ctrler.check() != define.TASK_STATUS_RUN {
			goto UPDATE
		}

		for index, thread := range ctrler.self.Threads {

			if thread.Cache.Isrun {
				continue
			}

		WAIT:
			for idx, number := 0, 0; idx < len(ctrler.self.Threads); idx++ {

				if ctrler.self.Threads[idx].Cache.Isrun {
					number++
				}

				if number >= 100 {
					time.Sleep(100 * time.Millisecond)
					goto WAIT
				}
			}

			thread.Cache.Isrun = true

				fmt.Println("REMARK-DEBUG: launching thread", index, "isrun=", thread.Cache.Isrun)
			go ctrler.thread_running(index, thread)
		}

	UPDATE:
		if ctrler.self.Cache.Update > 0 {

			ctrler.self.Cache.Update = 0

			if err := ctrler.object.Update(plugin.Bson{
				"config." + define.TASK_MODE_QZONEREMARK + ".robotdb": &ctrler.self.Robotdb,
			}); err != nil {
				ctrler.object.Cache.Error = fmt.Sprint("持久化存储异常：", err.Error())
			}
		}
	}
}

func (ctrler *ctrler_task_qzoneremark) check() (status int) {

	var finished bool = true

	if ctrler.object.Cache.Status == define.TASK_STATUS_FINISH {
		return
	}

	if ctrler.self = ctrler.object.Config.Qzoneremark; ctrler.self == nil {
		status = define.TASK_STATUS_CONFIG_NULL
		goto END
	}

	ctrler.self.Robotdb.Foreach(0, 0, func(stop *bool, value *model.QzoneremarkRobotdbValue) {
		if value.Status == model.QZONEGREET_ROBOTDB_SATAUS_RUN || value.Status == model.QZONEGREET_ROBOTDB_SATAUS_WAIT {
			finished = false
			*stop = true
		}
	})

	if finished {

	WAITS:
		for _, thread := range ctrler.self.Threads {
			if thread.Cache.Isrun {
				time.Sleep(100 * time.Millisecond)
				goto WAITS
			}
		}

		ctrler.object.Update(plugin.Bson{
			"switch": false,
		})

		ctrler.object.Switch = false

		status = define.TASK_STATUS_FINISH

		ctrler.self.Cache.Update++

		goto END
	}

	if !ctrler.object.Switch {
		status = define.TASK_STATUS_STOP
		goto END
	}

	if time.Now().Unix() < ctrler.object.StartupTime {
		status = define.TASK_STATUS_WAIT
		goto END
	}

	if ctrler.object = self.Tasks.Existed(ctrler.object.Id); ctrler.object == nil {
		status = define.TASK_STATUS_NULL
		return
	}

	if ctrler.user = self.Users.Existed(ctrler.object.UserId); ctrler.user == nil {
		status = define.TASK_STATUS_USER_NULL
		goto END
	}

	if !ctrler.self.ProxyExtractorId.IsZero() {

		if ctrler.proxy_extractor = self.ProxyExtractors.Existed(ctrler.self.ProxyExtractorId); ctrler.proxy_extractor == nil {
			status = define.TASK_STATUS_PROXY_EXTRACTOR_NULL
			goto END
		}
	}

	if ctrler.qzone_remark_rule = self.TaskQzoneRemarkRules.Existed(ctrler.self.QzoneRemarkRuleId); ctrler.qzone_remark_rule == nil ||
		len(ctrler.qzone_remark_rule.List) == 0 {
		status = define.TASK_STATUS_QZONE_REMARK_RULE_NULL
		goto END
	}

	for _, item := range ctrler.qzone_remark_rule.List {
		if item.Action == model.TASK_QZONE_REMARK_RULE_ACTION_COMMENT {
			if len(ctrler.qzone_remark_rule.Comments) == 0 {
				status = define.TASK_STATUS_QZONE_REMARK_RULE_NULL
				goto END
			}
			break
		}
	}

	if ctrler.qzonedb = self.TaskQzonedbs.Existed(ctrler.object.UserId, ctrler.self.QzonedbName); ctrler.qzonedb == nil {
		status = define.TASK_STATUS_QZONEDB_NULL
		goto END
	}

	status = define.TASK_STATUS_RUN
	ctrler.object.Cache.Error = ""

END:
	ctrler.object.Cache.Status = status

	return
}

func (ctrler *ctrler_task_qzoneremark) thread_running(index int, thread *model.QzoneremarkThread) (err error) {

	fmt.Println("REMARK-DEBUG: thread_running entered, RobotUid=", thread.RobotUid, "Step=", thread.Step, "Timer=", thread.Timer)
	defer func() {

		if info := recover(); info != nil {
			thread.Cache.Error = fmt.Sprint("系统错误：", info)
		}

		thread.Cache.Isrun = false
	}()

	fmt.Println("REMARK-DEBUG: about to call GetTimer, timer=", thread.Timer)
	if !thread.GetTimer(ctrler.qzone_remark_rule) {
	fmt.Println("REMARK-DEBUG: GetTimer returned false, timer not ready")
		return
	}

	defer thread.SetTimer(nil)

	if err = ctrler.robot_get(thread); err != nil {
		goto END
	}

	if ctrler.robot_lineup(thread) {
		return
	}

	switch thread.Step {

	case model.QZONEREMARK_STEP_QZONE:

		err = ctrler.qzone_handle(thread)

	case model.QZONEREMARK_STEP_APPLIST:

		err = ctrler.applist_handle(thread)

	case model.QZONEREMARK_STEP_REMARK:

		err = ctrler.remark_handle(thread)

	case model.QZONEREMARK_STEP_REMARK_REPLAY:

		err = ctrler.remark_replay_handle(thread)
	}

END:
	if err == nil {
		thread.Cache.Error = ""

	} else {

		thread.Cache.Error = err.Error()

		if !define.TaskThreadSafe(thread.Cache.Error) {
			ctrler.object.SetRobotLog(thread.RobotUid, "Error "+thread.Cache.Error)
		}

		if strings.Contains(thread.Cache.Error, "请你重新登录") || strings.Contains(thread.Cache.Error, "please login first") {
			ctrler.robot_offline(thread)
		}
	}

	if thread.Cache.Update > 0 {

		thread.Cache.Update = 0

		ctrler.object.Update(plugin.Bson{
			"config." + define.TASK_MODE_QZONEREMARK + ".threads." + strconv.Itoa(index): thread,
		})
	}

	return
}

func (ctrler *ctrler_task_qzoneremark) robot_lineup(thread *model.QzoneremarkThread) bool {

	if thread.Index <= ctrler.self.IndexLimit-1 && thread.Index <= len(ctrler.qzone_remark_rule.List)-1 {
		return false
	}

	ctrler.self.Robotdb.Foreach(0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.QzoneremarkRobotdbValue) {
		value.Status = model.QZONEREMARK_ROBOTDB_SATAUS_FINISH
		ctrler.self.Cache.Update++
	})

	thread.Clean()
	thread.Cache.Update++

	return true
}

func (ctrler *ctrler_task_qzoneremark) robot_offline(thread *model.QzoneremarkThread) (err error) {

	ctrler.self.Robotdb.Foreach(0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.QzoneremarkRobotdbValue) {
		value.Status = model.QZONEREMARK_ROBOTDB_SATAUS_OFFLINE
		ctrler.self.Cache.Update++
	})

	thread.Clean()
	thread.Cache.Update++

	return
}

func (ctrler *ctrler_task_qzoneremark) proxy_get(thread *model.QzoneremarkThread, callback func() error) (err error) {

	var (
		robot_proxy *model.Proxy
	)

	thread.Cache.ProxyConfig = nil

	if robot_proxy = self.Proxys.Existed(thread.Cache.Robot.ProxyId); robot_proxy == nil {
		return fmt.Errorf("机器人静态代理不存在")
	}

	if thread.Cache.ProxyConfig = ctrler.self.GetProxy(&robot_proxy.Config.City, nil); thread.Cache.ProxyConfig != nil {
		goto END
	}

	if thread.Cache.ProxyConfig = ctrler.self.GetProxy(nil, &robot_proxy.Config.Province); thread.Cache.ProxyConfig != nil {
		goto END
	}

	if thread.Cache.ProxyConfig, err = ctrler.proxy_extractor.Get(&model.ProxyExtractorRequest{Ipzan: model.ProxyExtractorRequestIpzan{
		City:   robot_proxy.Config.City,
		Minute: ctrler.self.ProxyMinute,
	}}); err == nil {
		goto BUY
	}

	if thread.Cache.ProxyConfig, err = ctrler.proxy_extractor.Get(&model.ProxyExtractorRequest{Ipzan: model.ProxyExtractorRequestIpzan{
		Province: robot_proxy.Config.Province,
		Minute:   ctrler.self.ProxyMinute,
	}}); err != nil {
		return fmt.Errorf("代理提取异常：%s", err.Error())
	}

BUY:
	ctrler.self.AddProxy(thread.Cache.ProxyConfig)

	thread.ProxyExtractorTimer++
	thread.Cache.Update++

END:
	if thread.Cache.ProxyConfig == nil {
		return fmt.Errorf("代理提取失败")
	}

	err = callback()

	return
}

func (ctrler *ctrler_task_qzoneremark) robot_get(thread *model.QzoneremarkThread) (err error) {

FIND:
	fmt.Println("REMARK-DEBUG: robot_get FIND, RobotUid=", thread.RobotUid)

	if thread.Cache.Robot = self.Robots.ExistedByUid(thread.RobotUid); thread.Cache.Robot != nil {
		goto NEXT
	}

	if thread.RobotUid != 0 {

		ctrler.self.Robotdb.Foreach(0, thread.RobotUid, func(stop *bool, value *model.QzoneremarkRobotdbValue) {
			value.Status = model.QZONEREMARK_ROBOTDB_SATAUS_LOSS
			ctrler.self.Cache.Update++
		})

		thread.RobotUid = 0
		thread.Cache.Update++
	}

	ctrler.self.Robotdb.Foreach(model.QZONEREMARK_ROBOTDB_SATAUS_WAIT, 0, func(stop *bool, value *model.QzoneremarkRobotdbValue) {

		value.Status = model.QZONEREMARK_ROBOTDB_SATAUS_RUN
		ctrler.self.Cache.Update++

		thread.RobotUid = value.RobotUid
		thread.Cache.Update++
	})

	if thread.RobotUid != 0 {
		goto FIND
	}

	return fmt.Errorf("已闲置")

NEXT:
	if err = ctrler.object.CheckRobotLog(thread.RobotUid); err != nil {
		return err
	}

	// PATCHED: skip ONLINE check
	if false {
		ctrler.robot_offline(thread)
		return fmt.Errorf("已闲置")
	}

	return
}

func (ctrler *ctrler_task_qzoneremark) qzone_handle(thread *model.QzoneremarkThread) (err error) {

	var (
		result model.RobotQzoneMainResult
	)

	if thread.MasterUid, err = ctrler.qzonedb.Extract(); err != nil {
		return fmt.Errorf("母料库已用完")
	}

	ctrler.self.Robotdb.Foreach(0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.QzoneremarkRobotdbValue) {
		value.MasterTotal++
		ctrler.self.Cache.Update++
	})

	if result, err = thread.Cache.Robot.GetQzoneMain(thread.MasterUid, nil); err != nil {
		return fmt.Errorf("访问空间请求异常：%s", err.Error())
	}

	if result.Visit.LastTime == 0 || result.Visit.TotalNum == 0 {
		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("空间：%d，无访问权限！", thread.MasterUid))
		return
	}

	ctrler.self.Robotdb.Foreach(0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.QzoneremarkRobotdbValue) {
		value.Master++
		ctrler.self.Cache.Update++
	})

	ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("空间：%d，访问成功。", thread.MasterUid))
	thread.Step = model.QZONEREMARK_STEP_APPLIST
	thread.Cache.Update++

	return
}

func (ctrler *ctrler_task_qzoneremark) applist_handle(thread *model.QzoneremarkThread) (err error) {

	var (
		result model.RobotApplistResult
	)

	thread.OrgLikeKey = ""
	thread.CurLikeKey = ""
	thread.CellId = ""
	thread.CellLikeNum = 0
	thread.CellCommentNum = 0

	if result, err = thread.Cache.Robot.GetApplist(thread.MasterUid, 20, nil); err != nil {
		return fmt.Errorf("查看说说请求异常：%s", err.Error())
	}

	find_number := 0

	if result.AllApplistData != nil {

		for _, app := range *result.AllApplistData {

			find_number++

			if thread.OrgLikeKey != "" {
				continue
			}

			thread.OrgLikeKey = app.CellComm.OrgLikeKey
			thread.CurLikeKey = app.CellComm.CurLikeKey
			thread.CellId = app.CellId.CellId
			thread.CellLikeNum = app.CellLike.Num
			thread.CellCommentNum = app.CellComment.Num
		}
	}

	if thread.OrgLikeKey == "" || thread.CurLikeKey == "" || thread.CellId == "" {
		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("空间：%d，未查找到已发布的说说！(%d)", thread.MasterUid, find_number))
		thread.Step = model.QZONEREMARK_STEP_QZONE
		thread.Cache.Update++
		return
	}

	ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("空间：%d，查找到已发布的说说。(%d)", thread.MasterUid, find_number))
	thread.Step = model.QZONEREMARK_STEP_REMARK
	thread.Cache.Update++

	return
}

func (ctrler *ctrler_task_qzoneremark) remark_handle(thread *model.QzoneremarkThread) (err error) {

	var (
		rule = ctrler.qzone_remark_rule.List[thread.Index]

		proxy_info string
		web_info   string
	)

	handle := func() (err error) {

		if thread.Cache.ProxyConfig != nil {
			proxy_info = fmt.Sprintf("代理：（%s:%d）", thread.Cache.ProxyConfig.Ip, thread.Cache.ProxyConfig.Port)
		}

		if rule.Web {
			web_info = "（网页模式）"
		}

		switch rule.Action {

		case model.TASK_QZONE_REMARK_RULE_ACTION_LIKE:

			if _, err = thread.Cache.Robot.QzoneLike(thread.MasterUid, thread.OrgLikeKey, thread.CurLikeKey, rule.Web, thread.Cache.ProxyConfig); err == nil {

				ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("空间：%d，点赞已完成。%s%s", thread.MasterUid, proxy_info, web_info))
			}

		case model.TASK_QZONE_REMARK_RULE_ACTION_COMMENT:

			var (
				comment = ctrler.qzone_remark_rule.Comments[utils.Random(0, len(ctrler.qzone_remark_rule.Comments)-1)]
			)

			if _, err = thread.Cache.Robot.QzoneComment(thread.MasterUid, thread.CellId, comment, rule.Web, thread.Cache.ProxyConfig); err == nil {

				ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("空间：%d，评论已完成：\"%s\"。%s%s", thread.MasterUid, comment, proxy_info, web_info))
			}
		}

		if err != nil {

			if strings.Contains(err.Error(), "socks") {
				return
			}

			ctrler.self.Robotdb.Foreach(0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.QzoneremarkRobotdbValue) {
				value.Warn++
				ctrler.self.Cache.Update++
			})

			ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("空间：%d，%s异常：%s", thread.MasterUid, model.TASK_QZONE_REMARK_RULE_ACTION_NAMES[rule.Action], err.Error()))

			thread.SetTimer(ctrler.qzone_remark_rule)

			thread.Index++

			thread.Step = model.QZONEREMARK_STEP_QZONE

			thread.Cache.Update++

			ctrler.robot_lineup(thread)

			return nil
		}

		return
	}

	if rule.IpDynamic {

		if ctrler.self.ProxyExtractorId.IsZero() {
			return fmt.Errorf("未配置代理生成器")
		}

		err = ctrler.proxy_get(thread, handle)

	} else {
		err = handle()
	}

	if err != nil {
		return
	}

	thread.Action = rule.Action

	thread.Step = model.QZONEREMARK_STEP_REMARK_REPLAY

	thread.Cache.Update++

	return
}

func (ctrler *ctrler_task_qzoneremark) remark_replay_handle(thread *model.QzoneremarkThread) (err error) {

	var (
		result  model.RobotApplistResult
		existed bool
	)

	thread.Cache.Robot.GetQzoneMain(thread.MasterUid, nil)

	time.Sleep(time.Duration(utils.Random(1000, 2000)) * time.Millisecond)

	if result, err = thread.Cache.Robot.GetApplist(thread.MasterUid, 20, nil); err != nil {
		return fmt.Errorf("查看说说请求异常：%s", err.Error())
	}

	if result.AllApplistData != nil {

		for _, app := range *result.AllApplistData {

			switch thread.Action {

			case model.TASK_QZONE_REMARK_RULE_ACTION_LIKE:

				if app.CellLike.Likemans != nil && len(*app.CellLike.Likemans) > 0 {

					for _, item := range *app.CellLike.Likemans {

						if item.User.Uin == thread.RobotUid {
							existed = true
							goto NEXT
						}
					}

				} else {

					if app.CellLike.Num > thread.CellLikeNum {
						existed = true
						goto NEXT
					}
				}

			case model.TASK_QZONE_REMARK_RULE_ACTION_COMMENT:

				if app.CellComment.Comment != nil && len(*app.CellComment.Comment) > 0 {

					for _, item := range *app.CellComment.Comment {

						if item.User.Uin == thread.RobotUid {
							existed = true
							goto NEXT
						}
					}

				} else {

					if app.CellComment.Num > thread.CellCommentNum {
						existed = true
						goto NEXT
					}
				}
			}
		}
	}

NEXT:

	if existed {

		ctrler.self.Robotdb.Foreach(0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.QzoneremarkRobotdbValue) {
			value.Succ++
			ctrler.self.Cache.Update++
		})

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("空间：%d，留痕成功。", thread.MasterUid))

	} else {

		ctrler.self.Robotdb.Foreach(0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.QzoneremarkRobotdbValue) {
			value.Fail++
			ctrler.self.Cache.Update++
		})

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("空间：%d，留痕失败！", thread.MasterUid))
	}

	thread.SetTimer(ctrler.qzone_remark_rule)

	thread.Index++

	thread.Step = model.QZONEREMARK_STEP_QZONE

	thread.Cache.Update++

	ctrler.robot_lineup(thread)

	return
}
