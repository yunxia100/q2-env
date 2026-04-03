package model

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"time"
	"ymlink-q2/define"
	"ymlink-q2/plugin"
	"ymlink-q2/utils"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ROBOT_MESSAGE_CHANNEL = struct {
	PERSION string
	GROUP   string
}{PERSION: "persion", GROUP: "group"}

type RobotMessageFilter struct {
	Timestart int64 `form:"timestart" bson:"timestart" json:"timestart"`
	Timestop  int64 `form:"timestop" bson:"timestop" json:"timestop"`
	Size      int64 `form:"size" bson:"size" json:"size"`
	Offset    int64 `form:"offset" bson:"offset" json:"offset"`
	Desc      bool  `form:"desc" bson:"desc" json:"desc"`

	Channel string `form:"channel" bson:"channel" json:"channel"`
	Type    string `form:"type" bson:"type" json:"type"`
	From    int    `form:"from" bson:"from" json:"from"` // 查群：to，查对话：from+to，查发言：from
	To      int    `form:"to" bson:"to" json:"to"`

	Count *int64 `form:"count" bson:"count" json:"count"`
}

// channel.from.to.values
type RobotMessageHistoryStorage []*RobotMessageValue

type RobotMessageValue struct {
	Channel string `form:"channel" bson:"channel" json:"channel"`
	Type    string `form:"type" bson:"type" json:"type"`
	From    int    `form:"from" bson:"from" json:"from"`
	To      int    `form:"to" bson:"to" json:"to"`
	Time    int64  `form:"time" bson:"time" json:"time"`
	Data    string `form:"data" bson:"data" json:"data"`
}

type Msg struct {
	SubType       int64  `json:"sub_type"`
	MsgTitle      string `json:"msg_title"`
	MsgDescribe   string `json:"msg_describe"`
	MsgAdditional string `json:"msg_additional"`
	MsgSource     string `json:"msg_source"`
	MsgDecided    string `json:"msg_decided"`
	SrcId         int64  `json:"src_id"`
	SubSrcId      int64  `json:"sub_src_id"`
	Relation      int64  `json:"relation"`
	ReqUinFaceid  int64  `json:"req_uin_faceid"`
	ReqUinNick    string `json:"req_uin_nick"`
	MsgDetail     string `json:"msg_detail"`
	ReqUinGender  int64  `json:"req_uin_gender"`
	ReqUinAge     int64  `json:"req_uin_age"`
}

type FriendNotice struct {
	MsgType int64 `json:"msg_type"`
	MsgSeq  int64 `json:"msg_seq"`
	MsgTime int64 `json:"msg_time"`
	ReqUin  int64 `json:"req_uin"`
	Msg     *Msg  `json:"msg"`
}

type RobotFriendNoticesResponse struct {
	LatestFriendSeq    int64           `json:"latest_friend_seq"`
	LatestGroupSeq     int64           `json:"latest_group_seq"`
	FollowingFriendSeq int64           `json:"following_friend_seq"`
	FriendNotices      []*FriendNotice `json:"friend_notices"`
	MsgDisplay         string          `json:"msg_display"`
	Over               int64           `json:"over"`
}

type RobotFriendPassRequest struct {
	RobotId  string `json:"robot_id"`
	ReqUin   int64  `json:"req_uin"`
	SrcId    int64  `json:"src_id"`
	SubSrcId int64  `json:"sub_src_id"`
}

func (storage *RobotMessageHistoryStorage) AddPoint(values []*RobotMessageValue) {

	if *storage == nil {
		*storage = []*RobotMessageValue{}
	}

	for _, value := range values {

		for _, item := range *storage {
			if item.Time == value.Time && item.From == value.From && item.To == value.To {
				goto NEXT
			}
		}

		if result, err := utils.GzipCompress(&value.Data); err != nil {
			value.Data = err.Error()
		} else {
			value.Data = base64.StdEncoding.EncodeToString(result)
		}

		*storage = append(*storage, value)

	NEXT:
	}
}

func (storage *RobotMessageHistoryStorage) Null() bool {

	return len(*storage) == 0
}

func (storage *RobotMessageHistoryStorage) Move() *RobotMessageHistoryStorage {

	_storage := *storage

	*storage = RobotMessageHistoryStorage{}

	return &_storage
}

func (storage *RobotMessageHistoryStorage) Write(influx *plugin.Influx, user_id primitive.ObjectID) (err error) {

	defer utils.ErrorRecover("RobotMessageHistoryStorage")

	batch := plugin.NewInfluxBatch(influx, define.INFLUX_BUCKET_HISTORY)

	for _, value := range *storage {

		tags := map[string]string{
			"channel": value.Channel,
			"type":    value.Type,
			"from":    strconv.Itoa(value.From),
			"to":      strconv.Itoa(value.To),
		}

		fields := map[string]interface{}{
			"value": value.Data,
		}

		batch.AddPoint(user_id.Hex(), tags, fields, value.Time)
	}

	return batch.Write()
}

func (storage *RobotMessageHistoryStorage) Read(influx *plugin.Influx, user_id primitive.ObjectID, filter *RobotMessageFilter) (count int64, err error) {

	var (
		cmd string

		cmd_fromto  = "true"
		cmd_channel = "true"
		cmd_type    = "true"

		query_results []map[string]interface{}
	)

	*storage = RobotMessageHistoryStorage{}

	// [PATCH] InfluxDB v2 不支持超大的 stop 时间戳（如年3000），会导致查询返回空
	if filter.Timestop == 0 {
		filter.Timestop = (time.Now().Unix() + 86400) * 1000 // now + 1 day
	}

	// [PATCH] InfluxDB v2 不支持 range(start: 0)，设置合理的默认起始时间（1年前）
	if filter.Timestart == 0 {
		filter.Timestart = (time.Now().Unix() - 365*24*3600) * 1000
	}

	if filter.Timestart/1000 >= filter.Timestop/1000 {
		filter.Timestart = filter.Timestop - 1000
	}

	if filter.Channel != "" {
		cmd_channel = fmt.Sprintf(`r.channel == "%s"`, filter.Channel)
	}

	if filter.Type != "" {
		cmd_type = fmt.Sprintf(`r.type == "%s"`, filter.Type)
	}

	switch {
	case filter.From != 0 && filter.To != 0:
		cmd_fromto = fmt.Sprintf(`( (r.from == "%d" and r.to == "%d") or (r.from == "%d" and r.to == "%d") )`, filter.From, filter.To, filter.To, filter.From)
	case filter.From != 0 && filter.To == 0:
		cmd_fromto = fmt.Sprintf(`r.from == "%d"`, filter.From)
	case filter.From == 0 && filter.To != 0:
		cmd_fromto = fmt.Sprintf(`r.to == "%d"`, filter.To)
	}

	if filter.Count != nil {

		cmd = fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %d, stop: %d)
		|> filter(fn: (r) => r._measurement == "%s" and %s and %s and %s) 
		|> group(columns: [])
		|> count()
		`,
			define.INFLUX_BUCKET_HISTORY,
			filter.Timestart/1000, filter.Timestop/1000,
			user_id.Hex(), cmd_channel, cmd_type, cmd_fromto,
		)

		if query_results, err = influx.Query(cmd); err != nil {
			return
		}

		if len(query_results) > 0 {
			count, _ = query_results[0]["_value"].(int64)
		}
	}

	cmd = fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %d, stop: %d)
		|> filter(fn: (r) => r._measurement == "%s" and %s and %s and %s) 
		|> group(columns: [])
		|> sort(columns: ["_time"], desc: %t)
		|> drop(columns: ["_start", "_stop", "_measurement", "_field"])
		|> limit(n: %d, offset: %d)
		|> yield()
		`,
		define.INFLUX_BUCKET_HISTORY,
		filter.Timestart/1000, filter.Timestop/1000,
		user_id.Hex(), cmd_channel, cmd_type, cmd_fromto,
		filter.Desc,
		filter.Size, (filter.Offset-1)*filter.Size,
	)

	logrus.WithFields(logrus.Fields{"cmd": cmd, "timestart": filter.Timestart, "timestop": filter.Timestop}).Info("[PATCH-DEBUG] InfluxDB query")

	if query_results, err = influx.Query(cmd); err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("[PATCH-DEBUG] InfluxDB query error")
		return
	}

	logrus.WithFields(logrus.Fields{"results_count": len(query_results)}).Info("[PATCH-DEBUG] InfluxDB query results")

	var (
		value *RobotMessageValue
		ok    bool
	)

	for _, query_result := range query_results {

		value = &RobotMessageValue{}

		if value.Channel, ok = query_result["channel"].(string); !ok {
			continue
		}

		if value.Type, ok = query_result["type"].(string); !ok {
			continue
		}

		if from, ok := query_result["from"].(string); !ok {
			continue
		} else if value.From, _ = strconv.Atoi(from); value.From == 0 {
			continue
		}

		if to, ok := query_result["to"].(string); !ok {
			continue
		} else if value.To, _ = strconv.Atoi(to); value.To == 0 {
			continue
		}

		if value.Time, ok = query_result["time"].(int64); !ok {
			continue
		}

		if value.Data, ok = query_result["_value"].(string); !ok {
			continue
		} else if base64_str, err := base64.StdEncoding.DecodeString(value.Data); err != nil {
			value.Data = err.Error()
		} else if value.Data, err = utils.GzipDecompress(base64_str); err != nil {
			value.Data = err.Error()
		}

		*storage = append(*storage, value)
	}

	return
}
