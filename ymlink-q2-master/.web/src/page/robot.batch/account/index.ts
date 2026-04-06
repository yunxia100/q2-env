import { store } from "@/store"
import { computed, reactive } from "vue"
import axios from "axios"
import { createGlobalState, useStorage } from "@vueuse/core"
import { ROBOT_LOGIN_MODE_ENUM, ROBOT_LOGIN_STATUS_ENUM, ROBOT_LOGIN_STATUS_OPTIONS } from "@/types/type.robot.e"
import { CopyText, GetHttpText, GetRandomNumber, GetRandomPassword } from "@/utils/text"
import { batch } from ".."
import qrcode from 'qrcode'

export const table = reactive({
    filter: <{ [object: string]: TableFilterType }>{
        "uid": {
            title: '号码',
            type: 'input',
            width: '150px',
            disabled: computed(() => {
            }),
            value: '',
            list: computed(() => {
            }),
            select: () => {
            },
            clear: () => {
                table.filter['uid'].value = ''
            },
            handle: (item: RobotType) => {
                if (table.filter['uid'].value == '') return true
                if (!item.kernel.UserLoginData.Uis.includes(table.filter['uid'].value)) return
                return true
            },
        },
        "proxy": {
            title: '代理状态',
            type: 'select',
            width: '170px',
            disabled: computed(() => {
            }),
            value: -1,
            list: computed(() => {
                return [
                    { label: '全部', value: -1 },
                    { label: '有', value: 1 },
                    { label: '无', value: 0 },
                ]
            }),
            select: () => {
            },
            clear: () => {
                table.filter['proxy'].value = -1
            },
            handle: (item: RobotType) => {
                if (table.filter['proxy'].value == -1) return true
                if (table.filter['proxy'].value == 1 && batch.proxy_urls[item.proxy_id]) return true
                if (table.filter['proxy'].value == 0 && !batch.proxy_urls[item.proxy_id]) return true
                return false
            },
        },
        "status.code": {
            title: '登录状态',
            type: 'select',
            width: '170px',
            disabled: computed(() => {
            }),
            value: -1,
            list: computed(() => {
                return [{ label: '全部', value: -1 }, ...ROBOT_LOGIN_STATUS_OPTIONS, { label: '下线', value: -2 }]
            }),
            select: () => {
            },
            clear: () => {
                table.filter['uid'].value = -1
            },
            handle: (item: RobotType) => {
                if (!table.status) return false
                if (table.filter['status.code'].value == -1) return true
                if (table.filter['status.code'].value == -2) {
                    if (!table.status[item.id].login) return false
                    if (table.status[item.id].login.code != 0) return false
                    if (table.status[item.id].renew_secret_key && table.status[item.id].renew_secret_key.code == 0 &&
                        table.status[item.id].renew_online && table.status[item.id].renew_online.code == 0) return false
                } else {
                    if (!table.status[item.id].login) return false
                    if (table.status[item.id].login.code != table.filter['status.code'].value) return false
                }
                return true
            },
        },
    },

    page: {
        index: 1,
        size: 100,
        sizes: [15, 30, 50, 100, 1000],
        count: computed((): number => {
            const count = Math.ceil(table.robots.length / table.page.size)
            if (table.page.index > count) table.page.index = count
            if (table.page.index < 1) table.page.index = 1
            return count
        }),
        is_show: (index: number): boolean => {
            if (index < (table.page.index - 1) * table.page.size) return false
            if (index >= table.page.index * table.page.size) return false
            return true
        },
    },

    robots: computed((): RobotType[] => {
        const list = []

        for (const item of batch.robots) {
            let result = true
            for (const object in table.filter) if (!table.filter[object].handle(item)) {
                result = false
                break
            }
            if (result) list.push(item)
        }

        return list
    }),

    proxy_url: computed((): string => {
        return batch.proxy ? `socks5://${batch.proxy.config.username}:${batch.proxy.config.password}@${batch.proxy.config.ip}:${batch.proxy.config.port}` : ''
    }),

    status: <{ [id: string]: RobotStatusType }>(undefined),

    RegisterClear: async (robot_id: string) => {
        const dialog = window['$dialog'].success({
            title: '重置注册',
            content: '即将清空机器人的注册状态，此过程不可逆。',
            positiveText: '确定',
            negativeText: '取消',
            maskClosable: false,
            onPositiveClick: async () => {
                dialog.loading = true
                await store.robot.RegisterClear(batch.object.key.value, robot_id, () => {
                    window['$message'].success('重置注册成功')
                })
                dialog.loading = false
            },
        })
    },

    LoginClearAll: async () => {

        const robot_ids = []

        for (const index in table.robots) {
            if (!table.page.is_show(Number(index))) continue
            const status = table.status[table.robots[index].id]
            if (!status || !status.login) continue
            if (status.login.code == 0) continue
            robot_ids.push(table.robots[index].id)
        }

        const dialog = window['$dialog'].success({
            title: '批量重置登录状态',
            content: `当前选择数量：${robot_ids.length}，此过程不可逆！`,
            positiveText: '确定',
            negativeText: '取消',
            maskClosable: false,
            onPositiveClick: async () => {
                dialog.loading = true
                let succ = 0
                for (const robot_id of robot_ids) {
                    await store.robot.LoginClear(batch.object.key.value, robot_id, () => {
                        succ++
                    })
                }
                window.$message.success(`总数：${robot_ids.length}，重置成功数：${succ}。`)
                dialog.loading = false
            },
        })
    },

    ProxyResetAll: async () => {

        const robot_ids = []

        for (const index in table.robots) {
            if (!table.page.is_show(Number(index))) continue
            robot_ids.push(table.robots[index].id)
        }

        const dialog = window['$dialog'].success({
            title: '批量重置代理',
            content: `当前选择数量：${robot_ids.length}，此过程不可逆！`,
            positiveText: '确定',
            negativeText: '取消',
            maskClosable: false,
            onPositiveClick: async () => {
                dialog.loading = true
                let succ = 0
                for (const robot_id of robot_ids) {
                    await store.robot.ProxyReset(batch.object.key.value, robot_id, () => {
                        succ++
                    })
                }
                window.$message.success(`总数：${robot_ids.length}，重置成功数：${succ}。`)
                dialog.loading = false
            },
        })
    },

    AssistantLoginQrcode: {
        loading: false,
        number: computed(() => {
            let number = 0
            for (const id in table.status) {
                if (table.status[id]?.login?.code == ROBOT_LOGIN_STATUS_ENUM.SUP) {
                    number++
                }
            }
            return number
        }),
        Handle: async () => {
            for (const item of table.robots) {
                const status = table.status[item.id]
                if (!status || !status.login || status.login.code != ROBOT_LOGIN_STATUS_ENUM.SUP) continue

                await store.robot.GetLoginAssistantQrcode(batch.object.key.value, item.id, status.login.value, async (result) => {
                    try {
                        const image_url = await qrcode.toDataURL(result)
                        const link = document.createElement('a')
                        link.href = image_url
                        link.download = `${item.submit.uid}.png`
                        document.body.appendChild(link)
                        link.click()
                        document.body.removeChild(link)

                    } catch (err_msg) {
                        window.$message.error(`扣号：${item.submit.uid}，生成二维码失败：${err_msg}。`)
                    }
                }, (err_msg) => {
                    window.$message.error(`扣号：${item.submit.uid}，生成二维码失败：${err_msg}。`)
                })
            }
        },
    },
})

export const table_batch = reactive({
    show: false,
    view: () => {
        table_batch.show = true
    },
    close: () => {
        table_batch.show = false
    },
    create: async () => {
    },
    update: async () => {
    },
})

export const table_login = reactive({
    show: false,
    value: {
        province: undefined,
        account_str: '',
        password: '',
        accounts: <{ uid: string, password: string, status: string }[]>[],
        account_update: (need_create: boolean) => {
            const map: { [uid: string]: string } = {}
            for (const item of table_login.value.account_str.split(/\n/)) {
                const split = item.replace(' ', '').split('----')
                if (split.length <= 0) continue
                if (5 > split[0].length || split[0].length > 12) continue
                if (split.length == 1) {
                    if (table_login.value.password.length == 0) {
                        window.$message.error('请输入公共密码')
                        return
                    }
                    map[split[0]] = table_login.value.password
                } else {
                    map[split[0]] = split[1]
                }
            }
            for (const uid in map) table_login.value.accounts.push({ uid, password: map[uid], status: '' })
            table_login.value.account_str = ''
            if (need_create) table_login.Handle()
        },
    },
    loading: false,
    view: () => {
        table_login.show = true
    },
    close: () => {
        table_login.show = false
    },
    clean: () => {
        table_login.value.province = undefined
        table_login.value.account_str = ''
        table_login.value.accounts = []
    },
    cleanSucc: () => {
        for (let idx = 0; idx < table_login.value.accounts.length; idx++) {
            if (table_login.value.accounts[idx].status == '创建成功') {
                table_login.value.accounts.splice(idx, 1)
                idx--
            }
        }
    },
    Handle: async () => {
        table_login.loading = true
        let succ = 0
        let limit = undefined
        let now_wait = 0
        let limit_used = false

        if (table_login.value.accounts.length == 0) {
            return window.$message.error('列表为空')
        }

        for (let idx = 0; idx < table_login.value.accounts.length; idx++) {

            if (limit != undefined && succ >= limit) {
                limit_used = true
                break
            }

            const account = table_login.value.accounts[idx]

            if (account.status == '操作成功') continue

            await store.robot.Create(batch.object.key.value, {
                uid: Number(account.uid),
                password: account.password,
                province: table_login.value.province,
            },
                () => {
                    account.status = '创建成功'
                    succ++
                }, (err_msg) => {
                    account.status = err_msg
                })
        }

        if (limit_used) {
            window.$message.warning('创建数量：' + succ + (limit_used ? `，限制登录！` : ''))
        } else {
            window.$message.success('创建数量：' + succ + (limit_used ? `，限制登录！` : ''))
        }
        table_login.loading = false
    },
})

export const table_login_sms = reactive({
    show: false,
    value: {
        province: undefined,
        account_str: '',
        password: '',
        accounts: <{ uid: number, password: string, com: number, url: string, province: string, status: string }[]>[],
        account_update: async (need_create: boolean) => {
            const map: { [uid: string]: string } = {}
            for (const item of table_login_sms.value.account_str.split(/\n/)) {
                const split = item.replace(' ', '').split('----')

                let uid: number, password: string, com: number, province: string, url: string

                switch (split.length) {
                    case 4:
                        uid = Number(split[0])
                        password = split[1]
                        com = Number(split[2].split(',')[0].replaceAll("COM", ""))
                        url = split[3]
                        break
                    case 5:
                        uid = Number(split[0])
                        password = split[1]
                        com = Number(split[3].split(',')[0].replaceAll("COM", ""))
                        url = split[4]
                        break
                    case 6:
                        uid = Number(split[0])
                        password = split[1]
                        com = Number(split[3].split(',')[0].replaceAll("COM", ""))
                        province = split[4].replaceAll("IP:", "")
                        url = split[5]
                        break
                }

                if (uid <= 0 || isNaN(uid)) continue
                if (password.length < 6) continue
                if (isNaN(com) || com == 0) continue
                if (!url.includes("http")) continue

                if (province) await store.proxy.GetRegion(province, (country, _province, city) => {
                    province = _province
                })

                table_login_sms.value.accounts.push({ uid, password, com, url, province, status: '' })
            }
            table_login_sms.value.account_str = ''
            if (need_create) table_login_sms.Handle()
        },
    },
    loading: false,
    view: () => {
        table_login_sms.show = true
    },
    close: () => {
        table_login_sms.show = false
    },
    clean: () => {
        table_login_sms.value.province = undefined
        table_login_sms.value.account_str = ''
        table_login_sms.value.accounts = []
    },
    cleanSucc: () => {
        for (let idx = 0; idx < table_login_sms.value.accounts.length; idx++) {
            if (table_login_sms.value.accounts[idx].status == '创建成功') {
                table_login_sms.value.accounts.splice(idx, 1)
                idx--
            }
        }
    },
    Handle: async () => {
        table_login_sms.loading = true
        let succ = 0
        let limit = undefined
        let now_wait = 0
        let limit_used = false

        if (table_login_sms.value.accounts.length == 0) {
            return window.$message.error('列表为空')
        }

        for (let idx = 0; idx < table_login_sms.value.accounts.length; idx++) {

            if (limit != undefined && succ >= limit) {
                limit_used = true
                break
            }

            const account = table_login_sms.value.accounts[idx]

            if (account.status == '创建成功') continue

            account.status = "执行中."

            await store.robot.Create(batch.object.key.value, {
                province: account.province,
                uid: Number(account.uid),
                password: account.password,
                szfangmm_url: account.url,
                szfangmm_com: account.com,
            },
                () => {
                    account.status = '创建成功'
                    succ++
                }, (err_msg) => {
                    account.status = err_msg
                })
        }

        if (limit_used) {
            window.$message.warning('创建数量：' + succ + (limit_used ? `，限制登录！` : ''))
        } else {
            window.$message.success('创建数量：' + succ + (limit_used ? `，限制登录！` : ''))
        }
        table_login_sms.loading = false
    },
})

export const table_register = reactive({
    show: false,
    value: {
        account_str: '',
        accounts: <{ mobile: number, nick: string, password: string, status: string }[]>[],
        account_update: async (need_create: boolean) => {

            if (!batch.nick_material) return window.$message.error('昵称用文字素材不存在')

            const map: { [uid: number]: string } = {}

            for (const item of table_register.value.account_str.split(/\n/)) {
                if (!/^(\+?d{1,4}?)?[1-9]\d{10}$/.test(item)) continue
                map[item] = GetRandomPassword()
            }

            for (const mobile in map) {

                const nick_file_name = batch.nick_material.file_names[GetRandomNumber(0, batch.nick_material.file_names.length - 1)]

                let nick = ''
                const baseUrl = import.meta.env.VITE_API_URL
                await GetHttpText(
                    `${baseUrl}/file/material/${batch.nick_material.user_id}/${batch.nick_material.mode}/${batch.nick_material.name}/${nick_file_name}`,
                    (text) => {
                        nick = text
                    })

                table_register.value.accounts.push({
                    mobile: Number(mobile),
                    password: map[mobile],
                    nick: nick,
                    status: '',
                })
            }

            table_register.value.account_str = ''

            if (need_create) table_register.Handle()
        },
    },
    loading: false,
    view: () => {
        table_register.show = true
    },
    close: () => {
        table_register.show = false
    },
    clean: () => {
        table_register.value.account_str = ''
        table_register.value.accounts = []
    },
    cleanSucc: () => {
        for (let idx = 0; idx < table_register.value.accounts.length; idx++) {
            if (table_register.value.accounts[idx].status == '创建成功') {
                table_register.value.accounts.splice(idx, 1)
                idx--
            }
        }
    },
    Handle: async () => {
        table_register.loading = true
        let succ = 0
        let limit = undefined
        let now_wait = 0
        let limit_used = false

        if (table_register.value.accounts.length == 0) {
            return window.$message.error('列表为空')
        }

        for (let idx = 0; idx < table_register.value.accounts.length; idx++) {

            if (limit != undefined && succ >= limit) {
                limit_used = true
                break
            }

            const account = table_register.value.accounts[idx]

            if (account.status == '创建成功') continue

            await store.robot.Create(batch.object.key.value, {
                mobile: Number(account.mobile),
                nick: account.nick,
                password: account.password,
            }, () => {
                account.status = '创建成功'
                succ++
            }, (err_msg) => {
                account.status = err_msg
            })
        }

        if (limit_used) {
            window.$message.warning('创建数量：' + succ + (limit_used ? `，限制登录！` : ''))
        } else {
            window.$message.success('创建数量：' + succ + (limit_used ? `，限制登录！` : ''))
        }
        table_register.loading = false
    },
})

export const table_assistant_login = reactive({
    loading: {},
    loading_copy: {},
    value: '',
    err_msg: '',
    view: async (robot_id: string, link: string) => {
        table_assistant_login.value = ''
        table_assistant_login.err_msg = undefined
        table_assistant_login.loading[robot_id] = true
        await store.robot.GetLoginAssistantQrcode(batch.object.key.value, robot_id, link, (result) => {
            table_assistant_login.value = result
        }, (err_msg) => {
            table_assistant_login.err_msg = err_msg
        })
        table_assistant_login.loading[robot_id] = false
    },
    copy: async (robot_id: string, link: string) => {
        table_assistant_login.value = ''
        table_assistant_login.err_msg = undefined
        table_assistant_login.loading_copy[robot_id] = true
        await store.robot.GetLoginAssistantQrcode(batch.object.key.value, robot_id, link, (result) => {
            table_assistant_login.value = result
            CopyText(table_assistant_login.value, '已复制')
        })
        table_assistant_login.loading_copy[robot_id] = false
    },
    close: () => {
    },
})

export const table_slider_login = reactive({
    show: false,
    loading: false,
    robot_id: '',
    value: '',
    view: (robot_id: string, url: string) => {
        table_slider_login.robot_id = robot_id
        table_slider_login.value = url
        table_slider_login.show = true
    },
    close: () => {
        table_slider_login.show = false
    },
})

export const table_slider_register = reactive({
    show: false,
    loading: false,
    robot_id: '',
    value: '',
    view: (robot_id: string, url: string) => {
        table_slider_register.robot_id = robot_id
        table_slider_register.value = url
        table_slider_register.show = true
    },
    close: () => {
        table_slider_register.show = false
    },
})

export const table_mobile = reactive({
    show: false,
    loading: false,
    robot_id: '',
    list: <MobileType[]>[],
    view: async (robot_id: string) => {
        table_mobile.robot_id = robot_id
        store.user.MobileFetch((list) => {
            table_mobile.list = list
        })
        table_mobile.show = true
    },
    close: () => {
        table_mobile.show = false
    },
})

export const table_outside_phone_register = reactive({
    show: false,
    loading: false,
    view: () => {
        table_outside_phone_register.show = true
    },
    close: () => {
        table_outside_phone_register.show = false
    },
})

export const table_sms_code = reactive({
    show: false,
    loading: false,
    value: "",
    view: async () => {
        table_sms_code.value = undefined
        table_sms_code.show = true
    },
    close: () => {
        table_sms_code.show = false
    },
    Handle: async (robot_id: string) => {
        if (table_sms_code.value.length != 6) return
        table_sms_code.loading = true
        await store.robot.Login(batch.object.key.value, robot_id, ROBOT_LOGIN_MODE_ENUM.SMS_CODE, table_sms_code.value, () => {
            window['$message'].success('请求成功')
            table_sms_code.close()
        })
        table_sms_code.loading = false
    },
})

// --- SUP 辅助验证短信 ---

export const table_sup_sms = reactive({
    loading: <{ [id: string]: boolean }>{},
    sms_sent: <{ [id: string]: boolean }>{},
    code_value: <{ [id: string]: string }>{},
    code_loading: <{ [id: string]: boolean }>{},
    // 发送短信验证码
    SendSms: async (robot_id: string) => {
        table_sup_sms.loading[robot_id] = true
        await store.robot.Login(batch.object.key.value, robot_id, ROBOT_LOGIN_MODE_ENUM.SMS_GET, '', () => {
            window['$message'].success('短信已发送，请查看手机')
            table_sup_sms.sms_sent[robot_id] = true
        }, (err_msg: string) => {
            window['$message'].error('发送失败: ' + err_msg)
        })
        table_sup_sms.loading[robot_id] = false
    },
    // 提交短信验证码
    SubmitCode: async (robot_id: string) => {
        const code = table_sup_sms.code_value[robot_id]
        if (!code || code.length < 4) {
            window['$message'].warning('请输入验证码')
            return
        }
        table_sup_sms.code_loading[robot_id] = true
        await store.robot.Login(batch.object.key.value, robot_id, ROBOT_LOGIN_MODE_ENUM.SMS_CODE, code, () => {
            window['$message'].success('验证成功')
            table_sup_sms.code_value[robot_id] = ''
            table_sup_sms.sms_sent[robot_id] = false
        }, (err_msg: string) => {
            window['$message'].error('验证失败: ' + err_msg)
        })
        table_sup_sms.code_loading[robot_id] = false
    },
})

// ---

const mevrtState = createGlobalState(() => useStorage(
    'ymlink-q2-robot-batch-mevrt',
    {
        info: {
            nickname: '',
            token: '',
        },
        config: {
            switch: false,
            robot_password: '',
        },
        list: <{ uid: string, used: boolean, upload: boolean }[]>[],
    },
))

export const account_brand_mevrt = reactive({
    show: false,
    loading: false,
    account: {
        uid: '',
        password: '',
    },
    value: mevrtState().value,
    view: () => {
        account_brand_mevrt.show = true
    },
    close: () => {
        account_brand_mevrt.show = false
    },
    Login: async () => {
        account_brand_mevrt.loading = true

        try {
            const baseUrl = import.meta.env.VITE_API_URL
            const response = await axios.get(`${baseUrl}/api/debug/proxy`, {
                headers: {
                    "proxy_url": table.proxy_url,
                    "target_url": 'http://api.xiangshang666.com/user/login',
                },
                params: {
                    uid: account_brand_mevrt.account.uid,
                    password: account_brand_mevrt.account.password,
                }
            });

            if (response.data?.code === 200) {
                window.$message.success('登录成功')
                account_brand_mevrt.value.info.nickname = response.data.result.nickname
                account_brand_mevrt.value.info.token = response.data.result.token
            } else {
                window.$message.error(response.data.msg)
            }

        } catch (error) {
            console.error('请求出错:', error)
        }

        account_brand_mevrt.loading = false
    },
    Loginout: () => {
        account_brand_mevrt.value.info.token = ''
        account_brand_mevrt.value.info.nickname = ''
        account_brand_mevrt.value.config.switch = false
        account_brand_mevrt.value.config.robot_password = ''
    },
    count: 0,
    total: 0,
    GetAccount: async () => {
        if (!account_brand_mevrt.value.config.switch) return
        let uid = undefined

        account_brand_mevrt.total++

        try {
            const baseUrl = import.meta.env.VITE_API_URL
            const response = await axios.get(`${baseUrl}/api/debug/proxy`, {
                headers: {
                    "proxy_url": table.proxy_url,
                    "target_url": 'http://api.xiangshang666.com/account/getAccount',
                },
                params: {
                    token: account_brand_mevrt.value.info.token,
                }
            });

            console.log("取号：", JSON.stringify(response.data, null, 2));

            if (response.data?.code === 200 && response.data.result && response.data?.result.acount) {
                uid = response.data.result.acount
                window.$message.success(`取号成功：${response.data?.result.acount}`)
                account_brand_mevrt.count++
            }

        } catch (error) {
        }

        if (uid) account_brand_mevrt.value.list.push({ uid, used: false, upload: false })
    },
    CreateRobot: async () => {
        let account = <typeof account_brand_mevrt.value.list[0]>undefined

        for (const item of account_brand_mevrt.value.list) if (!item.used) {
            account = item
            break
        }
        if (!account) return

        await store.robot.Create(batch.object.key.value, {
            uid: Number(account.uid),
            password: account_brand_mevrt.value.config.robot_password,
        }, () => {
            account.used = true
        })
    },
    UpdateAccountStatus: async () => {

        for (const account of account_brand_mevrt.value.list) {
            if (account.upload) continue

            for (const id in table.status) {

                let robot_uid = undefined

                for (const item of table.robots) if (item.id == id) {
                    robot_uid = item.kernel.UserLoginData.Uin
                    break
                }

                if (robot_uid == account.uid && table.status[id]) switch (table.status[id].login.code) {

                    case ROBOT_LOGIN_STATUS_ENUM.SUCC: {

                        try {
                            const baseUrl = import.meta.env.VITE_API_URL
                            const response = await axios.get(`${baseUrl}/api/debug/proxy`, {
                                headers: {
                                    "proxy_url": table.proxy_url,
                                    "target_url": 'http://api.xiangshang666.com/account/updateAccountStatus',
                                },
                                params: {
                                    token: account_brand_mevrt.value.info.token,
                                    account: account.uid,
                                    status: 2,
                                    source: 0,
                                    remark: '使用成功',
                                }
                            });

                            console.log("取号设置状态.2：", JSON.stringify(response.data, null, 2));

                            if (response.data.code === 200) {
                                account.upload = true
                            }

                        } catch (error) { }

                    } break

                    case ROBOT_LOGIN_STATUS_ENUM.SLIDER: {

                    } break

                    default: {

                        try {
                            const baseUrl = import.meta.env.VITE_API_URL
                            const response = await axios.get(`${baseUrl}/api/debug/proxy`, {
                                headers: {
                                    "proxy_url": table.proxy_url,
                                    "target_url": 'http://api.xiangshang666.com/account/updateAccountStatus',
                                },
                                params: {
                                    token: account_brand_mevrt.value.info.token,
                                    account: account.uid,
                                    status: 3,
                                    source: 0,
                                    remark: table.status[id].login.value,
                                }
                            });

                            console.log("设置状态.3：", JSON.stringify(response.data, null, 2));

                            if (response.data.code === 200) {
                                account.upload = true
                            }

                        } catch (error) { }

                    } break
                }
            }
        }
    },
})

account_brand_mevrt.value.config.switch = false

// ---

export const delete_handle = (robot: RobotType) => {

    const dialog = window['$dialog'].error({
        title: '删除扣号',
        content: `即将扣号：${robot.submit.uid}，此过程不可逆！`,
        positiveText: '确认删除？',
        negativeText: '取消',
        maskClosable: false,
        onPositiveClick: async () => {
            dialog.loading = true
            await store.robot.DeleteByBatch(batch.object.key.value, robot.id, async () => {
                await batch.Update()
                window['$message'].success('删除成功')
            }, (err_msg) => {
            })
            dialog.loading = false
        },
    })

    dialog.loading = true
    dialog.positiveText = '确认删除？'

    setTimeout(() => {
        dialog.positiveText = `确认`
        dialog.loading = false
    }, 1000)
}
