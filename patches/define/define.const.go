package define

import "strings"

var (
	TRUE  = true
	FALSE = false
)

var (
	SYSTEM_MODE_TEST    = "test"
	SYSTEM_MODE_DEBUG   = "debug"
	SYSTEM_MODE_RELEASE = "release"

	SystemInfo = struct {
		Cpu             float32 `form:"cpu" bson:"cpu" json:"cpu"`
		MemUsed         int     `form:"mem_used" bson:"mem_used" json:"mem_used"`
		MemTotal        int     `form:"mem_total" bson:"mem_total" json:"mem_total"`
		GoroutinesTotal int     `form:"goroutines_total" bson:"goroutines_total" json:"goroutines_total"`
	}{}
)

var (
	USER_IGNORE_PATHS = []string{

		"/api/user/signin",

		"/api/custservice/api/*",

		"/api/debug/*",

		"/file/task/*",
		"/file/material/*",
		"/file/message/*",

		"/api/mobile/*",

		"/api/webrobot/create",

		"/api/robot/create",
		"/api/robot/import_pack",
		"/api/robot/register",
		"/api/robot/login",
		"/api/robot/register_clear",
		"/api/robot/login_clear",
		"/api/robot/delete_by_batch",
		"/api/robot/proxy_reset",
		"/api/robot/login_assistant_qrcode",
		"/api/robot/set_password",

		"/api/robot/batch/info",
		"/api/robot/batch/status",
		"/api/robot/batch/account_submit",

		"/api/robot/message/*",

		"/api/proxy/get_region",
	}

	CUSTSERVICE_IGNORE_PATHS = []string{

		"/api/custservice/api/signin",
	}
)

var (
	INFLUX_BUCKET_REALTIME = "realtime"
	INFLUX_BUCKET_HISTORY  = "history"
	INFLUX_BUCKETS         = []string{
		INFLUX_BUCKET_REALTIME, INFLUX_BUCKET_HISTORY,
	}
)

var (
	MONGO_COLLECTION_USER                   = "user"
	MONGO_COLLECTION_CUSTSERVICE            = "custservice"
	MONGO_COLLECTION_PROXY                  = "proxy"
	MONGO_COLLECTION_PROXY_EXTRACTOR        = "proxy_extractor"
	MONGO_COLLECTION_WEBROBOT               = "webrobot"
	MONGO_COLLECTION_ROBOT                  = "robot"
	MONGO_COLLECTION_ROBOT_LABEL            = "robot_label"
	MONGO_COLLECTION_ROBOT_BATCH            = "robot_batch"
	MONGO_COLLECTION_ROBOT_FRIEND           = "robot_friend"
	MONGO_COLLECTION_ROBOT_MATERIAL         = "robot_material"
	MONGO_COLLECTION_TASK                   = "task"
	MONGO_COLLECTION_TASK_GREET_RULE        = "task_greet_rule"
	MONGO_COLLECTION_TASK_GREET_WORD        = "task_greet_word"
	MONGO_COLLECTION_TASK_QZONE_REMARK_RULE = "task_qzone_remark_rule"
	MONGO_COLLECTION_TASK_USEDB             = "task_usedb"
	MONGO_COLLECTION_TASK_QZONEDB           = "task_qzonedb"
	MONGO_COLLECTION_TASK_MATERIALDB        = "task_materialdb"
	MONGO_COLLECTION_TASK_REALINFODB        = "task_realinfodb"
	MONGO_COLLECTIONS                       = []string{
		MONGO_COLLECTION_USER, MONGO_COLLECTION_CUSTSERVICE, MONGO_COLLECTION_PROXY, MONGO_COLLECTION_PROXY_EXTRACTOR, MONGO_COLLECTION_ROBOT,
		MONGO_COLLECTION_ROBOT_LABEL, MONGO_COLLECTION_ROBOT_BATCH, MONGO_COLLECTION_TASK, MONGO_COLLECTION_TASK_GREET_RULE, MONGO_COLLECTION_TASK_GREET_WORD,
		MONGO_COLLECTION_TASK_QZONE_REMARK_RULE,
		MONGO_COLLECTION_TASK_USEDB, MONGO_COLLECTION_TASK_QZONEDB, MONGO_COLLECTION_TASK_MATERIALDB, MONGO_COLLECTION_TASK_REALINFODB, MONGO_COLLECTION_ROBOT_MATERIAL, // # 待完成，暂通过本地文件管理实现
		MONGO_COLLECTION_ROBOT_FRIEND,
	}
	// group_task
	MONGO_COLLECTION_GROUP_TASK = "group_task"
	// group_material
	MONGO_COLLECTION_GROUP_MATERIAL = "group_material"
	// risk_management
	MONGO_COLLECTION_RISK_MANAGEMENT = "risk_management"
)

var (
	CTX_USER_INFO        = "CTX_USER_INFO"
	CTX_CUSTSERVICE_INFO = "CTX_CUSTSERVICE_INFO"
)

var (
	PROXY_EXTRACTOR_IPZAN = "ipzan"
	PROXY_EXTRACTORS      = []string{
		PROXY_EXTRACTOR_IPZAN,
	}

	PROXY_EXTRACTOR_MUTEX_TIMEOUT = int64(200)
)

var (
	REGOIN_LEVEL_COUNTRY  = 1
	REGOIN_LEVEL_PROVINCE = 2
	REGOIN_LEVEL_CITY     = 3
	REGOIN_LEVELs         = []int{
		REGOIN_LEVEL_COUNTRY, REGOIN_LEVEL_PROVINCE, REGOIN_LEVEL_CITY,
	}
)

const (
	ROBOT_IMPORT_MODE_ANDROID_PACK = "android_pack"
	ROBOT_IMPORT_MODE_IOS_PACK     = "ios_pack"
	ROBOT_IMPORT_MODE_INI_PACK     = "ini_pack"
)

const (
	ROBOT_SOFTWARE_ANDROID = "androidQQ"
	ROBOT_SOFTWARE_IOS     = "iOSQQ"
	ROBOT_SOFTWARE_MACOS   = "macOSQQ"
	ROBOT_SOFTWARE_WIN     = "winQQ"
)

const (
	ROBOT_MUTEX_TIMEOUT = int64(1000)

	ROBOT_STATUS_SYSTEM_SUCC  = 0  // 正常
	ROBOT_STATUS_SYSTEM_FAIL  = -1 // 请求失败
	ROBOT_STATUS_SYSTEM_ERROR = -2 // 系统错误

	ROBOT_STATUS_PROXY_SUCC     = 0 // 正常
	ROBOT_STATUS_PROXY_DISABLED = 1 // 失效
	ROBOT_STATUS_PROXY_CHANGING = 2 // 更换中

	// 注册相关

	ROBOT_REGISTER_MODE_PHONE_CHECK    = "7"  // 检查手机号
	ROBOT_REGISTER_MODE_SMS_SLIDER     = "10" // 滑块验证
	ROBOT_REGISTER_MODE_SMS_PUSH       = "3"  // 已发1
	ROBOT_REGISTER_MODE_SMS_CODE       = "5"  // 提交短信验证码
	ROBOT_REGISTER_MODE_ACCOUNT_SUBMIT = "6"  // 提交昵称和密码

	ROBOT_REGISTER_STATUS_FAIL         = -1  // 失败
	ROBOT_REGISTER_STATUS_SUCC         = 0   // 成功
	ROBOT_REGISTER_STATUS_SMS          = 2   // 请发送短信
	ROBOT_REGISTER_STATUS_DANGER       = 7   // 手机号码存在风险
	ROBOT_REGISTER_STATUS_FREQUENT     = 201 // 操作过于频繁
	ROBOT_REGISTER_STATUS_CHECK_FRIEND = 51  // 需要好友辅助认证
	ROBOT_REGISTER_STATUS_SLIDER       = 58  // 需要滑块认证
	ROBOT_REGISTER_STATUS_CHECK_REAL   = 59  // 需要实名认证
	ROBOT_REGISTER_STATUS_BUSY         = 115 // 操作过于频繁
	// 不记得了 手机格式错误

	// 登录相关

	ROBOT_LOGIN_MODE_PASSWORD = "9" // 密码登录
	ROBOT_LOGIN_MODE_SLIDER   = "2" // 校验滑块
	ROBOT_LOGIN_MODE_SMS_GET  = "8" // 验证码登录
	ROBOT_LOGIN_MODE_SMS_CODE = "7" // 校验验证码

	ROBOT_LOGIN_STATUS_SUCC          = 0   // 登录成功
	ROBOT_LOGIN_STATUS_WRONG         = 1   // 账号或密码错误
	ROBOT_LOGIN_STATUS_SLIDER        = 2   // 需要滑块
	ROBOT_LOGIN_STATUS_CONNECTING    = 9   // 服务连接中
	ROBOT_LOGIN_STATUS_CONNECTING2   = 10  // 服务连接中，请稍后重试
	ROBOT_LOGIN_STATUS_DISABLED      = 15  // 身份验证失效
	ROBOT_LOGIN_STATUS_LOSE          = 16  // 登录失效
	ROBOT_LOGIN_STATUS_TIMEOUT       = 20  // 登录态过期
	ROBOT_LOGIN_STATUS_USERNAME_NONE = 32  // 账号不存在
	ROBOT_LOGIN_STATUS_LOCK          = 40  // 账号冻结
	ROBOT_LOGIN_STATUS_WARNING       = 45  // 禁止登录
	ROBOT_LOGIN_STATUS_SUP           = 239 // 需要辅助验证
	ROBOT_LOGIN_STATUS_SAFE          = 237 // 安全提醒
	ROBOT_LOGIN_STATUS_CON_TIMEOUT   = 154 // 服务器连接超时
	ROBOT_LOGIN_STATUS_FAIL          = 155 // 登录失败
	ROBOT_LOGIN_STATUS_SMS           = 160 // 需要短信验证
	ROBOT_LOGIN_STATUS_SMS_FAIL      = 163 // 短信验证失败
	ROBOT_LOGIN_STATUS_TIM           = 247 // 请在TIM设置手机号
)

const (
	ROBOT_MATERIAL_MODE_TEXT      = "text"
	ROBOT_MATERIAL_MODE_IMAGE     = "image"
	ROBOT_MATERIAL_MODE_AUDIO     = "audio"
	ROBOT_MATERIAL_MODE_VIDEO     = "video"
	ROBOT_MATERIAL_MODE_GROUPLINK = "grouplink"
)

const (
	ROBOT_MESSAGE_TYPE_ERROR           = "error"
	ROBOT_MESSAGE_TYPE_UNKNOW          = "unknow"
	ROBOT_MESSAGE_TYPE_NOP             = "nop"
	ROBOT_MESSAGE_TYPE_IMAGE_NOTONLINE = "image_notonline"
	ROBOT_MESSAGE_TYPE_TEXT            = "text"
	ROBOT_MESSAGE_TYPE_LINK            = "link"
	ROBOT_MESSAGE_TYPE_IMAGE           = "image"
	ROBOT_MESSAGE_TYPE_IMAGES          = "images"
	ROBOT_MESSAGE_TYPE_AUDIO           = "audio"
	ROBOT_MESSAGE_TYPE_VIDEO           = "video"
	ROBOT_MESSAGE_TYPE_FILE            = "file"
	ROBOT_MESSAGE_TYPE_FILE_NOTONLINE  = "file_notonline"
)

const (
	INTERVAL_RENEW_SECRET_KEY int64 = 60 * 60 * 24 * (10 - 3)
	INTERVAL_RENEW_ONLINE     int64 = 60 * (10 - 3)
	INTERVAL_FRIENDS          int64 = 60 * 60 * 2
	INTERVAL_INFO             int64 = 60 * 60 * 24 * 1
	INTERVAL_PROFILE          int64 = 60 * 60 * 24 * 30
	INTERVAL_APPLIST          int64 = 60 * 60 * 24 * 1
	INTERVAL_DEVICELIST       int64 = 60 * 60 * 24 * 1
	INTERVAL_QZONE_PERMISSION int64 = 60 * 60 * 24 * 1
	INTERVAL_QZONE_MAIN       int64 = 60 * 60 * 1
	INTERVAL_DAILY            int64 = 60 * 60 * 24 * 1
)

var (
	TASK_MODE_QZONEGREET     = "qzonegreet"
	TASK_MODE_MATERIALGREET  = "materialgreet"
	TASK_MODE_GROUPCHATPULL  = "groupchatpull"
	TASK_MODE_QZONEREMARK    = "qzoneremark"
	TASK_MODE_QZONEVISITOR   = "qzonevisitor"
	TASK_MODE_QSHOWRECOMMEND = "qshowrecommend"
	TASK_MODE_GROUPMENBER    = "groupmenber"
	TASK_MODE                = []string{
		TASK_MODE_QZONEGREET, TASK_MODE_MATERIALGREET, TASK_MODE_GROUPCHATPULL, TASK_MODE_QZONEREMARK, TASK_MODE_QZONEVISITOR, TASK_MODE_QSHOWRECOMMEND, TASK_MODE_GROUPMENBER,
	}
)

const (
	TASK_API_INTERVAL_MIN = 5
	TASK_API_INTERVAL_MAX = 10

	TASK_FILE_PATH           = "./file/task/%s"
	TASK_LOG_ROBOT_FILE_PATH = "./file/task/%s/log/robot/%d.txt"
	TASK_LOG_GROUP_FILE_PATH = "./file/task/%s/log/group/%d.txt"
)

const (
	TASK_STATUS_BLANK                  = 0  // 加载中
	TASK_STATUS_NULL                   = 1  // 表单不存在
	TASK_STATUS_CONFIG_NULL            = 2  // 配置丢失
	TASK_STATUS_RUN                    = 3  // 运行中
	TASK_STATUS_STOP                   = 4  // 已暂停
	TASK_STATUS_WAIT                   = 5  // 等待启动
	TASK_STATUS_PROXY_EXTRACTOR_NULL   = 6  // 代理生成器不存在
	TASK_STATUS_WORD_NULL              = 7  // 招呼语不存在
	TASK_STATUS_RULE_NULL              = 8  // 招呼规则不存在
	TASK_STATUS_USEDB_NULL             = 9  // 历史库不存在
	TASK_STATUS_QZONEDB_NULL           = 11 // 母料库不存在
	TASK_STATUS_MATERIALDB_NULL        = 12 // 子料库不存在
	TASK_STATUS_USER_NULL              = 13 // 用户不存在
	TASK_STATUS_ROBOTDB_NULL           = 14 // 任务缓存丢失
	TASK_STATUS_QZONE_REMARK_RULE_NULL = 15 // 空间留痕规则不存在
	TASK_STATUS_FINISH                 = 16 // 已完成
	TASK_STATUS_DIVISION               = 17 // 分裂失败
)

var (
	TASK_THREAD_SAFE_ERROR = []string{
		"已闲置",
		"子料库已用完",
		"母料库已用完",
		"代理提取异常",
	}

	TaskThreadSafe = func(err string) bool {
		for idx := range TASK_THREAD_SAFE_ERROR {
			if strings.Contains(err, TASK_THREAD_SAFE_ERROR[idx]) {
				return true
			}
		}
		return false
	}
)

const (
	ROBOT_STATUS_NONE = 0
	ROBOT_STATUS_NULL = 1
)

const (
	ROBOT_LOGIN_SLIDER_FILE_PATH = "file/login/%s/slider/%d.html"
	ROBOT_LOGIN_SUP_FILE_PATH    = "file/login/%s/sup/%d.html"
)

const (
	PROXY_EXTRACTOR_STATUS_NONE = 0
	PROXY_EXTRACTOR_STATUS_NULL = 1
)

var (
	PROXY_PROTOCOL_SOCKS5 = "socks5"
	PROXY_PROTOCOL_HTTP   = "http"
	PROXY_PROTOCOL_HTTPS  = "https"
)

const (
	GREET_RULE_CHANNELS_10028_1 = "10028-1" // 空间访客
	GREET_RULE_CHANNELS_2081_1  = "2081-1"  // 空间
	GREET_RULE_CHANNELS_2020_4  = "2020-4"  // 搜索
	GREET_RULE_CHANNELS_2050_1  = "2050-1"  // 小世界
	GREET_RULE_CHANNELS_2001_0  = "2001-0"  // 黑名单
	GREET_RULE_CHANNELS_2011_0  = "2011-0"  // 我看过谁
)

var (
	FILTER_SEX_NAME = map[int]interface{}{
		0: "男",
		1: "女",
		2: "未知",
	}
	FILTER_AUTH_TYPE_NAME = map[int]interface{}{
		0:   "允许任何人添加",
		1:   "需要验证信息",
		3:   "需要正确回答问题",
		4:   "需要回答问题并由对方确认",
		101: "原本就是好友，无需申请",
		// nil 不允许任何人添加
	}
)

var (
	ROBOT_BATCH_MODE_ACCOUNT = "account"
	ROBOT_BATCH_MODE_IMAGE   = "image"
	ROBOT_BATCH_MODE_COOKIE  = "cookie"
	ROBOT_BATCH_MODE_PACK    = "pack"
)

const (
	USEDB_PATH        = "./file/usedb"
	QZONEDB_PATH      = "./file/qzonedb"
	MATERIALDB_PATH   = "./file/materialdb"
	REALINFODB_PATH   = "./file/realinfodb"
	MATERIAL_PATH     = "./file/material"
	ANDROID_PACK_PATH = "./file/android_pack"
	IOS_PACK_PATH     = "./file/ios_pack"
	INI_PACK_PATH     = "./file/ini_pack"
	MESSAGE_PATH      = "./file/message"
)
