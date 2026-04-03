package ctrler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"ymlink-q2/apps/server/self"
	"ymlink-q2/define"
	"ymlink-q2/ini"
	"ymlink-q2/model"
	"ymlink-q2/plugin"
	"ymlink-q2/utils"

	"github.com/dustin/go-humanize"
)

func (ctrler *ctrler_task_materialgreet) greet_running() {

	defer func() {
		if info := recover(); info != nil {
		}
		ctrler.greet_isrun = false
	}()

	for {
		time.Sleep(time.Second)

		if !ctrler.greet_isrun || !ctrler.message_isrun || !ctrler.hello_isrun {
			return
		}

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

			go ctrler.thread_running(index, thread)
		}

	UPDATE:
		if ctrler.self.Cache.GreetUpdate > 0 {

			ctrler.self.Cache.GreetUpdate = 0

			if err := ctrler.object.Update(plugin.Bson{
				"config." + define.TASK_MODE_MATERIALGREET + ".robotdb": &ctrler.self.Robotdb,
			}); err != nil {
				ctrler.object.Cache.Error = fmt.Sprint("持久化存储异常：", err.Error())
			}
		}
	}
}

func (ctrler *ctrler_task_materialgreet) check() (status int) {

	var finished bool = true

	if ctrler.object.Cache.Status == define.TASK_STATUS_FINISH {
		return
	}

	if ctrler.self = ctrler.object.Config.Materialgreet; ctrler.self == nil {
		status = define.TASK_STATUS_CONFIG_NULL
		goto END
	}

	ctrler.self.Robotdb.Foreach(true, 0, 0, func(stop *bool, value *model.MaterialgreetRobotdbValue) {
		if value.Status == model.MATERIALGREET_ROBOTDB_SATAUS_RUN || value.Status == model.MATERIALGREET_ROBOTDB_SATAUS_WAIT {
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

		ctrler.self.Cache.GreetUpdate++

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

	if ctrler.user = self.Users.Existed(ctrler.object.UserId); ctrler.user == nil {
		status = define.TASK_STATUS_USER_NULL
		goto END
	}

	if ctrler.object = self.Tasks.Existed(ctrler.object.Id); ctrler.object == nil {
		status = define.TASK_STATUS_NULL
		return
	}

	// [PATCH] Allow running without proxy extractor - will use robot's static proxy
	ctrler.proxy_extractor = self.ProxyExtractors.Existed(ctrler.self.ProxyExtractorId)
	// proxy_extractor can be nil, proxy_get will handle fallback

	if ctrler.greet_word = self.TaskGreetWords.Existed(ctrler.self.GreetWordId); ctrler.greet_word == nil || len(ctrler.greet_word.List) == 0 {
		status = define.TASK_STATUS_WORD_NULL
		goto END
	}

	if ctrler.greet_word2 = self.TaskGreetWords.Existed(ctrler.self.GreetWordId); ctrler.greet_word2 == nil || len(ctrler.greet_word2.List) == 0 {
		status = define.TASK_STATUS_WORD_NULL
		goto END
	}

	if ctrler.greet_rule = self.TaskGreetRules.Existed(ctrler.self.GreetRuleId); ctrler.greet_rule == nil || len(ctrler.greet_rule.List) == 0 {
		status = define.TASK_STATUS_RULE_NULL
		goto END
	}

	if ctrler.materialdb = self.TaskMaterialdbs.Existed(ctrler.object.UserId, ctrler.self.MaterialdbName); ctrler.materialdb == nil {

		if ctrler.realinfodb = self.TaskRealinfodbs.Existed(ctrler.object.UserId, ctrler.self.RealinfodbName); ctrler.realinfodb == nil {
			status = define.TASK_STATUS_MATERIALDB_NULL
			goto END
		}
	}

	status = define.TASK_STATUS_RUN
	ctrler.object.Cache.Error = ""

END:
	ctrler.object.Cache.Status = status

	return
}

func (ctrler *ctrler_task_materialgreet) thread_running(index int, thread *model.MaterialgreetThread) (err error) {

	defer func() {

		if info := recover(); info != nil {
			thread.Cache.Error = fmt.Sprint("系统错误：", info)
		}

		thread.Cache.Isrun = false
	}()

	if !thread.GetTimer(ctrler.greet_rule) {
		return
	}

	defer thread.SetTimer(nil)

	if err = ctrler.robot_get(thread); err != nil {
		goto END
	}

	if ctrler.robot_lineup(thread) {
		return
	}

	if thread.Cache.ProxyBlackList == nil {
		thread.Cache.ProxyBlackList = []string{}
	}

	switch thread.Step {

	case model.MATERIALGREET_STEP_BEFORE:

		if err = ctrler.before_handle(thread); err != nil {
			thread.SetTimer(ctrler.greet_rule)
		}

	case model.MATERIALGREET_STEP_VISTER:

		err = ctrler.visitor_handle(thread)

	case model.MATERIALGREET_STEP_SETTING:

		err = ctrler.setting_handle(thread)

	case model.MATERIALGREET_STEP_INFO:

		err = ctrler.info_handle(thread)

	case model.MATERIALGREET_STEP_GREET:

		err = ctrler.proxy_get(thread, ctrler.greet_handle)

	case model.MATERIALGREET_STEP_GREET_REPLAY:

		err = ctrler.greet_replay_handle(thread)
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

		if strings.Contains(thread.Cache.Error, "网页续签失败") {
			ctrler.robot_web_renew_fail(thread)
		}
	}

	if thread.Cache.Update > 0 {

		thread.Cache.Update = 0

		ctrler.object.Update(plugin.Bson{
			"config." + define.TASK_MODE_MATERIALGREET + ".threads." + strconv.Itoa(index): thread,
		})
	}

	return
}

func (ctrler *ctrler_task_materialgreet) robot_lineup(thread *model.MaterialgreetThread) bool {

	if thread.Index <= ctrler.self.IndexLimit-1 && thread.Index <= len(ctrler.greet_rule.List)-1 {
		return false
	}

	ctrler.self.Robotdb.Foreach(false, 0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.MaterialgreetRobotdbValue) {
		value.Status = model.MATERIALGREET_ROBOTDB_SATAUS_FINISH
		ctrler.self.Cache.GreetUpdate++
	})

	thread.Clean()
	thread.Cache.Update++

	return true
}

func (ctrler *ctrler_task_materialgreet) robot_offline(thread *model.MaterialgreetThread) {

	ctrler.self.Robotdb.Foreach(false, 0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.MaterialgreetRobotdbValue) {
		value.Status = model.MATERIALGREET_ROBOTDB_SATAUS_OFFLINE
		ctrler.self.Cache.GreetUpdate++
		*stop = true
	})

	thread.Clean()
	thread.Cache.Update++
}

func (ctrler *ctrler_task_materialgreet) robot_web_renew_fail(thread *model.MaterialgreetThread) {

	ctrler.self.Robotdb.Foreach(false, 0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.MaterialgreetRobotdbValue) {
		value.Status = model.MATERIALGREET_ROBOTDB_SATAUS_WEB_RENEW_FAIL
		ctrler.self.Cache.GreetUpdate++
		*stop = true
	})

	thread.Clean()
	thread.Cache.Update++
}

func (ctrler *ctrler_task_materialgreet) proxy_get(thread *model.MaterialgreetThread, callback func(*model.MaterialgreetThread) error) (err error) {

	var (
		robot_proxy *model.Proxy
		province    string
	)

	thread.Cache.ProxyConfig = nil

	if robot_proxy = self.Proxys.Existed(thread.Cache.Robot.ProxyId); robot_proxy == nil {
		return fmt.Errorf("机器人静态代理不存在")
	}

	// [PATCH] If proxy extractor is available, use it; otherwise use robot's static proxy
	if ctrler.proxy_extractor != nil {
		if province = define.FindClosestProvince(robot_proxy.Config.Province, thread.Cache.ProxyBlackList); province == "" {
			province = robot_proxy.Config.Province
		}

		if thread.Cache.ProxyConfig, err = ctrler.proxy_extractor.Get(&model.ProxyExtractorRequest{Ipzan: model.ProxyExtractorRequestIpzan{
			Province: province,
			Minute:   1,
		}}); err != nil {
			// Fall back to static proxy on extractor failure
			thread.Cache.ProxyConfig = &robot_proxy.Config
		}
	} else {
		// No proxy extractor configured, use robot's static proxy directly
		thread.Cache.ProxyConfig = &robot_proxy.Config
	}

	thread.ProxyExtractorTimer++
	thread.Cache.Update++

	err = callback(thread)

	return
}

func (ctrler *ctrler_task_materialgreet) robot_get(thread *model.MaterialgreetThread) (err error) {

FIND:

	if thread.Cache.Robot = self.Robots.ExistedByUid(thread.RobotUid); thread.Cache.Robot != nil {
		goto NEXT
	}

	if thread.RobotUid != 0 {

		ctrler.self.Robotdb.Foreach(false, 0, thread.RobotUid, func(stop *bool, value *model.MaterialgreetRobotdbValue) {
			value.Status = model.MATERIALGREET_ROBOTDB_SATAUS_LOSS
			ctrler.self.Cache.GreetUpdate++
			*stop = true
		})

		thread.RobotUid = 0
		thread.Cache.Update++
	}

	ctrler.self.Robotdb.Foreach(true, model.MATERIALGREET_ROBOTDB_SATAUS_WAIT, 0, func(stop *bool, value *model.MaterialgreetRobotdbValue) {

		value.Status = model.MATERIALGREET_ROBOTDB_SATAUS_RUN
		ctrler.self.Cache.GreetUpdate++

		thread.RobotUid = value.RobotUid
		thread.Cache.Update++
		thread.Index = value.Succ + value.Fail + value.Warn

		*stop = true
	})

	if thread.RobotUid != 0 {
		goto FIND
	}

	return fmt.Errorf("已闲置")

NEXT:
	if err = ctrler.object.CheckRobotLog(thread.RobotUid); err != nil {
		return
	}

	if !utils.ContainsInt(&thread.Cache.Robot.Status.Statistic, model.ROBOT_STATISTIC_ONLINE) {
		ctrler.robot_offline(thread)
		return fmt.Errorf("已闲置")
	}

	return
}

func (ctrler *ctrler_task_materialgreet) before_handle(thread *model.MaterialgreetThread) (err error) {

	// 取料子

	switch {

	case ctrler.materialdb != nil:

		if ctrler.greet_rule.List[thread.Index].Channels == define.GREET_RULE_CHANNELS_10028_1 && !ctrler.materialdb.OnMaster {
			return fmt.Errorf(`不带母号的子料库，不支持加人通道类型：空间访客！`)
		}

		if thread.VisitorUid, thread.MasterUid, err = ctrler.materialdb.Extract(); err != nil {
			return fmt.Errorf("子料库已用完")
		}

		thread.Mobile = 0
		thread.Name = ""

		thread.AuthType = 0
		thread.SearchToken = ""

	case ctrler.realinfodb != nil:

		if thread.VisitorUid, thread.MasterUid, thread.Mobile, thread.Name, err = ctrler.realinfodb.Extract(); err != nil {
			return fmt.Errorf("实名子料库已用完")
		}

		thread.AuthType = 0
		thread.SearchToken = ""
	}

	ctrler.self.Robotdb.Foreach(false, 0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.MaterialgreetRobotdbValue) {
		value.Material++
		ctrler.self.Cache.GreetUpdate++
		*stop = true
	})

	// 前置动作

	switch ctrler.greet_rule.List[thread.Index].Channels {

	case define.GREET_RULE_CHANNELS_10028_1:

		var res model.RobotQzoneMainResult

		begin := time.Now().UnixMilli()
		res, err = thread.Cache.Robot.GetQzoneMain(thread.MasterUid, nil)
		usetime := time.Now().UnixMilli() - begin

		if err != nil {
			return fmt.Errorf("[%s] 母料空间：%d，访问请求异常：%s", humanize.Comma(usetime), thread.MasterUid, err.Error())

		} else if res.Visit.LastTime == 0 || res.Visit.TotalNum == 0 {
			ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 母料空间：%d，无访问权限！", humanize.Comma(usetime), thread.MasterUid))
			return
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 母料空间：%d，访问成功。", humanize.Comma(usetime), thread.MasterUid))
		thread.Step = model.MATERIALGREET_STEP_VISTER

	case define.GREET_RULE_CHANNELS_2081_1:

		begin := time.Now().UnixMilli()
		_, err = thread.Cache.Robot.GetQzoneMain(thread.VisitorUid, nil)
		usetime := time.Now().UnixMilli() - begin

		if err != nil {
			return fmt.Errorf("[%s] 子料空间：%d，访问请求异常：%s", humanize.Comma(usetime), thread.VisitorUid, err.Error())
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料空间：%d，访问成功。", humanize.Comma(usetime), thread.VisitorUid))
		thread.Step = model.MATERIALGREET_STEP_SETTING

	case define.GREET_RULE_CHANNELS_2020_4:

		var res model.RobotSearchResult

		begin := time.Now().UnixMilli()
		res, err = thread.Cache.Robot.Search(strconv.Itoa(thread.VisitorUid))
		usetime := time.Now().UnixMilli() - begin

		if err != nil {
			return fmt.Errorf("[%s] 子料：%d，搜索用户请求异常：%s", humanize.Comma(usetime), thread.VisitorUid, err.Error())
		}

		for _, item := range *res.List {
			if item.Uin == thread.VisitorUid {
				thread.SearchToken = item.Token
				break
			}
		}

		if thread.SearchToken == "" {
			ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，搜索不到用户！", humanize.Comma(usetime), thread.VisitorUid))
			break
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，搜索用户成功。", humanize.Comma(usetime), thread.VisitorUid))
		thread.Step = model.MATERIALGREET_STEP_SETTING

	case define.GREET_RULE_CHANNELS_2050_1:

		var res model.RobotShortvideoMainResult

		begin := time.Now().UnixMilli()
		res, err = thread.Cache.Robot.GetShortvideoMain(thread.VisitorUid)
		usetime := time.Now().UnixMilli() - begin

		if err != nil {
			return fmt.Errorf("[%s] 子料：%d，访问小世界请求异常：%s", humanize.Comma(usetime), thread.VisitorUid, err.Error())
		}

		if res.TemplateInfo.QQLogo == "" {
			ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，访问小世界失败！", humanize.Comma(usetime), thread.VisitorUid))
			break
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，访问小世界成功。", humanize.Comma(usetime), thread.VisitorUid))
		thread.Step = model.MATERIALGREET_STEP_SETTING

	case define.GREET_RULE_CHANNELS_2001_0:

		begin := time.Now().UnixMilli()
		res, err := thread.Cache.Robot.Search(strconv.Itoa(thread.VisitorUid))
		usetime := time.Now().UnixMilli() - begin

		if err != nil {
			return fmt.Errorf("[%s] 子料：%d，搜索用户请求异常：%s", humanize.Comma(usetime), thread.VisitorUid, err.Error())
		} else {
			existed := false
			for _, item := range *res.List {
				if item.Uin == thread.VisitorUid {
					existed = true
					break
				}
			}
			if !existed {
				ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，搜索不到用户！", humanize.Comma(usetime), thread.VisitorUid))
				break
			}
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，搜索用户成功。", humanize.Comma(usetime), thread.VisitorUid))
		time.Sleep(time.Duration(utils.Random(1, 3)) * time.Second)

		begin = time.Now().UnixMilli()
		res2, err := thread.Cache.Robot.AddBlacklist(thread.VisitorUid)
		usetime = time.Now().UnixMilli() - begin

		if err != nil {
			return fmt.Errorf("[%s] 子料：%d，添加黑名单请求异常：%s", humanize.Comma(usetime), thread.VisitorUid, err.Error())
		} else if len(res2) == 0 {
			ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，添加黑名单失败！", humanize.Comma(usetime), thread.VisitorUid))
			break
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，添加黑名单成功。", humanize.Comma(usetime), thread.VisitorUid))
		time.Sleep(time.Duration(utils.Random(1, 3)) * time.Second)

		begin = time.Now().UnixMilli()
		res3, err := thread.Cache.Robot.GetBlacklist()
		usetime = time.Now().UnixMilli() - begin

		if err != nil {
			return fmt.Errorf("[%s] 子料：%d，查看黑名单请求异常：%s", humanize.Comma(usetime), thread.VisitorUid, err.Error())
		} else {
			existed := false
			for _, item := range res3.Items {
				if item.UinWrap.Uin == thread.VisitorUid {
					existed = true
					break
				}
			}
			if !existed {
				ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，黑名单查询不到目标！", humanize.Comma(usetime), thread.VisitorUid))
				break
			}
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，黑名单查询目标成功。", humanize.Comma(usetime), thread.VisitorUid))
		time.Sleep(time.Duration(utils.Random(1, 3)) * time.Second)

		begin = time.Now().UnixMilli()
		res4, err := thread.Cache.Robot.DelBlacklist(thread.VisitorUid)
		usetime = time.Now().UnixMilli() - begin

		if err != nil {
			return fmt.Errorf("[%s] 子料：%d，删除黑名单请求异常：%s", humanize.Comma(usetime), thread.VisitorUid, err.Error())
		} else if len(res4) == 0 {
			ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，删除黑名单失败！", humanize.Comma(usetime), thread.VisitorUid))
			break
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，删除黑名单成功。", humanize.Comma(usetime), thread.VisitorUid))
		thread.Step = model.MATERIALGREET_STEP_SETTING

	case define.GREET_RULE_CHANNELS_2011_0:

		begin := time.Now().UnixMilli()
		res, err := thread.Cache.Robot.Search(strconv.Itoa(thread.VisitorUid))
		usetime := time.Now().UnixMilli() - begin

		if err != nil {
			return fmt.Errorf("[%s] 子料：%d，搜索用户请求异常：%s", humanize.Comma(usetime), thread.VisitorUid, err.Error())
		} else {
			existed := false
			for _, item := range *res.List {
				if item.Uin == thread.VisitorUid {
					existed = true
					break
				}
			}
			if !existed {
				ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，搜索不到用户！", humanize.Comma(usetime), thread.VisitorUid))
				break
			}
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，搜索用户成功。", humanize.Comma(usetime), thread.VisitorUid))
		time.Sleep(time.Duration(utils.Random(1, 3)) * time.Second)

		begin = time.Now().UnixMilli()
		_, err = thread.Cache.Robot.GetQzoneMain(thread.VisitorUid, nil)
		usetime = time.Now().UnixMilli() - begin

		if err != nil {
			return fmt.Errorf("[%s] 子料：%d，访问空间请求异常：%s", humanize.Comma(usetime), thread.VisitorUid, err.Error())
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，访问空间成功。", humanize.Comma(usetime), thread.VisitorUid))
		time.Sleep(time.Duration(utils.Random(1, 3)) * time.Second)

		begin = time.Now().UnixMilli()
		_, err = thread.Cache.Robot.GetQzoneMain(thread.Cache.Robot.Kernel.UserLoginData.Uin, nil)
		usetime = time.Now().UnixMilli() - begin

		if err != nil {
			return fmt.Errorf("[%s] 访问自己空间请求异常：%s", humanize.Comma(usetime), err.Error())
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 访问自己空间成功。", humanize.Comma(usetime)))
		time.Sleep(time.Duration(utils.Random(1, 3)) * time.Second)

		begin = time.Now().UnixMilli()
		res2, err := thread.Cache.Robot.GetVisitorRecord(nil)
		usetime = time.Now().UnixMilli() - begin

		if err != nil {
			return fmt.Errorf("[%s] 查我看过谁请求异常：%s", humanize.Comma(usetime), err.Error())
		} else if len(res2.Data.Visit.Datalist) == 0 {
			ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，未查询到访问其空间的记录！", humanize.Comma(usetime), thread.VisitorUid))
			break
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，\"我看过谁\" 中查询目标成功。", humanize.Comma(usetime), thread.VisitorUid))
		thread.Step = model.MATERIALGREET_STEP_SETTING

	default:
		return fmt.Errorf("不支持的通道：%s", ctrler.greet_rule.List[thread.Index].Channels)
	}

	thread.Cache.Update++

	return
}

func (ctrler *ctrler_task_materialgreet) visitor_handle(thread *model.MaterialgreetThread) (err error) {

	var (
		result     model.RobotQzoneVisitorResult
		try_number int
	)

TRY:
	begin := time.Now().UnixMilli()
	result, err = thread.Cache.Robot.GetQzoneVisitor(thread.MasterUid, nil)
	usetime := time.Now().UnixMilli() - begin

	if err != nil {
		return err
	}

	if result.Ret != 0 || result.Data.Visit.TotalNum == 0 || len(result.Data.Visit.DataList) == 0 {

		if try_number++; try_number <= 3 {
			time.Sleep(time.Second)
			goto TRY
		}

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 空间：%d，查看访客列表失败！（%s）", humanize.Comma(usetime), thread.MasterUid, result.Msg))
		thread.Step = model.MATERIALGREET_STEP_BEFORE
		thread.Cache.Update++

		return
	}

	existed := false

	for _, visitor := range result.Data.Visit.DataList {
		if visitor.Uin == thread.VisitorUid {
			existed = true
			break
		}
	}

	if !existed {
		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 空间：%d，查看访客列表，目标子料：%d 不存在！", humanize.Comma(usetime), thread.MasterUid, thread.VisitorUid))
		thread.Step = model.MATERIALGREET_STEP_BEFORE
		thread.Cache.Update++
		return
	}

	ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 空间：%d，查看访客列表，存在目标子料：%d。", humanize.Comma(usetime), thread.MasterUid, thread.VisitorUid))
	thread.Step = model.MATERIALGREET_STEP_SETTING
	thread.Cache.Update++

	return
}

func (ctrler *ctrler_task_materialgreet) setting_handle(thread *model.MaterialgreetThread) (err error) {

	var (
		result       model.AddFriendSettingResult
		setting_name interface{}

		channels = ctrler.greet_rule.List[thread.Index].Channels
		question string
	)

	begin := time.Now().UnixMilli()
	result, err = thread.Cache.Robot.GetAddFriendSetting(thread.VisitorUid, channels, nil)
	usetime := time.Now().UnixMilli() - begin

	if err != nil {
		return fmt.Errorf("[%s] 查询添加方式异常：%s", humanize.Comma(usetime), err.Error())
	}

	for idx, item := range result.UserQuestion {
		question += item
		if idx != len(result.UserQuestion)-1 {
			question += ","
		}
	}

	if question != "" {
		question = "（" + question + "）"
	}

	if setting_name = define.FILTER_AUTH_TYPE_NAME[result.QueryUinSetting]; setting_name == nil {
		setting_name = fmt.Sprintf("未知的添加方式：%d", result.QueryUinSetting)
	}

	if !ctrler.self.AuthTypeFiltering(result.QueryUinSetting) {
		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，查询添加方式，当前为：%s，不符合加人要求。%s", humanize.Comma(usetime), thread.VisitorUid, setting_name, question))
		thread.Step = model.MATERIALGREET_STEP_BEFORE
		thread.Cache.Update++
		return
	}

	thread.AuthType = result.QueryUinSetting
	thread.Questions = result.UserQuestion
	thread.Cache.Update++

	ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，查询添加方式，当前为：%s，符合加人要求。%s", humanize.Comma(usetime), thread.VisitorUid, setting_name, question))
	thread.Step = model.MATERIALGREET_STEP_INFO

	return
}

func (ctrler *ctrler_task_materialgreet) info_handle(thread *model.MaterialgreetThread) (err error) {

	var (
		result model.SummaryCardResult
	)

	switch ctrler.greet_rule.List[thread.Index].Channels {
	case define.GREET_RULE_CHANNELS_2050_1, define.GREET_RULE_CHANNELS_2001_0:
	default:
		thread.Step = model.MATERIALGREET_STEP_GREET
		thread.Cache.Update++
		return
	}

	begin := time.Now().UnixMilli()
	result, err = thread.Cache.Robot.GetSummaryCard(thread.VisitorUid, nil)
	usetime := time.Now().UnixMilli() - begin

	if err != nil {
		return fmt.Errorf("[%s] 查询资料异常：%s", humanize.Comma(usetime), err.Error())
	}

	if ctrler.self.Filtered {

		if !ctrler.self.InfoFiltering(result.Sex, result.Age, result.Level) || result.ULoginDays < ctrler.self.LoginDays {
			ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，查资料，昵称：%s，性别：%s，年龄：%d，等级：%d，达人天数：%d，不符合筛选要求。", humanize.Comma(usetime), thread.VisitorUid, result.Nick, define.FILTER_SEX_NAME[result.Sex], result.Age, result.Level, result.ULoginDays))
			thread.Step = model.MATERIALGREET_STEP_BEFORE
			thread.Cache.Update++
			return
		}
	}

	ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，查资料，昵称：%s，性别：%s，年龄：%d，等级：%d，达人天数：%d，符合要求。", humanize.Comma(usetime), thread.VisitorUid, result.Nick, define.FILTER_SEX_NAME[result.Sex], result.Age, result.Level, result.ULoginDays))

	thread.Step = model.MATERIALGREET_STEP_GREET
	thread.Cache.Update++

	return
}

func (ctrler *ctrler_task_materialgreet) greet_handle(thread *model.MaterialgreetThread) (err error) {

	var (
		result model.GreetResult

		channels   = ctrler.greet_rule.List[thread.Index].Channels
		remark     = fmt.Sprintf("%d-%d", thread.MasterUid, thread.VisitorUid)
		greet_word = ""

		busy_stop   = false
		fail_stop   = false
		socks5_stop = false
	)

	switch thread.AuthType {
	case 4:
		for idx := range thread.Questions {
			var answer string
			if idx < len(ctrler.greet_word2.List) {
				answer = ctrler.greet_word2.List[idx]
			} else {
				answer = ctrler.greet_word2.List[len(ctrler.greet_word2.List)-1]
			}
			greet_word += fmt.Sprintf("问题%d:%s\n回答:%s", idx+1, thread.Questions[idx], answer)
			if idx != len(thread.Questions)-1 {
				greet_word += "\n"
			}
		}
	default:
		greet_word = ctrler.greet_word.List[utils.Random(0, len(ctrler.greet_word.List)-1)]
	}

	if thread.Name != "" {
		greet_word = strings.ReplaceAll(greet_word, "[姓名]", thread.Name)
		greet_word = strings.ReplaceAll(greet_word, "[名字]", thread.Name)
	}

	if thread.Mobile != 0 {
		greet_word = strings.ReplaceAll(greet_word, "[电话]", strconv.Itoa(thread.Mobile))
		greet_word = strings.ReplaceAll(greet_word, "[手机]", strconv.Itoa(thread.Mobile))
		greet_word = strings.ReplaceAll(greet_word, "[号码]", strconv.Itoa(thread.Mobile))
	}

	begin := time.Now().UnixMilli()
	result, err = thread.Cache.Robot.Greet(thread.VisitorUid, thread.AuthType, channels, remark, greet_word, thread.SearchToken, thread.Cache.ProxyConfig)
	usetime := time.Now().UnixMilli() - begin

	if result.ErrorCode == 170 {
		result.ErrorCode = 0
		err = nil
	}

	if err == nil && (result.Result != 0 || result.ErrorCode != 0) {
		err = fmt.Errorf("[%s] %d: %s%s", humanize.Comma(usetime), result.ErrorCode, result.Errorstring, result.ErrorString)
	}

	if err != nil {

		ctrler.self.Robotdb.Foreach(false, 0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.MaterialgreetRobotdbValue) {

			if strings.Contains(err.Error(), "请稍后再试") {

				value.Busy++

			} else if strings.Contains(err.Error(), "socks5") {

				socks5_stop = true

			} else {
				value.Fail++
			}

			if ctrler.self.BusyLimit > 0 && value.Busy >= ctrler.self.BusyLimit {
				value.Status = model.MATERIALGREET_ROBOTDB_SATAUS_BUSY
				busy_stop = true
			}

			if ctrler.self.FailLimit > 0 && value.Fail >= ctrler.self.FailLimit {
				value.Status = model.MATERIALGREET_ROBOTDB_SATAUS_FAIL
				fail_stop = true
			}

			thread.Cache.Update++
			*stop = true
		})

		ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf("[%s] 子料：%d，好友申请发送失败：%s", humanize.Comma(usetime), thread.VisitorUid, err.Error()))

		if strings.Contains(err.Error(), "Ban state forbit") {

			ctrler.self.Robotdb.Foreach(false, 0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.MaterialgreetRobotdbValue) {

				ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf(`[%s] 机器人：%d，社交功能被限制，中止任务！`, humanize.Comma(usetime), thread.RobotUid))

				value.Status = model.MATERIALGREET_ROBOTDB_SATAUS_Limit
				ctrler.self.Cache.GreetUpdate++

				thread.Clean()
				thread.Cache.Update++

				*stop = true
			})

		} else if busy_stop || fail_stop {

			thread.Clean()
			thread.Cache.Update++

		} else if socks5_stop {

			thread.Step = model.MATERIALGREET_STEP_GREET
			thread.Timer = time.Now().UnixMilli() + 5*60*1000
			thread.Cache.Update++

		} else {

			thread.SetTimer(ctrler.greet_rule)
			thread.Step = model.MATERIALGREET_STEP_BEFORE
			thread.Cache.Update++
		}

		err = nil

		return
	}

	ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf(`[%s] 子料：%d，好友申请发送成功！招呼语："%s"，代理：%s:%d。`, humanize.Comma(usetime), thread.VisitorUid, greet_word, thread.Cache.ProxyConfig.Ip, thread.Cache.ProxyConfig.Port))
	thread.Step = model.MATERIALGREET_STEP_GREET_REPLAY
	thread.Cache.Update++

	return
}

func (ctrler *ctrler_task_materialgreet) greet_replay_handle(thread *model.MaterialgreetThread) (err error) {

	var (
		result model.GreetReplyResult

		channels = ctrler.greet_rule.List[thread.Index].Channels

		warn_existed = false
		warn_stop    = false
	)

	begin := time.Now().UnixMilli()
	result, err = thread.Cache.Robot.GetGreetReply(thread.VisitorUid, thread.AuthType, channels, nil)
	usetime := time.Now().UnixMilli() - begin

	if err != nil {
		thread.Step = model.MATERIALGREET_STEP_BEFORE
		thread.Cache.Update++
		return fmt.Errorf("[%s] 查看好友申请过滤结果异常：%s", humanize.Comma(usetime), err.Error())
	}

	ctrler.self.Robotdb.Foreach(true, 0, thread.Cache.Robot.Kernel.UserLoginData.Uin, func(stop *bool, value *model.MaterialgreetRobotdbValue) {

		if result.Remark == "" {

			value.Warn++

			warn_existed = true

			if ctrler.self.ChangeProxy {
				if robot_proxy := self.Proxys.Existed(thread.Cache.Robot.ProxyId); robot_proxy != nil {
					thread.Cache.ProxyBlackList = append(thread.Cache.ProxyBlackList, robot_proxy.Config.Province)
				}
			}

			if ctrler.self.WarnLimit > 0 && value.Warn >= ctrler.self.WarnLimit {
				value.Status = model.MATERIALGREET_ROBOTDB_SATAUS_WARN
				warn_stop = true
			}

			ctrler.object.SetRobotLog(thread.RobotUid, fmt.Sprintf(`[%s] 机器人：%d，警告！好友申请被列入过滤通知！次数：%d。`, humanize.Comma(usetime), thread.RobotUid, value.Warn))

		} else {
			value.Succ++
		}

		ctrler.self.Cache.GreetUpdate++

		*stop = true
	})

	thread.Cache.Update++

	if warn_stop {
		thread.Clean()
		return
	}

	if !warn_existed {
		thread.Index++
	}

	if thread.Index < len(ctrler.greet_rule.List) {
		thread.SetTimer(ctrler.greet_rule)
	}

	ini.Friendb1.Foreach(plugin.FriendbFilter{Wlock: true, Blank: true}, func(stop *bool, value *plugin.FriendbValue) {

		value.Status = plugin.FRIENDB_STATUS_USED
		value.Uid = thread.VisitorUid
		value.RobotUid = thread.Cache.Robot.Kernel.UserLoginData.Uin
		value.CreateTime = time.Now().Unix()
		value.UserMark = ctrler.user.Mark

		switch {
		case ctrler.materialdb != nil:
			value.FromMark = ctrler.materialdb.Mark
		case ctrler.realinfodb != nil:
			value.FromMark = ctrler.realinfodb.Mark
		}

		value.TaskMark = ctrler.object.Mark

		*stop = true
	})

	thread.Step = model.MATERIALGREET_STEP_BEFORE

	return
}
