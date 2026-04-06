import { reactive, computed } from 'vue'
import { store } from '@/store'
import { get, post } from '@/api/http'
import { ResultEnum } from '@/api/axios'
import { ymlink_q2_win_api_axiospre } from './../../../../../../setting'

export interface FriendNoticeItem {
    msg_type: number
    msg_seq: number
    msg_time: number
    req_uin: number
    nick: string
    gender: number
    age: number
    src_id: number
    sub_src_id: number
    msg_title: string
    msg_additional: string
    msg_source: string
    msg_detail: string
}

export const robotOptions = computed(() => {
    return store.robot.list.map((robot: any) => {
        const qqNum = robot.kernel?.UserLoginData?.Uin || robot.submit?.uid || ''
        const nick = robot.submit?.nick || ''
        return {
            label: `${qqNum}${nick ? ' (' + nick + ')' : ''}`,
            value: robot.id,
        }
    })
})

export const formatTime = (timestamp: number) => {
    if (!timestamp) return '-'
    const d = new Date(timestamp * 1000)
    return `${d.getMonth() + 1}-${d.getDate()} ${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
}

export const friend_notice = reactive({
    show: false,
    loading: false,
    fetched: false,
    robot_id: '' as string,
    list: [] as FriendNoticeItem[],
    passing: {} as { [uin: number]: boolean },
    passed: {} as { [uin: number]: boolean },

    view: () => {
        friend_notice.show = true
        friend_notice.list = []
        friend_notice.fetched = false
        friend_notice.passing = {}
        friend_notice.passed = {}
        if (store.robot.list.length == 1) {
            friend_notice.robot_id = store.robot.list[0].id
        }
    },
    close: () => {
        friend_notice.show = false
    },
    fetch: async () => {
        if (!friend_notice.robot_id) return
        friend_notice.loading = true
        friend_notice.list = []
        friend_notice.passed = {}
        friend_notice.passing = {}
        try {
            const res = await get<FriendNoticeItem[]>(
                `${ymlink_q2_win_api_axiospre}/robot/friend_notices`,
                { robot_id: friend_notice.robot_id }
            )
            if (res.code == ResultEnum.REQUEST_SUCCESS) {
                friend_notice.list = res.data || []
            } else {
                window.$message.error(res.msg || '获取好友请求失败')
            }
        } catch (err) {
            window.$message.error('获取好友请求失败: ' + String(err))
        }
        friend_notice.loading = false
        friend_notice.fetched = true
    },
    pass: async (item: FriendNoticeItem) => {
        friend_notice.passing[item.req_uin] = true
        try {
            const res = await post<any>(
                `${ymlink_q2_win_api_axiospre}/robot/friend_pass`,
                {},
                {
                    robot_id: friend_notice.robot_id,
                    req_uin: item.req_uin,
                    src_id: item.src_id,
                    sub_src_id: item.sub_src_id,
                }
            )
            if (res.code == ResultEnum.REQUEST_SUCCESS) {
                friend_notice.passed[item.req_uin] = true
                window.$message.success('已通过好友请求')
            } else {
                window.$message.error(res.msg || '通过好友请求失败')
            }
        } catch (err) {
            window.$message.error('通过好友请求失败: ' + String(err))
        }
        friend_notice.passing[item.req_uin] = false
    },
})
