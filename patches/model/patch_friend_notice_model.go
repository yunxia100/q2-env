package model

import (
	"encoding/json"
	"fmt"
	"ymlink-q2/plugin"
)

// ========== 好友通知 (drive层原始返回) ==========

type PatchFriendMsg struct {
	Version int   `json:"version"`
	MsgType int64 `json:"msgType"`
	MsgSeq  int64 `json:"msgSeq"`
	MsgTime int64 `json:"msgTime"`
	ReqUin  int64 `json:"reqUin"`
	Msg     *struct {
		SubType       int64  `json:"subType"`
		MsgTitle      string `json:"msgTitle"`
		MsgDescribe   string `json:"msgDescribe"`
		MsgAdditional string `json:"msgAdditional"`
		MsgSource     string `json:"msgSource"`
		MsgDecided    string `json:"msgDecided"`
		SrcId         int64  `json:"srcId"`
		SubSrcId      int64  `json:"subSrcId"`
		Relation      int64  `json:"relation"`
		ReqUinFaceid  int64  `json:"reqUinFaceid"`
		ReqUinNick    string `json:"reqUinNick"`
		MsgDetail     string `json:"msgDetail"`
		ReqUinGender  int64  `json:"reqUinGender"`
		ReqUinAge     int64  `json:"reqUinAge"`
	} `json:"msg"`
}

type PatchFriendNoticesResult struct {
	Head               any               `json:"head"`
	LatestFriendSeq    int64             `json:"latestFriendSeq"`
	LatestGroupSeq     int64             `json:"latestGroupSeq"`
	FollowingFriendSeq int64             `json:"followingFriendSeq"`
	FriendMsgs         []*PatchFriendMsg `json:"friendMsg"`
	MsgDisplay         string            `json:"msgDisplay"`
	Over               int64             `json:"over"`
}

func (robot *Robot) PatchFriendNotices() (*PatchFriendNoticesResult, error) {
	code, _, content, err := robot.Client().PostForm(
		"/device/ProfileService.Pb.ReqSystemMsgNew",
		map[string]string{
			"Accept": "application/json",
		},
		map[string]string{"objid": robot.Kernel.Objid},
		map[string]string{},
	)
	if err != nil {
		return nil, fmt.Errorf("获取好友通知列表请求错误: %v", err)
	}
	if code-code%plugin.REQUEST_SUCCESS != plugin.REQUEST_SUCCESS {
		return nil, fmt.Errorf("code: %d, content: %s", code, string(content))
	}
	result := new(PatchFriendNoticesResult)
	if err = json.Unmarshal(content, result); err != nil {
		return nil, fmt.Errorf("获取好友响应数据格式异常: %s", string(content))
	}
	return result, nil
}

// ========== 通过好友请求 ==========

type PatchFriendPassResult struct {
	Head struct {
		Result  int    `json:"result"`
		MsgFail string `json:"msgFail"`
	} `json:"head"`
	MsgDetail string `json:"msgDetail"`
}

func (robot *Robot) PatchFriendPass(reqUin, srcId, subSrcId string) (*PatchFriendPassResult, error) {
	code, _, content, err := robot.Client().PostForm(
		"/device/ProfileService.Pb.ReqSystemMsgAction.Friend",
		map[string]string{
			"Accept": "application/json",
		},
		map[string]string{"objid": robot.Kernel.Objid},
		map[string]string{"reqUin": reqUin, "srcId": srcId, "subSrcId": subSrcId},
	)
	if err != nil {
		return nil, fmt.Errorf("通过好友请求错误: %v", err)
	}
	if code-code%plugin.REQUEST_SUCCESS != plugin.REQUEST_SUCCESS {
		return nil, fmt.Errorf("code: %d, content: %s", code, string(content))
	}
	result := new(PatchFriendPassResult)
	if err = json.Unmarshal(content, result); err != nil {
		return nil, fmt.Errorf("通过好友响应数据格式异常: %s", string(content))
	}
	return result, nil
}

// ========== 兼容别名：让 ctrler.custservice.api.go 能编译 ==========

// RobotFriendNoticesResult is an alias for PatchFriendNoticesResult
type RobotFriendNoticesResult = PatchFriendNoticesResult

// RobotFriendPassResult is an alias for PatchFriendPassResult
type RobotFriendPassResult = PatchFriendPassResult

// FriendNotices wraps PatchFriendNotices with the expected method name
func (robot *Robot) FriendNotices() (*RobotFriendNoticesResult, error) {
	return robot.PatchFriendNotices()
}

// FriendPass wraps PatchFriendPass with the expected method name
func (robot *Robot) FriendPass(reqUin, srcId, subSrcId string) (*RobotFriendPassResult, error) {
	return robot.PatchFriendPass(reqUin, srcId, subSrcId)
}

// ========== 前端所需的响应类型 ==========

type PatchFriendNoticeItem struct {
	MsgType       int    `json:"msg_type"`
	MsgSeq        int64  `json:"msg_seq"`
	MsgTime       int64  `json:"msg_time"`
	ReqUin        int64  `json:"req_uin"`
	Nick          string `json:"nick"`
	Gender        int64  `json:"gender"`
	Age           int64  `json:"age"`
	SrcId         int64  `json:"src_id"`
	SubSrcId      int64  `json:"sub_src_id"`
	MsgTitle      string `json:"msg_title"`
	MsgAdditional string `json:"msg_additional"`
	MsgSource     string `json:"msg_source"`
	MsgDetail     string `json:"msg_detail"`
}

type PatchFriendPassRequest struct {
	RobotId  string `json:"robot_id" form:"robot_id"`
	ReqUin   int64  `json:"req_uin" form:"req_uin"`
	SrcId    int64  `json:"src_id" form:"src_id"`
	SubSrcId int64  `json:"sub_src_id" form:"sub_src_id"`
}
