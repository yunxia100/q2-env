#!/bin/bash
# ============================================================
# 一键部署好友请求功能 - 在服务器上执行
# 用法: bash deploy_all.sh
# ============================================================
set -e
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
cd /opt/ymlink-q2

echo -e "${CYAN}══════════════════════════════════════════${NC}"
echo -e "${CYAN}  部署好友请求功能补丁${NC}"
echo -e "${CYAN}══════════════════════════════════════════${NC}"
echo ""

# ===== 1. 恢复原始 main.go =====
echo -e "${CYAN}[1/5] 恢复原始 main.go...${NC}"
if git diff --name-only 2>/dev/null | grep -q 'apps/server/main.go'; then
    git checkout apps/server/main.go
    echo -e "${GREEN}  ✓ 从git恢复${NC}"
elif [ -f apps/server/main.go.bak ]; then
    cp apps/server/main.go.bak apps/server/main.go
    echo -e "${GREEN}  ✓ 从备份恢复${NC}"
else
    echo -e "${YELLOW}  ! main.go未被修改, 使用当前版本${NC}"
fi

# 备份
cp apps/server/main.go apps/server/main.go.bak.$(date +%s)
echo -e "${GREEN}  ✓ 已备份${NC}"

# ===== 2. 写入 model 补丁文件 =====
echo ""
echo -e "${CYAN}[2/5] 写入 model 补丁...${NC}"
base64 -d << 'MODEL_EOF' > model/patch_friend_notice_model.go
cGFja2FnZSBtb2RlbAoKaW1wb3J0ICgKCSJlbmNvZGluZy9qc29uIgoJImZtdCIKCSJ5bWxpbmstcTIvcGx1Z2luIgopCgovLyA9PT09PT09PT09IOWlveWPi+mAmuefpSAoZHJpdmXlsYLljp/lp4vov5Tlm54pID09PT09PT09PT0KCnR5cGUgUGF0Y2hGcmllbmRNc2cgc3RydWN0IHsKCVZlcnNpb24gaW50ICAgYGpzb246InZlcnNpb24iYAoJTXNnVHlwZSBpbnQgICBganNvbjoibXNnVHlwZSJgCglNc2dTZXEgIGludDY0IGBqc29uOiJtc2dTZXEiYAoJTXNnVGltZSBpbnQ2NCBganNvbjoibXNnVGltZSJgCglSZXFVaW4gIGludDY0IGBqc29uOiJyZXFVaW4iYAoJTXNnICAgICAqc3RydWN0IHsKCQlTdWJUeXBlICAgICAgIGludCAgICBganNvbjoic3ViVHlwZSJgCgkJTXNnVGl0bGUgICAgICBzdHJpbmcgYGpzb246Im1zZ1RpdGxlImAKCQlNc2dEZXNjcmliZSAgIHN0cmluZyBganNvbjoibXNnRGVzY3JpYmUiYAoJCU1zZ0FkZGl0aW9uYWwgc3RyaW5nIGBqc29uOiJtc2dBZGRpdGlvbmFsImAKCQlNc2dTb3VyY2UgICAgIHN0cmluZyBganNvbjoibXNnU291cmNlImAKCQlNc2dEZWNpZGVkICAgIHN0cmluZyBganNvbjoibXNnRGVjaWRlZCJgCgkJU3JjSWQgICAgICAgICBpbnQ2NCAgYGpzb246InNyY0lkImAKCQlTdWJTcmNJZCAgICAgIGludDY0ICBganNvbjoic3ViU3JjSWQiYAoJCVJlbGF0aW9uICAgICAgaW50NjQgIGBqc29uOiJyZWxhdGlvbiJgCgkJUmVxVWluRmFjZWlkICBpbnQ2NCAgYGpzb246InJlcVVpbkZhY2VpZCJgCgkJUmVxVWluTmljayAgICBzdHJpbmcgYGpzb246InJlcVVpbk5pY2siYAoJCU1zZ0RldGFpbCAgICAgc3RyaW5nIGBqc29uOiJtc2dEZXRhaWwiYAoJCVJlcVVpbkdlbmRlciAgaW50NjQgIGBqc29uOiJyZXFVaW5HZW5kZXIiYAoJCVJlcVVpbkFnZSAgICAgaW50NjQgIGBqc29uOiJyZXFVaW5BZ2UiYAoJfSBganNvbjoibXNnImAKfQoKdHlwZSBQYXRjaEZyaWVuZE5vdGljZXNSZXN1bHQgc3RydWN0IHsKCUhlYWQgICAgICAgICAgICAgICBhbnkgICAgICAgICAgICAgICBganNvbjoiaGVhZCJgCglMYXRlc3RGcmllbmRTZXEgICAgaW50NjQgICAgICAgICAgICAgYGpzb246ImxhdGVzdEZyaWVuZFNlcSJgCglMYXRlc3RHcm91cFNlcSAgICAgaW50NjQgICAgICAgICAgICAgYGpzb246ImxhdGVzdEdyb3VwU2VxImAKCUZvbGxvd2luZ0ZyaWVuZFNlcSBpbnQ2NCAgICAgICAgICAgICBganNvbjoiZm9sbG93aW5nRnJpZW5kU2VxImAKCUZyaWVuZE1zZ3MgICAgICAgICBbXSpQYXRjaEZyaWVuZE1zZyBganNvbjoiZnJpZW5kTXNnImAKCU1zZ0Rpc3BsYXkgICAgICAgICBzdHJpbmcgICAgICAgICAgICBganNvbjoibXNnRGlzcGxheSJgCglPdmVyICAgICAgICAgICAgICAgaW50NjQgICAgICAgICAgICAgYGpzb246Im92ZXIiYAp9CgpmdW5jIChyb2JvdCAqUm9ib3QpIFBhdGNoRnJpZW5kTm90aWNlcygpICgqUGF0Y2hGcmllbmROb3RpY2VzUmVzdWx0LCBlcnJvcikgewoJY29kZSwgXywgY29udGVudCwgZXJyIDo9IHJvYm90LkNsaWVudCgpLlBvc3RGb3JtKAoJCSIvZGV2aWNlL1Byb2ZpbGVTZXJ2aWNlLlBiLlJlcVN5c3RlbU1zZ05ldyIsCgkJbWFwW3N0cmluZ11zdHJpbmd7CgkJCSJBY2NlcHQiOiAiYXBwbGljYXRpb24vanNvbiIsCgkJfSwKCQltYXBbc3RyaW5nXXN0cmluZ3sib2JqaWQiOiByb2JvdC5LZXJuZWwuT2JqaWR9LAoJCW1hcFtzdHJpbmddc3RyaW5ne30sCgkpCglpZiBlcnIgIT0gbmlsIHsKCQlyZXR1cm4gbmlsLCBmbXQuRXJyb3JmKCLojrflj5blpb3lj4vpgJrnn6XliJfooajor7fmsYLplJnor686ICV2IiwgZXJyKQoJfQoJaWYgY29kZS1jb2RlJXBsdWdpbi5SRVFVRVNUX1NVQ0NFU1MgIT0gcGx1Z2luLlJFUVVFU1RfU1VDQ0VTUyB7CgkJcmV0dXJuIG5pbCwgZm10LkVycm9yZigiY29kZTogJWQsIGNvbnRlbnQ6ICVzIiwgY29kZSwgc3RyaW5nKGNvbnRlbnQpKQoJfQoJcmVzdWx0IDo9IG5ldyhQYXRjaEZyaWVuZE5vdGljZXNSZXN1bHQpCglpZiBlcnIgPSBqc29uLlVubWFyc2hhbChjb250ZW50LCByZXN1bHQpOyBlcnIgIT0gbmlsIHsKCQlyZXR1cm4gbmlsLCBmbXQuRXJyb3JmKCLojrflj5blpb3lj4vlk43lupTmlbDmja7moLzlvI/lvILluLg6ICVzIiwgc3RyaW5nKGNvbnRlbnQpKQoJfQoJcmV0dXJuIHJlc3VsdCwgbmlsCn0KCi8vID09PT09PT09PT0g6YCa6L+H5aW95Y+L6K+35rGCID09PT09PT09PT0KCnR5cGUgUGF0Y2hGcmllbmRQYXNzUmVzdWx0IHN0cnVjdCB7CglIZWFkIHN0cnVjdCB7CgkJUmVzdWx0ICBpbnQgICAgYGpzb246InJlc3VsdCJgCgkJTXNnRmFpbCBzdHJpbmcgYGpzb246Im1zZ0ZhaWwiYAoJfSBganNvbjoiaGVhZCJgCglNc2dEZXRhaWwgc3RyaW5nIGBqc29uOiJtc2dEZXRhaWwiYAp9CgpmdW5jIChyb2JvdCAqUm9ib3QpIFBhdGNoRnJpZW5kUGFzcyhyZXFVaW4sIHNyY0lkLCBzdWJTcmNJZCBzdHJpbmcpICgqUGF0Y2hGcmllbmRQYXNzUmVzdWx0LCBlcnJvcikgewoJY29kZSwgXywgY29udGVudCwgZXJyIDo9IHJvYm90LkNsaWVudCgpLlBvc3RGb3JtKAoJCSIvZGV2aWNlL1Byb2ZpbGVTZXJ2aWNlLlBiLlJlcVN5c3RlbU1zZ0FjdGlvbi5GcmllbmQiLAoJCW1hcFtzdHJpbmddc3RyaW5newoJCQkiQWNjZXB0IjogImFwcGxpY2F0aW9uL2pzb24iLAoJCX0sCgkJbWFwW3N0cmluZ11zdHJpbmd7Im9iamlkIjogcm9ib3QuS2VybmVsLk9iamlkfSwKCQltYXBbc3RyaW5nXXN0cmluZ3sicmVxVWluIjogcmVxVWluLCAic3JjSWQiOiBzcmNJZCwgInN1YlNyY0lkIjogc3ViU3JjSWR9LAoJKQoJaWYgZXJyICE9IG5pbCB7CgkJcmV0dXJuIG5pbCwgZm10LkVycm9yZigi6YCa6L+H5aW95Y+L6K+35rGC6ZSZ6K+vOiAldiIsIGVycikKCX0KCWlmIGNvZGUtY29kZSVwbHVnaW4uUkVRVUVTVF9TVUNDRVNTICE9IHBsdWdpbi5SRVFVRVNUX1NVQ0NFU1MgewoJCXJldHVybiBuaWwsIGZtdC5FcnJvcmYoImNvZGU6ICVkLCBjb250ZW50OiAlcyIsIGNvZGUsIHN0cmluZyhjb250ZW50KSkKCX0KCXJlc3VsdCA6PSBuZXcoUGF0Y2hGcmllbmRQYXNzUmVzdWx0KQoJaWYgZXJyID0ganNvbi5Vbm1hcnNoYWwoY29udGVudCwgcmVzdWx0KTsgZXJyICE9IG5pbCB7CgkJcmV0dXJuIG5pbCwgZm10LkVycm9yZigi6YCa6L+H5aW95Y+L5ZON5bqU5pWw5o2u5qC85byP5byC5bi4OiAlcyIsIHN0cmluZyhjb250ZW50KSkKCX0KCXJldHVybiByZXN1bHQsIG5pbAp9CgovLyA9PT09PT09PT09IOWJjeerr+aJgOmcgOeahOWTjeW6lOexu+WeiyA9PT09PT09PT09Cgp0eXBlIFBhdGNoRnJpZW5kTm90aWNlSXRlbSBzdHJ1Y3QgewoJTXNnVHlwZSAgICAgICBpbnQgICAgYGpzb246Im1zZ190eXBlImAKCU1zZ1NlcSAgICAgICAgaW50NjQgIGBqc29uOiJtc2dfc2VxImAKCU1zZ1RpbWUgICAgICAgaW50NjQgIGBqc29uOiJtc2dfdGltZSJgCglSZXFVaW4gICAgICAgIGludDY0ICBganNvbjoicmVxX3VpbiJgCglOaWNrICAgICAgICAgIHN0cmluZyBganNvbjoibmljayJgCglHZW5kZXIgICAgICAgIGludDY0ICBganNvbjoiZ2VuZGVyImAKCUFnZSAgICAgICAgICAgaW50NjQgIGBqc29uOiJhZ2UiYAoJU3JjSWQgICAgICAgICBpbnQ2NCAgYGpzb246InNyY19pZCJgCglTdWJTcmNJZCAgICAgIGludDY0ICBganNvbjoic3ViX3NyY19pZCJgCglNc2dUaXRsZSAgICAgIHN0cmluZyBganNvbjoibXNnX3RpdGxlImAKCU1zZ0FkZGl0aW9uYWwgc3RyaW5nIGBqc29uOiJtc2dfYWRkaXRpb25hbCJgCglNc2dTb3VyY2UgICAgIHN0cmluZyBganNvbjoibXNnX3NvdXJjZSJgCglNc2dEZXRhaWwgICAgIHN0cmluZyBganNvbjoibXNnX2RldGFpbCJgCn0KCnR5cGUgUGF0Y2hGcmllbmRQYXNzUmVxdWVzdCBzdHJ1Y3QgewoJUm9ib3RJZCAgc3RyaW5nIGBqc29uOiJyb2JvdF9pZCIgZm9ybToicm9ib3RfaWQiYAoJUmVxVWluICAgaW50NjQgIGBqc29uOiJyZXFfdWluIiBmb3JtOiJyZXFfdWluImAKCVNyY0lkICAgIGludDY0ICBganNvbjoic3JjX2lkIiBmb3JtOiJzcmNfaWQiYAoJU3ViU3JjSWQgaW50NjQgIGBqc29uOiJzdWJfc3JjX2lkIiBmb3JtOiJzdWJfc3JjX2lkImAKfQo=
MODEL_EOF
echo -e "${GREEN}  ✓ model/patch_friend_notice_model.go${NC}"

# ===== 3. 写入 controller 补丁文件 =====
echo ""
echo -e "${CYAN}[3/5] 写入 controller 补丁...${NC}"
base64 -d << 'CTRLER_EOF' > apps/server/ctrler/patch_friend_notice_ctrler.go
cGFja2FnZSBjdHJsZXIKCmltcG9ydCAoCgkiZm10IgoJInN0cmNvbnYiCgkieW1saW5rLXEyL2FwcHMvc2VydmVyL3NlbGYiCgkieW1saW5rLXEyL21vZGVsIgoJInltbGluay1xMi9wbHVnaW4iCgoJImdpdGh1Yi5jb20vZ2luLWdvbmljL2dpbiIKCSJnby5tb25nb2RiLm9yZy9tb25nby1kcml2ZXIvYnNvbi9wcmltaXRpdmUiCikKCi8vIEdFVCAvYXBpL3JvYm90L2ZyaWVuZF9ub3RpY2VzP3JvYm90X2lkPXh4eApmdW5jIChjdHJsZXIgKmN0cmxlcl9yb2JvdCkgRnJpZW5kTm90aWNlcyhjdHggKmdpbi5Db250ZXh0KSB7Cglyb2JvdElkIDo9IGN0eC5RdWVyeSgicm9ib3RfaWQiKQoJaWYgcm9ib3RJZCA9PSAiIiB7CgkJcGx1Z2luLkh0dHBEZWZhdWx0KGN0eCwgcGx1Z2luLlJFUVVFU1RfQkFELCAi5py65Zmo5Lq6SUTlj4LmlbDlv4XpobvkuIrkvKAiLCBuaWwpCgkJcmV0dXJuCgl9CgoJb1JvYm90SWQsIGVyciA6PSBwcmltaXRpdmUuT2JqZWN0SURGcm9tSGV4KHJvYm90SWQpCglpZiBlcnIgIT0gbmlsIHsKCQlwbHVnaW4uSHR0cERlZmF1bHQoY3R4LCBwbHVnaW4uUkVRVUVTVF9CQUQsICLkuIrkvKDnmoTmnLrlmajkurpJROWPguaVsOagvOW8j+mUmeivryIsIG5pbCkKCQlyZXR1cm4KCX0KCglyb2JvdCA6PSBzZWxmLlJvYm90cy5FeGlzdGVkKG9Sb2JvdElkKQoJaWYgcm9ib3QgPT0gbmlsIHsKCQlwbHVnaW4uSHR0cERlZmF1bHQoY3R4LCBwbHVnaW4uUkVRVUVTVF9CQUQsICLmnLrlmajkurrkv6Hmga/kuI3lrZjlnKgiLCBuaWwpCgkJcmV0dXJuCgl9CgoJcmVzdWx0LCBlcnIgOj0gcm9ib3QuUGF0Y2hGcmllbmROb3RpY2VzKCkKCWlmIGVyciAhPSBuaWwgewoJCXBsdWdpbi5IdHRwRGVmYXVsdChjdHgsIHBsdWdpbi5SRVFVRVNUX0JBRCwgZXJyLkVycm9yKCksIG5pbCkKCQlyZXR1cm4KCX0KCgkvLyDovazmjaLkuLrliY3nq6/miYDpnIDmoLzlvI8KCWl0ZW1zIDo9IG1ha2UoW10qbW9kZWwuUGF0Y2hGcmllbmROb3RpY2VJdGVtLCAwKQoJZm9yIF8sIG1zZyA6PSByYW5nZSByZXN1bHQuRnJpZW5kTXNncyB7CgkJaWYgbXNnLk1zZyA9PSBuaWwgewoJCQljb250aW51ZQoJCX0KCQlpdGVtcyA9IGFwcGVuZChpdGVtcywgJm1vZGVsLlBhdGNoRnJpZW5kTm90aWNlSXRlbXsKCQkJTXNnVHlwZTogICAgICAgbXNnLk1zZ1R5cGUsCgkJCU1zZ1NlcTogICAgICAgIG1zZy5Nc2dTZXEsCgkJCU1zZ1RpbWU6ICAgICAgIG1zZy5Nc2dUaW1lLAoJCQlSZXFVaW46ICAgICAgICBtc2cuUmVxVWluLAoJCQlOaWNrOiAgICAgICAgICBtc2cuTXNnLlJlcVVpbk5pY2ssCgkJCUdlbmRlcjogICAgICAgIG1zZy5Nc2cuUmVxVWluR2VuZGVyLAoJCQlBZ2U6ICAgICAgICAgICBtc2cuTXNnLlJlcVVpbkFnZSwKCQkJU3JjSWQ6ICAgICAgICAgbXNnLk1zZy5TcmNJZCwKCQkJU3ViU3JjSWQ6ICAgICAgbXNnLk1zZy5TdWJTcmNJZCwKCQkJTXNnVGl0bGU6ICAgICAgbXNnLk1zZy5Nc2dUaXRsZSwKCQkJTXNnQWRkaXRpb25hbDogbXNnLk1zZy5Nc2dBZGRpdGlvbmFsLAoJCQlNc2dTb3VyY2U6ICAgICBtc2cuTXNnLk1zZ1NvdXJjZSwKCQkJTXNnRGV0YWlsOiAgICAgbXNnLk1zZy5Nc2dEZXRhaWwsCgkJfSkKCX0KCglwbHVnaW4uSHR0cFN1Y2Nlc3MoY3R4LCBpdGVtcykKfQoKLy8gUE9TVCAvYXBpL3JvYm90L2ZyaWVuZF9wYXNzCmZ1bmMgKGN0cmxlciAqY3RybGVyX3JvYm90KSBGcmllbmRQYXNzKGN0eCAqZ2luLkNvbnRleHQpIHsKCXZhciByZXF1ZXN0IG1vZGVsLlBhdGNoRnJpZW5kUGFzc1JlcXVlc3QKCWlmIGVyciA6PSBjdHguU2hvdWxkQmluZCgmcmVxdWVzdCk7IGVyciAhPSBuaWwgewoJCXBsdWdpbi5IdHRwRGVmYXVsdChjdHgsIHBsdWdpbi5SRVFVRVNUX0JBRCwgIuivt+axguWPguaVsOmUmeivryIsIG5pbCkKCQlyZXR1cm4KCX0KCglpZiByZXF1ZXN0LlJvYm90SWQgPT0gIiIgewoJCXBsdWdpbi5IdHRwRGVmYXVsdChjdHgsIHBsdWdpbi5SRVFVRVNUX0JBRCwgIuacuuWZqOS6uklE5Y+C5pWw5b+F6aG75LiK5LygIiwgbmlsKQoJCXJldHVybgoJfQoKCW9Sb2JvdElkLCBlcnIgOj0gcHJpbWl0aXZlLk9iamVjdElERnJvbUhleChyZXF1ZXN0LlJvYm90SWQpCglpZiBlcnIgIT0gbmlsIHsKCQlwbHVnaW4uSHR0cERlZmF1bHQoY3R4LCBwbHVnaW4uUkVRVUVTVF9CQUQsICLor7fmsYLmnLrlmajkurpJROWPguaVsOagvOW8j+mUmeivryIsIG5pbCkKCQlyZXR1cm4KCX0KCglyb2JvdCA6PSBzZWxmLlJvYm90cy5FeGlzdGVkKG9Sb2JvdElkKQoJaWYgcm9ib3QgPT0gbmlsIHsKCQlwbHVnaW4uSHR0cERlZmF1bHQoY3R4LCBwbHVnaW4uUkVRVUVTVF9CQUQsICLmnLrlmajkurrkv6Hmga/kuI3lrZjlnKgiLCBuaWwpCgkJcmV0dXJuCgl9CgoJcmVzdWx0LCBlcnIgOj0gcm9ib3QuUGF0Y2hGcmllbmRQYXNzKAoJCXN0cmNvbnYuRm9ybWF0SW50KHJlcXVlc3QuUmVxVWluLCAxMCksCgkJc3RyY29udi5Gb3JtYXRJbnQocmVxdWVzdC5TcmNJZCwgMTApLAoJCXN0cmNvbnYuRm9ybWF0SW50KHJlcXVlc3QuU3ViU3JjSWQsIDEwKSwKCSkKCWlmIGVyciAhPSBuaWwgewoJCXBsdWdpbi5IdHRwRGVmYXVsdChjdHgsIHBsdWdpbi5SRVFVRVNUX0JBRCwgZXJyLkVycm9yKCksIG5pbCkKCQlyZXR1cm4KCX0KCglpZiByZXN1bHQuSGVhZC5SZXN1bHQgPT0gLTEgewoJCXBsdWdpbi5IdHRwRGVmYXVsdChjdHgsIHBsdWdpbi5SRVFVRVNUX0JBRCwgZm10LlNwcmludGYoIumAmui/h+WlveWPi+WTjeW6lOmUmeivryxjb2RlOiVkLG1lc3NhZ2U6JXMiLCByZXN1bHQuSGVhZC5SZXN1bHQsIHJlc3VsdC5IZWFkLk1zZ0ZhaWwpLCBuaWwpCgkJcmV0dXJuCgl9CgoJcGx1Z2luLkh0dHBTdWNjZXNzKGN0eCwgcGx1Z2luLkJzb257fSkKfQo=
CTRLER_EOF
echo -e "${GREEN}  ✓ apps/server/ctrler/patch_friend_notice_ctrler.go${NC}"

# ===== 4. 添加路由到 main.go =====
echo ""
echo -e "${CYAN}[4/5] 添加路由到 main.go...${NC}"
if grep -q 'friend_notices' apps/server/main.go; then
    echo -e "${YELLOW}  ! 路由已存在, 跳过${NC}"
else
    # 在 ROBOT.GET("/set_password" 行后添加两个路由
    sed -i '/ROBOT.GET("\/set_password"/a\\t\t\tROBOT.GET("/friend_notices", ctrler.Robot.FriendNotices)\n\t\t\tROBOT.POST("/friend_pass", ctrler.Robot.FriendPass)' apps/server/main.go
    if grep -q 'friend_notices' apps/server/main.go; then
        echo -e "${GREEN}  ✓ 路由添加成功${NC}"
    else
        echo -e "${RED}  ✗ 路由添加失败!${NC}"
        echo -e "${YELLOW}  尝试在 LABEL 组之前添加...${NC}"
        # 备选: 在 LABEL := ROBOT.Group("/label") 之前添加
        sed -i '/LABEL := ROBOT.Group/i\\t\t\tROBOT.GET("/friend_notices", ctrler.Robot.FriendNotices)\n\t\t\tROBOT.POST("/friend_pass", ctrler.Robot.FriendPass)\n' apps/server/main.go
        if grep -q 'friend_notices' apps/server/main.go; then
            echo -e "${GREEN}  ✓ 路由添加成功 (备选位置)${NC}"
        else
            echo -e "${RED}  ✗ 无法自动添加路由! 请手动添加${NC}"
            exit 1
        fi
    fi
fi

# 验证
echo ""
echo -e "${CYAN}[验证] 检查所有文件...${NC}"
echo -n "  model: "; [ -f model/patch_friend_notice_model.go ] && echo -e "${GREEN}OK${NC}" || echo -e "${RED}MISSING${NC}"
echo -n "  ctrler: "; [ -f apps/server/ctrler/patch_friend_notice_ctrler.go ] && echo -e "${GREEN}OK${NC}" || echo -e "${RED}MISSING${NC}"
echo -n "  routes: "; grep -q 'friend_notices' apps/server/main.go && echo -e "${GREEN}OK${NC}" || echo -e "${RED}MISSING${NC}"

# ===== 5. 重建Docker =====
echo ""
echo -e "${CYAN}[5/5] 重建并重启 Docker server 容器...${NC}"
docker compose build server 2>&1 | tail -5
docker compose up -d server
echo ""
echo -e "${GREEN}══════════════════════════════════════════${NC}"
echo -e "${GREEN}  部署完成! 请测试好友请求功能${NC}"
echo -e "${GREEN}══════════════════════════════════════════${NC}"
