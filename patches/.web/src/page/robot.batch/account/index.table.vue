<template>
    <div class="robot-table">
        <tool-table :empty="table.robots.length == 0">
            <template v-slot:head>
                <tr>

                    <th class="index">序号</th>
                    <th class="time">创建时间</th>
                    <th class="proxy_url">代理</th>
                    <th class="proxy_province">代理地区</th>
                    <th class="nick">昵称</th>
                    <th class="mobile">手机号</th>
                    <th class="uid">扣号</th>
                    <th class="pwd">密码</th>
                    <th class="status">状态</th>
                    <th class="msg">提醒</th>
                    <th class="try">自动重置</th>
                    <th class="ctrl">操作</th>
                </tr>
            </template>

            <template v-slot:body v-if="table.status != undefined">
                <tool-transparent v-for="(item, index) in table.robots">
                    <tr v-if="table.page.is_show(index)">

                        <td class="index">
                            <n-text type="primary">{{ index + 1 }}</n-text>
                        </td>
                        <td class="time">
                            {{ UnixToString(ObjectIdTimeUnix(item.id)) }}
                        </td>
                        <td class="proxy_url">
                            <n-text v-if="batch.proxy_urls != undefined">
                                {{ batch.proxy_urls[item.proxy_id] }}
                            </n-text>
                            <n-text v-else>
                                -
                            </n-text>
                        </td>
                        <td class="proxy_province">
                            <n-text v-if="!item.submit?.province || item.submit?.province == ''">
                                -
                            </n-text>
                            <n-text v-else>
                                {{ item.submit.province }}
                            </n-text>
                        </td>
                        <td class="nick">
                            <n-text v-if="item.submit.mobile == 0">
                                -
                            </n-text>
                            <n-text v-else>
                                {{ item.submit.nick }}
                            </n-text>
                        </td>
                        <td class="mobile">
                            <n-text v-if="item.submit.mobile == 0">
                                -
                            </n-text>
                            <n-text v-else>
                                {{ item.submit.mobile }}
                            </n-text>
                        </td>
                        <td class="uid">
                            <n-text v-if="item.submit.uid == 0">
                                -
                            </n-text>
                            <n-text v-else>
                                {{ item.submit.uid }}
                            </n-text>
                        </td>
                        <td class="pwd">
                            <n-text v-if="item.submit.password == ''">
                                -
                            </n-text>
                            <n-text v-else>
                                {{ item.submit.password }}
                            </n-text>
                        </td>
                        <tool-transparent
                            v-if="!table.status[item.id]?.register || table.status[item.id]?.register?.code != 0">
                            <td class="status">
                                <tool-transparent v-if="!table.status[item.id]?.register">
                                    -
                                </tool-transparent>
                                <tool-transparent v-else>
                                    <n-button size="tiny" text
                                        :type="RegisterStatusType(table.status[item.id].register.code)">
                                        {{ ROBOT_REGISTER_STATUS_NAME[table.status[item.id].register.code] ?
                                            ROBOT_REGISTER_STATUS_NAME[table.status[item.id].register.code] :
                                            `未知失败码:${table.status[item.id].register.code}` }}
                                    </n-button>
                                </tool-transparent>
                            </td>
                            <td class="msg">
                                <n-ellipsis expand-trigger="click" line-clamp="2" :tooltip="false">
                                    <tool-transparent v-if="!table.status[item.id]?.register">
                                        -
                                    </tool-transparent>
                                    <n-text v-else type="error">
                                        {{ table.status[item.id]?.register?.value }}
                                    </n-text>
                                </n-ellipsis>
                            </td>
                        </tool-transparent>
                        <tool-transparent v-else>
                            <td class="status">
                                <tool-transparent v-if="table.status[item.id]?.login == undefined">
                                    -
                                </tool-transparent>
                                <tool-transparent v-else-if="
                                    table.status[item.id].login.code == 0 &&
                                    table.status[item.id].renew_secret_key &&
                                    table.status[item.id].renew_online &&
                                    (table.status[item.id].renew_secret_key.code != 0 || table.status[item.id].renew_online.code != 0)
                                ">
                                    <n-button size="tiny" text type="error">
                                        下线
                                    </n-button>
                                </tool-transparent>
                                <tool-transparent v-else>
                                    <n-button size="tiny" text
                                        :type="LoginStatusType(table.status[item.id].login.code)">
                                        {{ ROBOT_LOGIN_STATUS_NAME[table.status[item.id].login.code] ?
                                            ROBOT_LOGIN_STATUS_NAME[table.status[item.id].login.code] :
                                            `未知失败码:${table.status[item.id].login.code}` }}
                                    </n-button>
                                </tool-transparent>
                            </td>
                            <td class="msg">
                                <n-ellipsis expand-trigger="click" line-clamp="2" :tooltip="false">

                                    <tool-transparent v-if="table.status[item.id]?.login == undefined">
                                        -
                                    </tool-transparent>
                                    <tool-transparent
                                        v-else-if="table.status[item.id].login.code == ROBOT_LOGIN_STATUS_ENUM.SUCC">

                                        <n-text type="warning"
                                            v-if="!table.status[item.id].renew_secret_key || !table.status[item.id].renew_online">
                                            上线中.
                                        </n-text>

                                        <n-text style="color:#999;"
                                            v-else-if="table.status[item.id].renew_online.code != 0">
                                            消息续签失败：
                                            <br>
                                            {{ table.status[item.id].renew_online.value }}
                                        </n-text>

                                        <n-text style="color:#999;"
                                            v-else-if="table.status[item.id].renew_secret_key.code != 0">
                                            秘钥续签失败：
                                            <br>
                                            {{ table.status[item.id].renew_secret_key.value }}
                                        </n-text>

                                        <n-text type="success" v-else>
                                            上线成功
                                        </n-text>

                                    </tool-transparent>
                                    <tool-transparent
                                        v-else-if="table.status[item.id].login.code == ROBOT_LOGIN_STATUS_ENUM.SLIDER">
                                        <n-text v-if="3 * 60 - (store.now - table.status[item.id].login.time) > 0"
                                            type="warning">
                                            等待中：{{ Countdown(3 * 60 - (store.now - table.status[item.id].login.time)) }}
                                        </n-text>
                                        <n-text v-else type="error">
                                            已超时: {{ UnixToString(table.status[item.id].login.time) }}
                                        </n-text>
                                    </tool-transparent>
                                    <tool-transparent
                                        v-else-if="table.status[item.id].login.code == ROBOT_LOGIN_STATUS_ENUM.SUP">
                                        <n-text v-if="10 * 60 - (store.now - table.status[item.id].login.time) > 0"
                                            type="warning">
                                            等待中：{{ Countdown(10 * 60 - (store.now - table.status[item.id].login.time))
                                            }}
                                        </n-text>
                                        <n-text v-else type="error">
                                            已超时: {{ UnixToString(table.status[item.id].login.time) }}
                                        </n-text>
                                    </tool-transparent>
                                    <tool-transparent
                                        v-else-if="table.status[item.id].login.code == ROBOT_LOGIN_STATUS_ENUM.SMS">
                                        <n-text v-if="10 * 60 - (store.now - table.status[item.id].login.time) > 0"
                                            type="warning">
                                            等待中：{{ Countdown(10 * 60 - (store.now - table.status[item.id].login.time))
                                            }}
                                        </n-text>
                                        <n-text v-else type="error">
                                            已超时: {{ UnixToString(table.status[item.id].login.time) }}
                                        </n-text>
                                    </tool-transparent>
                                    <tool-transparent v-else>
                                        <n-text style="color:#999;">
                                            {{ table.status[item.id].login.value }}
                                        </n-text>
                                    </tool-transparent>
                                </n-ellipsis>
                            </td>
                        </tool-transparent>
                        <td class="try">
                            <n-text v-if="table.status[item.id]?.login_tryed > 0" type="warning">
                                {{ table.status[item.id]?.login_tryed }}
                            </n-text>
                            <n-text v-else>
                                -
                            </n-text>
                        </td>
                        <td class="ctrl">

                            <tool-transparent
                                v-if="!table.status[item.id]?.register || table.status[item.id]?.register?.code != 0">
                                <div class="func-place" />
                                <n-button text size="tiny" type="info"
                                    @click="batch.SetPassword.view(item)">修改密码</n-button>
                                <div class="func-place-2" />
                                <n-button text size="tiny" type="success"
                                    @click="table.RegisterClear(item.id)">重置注册</n-button>
                                <div class="func-place-2" />
                                <n-button text size="tiny" type="error">删除</n-button>
                                <div class="func-place" />
                            </tool-transparent>

                            <tool-transparent v-else>
                                <div class="func-place" />
                                <n-button text size="tiny" type="info"
                                    @click="batch.SetPassword.view(item)">修改密码</n-button>
                                <div class="func-place-2" />
                                <n-button text size="tiny" type="success"
                                    @click="batch.ProxyReset(item.id)">更换代理</n-button>
                                <div class="func-place-2" />
                                <tool-transparent v-if="table.status[item.id]?.login">
                                    <n-button text size="tiny" type="success"
                                        @click="batch.LoginClear(item.id)">重置登录</n-button>
                                    <div class="func-place-2" />
                                </tool-transparent>
                                <tool-transparent v-if="table.status[item.id]?.login == undefined">
                                    <n-button text size="tiny" type="info" @click="batch.Login(item.id)">登录</n-button>
                                    <div class="func-place-2" />
                                </tool-transparent>
                                <tool-transparent
                                    v-else-if="table.status[item.id].login.code == ROBOT_LOGIN_STATUS_ENUM.SUCC">
                                    <n-text type="info" style="cursor:pointer;font-size:12px;"
                                        @click="batch.Relogin(item.id)">重登</n-text>
                                    <div class="func-place-2" />
                                </tool-transparent>
                                <tool-transparent
                                    v-else-if="table.status[item.id].login.code == ROBOT_LOGIN_STATUS_ENUM.SLIDER">
                                    <n-button text size="tiny" type="info"
                                        @click="table_slider_login.view(item.id, table.status[item.id].login.value)">
                                        本地滑块
                                    </n-button>
                                    <div class="func-place-2" />
                                </tool-transparent>
                                <tool-transparent
                                    v-else-if="table.status[item.id].login.code == ROBOT_LOGIN_STATUS_ENUM.SUP">
                                    <n-popover trigger="click" placement="bottom"
                                        style="border-radius: 6px; padding: 6px !important;">
                                        <template #trigger>
                                            <n-button text size="tiny" type="warning">
                                                短信验证
                                            </n-button>
                                        </template>
                                        <div style="width: 260px;">
                                            <n-button size="small" block type="info"
                                                :loading="table_sup_sms.loading[item.id]"
                                                :disabled="table_sup_sms.sms_sent[item.id]"
                                                @click="table_sup_sms.SendSms(item.id)"
                                                style="margin-bottom: 8px;">
                                                {{ table_sup_sms.sms_sent[item.id] ? '短信已发送' : '发送短信验证码' }}
                                            </n-button>
                                            <n-input-group v-if="table_sup_sms.sms_sent[item.id]">
                                                <n-input size="small"
                                                    v-model:value="table_sup_sms.code_value[item.id]"
                                                    placeholder="输入验证码" style="flex:1;" />
                                                <n-button size="small" type="primary"
                                                    :loading="table_sup_sms.code_loading[item.id]"
                                                    @click="table_sup_sms.SubmitCode(item.id)">
                                                    提交验证码
                                                </n-button>
                                            </n-input-group>
                                        </div>
                                    </n-popover>
                                    <n-popover trigger="manual" placement="bottom" v-model:show="popover_show[item.id]"
                                        style="border-radius: 6px; padding: 6px !important;">
                                        <template #trigger>
                                            <n-button text size="tiny" type="info"
                                                :loading="table_assistant_login.loading_copy[item.id]" @contextmenu="(event) => {
                                                    event.preventDefault()
                                                    if (event.button != 2) return
                                                    table_assistant_login.view(item.id, table.status[item.id].login.value)
                                                    popover_show[item.id] = true
                                                }" @click="(event) => {
                                                    if (event.button != 0) return
                                                    event.preventDefault()
                                                    table_assistant_login.copy(item.id, table.status[item.id].login.value)
                                                }">
                                                复制链接
                                            </n-button>
                                        </template>
                                        <div style="width: 300px;">
                                            <n-input-group style="margin-bottom: 6px;">
                                                <n-input size="small" :value="table_assistant_login.value"
                                                    style="flex:1;" placeholder="获取中." />
                                                <n-button size="small" @click="popover_show[item.id] = false">
                                                    关闭
                                                </n-button>
                                            </n-input-group>
                                            <div style="height: 300px;width: 100%;"
                                                v-if="table_assistant_login.loading[item.id]">
                                                <tool-loading />
                                            </div>
                                            <div class="center" style="height: 300px;width: 300px;"
                                                v-else-if="table_assistant_login.err_msg">
                                                {{ table_assistant_login.err_msg }}
                                            </div>
                                            <tool-transparent v-else>
                                                <vue-qrcode class="qrcode" :value="table_assistant_login.value"
                                                    :color="{ dark: '#000000ff', light: '#ffffffff' }"
                                                    :type="'image/png'" style="width: 100%;" />
                                            </tool-transparent>
                                        </div>
                                    </n-popover>
                                    <div class="func-place-2" />
                                </tool-transparent>
                                <tool-transparent
                                    v-else-if="table.status[item.id].login.code == ROBOT_LOGIN_STATUS_ENUM.SMS || table.status[item.id].login.code == ROBOT_LOGIN_STATUS_ENUM.SMS_FAIL">
                                    <n-popover trigger="click" style="border-radius: 6px; padding: 6px !important;">
                                        <template #trigger>
                                            <n-button text size="tiny" type="info" @click="table_sms_code.view">
                                                填写验证码
                                            </n-button>
                                        </template>
                                        <div style="width: 150px;">
                                            <n-input-group>
                                                <n-button size="tiny" style="width: 48px;">查看</n-button>
                                                <n-button size="tiny" type="primary" secondary style="flex:1;"
                                                    @click="sms_open(item.submit.szfangmm_url)">
                                                    点击跳转
                                                </n-button>
                                            </n-input-group>

                                            <n-input-group style="margin-top: 5px;">
                                                <n-button size="tiny" style="width: 48px;">COM</n-button>
                                                <n-input-number size="tiny" :value="item.submit.szfangmm_com"
                                                    :show-button="false" />
                                            </n-input-group>

                                            <n-input-group style="margin-top: 5px;">
                                                <n-button size="tiny" style="width: 48px;">码</n-button>
                                                <n-input size="tiny" :show-button="false"
                                                    v-model:value="table_sms_code.value"
                                                    :disabled="table_sms_code.loading" />
                                            </n-input-group>

                                            <n-divider title-placement="left" dashed
                                                style="font-size: 12px;margin:5px 0 0 0;color:#999;">
                                            </n-divider>

                                            <n-input-group style="margin-top: 5px;">
                                                <div class="func-place" />
                                                <n-button size="tiny" type="success"
                                                    @click="table_sms_code.Handle(item.id)"
                                                    :loading="table_sms_code.loading">确认</n-button>
                                                <div class="func-place" />
                                            </n-input-group>
                                        </div>
                                    </n-popover>

                                    <div class="func-place-2" />
                                </tool-transparent>
                                <tool-transparent
                                    v-else-if="table.status[item.id].login.code == ROBOT_LOGIN_STATUS_ENUM.SAFE">
                                    <!-- 待完成 -->
                                </tool-transparent>
                                <tool-transparent
                                    v-else-if="table.status[item.id].login.code == ROBOT_LOGIN_STATUS_ENUM.FAIL">
                                    <n-button text size="tiny" type="info">登录</n-button>
                                    <div class="func-place-2" />
                                </tool-transparent>
                                <n-button text size="tiny" type="error" @click="delete_handle(item)">删除</n-button>
                                <div class="func-place" />
                            </tool-transparent>
                        </td>
                    </tr>
                </tool-transparent>
            </template>

        </tool-table>

        <table-slider-register />
        <table-slider-login />
    </div>

</template>

<script setup lang="ts">
import ToolTable from '@/public/tool.table.vue'
import toolTransparent from '@/public/tool.transparent.vue'
import { CopyText, Countdown, ObjectIdTimeUnix, UnixToString } from '@/utils/text';
import { store } from '@/store';
import { ROBOT_LOGIN_STATUS_ENUM, ROBOT_LOGIN_STATUS_NAME, ROBOT_REGISTER_STATUS_ENUM, ROBOT_REGISTER_STATUS_NAME } from '@/types/type.robot.e';
import { onMounted, onUnmounted, ref } from 'vue';
import { delete_handle, table, table_assistant_login, table_sms_code, table_sup_sms, table_slider_login } from '.';
import TableSliderRegister from './index.table.slider.register.vue'
import TableSliderLogin from './index.table.slider.login.vue'
import toolLoading from '@/public/tool.loading.vue';
import { batch } from '..';

let update_switch = true

const robot_update_time = ref(0)

const update_time = async () => {

    if (!update_switch) return

    if (store.view.active) {

        await store.robot_batch.GetStatus(batch.object.key.value, (data) => {
            table.status = data

        }, (err_str) => {
        })
    }

    await batch.Update(true)

    if (robot_update_time.value < batch.object.cache.robot_update_time) {
        robot_update_time.value = Math.round(new Date() as any)
        await batch.Update()
    }

    setTimeout(update_time, 1000)
}

const popover_show = ref({})

onMounted(async () => {
    await update_time()
})

onUnmounted(() => {
    update_switch = false
})

const LoginStatusType = (code: number) => {
    switch (code) {
        case ROBOT_LOGIN_STATUS_ENUM.SUCC:
            return 'success'
        case ROBOT_LOGIN_STATUS_ENUM.SLIDER:
        case ROBOT_LOGIN_STATUS_ENUM.SUP:
        case ROBOT_LOGIN_STATUS_ENUM.SMS:
            return 'warning'
        default:
            return 'error'
    }
}

const RegisterStatusType = (code: number) => {
    switch (code) {
        case ROBOT_REGISTER_STATUS_ENUM.SUCC:
            return 'success'
        case ROBOT_REGISTER_STATUS_ENUM.BUSY:
            return 'warning'
        default:
            return 'error'
    }
}

const sms_open = (url: string) => {
    window.open(url, 'newWindowName', 'width=1600px,height=900px')
}

</script>

<style lang="scss" scoped>
.robot-table {
    height: 100%;

    td {
        user-select: text !important;
    }

    td,
    th {
        flex: 1;
    }

    .select {
        flex: unset;
        width: 40px;
    }

    .index {
        flex: unset;
        min-height: 32px;
        width: 60px;
    }

    .proxy_url,
    .time {
        flex: unset;
        width: 190px;
    }

    .try {
        flex: unset;
        width: 80px;
    }

    .ctrl {
        flex: unset;
        width: 290px;
    }
}

.ellipsis {
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}

:deep() {

    .n-input,
    .n-select {
        * {
            text-align: center;
        }
    }

    .n-base-select-option {
        text-align: center;
    }

    .n-base-select-option__content {
        width: 100%;
        text-align: center;
        font-size: 13px;
        color: #ccc !important;
    }

    .n-time-picker {
        text-align: center;

        .n-input__suffix {
            display: none;
        }
    }
}
</style>
