import { computed, reactive, ref } from "vue"

import { store } from "@/store"
import { ROBOT_BATCH_MODE_ENUM, ROBOT_BATCH_MODE_NAME } from "@/types/type.robot.batch.e"
import { ROBOT_LOGIN_MODE_ENUM, ROBOT_LOGIN_STATUS_ENUM, ROBOT_LOGIN_STATUS_OPTIONS } from "@/types/type.robot.e"
import { GetQueryParams } from "@/utils/text"

import AccountFunc from './account/index.func.vue'
import AccountFilter from './account/index.filter.vue'
import AccountTable from './account/index.table.vue'
import AccountFoot from './account/index.foot.vue'

import ImageFunc from './image/index.func.vue'
import ImageFilter from './image/index.filter.vue'
import ImageTable from './image/index.table.vue'
import ImageFoot from './image/index.foot.vue'

import CookieFunc from './cookie/index.func.vue'
import CookieFilter from './cookie/index.filter.vue'
import CookieTable from './cookie/index.table.vue'
import CookieFoot from './cookie/index.foot.vue'

import PackFunc from './pack/index.func.vue'
import PackFilter from './pack/index.filter.vue'
import PackTable from './pack/index.table.vue'
import PackFoot from './pack/index.foot.vue'

export const menu = reactive<MenuUniversalType>({
    select: 0,
    loading: false,
    skeleton: false,
    SelectHandle: (index: number) => {
    }
})

export const menu_list: MenuSelectUniversalType[] = [
    {
        title: ROBOT_BATCH_MODE_NAME[ROBOT_BATCH_MODE_ENUM.ACCOUNT],
        component: {
            func: AccountFunc,
            filter: AccountFilter,
            table: AccountTable,
            foot: AccountFoot,
        },
    },
    // {
    //     title: ROBOT_BATCH_MODE_NAME[ROBOT_BATCH_MODE_ENUM.IMAGE],
    //     component: {
    //         func: ImageFunc,
    //         filter: ImageFilter,
    //         table: ImageTable,
    //         foot: ImageFoot,
    //     },
    // },
    {
        title: ROBOT_BATCH_MODE_NAME[ROBOT_BATCH_MODE_ENUM.COOKIE],
        component: {
            func: CookieFunc,
            filter: CookieFilter,
            table: CookieTable,
            foot: CookieFoot,
        },
    },
    {
        title: ROBOT_BATCH_MODE_NAME[ROBOT_BATCH_MODE_ENUM.PACK],
        component: {
            func: PackFunc,
            filter: PackFilter,
            table: PackTable,
            foot: PackFoot,
        },
    },
]

export const batch = reactive({
    object: <RobotBatchType>(undefined as any),
    loading: true,

    proxy_urls: <{ [id: string]: string }>{},
    proxy: <ProxyType>undefined,

    robots: <RobotType[]>[],
    webrobots: <WebrobotType[]>[],

    nick_material: <RobotMaterialType>undefined,

    // update_loading: false,
    Update: async (no_robots?: boolean) => {
        // if (!no_robots) table.update_loading = true
        await store.robot_batch.GetInfo(GetQueryParams()['key'], no_robots, (robot_batch, proxy, proxy_urls, webrobots, robots, nick_material) => {
            let existed = false
            for (const [index, item] of menu_list.entries()) {
                if (ROBOT_BATCH_MODE_NAME[robot_batch.mode] == item.title) {
                    menu.select = index
                    batch.object = robot_batch
                    batch.proxy = proxy

                    if (!no_robots) {
                        if (robots) batch.robots = robots.reverse()
                        if (webrobots) batch.webrobots = webrobots.reverse()
                        batch.proxy_urls = proxy_urls
                        batch.nick_material = nick_material
                    }

                    existed = true

                } else {
                    item.disabled = true
                }
            }
            if (!existed) window['$message'].error('不支持的类型')
        })
        batch.loading = false
        // if (!no_robots) table.update_loading = false
    },

    ProxyReset: async (robot_id: string) => {
        const dialog = window['$dialog'].success({
            title: '更换代理',
            content: '即将更换机器人的静态代理，此过程不可逆。',
            positiveText: '确定',
            negativeText: '取消',
            maskClosable: false,
            onPositiveClick: async () => {
                dialog.loading = true
                await store.robot.ProxyReset(batch.object.key.value, robot_id, () => {
                    window['$message'].success('更换成功')
                })
                dialog.loading = false
            },
        })
    },

    SetPassword: {
        show: false,
        robot: <RobotType>undefined,
        value: {
            password: '',
        },
        loading: false,
        view: (item: RobotType) => {
            batch.SetPassword.robot = item
            batch.SetPassword.value.password = item.kernel.UserLoginData.Password
            batch.SetPassword.show = true
        },
        close: () => {
            batch.SetPassword.show = false
        },
        Handle: async () => {
            batch.SetPassword.loading = true
            await store.robot.SetPassword(batch.object.key.value, batch.SetPassword.robot.id, batch.SetPassword.value.password, () => {
                window['$message'].success('修改成功')
                batch.Update()
                batch.SetPassword.close()
            })
            batch.SetPassword.loading = false
        },
    },

    Login: async (robot_id: string) => {
        const dialog = window['$dialog'].success({
            title: '手动登录',
            content: '即将通过账号密码进行登录。',
            positiveText: '确定',
            negativeText: '取消',
            maskClosable: false,
            onPositiveClick: async () => {
                dialog.loading = true
                await store.robot.Login(batch.object.key.value, robot_id, ROBOT_LOGIN_MODE_ENUM.PASSWORD, "", () => {
                    window['$message'].success('登录请求成功，正在启动同步...')
                    store.robot.UpdateStop(robot_id, false, () => {
                        window['$message'].success('已启动，数据同步中')
                    })
                })
                dialog.loading = false
            },
        })
    },

    Relogin: async (robot_id: string) => {
        const dialog = window['$dialog'].warning({
            title: '重新登录',
            content: '即将重新登录该机器人，会先重置登录状态再重新登录。',
            positiveText: '确定',
            negativeText: '取消',
            maskClosable: false,
            onPositiveClick: async () => {
                dialog.loading = true
                await store.robot.LoginClear(batch.object.key.value, robot_id, async () => {
                    await store.robot.Login(batch.object.key.value, robot_id, ROBOT_LOGIN_MODE_ENUM.PASSWORD, "", () => {
                        window['$message'].success('重登请求成功，正在启动同步...')
                        store.robot.UpdateStop(robot_id, false, () => {
                            window['$message'].success('已启动，数据同步中')
                        })
                    })
                })
                dialog.loading = false
            },
        })
    },

    LoginClear: async (robot_id: string) => {
        const dialog = window['$dialog'].success({
            title: '重置登录',
            content: '即将清空机器人的登录状态，此过程不可逆。',
            positiveText: '确定',
            negativeText: '取消',
            maskClosable: false,
            onPositiveClick: async () => {
                dialog.loading = true
                await store.robot.LoginClear(batch.object.key.value, robot_id, () => {
                    window['$message'].success('重置登录成功')
                })
                dialog.loading = false
            },
        })
    },

    OutSidePhoneLogin: {
        show: false,
        loading: false,
        view: () => {
            batch.OutSidePhoneLogin.show = true
        },
        close: () => {
            batch.OutSidePhoneLogin.show = false
        },
    },
})
