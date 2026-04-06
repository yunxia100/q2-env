package plugin

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/bwmarrin/snowflake"
	"github.com/tjfoc/gmsm/sm3"
)

const (
	apiTextUrl      = "http://as.dun.163.com/v5/text/check"
	apiImageUrl     = "http://as.dun.163.com/v5/image/base64Check"
	version         = "v5.2"
	secretId        = "1525c4cfeb818ff7e4250d37a6d99cbe" // 产品密钥ID - ymlink客服系统(YD00791627652590)
	secretKey       = "dfa7e6ab9fdc9ac5daf7c4f08a47bf1d" // 产品私有密钥
	businessTextId  = "15ae4cf1c93bf16ec86ce430c5dc4be3" // 业务ID - 普通文本
	businessImageId = "18407282591c76ecff1b0400302a8eae" // 业务ID - 分类图片(普通图片)
)

var node *snowflake.Node

func init() {
	var err error
	if node, err = snowflake.NewNode(1); err != nil {
		panic(err)
	}
}

type ResultResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TaskId     string `json:"taskId"`
		Suggestion int64  `json:"suggestion"`
		Labels     []any  `json:"labels"`
		HitKeyword string `json:"hitKeyword"` // [PATCH] 命中的关键词
	}
}

// 请求易盾接口
func check(params url.Values, apiUrl string) *simplejson.Json {
	params["secretId"] = []string{secretId}
	if apiUrl == apiTextUrl {
		params["businessId"] = []string{businessTextId}
	} else if apiUrl == apiImageUrl {
		params["businessId"] = []string{businessImageId}
	}
	params["version"] = []string{version}
	params["timestamp"] = []string{strconv.FormatInt(time.Now().UnixNano()/1000000, 10)}
	params["nonce"] = []string{strconv.FormatInt(rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(10000000000), 10)}
	// params["signatureMethod"] = []string{"SM3"} // 签名方法支持国密SM3，默认MD5
	params["signature"] = []string{genSignature(params)}

	resp, err := http.Post(apiUrl, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()))

	if err != nil {
		fmt.Println("调用API接口失败:", err)
		return nil
	}

	defer resp.Body.Close()

	contents, _ := io.ReadAll(resp.Body)
	result, _ := simplejson.NewJson(contents)
	return result
}

// 生成签名信息
func genSignature(params url.Values) string {
	var paramStr string
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		paramStr += key + params[key][0]
	}
	paramStr += secretKey
	if params["signatureMethod"] != nil && params["signatureMethod"][0] == "SM3" {
		sm3Reader := sm3.New()
		sm3Reader.Write([]byte(paramStr))
		return hex.EncodeToString(sm3Reader.Sum(nil))
	} else {
		md5Reader := md5.New()
		md5Reader.Write([]byte(paramStr))
		return hex.EncodeToString(md5Reader.Sum(nil))
	}
}

func CheckText(content string) *ResultResp {
	dataId := uuid.New().String()
	params := url.Values{
		"dataId":  []string{dataId},
		"content": []string{content},
	}

	ret := check(params, apiTextUrl)

	code, _ := ret.Get("code").Int()
	message, _ := ret.Get("msg").String()
	resp := &ResultResp{Code: code, Msg: message}

	if code == 200 {
		result := ret.Get("result")
		antispam := result.Get("antispam")
		if antispam != nil {
			taskId, _ := antispam.Get("taskId").String()
			suggestion, _ := antispam.Get("suggestion").Int64()
			labels, _ := antispam.Get("labels").Array()

			resp.Data.TaskId = taskId
			resp.Data.Suggestion = suggestion
			resp.Data.Labels = labels

			// [PATCH] 提取命中的关键词
			if suggestion > 0 {
				if kw := antispam.GetPath("labels").GetIndex(0).GetPath("subLabels").GetIndex(0).GetPath("details", "keywords").GetIndex(0).Get("word"); kw != nil {
					resp.Data.HitKeyword, _ = kw.String()
				}
			}
		}
	}

	return resp
}

func CheckImage(content, account string) *ResultResp {
	var images []map[string]string
	imageBase64 := map[string]string{
		"name": fmt.Sprintf("{\"imageId\":%d,\"contentId\":%d}", node.Generate().Int64(), node.Generate().Int64()),
		"type": "2",
		"data": base64.StdEncoding.EncodeToString([]byte(content)),
	}

	images = append(images, imageBase64)
	jsonString, _ := json.Marshal(images)

	params := url.Values{
		"images":  []string{string(jsonString)},
		"account": []string{account},
		"ip":      []string{"154.91.231.29"},
	}

	ret := check(params, apiImageUrl)

	code, _ := ret.Get("code").Int()
	message, _ := ret.Get("msg").String()
	resp := &ResultResp{Code: code, Msg: message}

	if code == 200 {
		results, _ := ret.Get("result").Array()
		if len(results) != 1 {
			resp.Code = 400
			resp.Msg = "系统错误"
			return resp
		}

		if resultMap, ok := results[0].(map[string]interface{}); !ok {
			resp.Code = 400
			resp.Msg = "系统错误"
			return resp
		} else {
			if resultMap["antispam"] == nil {
				resp.Code = 400
				resp.Msg = "系统错误"
				return resp
			}

			antispam, _ := resultMap["antispam"].(map[string]interface{})
			taskId := antispam["taskId"].(string)
			status, _ := antispam["status"].(json.Number).Int64()
			// 检测状态：2 检测成功、3 检测失败
			if status == 2 {
				resp.Code = 0
				resp.Msg = "Success"

				resp.Data.TaskId = taskId
				resp.Data.Suggestion, _ = antispam["suggestion"].(json.Number).Int64()
				resp.Data.Labels = antispam["labels"].([]interface{})
			} else {
				resp.Code = 400
				// 检测失败原因，当status为3（检测失败）时返回：610 图片下载失败、620 图片格式错误、630 其他
				failureReason, _ := antispam["failureReason"].(json.Number).Int64()
				switch failureReason {
				case 610:
					resp.Msg = "图片下载失败"
				case 620:
					resp.Msg = "图片格式错误"
				default:
					resp.Msg = "未知错误"
				}
			}
		}
	}

	return resp
}
