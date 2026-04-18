package plugin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	REQUEST_SUCCESS            = 200 // 请求成功
	REQUEST_CREATED            = 201 // 创建成功
	REQUEST_BAD                = 400 // 请求参数有误或格式不正确
	REQUEST_UNAUTHORIZED       = 401 // 未通过用户验证
	REQUEST_FORBIDDEN          = 403 // 拒绝访问
	REQUEST_NOT_FOUND          = 404 // 资源不存在
	REQUEST_METHOD_NOT_ALLOWED = 405 // 方法不允许
	REQUEST_SERVER_ERROR       = 500 // 服务器异常
)

type HttpServerConfig struct {
	Url      string `form:"url" bson:"url" json:"url"`
	Mode     string `form:"mode" bson:"mode" json:"mode"`
	HtmlPath string `form:"html_path" bson:"html_path" json:"html_path"`
	Ssl      struct {
		Crt string `form:"crt" bson:"crt" json:"crt"`
		Key string `form:"key" bson:"key" json:"key"`
	} `form:"ssl" bson:"ssl" json:"ssl"`
}

type HttpServer struct {
	Router *gin.Engine
	Object *http.Server

	Config HttpServerConfig
}

type JSON map[string]interface{}

func NewHttpServer(config HttpServerConfig) (http_server *HttpServer) {

	http_server = &HttpServer{
		Config: config,
	}

	if http_server.Config.Mode == "" {
		http_server.Config.Mode = "release"
	}

	gin.SetMode(http_server.Config.Mode)

	http_server.Router = gin.New()

	// ============================================================
	// [PATCH] Gin 请求日志 + Panic 恢复中间件
	// ============================================================

	// 请求访问日志 → 写入 ./log/access.log + 控制台
	os.MkdirAll("./log", 0755)
	accessLog, _ := os.OpenFile("./log/access.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	var accessWriter io.Writer
	if accessLog != nil {
		accessWriter = io.MultiWriter(os.Stdout, accessLog)
	} else {
		accessWriter = os.Stdout
	}

	http_server.Router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[GIN] %s | %3d | %13v | %15s | %-7s %s\n",
				param.TimeStamp.Format(time.DateTime),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Path,
			)
		},
		Output: accessWriter,
	}))

	// Panic 恢复 → 写入 ./log/error.log + 控制台
	errorLog, _ := os.OpenFile("./log/error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	var errorWriter io.Writer
	if errorLog != nil {
		errorWriter = io.MultiWriter(os.Stderr, errorLog)
	} else {
		errorWriter = os.Stderr
	}

	http_server.Router.Use(gin.RecoveryWithWriter(errorWriter))

	// ============================================================
	// [END PATCH]
	// ============================================================

	http_server.Router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},                                                // 允许所有来源
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"}, // 允许的 HTTP 方法
		AllowHeaders:     []string{"*"},                                                // 允许所有请求头字段
		AllowCredentials: true,
	}))

	if http_server.Config.HtmlPath != "" {

		http_server.Router.Static("/monacoeditorwork", http_server.Config.HtmlPath+"/monacoeditorwork")
		http_server.Router.Static("/assets", http_server.Config.HtmlPath+"/assets")
		http_server.Router.Static("/images", http_server.Config.HtmlPath+"/images")
		http_server.Router.Static("/log.svg", http_server.Config.HtmlPath+"/log.svg")

		// [PATCH] 提供补丁 JS
		http_server.Router.GET("/patch.js", func(c *gin.Context) {
			c.Data(200, "application/javascript; charset=utf-8", []byte(patchJS))
		})

		// [PATCH] 返回 index.html 时自动注入 patch.js
		serveIndexWithPatch := func(c *gin.Context) {
			htmlBytes, err := os.ReadFile(http_server.Config.HtmlPath + "/index.html")
			if err != nil {
				c.Status(404)
				return
			}
			html := string(htmlBytes)
			inject := `<script src="/patch.js"></script>`
			if idx := strings.Index(html, "</body>"); idx != -1 {
				html = html[:idx] + inject + html[idx:]
			} else {
				html = strings.Replace(html, "</head>", inject+"</head>", 1)
			}
			c.Data(200, "text/html; charset=utf-8", []byte(html))
		}

		http_server.Router.GET("/", serveIndexWithPatch)

		http_server.Router.NoRoute(func(c *gin.Context) {
			serveIndexWithPatch(c)
		})

		// http_server.Router.Use(static.Serve("/", static.LocalFile(http_server.Config.HtmlPath+"/index.html", false)))
	}

	http_server.Object = &http.Server{
		Addr:    http_server.Config.Url,
		Handler: http_server.Router,
	}

	return
}

func (http_server *HttpServer) Running() (err error) {

	if http_server.Config.Ssl.Crt+http_server.Config.Ssl.Key == "" {
		goto NEXT
	}

	return http_server.Router.RunTLS(http_server.Config.Url, http_server.Config.Ssl.Crt, http_server.Config.Ssl.Key)

NEXT:
	return http_server.Router.Run(http_server.Config.Url)
}

func (http_server *HttpServer) Stopping() (err error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	cancel()

	if err = http_server.Object.Shutdown(ctx); err != nil {
		return
	}

	http_server.Object = nil

	return
}

func HttpSuccess(ctx *gin.Context, data interface{}) {

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"msg":  "succ",
		"code": REQUEST_SUCCESS,
		"data": data,
	})
}

func HttpHtml(ctx *gin.Context, html_str string) {

	ctx.Data(200, "text/html; charset=utf-8", []byte(html_str))
}

func HttpDefault(ctx *gin.Context, code int, msg interface{}, data interface{}) {

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"msg":  msg,
		"code": code,
		"data": data,
	})
}

func HttpBindObjectId(ctx *gin.Context, key string, object_id *primitive.ObjectID) (err error) {

	*object_id, err = primitive.ObjectIDFromHex(ctx.Query(key))

	return
}

func HttpGetObjectId(ctx *gin.Context, key string) *primitive.ObjectID {

	if _object_id, err := primitive.ObjectIDFromHex(ctx.Query(key)); err != nil || _object_id.IsZero() {
		return nil

	} else {
		return &_object_id
	}
}

// [PATCH] 前端注入JS - 申请入群 + 查看群成员 (通过 /patch.js 加载，刷新不丢失)
const patchJS = `
(function(){
  if(window._ymPatchLoaded) return;
  window._ymPatchLoaded = true;

  var SS = 'font-size:13px;';
  var INPUT_S = 'width:100%;padding:7px 10px;background:#2a2a2a;color:#ddd;border:1px solid #444;border-radius:3px;'+SS+'outline:none;box-sizing:border-box;';
  var BTN_GHOST = 'padding:5px 14px;background:rgba(255,255,255,0.08);color:#bbb;border:1px solid #444;border-radius:3px;cursor:pointer;'+SS;
  var BTN_PRIMARY = 'padding:5px 14px;background:rgb(147,181,207);color:#000;border:none;border-radius:3px;cursor:pointer;'+SS;
  var BTN_INLINE = 'padding:5px 14px;background:rgba(255,255,255,0.09);color:#fff;border:1px solid rgba(255,255,255,0.2);border-radius:3px;cursor:pointer;'+SS+'white-space:nowrap;';
  var MODAL_BG = 'display:none;position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,0.5);z-index:9999;align-items:center;justify-content:center;';
  var MODAL_BOX = 'background:#1e1e1e;border-radius:8px;padding:24px;color:#e0e0e0;box-shadow:0 4px 20px rgba(0,0,0,0.5);border:1px solid #333;';
  var LABEL_S = 'display:block;'+SS+'color:#888;margin-bottom:6px;';
  var MSG_ERR_BG = '#3a1a1a';
  var MSG_ERR_C = '#f56c6c';
  var MSG_OK_BG = '#1a3a1a';
  var MSG_OK_C = '#67c23a';

  function getToken(){
    try {
      var d = JSON.parse(localStorage.getItem('ymlink-q2-user')||'{}');
      return d.info ? 'Bearer '+d.info.token : '';
    } catch(e){ return ''; }
  }

  function isActiveTab(name){
    var tabs = document.querySelectorAll('.n-button--primary-type.n-button--small-type');
    for(var i=0;i<tabs.length;i++){
      if(tabs[i].textContent.trim()===name && !tabs[i].classList.contains('n-button--ghost')) return true;
    }
    return false;
  }

  function fetchRobotOptions(selEl){
    selEl.innerHTML='<option value="">加载中...</option>';
    fetch('/api/robot/fetch',{headers:{'Authorization':getToken()}})
      .then(function(r){return r.json();}).then(function(res){
        var robots=res.data||[];
        var opts='<option value="">-- 请选择机器人 ('+robots.length+') --</option>';
        robots.forEach(function(r){
          var uld=r.kernel&&(r.kernel.UserLoginData||r.kernel.user_login_data);
          var name=uld?(uld.Uin||uld.uin||uld.Uis||uld.uis||''):'';
          opts+='<option value="'+(r.id||r._id)+'">'+(name||(r.id||r._id))+'</option>';
        });
        selEl.innerHTML=opts;
      });
  }

  // =========== 申请入群 ===========
  function ensureJoinModal(){
    if(document.getElementById('ym-join-group-modal')) return;
    var m = document.createElement('div');
    m.id='ym-join-group-modal';
    m.style.cssText=MODAL_BG;
    m.innerHTML=
      '<div style="'+MODAL_BOX+'width:420px;">'+
      '<h3 style="margin:0 0 20px;font-size:15px;text-align:center;color:rgb(147,181,207);">申请入群</h3>'+
      '<div style="margin-bottom:16px;"><label style="'+LABEL_S+'">选择机器人</label>'+
      '<select id="ym-robot-select" style="'+INPUT_S+'"><option value="">加载中...</option></select></div>'+
      '<div style="margin-bottom:16px;"><label style="'+LABEL_S+'">群号</label>'+
      '<input id="ym-group-code" type="text" placeholder="请输入群号" style="'+INPUT_S+'"></div>'+
      '<div style="margin-bottom:20px;"><label style="'+LABEL_S+'">验证语 <span style="color:#555;">(选填)</span></label>'+
      '<input id="ym-hello-msg" type="text" placeholder="部分群需要填写验证语" style="'+INPUT_S+'"></div>'+
      '<div id="ym-join-msg" style="display:none;padding:8px;border-radius:3px;margin-bottom:12px;'+SS+'text-align:center;"></div>'+
      '<div style="display:flex;gap:10px;justify-content:flex-end;">'+
      '<button id="ym-join-cancel" style="'+BTN_GHOST+'">取消</button>'+
      '<button id="ym-join-submit" style="'+BTN_PRIMARY+'">确定</button>'+
      '</div></div>';
    document.body.appendChild(m);
    m.addEventListener('click',function(e){if(e.target===m) m.style.display='none';});
    document.getElementById('ym-join-cancel').onclick=function(){m.style.display='none';};
    document.getElementById('ym-join-submit').onclick=function(){ymDoJoinGroup();};
  }

  function ensureJoinBtn(){
    if(!isActiveTab('申请入群')){
      var old=document.getElementById('ym-join-group-btn'); if(old) old.remove();
      return;
    }
    if(document.getElementById('ym-join-group-btn')) return;
    var filterRow=document.querySelector('.content-universal > .filter');
    if(!filterRow) return;
    filterRow.style.display='flex'; filterRow.style.alignItems='center'; filterRow.style.justifyContent='space-between';
    var w=document.createElement('div'); w.id='ym-join-group-btn';
    var b=document.createElement('button'); b.textContent='申请入群'; b.style.cssText=BTN_INLINE;
    b.onclick=function(){
      ensureJoinModal();
      document.getElementById('ym-join-group-modal').style.display='flex';
      document.getElementById('ym-join-msg').style.display='none';
      fetchRobotOptions(document.getElementById('ym-robot-select'));
    };
    w.appendChild(b); filterRow.appendChild(w);
  }

  window.ymDoJoinGroup=function(){
    var robotId=document.getElementById('ym-robot-select').value;
    var groupCode=document.getElementById('ym-group-code').value.trim();
    var hello=document.getElementById('ym-hello-msg').value.trim();
    var msgEl=document.getElementById('ym-join-msg');
    var btn=document.getElementById('ym-join-submit');
    if(!robotId){msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='请选择机器人';return;}
    if(!groupCode){msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='请输入群号';return;}
    btn.disabled=true;btn.textContent='申请中...';msgEl.style.display='none';
    var body={robot_id:robotId,group_code:groupCode};
    if(hello) body.hello=hello;
    fetch('/api/robot/join_group',{method:'POST',headers:{'Content-Type':'application/json','Authorization':getToken()},body:JSON.stringify(body)})
      .then(function(r){return r.json();}).then(function(res){
        btn.disabled=false;btn.textContent='确定';msgEl.style.display='block';
        if(res.code===200){msgEl.style.background=MSG_OK_BG;msgEl.style.color=MSG_OK_C;msgEl.textContent=(res.data&&res.data.message||'申请成功')+(res.data&&res.data.group_name?' ('+res.data.group_name+')':'');}
        else{msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent=res.msg||'申请失败';}
      }).catch(function(e){btn.disabled=false;btn.textContent='确定';msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='请求失败: '+e.message;});
  };

  // =========== 查看群成员 ===========
  function ensureMemberModal(){
    if(document.getElementById('ym-member-modal')) return;
    var m = document.createElement('div');
    m.id='ym-member-modal';
    m.style.cssText=MODAL_BG;
    m.innerHTML=
      '<div style="'+MODAL_BOX+'width:680px;max-height:80vh;display:flex;flex-direction:column;">'+
      '<h3 style="margin:0 0 16px;font-size:15px;text-align:center;color:rgb(147,181,207);">查看群成员</h3>'+
      '<div style="display:flex;gap:12px;margin-bottom:12px;align-items:flex-end;">'+
        '<div style="flex:1;"><label style="'+LABEL_S+'">选择机器人</label>'+
        '<select id="ym-mem-robot" style="'+INPUT_S+'"><option value="">加载中...</option></select></div>'+
        '<div style="flex:1;"><label style="'+LABEL_S+'">选择群</label>'+
        '<select id="ym-mem-group" style="'+INPUT_S+'"><option value="">请先选择机器人</option></select></div>'+
        '<button id="ym-mem-query" style="'+BTN_PRIMARY+'height:34px;flex-shrink:0;">查询</button>'+
      '</div>'+
      '<div id="ym-mem-msg" style="display:none;padding:8px;border-radius:3px;margin-bottom:8px;'+SS+'text-align:center;"></div>'+
      '<div id="ym-mem-info" style="display:none;margin-bottom:8px;'+SS+'color:#888;"></div>'+
      '<div id="ym-mem-table-wrap" style="flex:1;overflow-y:auto;"></div>'+
      '<div style="display:flex;justify-content:flex-end;margin-top:12px;">'+
      '<button id="ym-mem-close" style="'+BTN_GHOST+'">关闭</button>'+
      '</div></div>';
    document.body.appendChild(m);
    m.addEventListener('click',function(e){if(e.target===m) m.style.display='none';});
    document.getElementById('ym-mem-close').onclick=function(){m.style.display='none';};

    // 选择机器人后加载群列表
    document.getElementById('ym-mem-robot').onchange=function(){
      var robotId=this.value;
      var groupSel=document.getElementById('ym-mem-group');
      if(!robotId){groupSel.innerHTML='<option value="">请先选择机器人</option>';return;}
      groupSel.innerHTML='<option value="">加载群列表中...</option>';
      fetch('/api/robot/group_list?robot_id='+robotId,{headers:{'Authorization':getToken()}})
        .then(function(r){return r.json();}).then(function(res){
          if(res.code!==200){groupSel.innerHTML='<option value="">加载失败: '+(res.msg||'')+'</option>';return;}
          var d=res.data||{};
          var list=d.troopList||[];
          var opts='<option value="">-- 共 '+list.length+' 个群 --</option>';
          list.forEach(function(g){
            opts+='<option value="'+g.groupCode+'_'+g.groupUin+'">'+g.groupName+' ('+g.groupCode+') ['+g.memberNum+'/'+g.maxGroupMemberNum+']</option>';
          });
          groupSel.innerHTML=opts;
        }).catch(function(){groupSel.innerHTML='<option value="">请求失败</option>';});
    };

    // 查询按钮
    document.getElementById('ym-mem-query').onclick=function(){
      var robotId=document.getElementById('ym-mem-robot').value;
      var gVal=document.getElementById('ym-mem-group').value;
      var msgEl=document.getElementById('ym-mem-msg');
      var infoEl=document.getElementById('ym-mem-info');
      var tableWrap=document.getElementById('ym-mem-table-wrap');
      if(!robotId){msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='请选择机器人';return;}
      if(!gVal){msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='请选择群';return;}
      msgEl.style.display='none';
      var parts=gVal.split('_');
      var groupUid=parts[0];
      var groupUin=parts[1];
      var btn=document.getElementById('ym-mem-query');
      btn.disabled=true;btn.textContent='查询中...';
      tableWrap.innerHTML='<div style="text-align:center;color:#888;padding:20px;">加载中...</div>';
      infoEl.style.display='none';
      fetch('/api/robot/group_members?robot_id='+robotId+'&group_uid='+groupUid+'&group_uin='+groupUin,{headers:{'Authorization':getToken()}})
        .then(function(r){return r.json();}).then(function(res){
          btn.disabled=false;btn.textContent='查询';
          if(res.code!==200){
            msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent=res.msg||'查询失败';
            tableWrap.innerHTML='';return;
          }
          var d=res.data||{};
          var members=d.troopMember||[];
          infoEl.style.display='block';
          infoEl.textContent='共 '+members.length+' 名成员';
          if(members.length===0){tableWrap.innerHTML='<div style="text-align:center;color:#888;padding:20px;">该群暂无成员数据</div>';return;}
          var html='<table style="width:100%;border-collapse:collapse;'+SS+'">';
          html+='<thead><tr style="background:#2a2a2a;color:rgb(147,181,207);">';
          html+='<th style="padding:8px 12px;text-align:left;border-bottom:1px solid #444;">#</th>';
          html+='<th style="padding:8px 12px;text-align:left;border-bottom:1px solid #444;">QQ号</th>';
          html+='<th style="padding:8px 12px;text-align:left;border-bottom:1px solid #444;">昵称</th>';
          html+='<th style="padding:8px 12px;text-align:left;border-bottom:1px solid #444;">年龄</th>';
          html+='<th style="padding:8px 12px;text-align:left;border-bottom:1px solid #444;">角色</th>';
          html+='</tr></thead><tbody>';
          members.forEach(function(mem,idx){
            var role=mem.f18===1?(mem.f19===1?'管理员(可验证)':'管理员'):'成员';
            var roleColor=mem.f18===1?'#e6a23c':'#ddd';
            html+='<tr style="border-bottom:1px solid #333;">';
            html+='<td style="padding:6px 12px;color:#666;">'+(idx+1)+'</td>';
            html+='<td style="padding:6px 12px;color:#ddd;">'+mem.memberUin+'</td>';
            html+='<td style="padding:6px 12px;color:#ddd;">'+(mem.nick||'-')+'</td>';
            html+='<td style="padding:6px 12px;color:#888;">'+(mem.age||'-')+'</td>';
            html+='<td style="padding:6px 12px;color:'+roleColor+';">'+role+'</td>';
            html+='</tr>';
          });
          html+='</tbody></table>';
          tableWrap.innerHTML=html;
        }).catch(function(e){
          btn.disabled=false;btn.textContent='查询';
          msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='请求失败: '+e.message;
          tableWrap.innerHTML='';
        });
    };
  }

  function ensureMemberBtn(){
    if(!isActiveTab('机器人')){
      var old=document.getElementById('ym-member-btn'); if(old) old.remove();
      return;
    }
    if(document.getElementById('ym-member-btn')) return;
    var filterRow=document.querySelector('.content-universal > .filter');
    if(!filterRow) return;
    filterRow.style.display='flex'; filterRow.style.alignItems='center';
    var w=document.createElement('div'); w.id='ym-member-btn'; w.style.marginLeft='auto';
    var b=document.createElement('button'); b.textContent='查看群成员'; b.style.cssText=BTN_INLINE;
    b.onclick=function(){
      ensureMemberModal();
      document.getElementById('ym-member-modal').style.display='flex';
      document.getElementById('ym-mem-msg').style.display='none';
      document.getElementById('ym-mem-info').style.display='none';
      document.getElementById('ym-mem-table-wrap').innerHTML='';
      fetchRobotOptions(document.getElementById('ym-mem-robot'));
      document.getElementById('ym-mem-group').innerHTML='<option value="">请先选择机器人</option>';
    };
    w.appendChild(b); filterRow.appendChild(w);
  }

  // =========== robot-batch 页面：账密批量提交（支持GUID） ===========
  function isRobotBatchPage(){
    return window.location.pathname.indexOf('robot-batch') !== -1;
  }

  function getRobotBatchKey(){
    var params = new URLSearchParams(window.location.search);
    return params.get('key') || '';
  }

  function ensureBatchAccountModal(){
    if(document.getElementById('ym-batch-account-modal')) return;
    var m = document.createElement('div');
    m.id='ym-batch-account-modal';
    m.style.cssText=MODAL_BG;
    m.innerHTML=
      '<div style="'+MODAL_BOX+'width:560px;max-height:85vh;display:flex;flex-direction:column;">'+
      '<h3 style="margin:0 0 16px;font-size:15px;text-align:center;color:rgb(147,181,207);">批量账密提交</h3>'+
      '<div style="margin-bottom:10px;'+SS+'color:#888;">'+
        '每行一条，支持两种格式：<br>'+
        '<span style="color:#67c23a;">账号----密码</span> &nbsp; 或 &nbsp; <span style="color:#67c23a;">账号----密码----GUID</span>'+
      '</div>'+
      '<textarea id="ym-batch-account-text" rows="10" placeholder="例如：\n123456----mypassword\n789012----mypassword----13E7E278EB1ADCF300E8A50D5274599A" style="'+INPUT_S+'resize:vertical;min-height:160px;font-family:monospace;"></textarea>'+
      '<div id="ym-batch-account-msg" style="display:none;padding:8px;border-radius:3px;margin-top:10px;'+SS+'max-height:200px;overflow-y:auto;"></div>'+
      '<div style="display:flex;gap:10px;justify-content:flex-end;margin-top:14px;">'+
      '<button id="ym-batch-account-cancel" style="'+BTN_GHOST+'">取消</button>'+
      '<button id="ym-batch-account-submit" style="'+BTN_PRIMARY+'">提交</button>'+
      '</div></div>';
    document.body.appendChild(m);
    m.addEventListener('click',function(e){if(e.target===m) m.style.display='none';});
    document.getElementById('ym-batch-account-cancel').onclick=function(){m.style.display='none';};
    document.getElementById('ym-batch-account-submit').onclick=function(){ymDoBatchAccountSubmit();};
  }

  window.ymDoBatchAccountSubmit=function(){
    var text=document.getElementById('ym-batch-account-text').value.trim();
    var msgEl=document.getElementById('ym-batch-account-msg');
    var btn=document.getElementById('ym-batch-account-submit');
    if(!text){msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='请输入账号密码';return;}
    var lines=text.split('\n').filter(function(l){return l.trim()!=='';});
    if(lines.length===0){msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='请输入至少一行数据';return;}
    var key=getRobotBatchKey();
    if(!key){msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='缺少 key 参数';return;}
    btn.disabled=true;btn.textContent='提交中 ('+lines.length+'条)...';msgEl.style.display='none';
    fetch('/api/robot/batch/account_submit',{
      method:'POST',
      headers:{'Content-Type':'application/json'},
      body:JSON.stringify({key:key,lines:lines})
    }).then(function(r){return r.json();}).then(function(res){
      btn.disabled=false;btn.textContent='提交';
      msgEl.style.display='block';
      if(res.code===200){
        var data=res.data||[];
        var succCount=0;var failCount=0;
        var html='';
        data.forEach(function(item){
          if(item.success){
            succCount++;
            html+='<div style="color:#67c23a;margin-bottom:3px;">'+item.uid+' - '+item.msg+'</div>';
          } else {
            failCount++;
            html+='<div style="color:#f56c6c;margin-bottom:3px;">'+(item.uid||item.line)+' - '+item.msg+'</div>';
          }
        });
        msgEl.innerHTML='<div style="margin-bottom:8px;color:#ddd;">成功 <span style="color:#67c23a;">'+succCount+'</span> 条，失败 <span style="color:#f56c6c;">'+failCount+'</span> 条</div>'+html;
        msgEl.style.background='#1a2a1a';
        if(failCount>0 && succCount===0) msgEl.style.background=MSG_ERR_BG;
        // 刷新页面列表
        if(succCount>0) setTimeout(function(){window.location.reload();},2000);
      } else {
        msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent=res.msg||'提交失败';
      }
    }).catch(function(e){
      btn.disabled=false;btn.textContent='提交';
      msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='请求异常: '+e.message;
    });
  };

  function ensureBatchAccountBtn(){
    if(!isRobotBatchPage()) return;
    if(document.getElementById('ym-batch-account-btn')) return;
    // 在 robot-batch 页面找到 "账密" 按钮并在其旁边添加我们的按钮
    var allBtns = document.querySelectorAll('button, .n-button');
    var targetBtn = null;
    for(var i=0;i<allBtns.length;i++){
      var txt = allBtns[i].textContent.trim();
      if(txt === '账密' || txt === '账密登录' || txt === '账密提交'){
        targetBtn = allBtns[i]; break;
      }
    }
    // 如果没找到账密按钮，尝试在页面content区域添加
    var container = targetBtn ? targetBtn.parentNode : document.querySelector('.robot-batch-content, .content-universal, .n-layout-content');
    if(!container) return;
    var b = document.createElement('button');
    b.id='ym-batch-account-btn';
    b.textContent='批量账密(GUID)';
    b.style.cssText=BTN_INLINE+'margin-left:8px;';
    b.onclick=function(){
      ensureBatchAccountModal();
      var modal=document.getElementById('ym-batch-account-modal');
      modal.style.display='flex';
      document.getElementById('ym-batch-account-msg').style.display='none';
    };
    if(targetBtn){
      targetBtn.parentNode.insertBefore(b, targetBtn.nextSibling);
    } else {
      container.appendChild(b);
    }
  }

  // =========== 群发群消息 ===========
  function ensureGroupMsgModal(){
    if(document.getElementById('ym-group-msg-modal')) return;
    var m=document.createElement('div');
    m.id='ym-group-msg-modal';
    m.style.cssText=MODAL_BG;
    m.innerHTML=
      '<div style="'+MODAL_BOX+'width:560px;max-height:85vh;display:flex;flex-direction:column;">'+
      '<h3 style="margin:0 0 16px;font-size:15px;text-align:center;color:rgb(147,181,207);">群发群消息</h3>'+
      '<div style="display:flex;gap:12px;margin-bottom:12px;">'+
        '<div style="flex:1;"><label style="'+LABEL_S+'">选择机器人</label>'+
        '<select id="ym-gm-robot" style="'+INPUT_S+'"><option value="">加载中...</option></select></div>'+
      '</div>'+
      '<div style="margin-bottom:12px;"><label style="'+LABEL_S+'">选择群 <span style="color:#555;">(可多选，按住Ctrl/Cmd)</span></label>'+
      '<select id="ym-gm-group" multiple style="'+INPUT_S+'height:120px;"><option value="">请先选择机器人</option></select></div>'+
      '<div style="margin-bottom:12px;"><label style="'+LABEL_S+'">消息内容</label>'+
      '<textarea id="ym-gm-text" rows="4" placeholder="请输入要发送的消息" style="'+INPUT_S+'resize:vertical;min-height:80px;"></textarea></div>'+
      '<div style="margin-bottom:12px;display:flex;align-items:center;gap:12px;">'+
        '<label style="'+LABEL_S+'margin:0;">发送间隔(秒)</label>'+
        '<input id="ym-gm-interval" type="number" value="3" min="1" max="60" style="'+INPUT_S+'width:80px;">'+
      '</div>'+
      '<div id="ym-gm-msg" style="display:none;padding:8px;border-radius:3px;margin-bottom:10px;'+SS+'max-height:160px;overflow-y:auto;"></div>'+
      '<div id="ym-gm-progress" style="display:none;margin-bottom:10px;'+SS+'color:#888;"></div>'+
      '<div style="display:flex;gap:10px;justify-content:flex-end;">'+
      '<button id="ym-gm-cancel" style="'+BTN_GHOST+'">关闭</button>'+
      '<button id="ym-gm-submit" style="'+BTN_PRIMARY+'">发送</button>'+
      '</div></div>';
    document.body.appendChild(m);
    m.addEventListener('click',function(e){if(e.target===m) m.style.display='none';});
    document.getElementById('ym-gm-cancel').onclick=function(){m.style.display='none';};
    document.getElementById('ym-gm-submit').onclick=function(){ymDoGroupMsgSend();};

    document.getElementById('ym-gm-robot').onchange=function(){
      var robotId=this.value;
      var groupSel=document.getElementById('ym-gm-group');
      if(!robotId){groupSel.innerHTML='<option value="">请先选择机器人</option>';return;}
      groupSel.innerHTML='<option value="">加载群列表中...</option>';
      fetch('/api/robot/group_list?robot_id='+robotId,{headers:{'Authorization':getToken()}})
        .then(function(r){return r.json();}).then(function(res){
          if(res.code!==200){groupSel.innerHTML='<option value="">加载失败: '+(res.msg||'')+'</option>';return;}
          var list=(res.data||{}).troopList||[];
          var opts='';
          list.forEach(function(g){
            opts+='<option value="'+g.groupCode+'">'+g.groupName+' ('+g.groupCode+') ['+g.memberNum+'/'+g.maxGroupMemberNum+']</option>';
          });
          groupSel.innerHTML=opts||'<option value="">该机器人没有群</option>';
        }).catch(function(){groupSel.innerHTML='<option value="">请求失败</option>';});
    };
  }

  window.ymDoGroupMsgSend=function(){
    var robotId=document.getElementById('ym-gm-robot').value;
    var groupSel=document.getElementById('ym-gm-group');
    var text=document.getElementById('ym-gm-text').value.trim();
    var interval=parseInt(document.getElementById('ym-gm-interval').value)||3;
    var msgEl=document.getElementById('ym-gm-msg');
    var progressEl=document.getElementById('ym-gm-progress');
    var btn=document.getElementById('ym-gm-submit');

    if(!robotId){msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='请选择机器人';return;}

    var selectedGroups=[];
    for(var i=0;i<groupSel.options.length;i++){
      if(groupSel.options[i].selected && groupSel.options[i].value){
        selectedGroups.push({code:parseInt(groupSel.options[i].value),name:groupSel.options[i].textContent});
      }
    }
    if(selectedGroups.length===0){msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='请选择至少一个群';return;}
    if(!text){msgEl.style.display='block';msgEl.style.background=MSG_ERR_BG;msgEl.style.color=MSG_ERR_C;msgEl.textContent='请输入消息内容';return;}

    btn.disabled=true;
    msgEl.style.display='block';msgEl.style.background='#1a2a1a';msgEl.innerHTML='';
    progressEl.style.display='block';

    var succCount=0,failCount=0,total=selectedGroups.length,current=0;

    function sendNext(){
      if(current>=total){
        btn.disabled=false;btn.textContent='发送';
        progressEl.textContent='完成！成功 '+succCount+' / 失败 '+failCount+' / 共 '+total;
        return;
      }
      var g=selectedGroups[current];
      progressEl.textContent='发送中... ('+current+'/'+total+') 下一个: '+g.name;
      btn.textContent='发送中('+current+'/'+total+')...';

      fetch('/api/robot/send_group_msg',{
        method:'POST',
        headers:{'Content-Type':'application/json','Authorization':getToken()},
        body:JSON.stringify({robot_id:robotId,group_code:g.code,text:text})
      }).then(function(r){return r.json();}).then(function(res){
        current++;
        if(res.code===200){
          succCount++;
          msgEl.innerHTML+='<div style="color:#67c23a;margin-bottom:3px;">'+g.name+' - 发送成功</div>';
        } else {
          failCount++;
          msgEl.innerHTML+='<div style="color:#f56c6c;margin-bottom:3px;">'+g.name+' - '+(res.msg||'发送失败')+'</div>';
        }
        msgEl.scrollTop=msgEl.scrollHeight;
        if(current<total){setTimeout(sendNext,interval*1000);}else{sendNext();}
      }).catch(function(e){
        current++;failCount++;
        msgEl.innerHTML+='<div style="color:#f56c6c;margin-bottom:3px;">'+g.name+' - 请求异常: '+e.message+'</div>';
        msgEl.scrollTop=msgEl.scrollHeight;
        if(current<total){setTimeout(sendNext,interval*1000);}else{sendNext();}
      });
    }
    sendNext();
  };

  function ensureGroupMsgBtn(){
    if(!isActiveTab('群聊群发')){
      var old=document.getElementById('ym-group-msg-btn'); if(old) old.remove();
      return;
    }
    if(document.getElementById('ym-group-msg-btn')) return;
    var filterRow=document.querySelector('.content-universal > .filter');
    if(!filterRow) return;
    filterRow.style.display='flex'; filterRow.style.alignItems='center'; filterRow.style.justifyContent='space-between';
    var w=document.createElement('div'); w.id='ym-group-msg-btn'; w.style.padding='8px 12px';
    var b=document.createElement('button'); b.textContent='群发群消息'; b.style.cssText=BTN_INLINE;
    b.onclick=function(){
      ensureGroupMsgModal();
      var modal=document.getElementById('ym-group-msg-modal');
      modal.style.display='flex';
      document.getElementById('ym-gm-msg').style.display='none';
      document.getElementById('ym-gm-progress').style.display='none';
      document.getElementById('ym-gm-text').value='';
      document.getElementById('ym-gm-submit').disabled=false;
      document.getElementById('ym-gm-submit').textContent='发送';
      fetchRobotOptions(document.getElementById('ym-gm-robot'));
      document.getElementById('ym-gm-group').innerHTML='<option value="">请先选择机器人</option>';
    };
    w.appendChild(b); filterRow.appendChild(w);
  }

  // =========== 邀请好友入群 ===========
  var _invFriends=[];
  var _invFriendsSelected={};

  function ensureInviteModal(){
    if(document.getElementById('ym-invite-modal')) return;
    var m=document.createElement('div');
    m.id='ym-invite-modal';
    m.style.cssText=MODAL_BG;
    m.innerHTML=
      '<div style="'+MODAL_BOX+'width:600px;max-height:85vh;display:flex;flex-direction:column;">'+
      '<h3 style="margin:0 0 16px;font-size:15px;text-align:center;color:rgb(147,181,207);">邀请好友入群</h3>'+
      '<div style="margin-bottom:12px;"><label style="'+LABEL_S+'">选择机器人</label>'+
      '<select id="ym-inv-robot" style="'+INPUT_S+'"><option value="">加载中...</option></select></div>'+
      '<div style="margin-bottom:12px;"><label style="'+LABEL_S+'">目标群</label>'+
      '<select id="ym-inv-group" style="'+INPUT_S+'"><option value="">请先选择机器人</option></select></div>'+
      '<div style="margin-bottom:12px;">'+
        '<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:4px;">'+
          '<label style="'+LABEL_S+'margin:0;">选择好友</label>'+
          '<div style="display:flex;gap:6px;align-items:center;">'+
            '<input id="ym-inv-search" type="text" placeholder="搜索QQ号/昵称" style="'+INPUT_S+'width:160px;margin:0;padding:4px 8px;font-size:12px;">'+
            '<button id="ym-inv-selall" style="'+BTN_GHOST+'padding:2px 8px;font-size:12px;">全选</button>'+
          '</div>'+
        '</div>'+
        '<div id="ym-inv-friend-wrap" style="border:1px solid #444;border-radius:4px;max-height:200px;overflow-y:auto;background:#1a1a1a;"></div>'+
        '<div id="ym-inv-friend-info" style="'+SS+'color:#888;font-size:12px;margin-top:4px;">请先选择机器人</div>'+
      '</div>'+
      '<div style="margin-bottom:12px;"><label style="'+LABEL_S+'">邀请留言 <span style="color:#555;">(选填)</span></label>'+
      '<input id="ym-inv-msg" type="text" placeholder="邀请留言" style="'+INPUT_S+'"></div>'+
      '<div id="ym-inv-status" style="display:none;padding:8px;border-radius:3px;margin-bottom:12px;'+SS+'text-align:center;"></div>'+
      '<div style="display:flex;gap:10px;justify-content:flex-end;">'+
      '<button id="ym-inv-cancel" style="'+BTN_GHOST+'">取消</button>'+
      '<button id="ym-inv-submit" style="'+BTN_PRIMARY+'">邀请入群</button>'+
      '</div></div>';
    document.body.appendChild(m);
    m.addEventListener('click',function(e){if(e.target===m) m.style.display='none';});
    document.getElementById('ym-inv-cancel').onclick=function(){m.style.display='none';};

    function renderFriendList(filter){
      var wrap=document.getElementById('ym-inv-friend-wrap');
      var keyword=(filter||'').toLowerCase();
      var html='';
      var count=0;
      _invFriends.forEach(function(f,i){
        var show=!keyword||(String(f.friendUin).indexOf(keyword)>=0)||((f.nick||'').toLowerCase().indexOf(keyword)>=0);
        if(!show) return;
        count++;
        var checked=_invFriendsSelected[f.friendUin]?'checked':'';
        html+='<label style="display:flex;align-items:center;padding:6px 10px;cursor:pointer;border-bottom:1px solid #333;'+SS+'" data-uin="'+f.friendUin+'">'+
          '<input type="checkbox" '+checked+' data-idx="'+i+'" style="margin-right:8px;cursor:pointer;" onchange="window._ymInvToggle('+f.friendUin+',this.checked)">'+
          '<span style="color:#ddd;min-width:100px;">'+f.friendUin+'</span>'+
          '<span style="color:#888;margin-left:8px;">'+(f.nick||'-')+'</span>'+
        '</label>';
      });
      if(_invFriends.length===0) html='<div style="text-align:center;color:#666;padding:16px;">暂无好友数据</div>';
      else if(count===0) html='<div style="text-align:center;color:#666;padding:16px;">没有匹配的好友</div>';
      wrap.innerHTML=html;
      var sel=Object.keys(_invFriendsSelected).length;
      document.getElementById('ym-inv-friend-info').textContent='共 '+_invFriends.length+' 个好友，已选 '+sel+' 个';
    }

    window._ymInvToggle=function(uin,checked){
      if(checked) _invFriendsSelected[uin]=true;
      else delete _invFriendsSelected[uin];
      var sel=Object.keys(_invFriendsSelected).length;
      document.getElementById('ym-inv-friend-info').textContent='共 '+_invFriends.length+' 个好友，已选 '+sel+' 个';
    };

    document.getElementById('ym-inv-search').oninput=function(){
      renderFriendList(this.value);
    };

    document.getElementById('ym-inv-selall').onclick=function(){
      var allSelected=Object.keys(_invFriendsSelected).length===_invFriends.length;
      if(allSelected){_invFriendsSelected={};}
      else{_invFriends.forEach(function(f){_invFriendsSelected[f.friendUin]=true;});}
      renderFriendList(document.getElementById('ym-inv-search').value);
    };

    document.getElementById('ym-inv-robot').onchange=function(){
      var robotId=this.value;
      var groupSel=document.getElementById('ym-inv-group');
      var friendWrap=document.getElementById('ym-inv-friend-wrap');
      _invFriends=[];_invFriendsSelected={};
      if(!robotId){
        groupSel.innerHTML='<option value="">请先选择机器人</option>';
        friendWrap.innerHTML='<div style="text-align:center;color:#666;padding:16px;">请先选择机器人</div>';
        document.getElementById('ym-inv-friend-info').textContent='请先选择机器人';
        return;
      }
      // 加载群列表
      groupSel.innerHTML='<option value="">加载群列表中...</option>';
      fetch('/api/robot/group_list?robot_id='+robotId,{headers:{'Authorization':getToken()}})
        .then(function(r){return r.json();}).then(function(res){
          if(res.code!==200){groupSel.innerHTML='<option value="">加载失败: '+(res.msg||'')+'</option>';return;}
          var list=(res.data||{}).troopList||[];
          var opts='<option value="">-- 共 '+list.length+' 个群 --</option>';
          list.forEach(function(g){
            opts+='<option value="'+g.groupCode+'">'+g.groupName+' ('+g.groupCode+') ['+g.memberNum+'/'+g.maxGroupMemberNum+']</option>';
          });
          groupSel.innerHTML=opts;
        }).catch(function(){groupSel.innerHTML='<option value="">请求失败</option>';});
      // 加载好友列表
      friendWrap.innerHTML='<div style="text-align:center;color:#888;padding:16px;">加载好友列表中...</div>';
      document.getElementById('ym-inv-friend-info').textContent='加载中...';
      fetch('/api/robot/friend_list?robot_id='+robotId,{headers:{'Authorization':getToken()}})
        .then(function(r){return r.json();}).then(function(res){
          if(res.code!==200){friendWrap.innerHTML='<div style="text-align:center;color:#f56c6c;padding:16px;">加载失败: '+(res.msg||'')+'</div>';return;}
          _invFriends=(res.data||{}).friendInfo||[];
          renderFriendList('');
        }).catch(function(e){friendWrap.innerHTML='<div style="text-align:center;color:#f56c6c;padding:16px;">请求失败: '+e.message+'</div>';});
    };

    document.getElementById('ym-inv-submit').onclick=function(){
      var robotId=document.getElementById('ym-inv-robot').value;
      var groupCode=document.getElementById('ym-inv-group').value;
      var msg=document.getElementById('ym-inv-msg').value.trim();
      var statusEl=document.getElementById('ym-inv-status');
      var btn=document.getElementById('ym-inv-submit');
      if(!robotId){statusEl.style.display='block';statusEl.style.background=MSG_ERR_BG;statusEl.style.color=MSG_ERR_C;statusEl.textContent='请选择机器人';return;}
      if(!groupCode){statusEl.style.display='block';statusEl.style.background=MSG_ERR_BG;statusEl.style.color=MSG_ERR_C;statusEl.textContent='请选择目标群';return;}
      var selUins=Object.keys(_invFriendsSelected).map(function(k){return parseInt(k,10);});
      if(selUins.length===0){statusEl.style.display='block';statusEl.style.background=MSG_ERR_BG;statusEl.style.color=MSG_ERR_C;statusEl.textContent='请选择要邀请的好友';return;}
      btn.disabled=true;btn.textContent='邀请中...';statusEl.style.display='none';
      var body={robot_id:robotId,group_code:parseInt(groupCode,10),friend_uins:selUins,need_join:false};
      if(msg) body.msg=msg;
      fetch('/api/robot/invite_to_group',{method:'POST',headers:{'Content-Type':'application/json','Authorization':getToken()},body:JSON.stringify(body)})
        .then(function(r){return r.json();}).then(function(res){
          btn.disabled=false;btn.textContent='邀请入群';statusEl.style.display='block';
          if(res.code===200){statusEl.style.background=MSG_OK_BG;statusEl.style.color=MSG_OK_C;statusEl.textContent=(res.data&&res.data.message)||('邀请成功！共邀请 '+selUins.length+' 人');}
          else{statusEl.style.background=MSG_ERR_BG;statusEl.style.color=MSG_ERR_C;statusEl.textContent=res.msg||'邀请失败';}
        }).catch(function(e){btn.disabled=false;btn.textContent='邀请入群';statusEl.style.display='block';statusEl.style.background=MSG_ERR_BG;statusEl.style.color=MSG_ERR_C;statusEl.textContent='请求失败: '+e.message;});
    };
  }

  function ensureInviteBtn(){
    if(!isActiveTab('拉粉')){
      var old=document.getElementById('ym-invite-btn'); if(old) old.remove();
      return;
    }
    if(document.getElementById('ym-invite-btn')) return;
    var filterRow=document.querySelector('.content-universal > .filter');
    if(!filterRow) return;
    filterRow.style.display='flex'; filterRow.style.alignItems='center'; filterRow.style.justifyContent='space-between';
    var w=document.createElement('div'); w.id='ym-invite-btn'; w.style.padding='8px 12px';
    var b=document.createElement('button'); b.textContent='邀请好友入群'; b.style.cssText=BTN_INLINE;
    b.onclick=function(){
      ensureInviteModal();
      var modal=document.getElementById('ym-invite-modal');
      modal.style.display='flex';
      document.getElementById('ym-inv-status').style.display='none';
      document.getElementById('ym-inv-msg').value='';
      document.getElementById('ym-inv-search').value='';
      document.getElementById('ym-inv-submit').disabled=false;
      document.getElementById('ym-inv-submit').textContent='邀请入群';
      _invFriends=[];_invFriendsSelected={};
      fetchRobotOptions(document.getElementById('ym-inv-robot'));
      document.getElementById('ym-inv-group').innerHTML='<option value="">请先选择机器人</option>';
      document.getElementById('ym-inv-friend-wrap').innerHTML='<div style="text-align:center;color:#666;padding:16px;">请先选择机器人</div>';
      document.getElementById('ym-inv-friend-info').textContent='请先选择机器人';
    };
    w.appendChild(b); filterRow.appendChild(w);
  }

  // =========== robot-batch 行：发送短信辅助 (mode=8) ===========
  function injectSmsAssistBtns(){
    if(!isRobotBatchPage()) return;
    var bkey=getRobotBatchKey();
    if(!bkey) return;
    // 实际DOM：<tr data-v-xxx> 无class；操作列 <td class="ctrl">；扣号列 <td class="uid">；序号列 <td class="index">
    var rows=document.querySelectorAll('tr');
    rows.forEach(function(row){
      if(row.querySelector('.ym-sms-assist')) return;
      var opCell=row.querySelector('td.ctrl');
      if(!opCell) return; // 非数据行（无操作列）跳过
      // 从专用列读取扣号和序号
      var uidCell=row.querySelector('td.uid');
      var idxCell=row.querySelector('td.index');
      var qqNum=uidCell?uidCell.textContent.trim():'';
      var seqNum=idxCell?parseInt(idxCell.textContent.trim(),10):0;
      // 注入按钮
      var sep=document.createTextNode('\u00a0');
      var btn=document.createElement('span');
      btn.className='ym-sms-assist';
      btn.textContent='发送短信';
      btn.style.cssText='color:#e6a23c;cursor:pointer;font-size:13px;user-select:none;white-space:nowrap;';
      btn.title='触发短信验证码发送 (mode=8)';
      btn.onclick=function(e){
        e.stopPropagation();
        if(btn._busy) return;
        btn._busy=true;btn.style.opacity='0.5';btn.textContent='查找中...';
        // batch端点用key参数认证，无需Authorization header
        fetch('/api/robot/batch/info?key='+encodeURIComponent(bkey))
          .then(function(r){return r.json();}).then(function(res){
            var robots=(res.data&&res.data.robots)||[];
            var robot=null;
            // 方法1：按扣号(QQ)精确匹配
            if(qqNum){
              robots.forEach(function(r){
                var uld=r.kernel&&(r.kernel.UserLoginData||r.kernel.user_login_data);
                var uin=uld?String(uld.Uin||uld.uin||uld.Uis||uld.uis||''):'';
                if(uin===qqNum) robot=r;
              });
            }
            // 方法2：按序号索引兜底
            if(!robot&&seqNum>=1&&seqNum<=robots.length) robot=robots[robots.length-seqNum]; // info接口返回reverse顺序
            if(!robot&&robots.length>0) robot=robots[0]; // 最终兜底
            if(!robot){btn._busy=false;btn.style.opacity='1';btn.textContent='发送短信';alert('找不到机器人(扣号:'+qqNum+' 序号:'+seqNum+')');return;}
            var robotId=robot.id||robot._id||'';
            btn.textContent='发送中...';
            fetch('/api/robot/login?key='+encodeURIComponent(bkey)+'&robot_id='+encodeURIComponent(robotId)+'&mode=8')
            .then(function(r){return r.json();}).then(function(d){
              btn._busy=false;btn.style.opacity='1';btn.textContent='发送短信';
              if(d.success===true||d.code===200){
                btn.style.color='#67c23a';setTimeout(function(){btn.style.color='#e6a23c';},3000);
              } else {
                btn.style.color='#f56c6c';setTimeout(function(){btn.style.color='#e6a23c';},5000);
                alert('发送失败: '+(d.msg||d.data||JSON.stringify(d)));
              }
            }).catch(function(err){btn._busy=false;btn.style.opacity='1';btn.textContent='发送短信';alert('请求失败: '+err.message);});
          }).catch(function(err){btn._busy=false;btn.style.opacity='1';btn.textContent='发送短信';alert('查询失败: '+err.message);});
      };
      opCell.appendChild(sep);
      opCell.appendChild(btn);
    });
  }

  // =========== MutationObserver 统一监听 ===========
  new MutationObserver(function(){ ensureJoinBtn(); ensureMemberBtn(); ensureBatchAccountBtn(); ensureGroupMsgBtn(); ensureInviteBtn(); injectSmsAssistBtns(); }).observe(document.body,{childList:true,subtree:true});
  setTimeout(function(){ ensureJoinBtn(); ensureMemberBtn(); ensureBatchAccountBtn(); ensureGroupMsgBtn(); ensureInviteBtn(); injectSmsAssistBtns(); }, 500);
})();
`
