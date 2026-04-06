import { ResultEnum, errorMsg } from "./axios";
import { RequestHttpEnum, http } from "./http"
import { store } from "@/store";

export const SigninApi = async (username: string, password: string, scuccess: (username: string, token: string, id) => void, error: (err_msg: string) => void) => {
    try {
        const baseUrl = import.meta.env.ROBOT_BASE_URL
        const res = await http(RequestHttpEnum.POST)<{ username: string, token: string, id: string }>(`${baseUrl}/user/signin`, {}, { username, password })
        if (res.code != ResultEnum.REQUEST_SUCCESS) error(errorMsg(res))
        else scuccess(res.data.username, res.data.token, res.data.id)

    } catch (err_msg) {
        if (error) error(String(err_msg))
    }
}

export const EnvironmentApi = async (scuccess: (update_time: { [object: string]: number }, system_info: SystemInfoType) => void, error: (err_msg: string) => void) => {
    try {
        const baseUrl = import.meta.env.ROBOT_BASE_URL
        const res = await http(RequestHttpEnum.GET)<{ update_time: { [object: string]: number }, system_info: SystemInfoType }>(`${baseUrl}/user/environment`)
        if (res.code != ResultEnum.REQUEST_SUCCESS) error(errorMsg(res))
        else scuccess(res.data.update_time, res.data.system_info)

    } catch (err_msg) {
        if (error) error(String(err_msg))
    }
}

export const UserFetchApi = async (scuccess: (list: UserType[]) => void, error: (err_msg: string) => void) => {
    try {
        const baseUrl = import.meta.env.ROBOT_BASE_URL
        const res = await http(RequestHttpEnum.GET)<UserType[]>(`${baseUrl}/user/fetch`)
        if (res.code != ResultEnum.REQUEST_SUCCESS) error(errorMsg(res))
        else scuccess(res.data.reverse())

    } catch (err_msg) {
        if (error) error(String(err_msg))
    }
}

export const UserCreateApi = async (table: UserType, scuccess: () => void, error: (err_msg: string) => void) => {
    try {
        const baseUrl = import.meta.env.ROBOT_BASE_URL
        const res = await http(RequestHttpEnum.POST)<undefined>(`${baseUrl}/user/create`, {}, table)
        if (res.code != ResultEnum.REQUEST_SUCCESS) error(errorMsg(res))
        else scuccess()

    } catch (err_msg) {
        if (error) error(String(err_msg))
    }
}

export const UserUpdateApi = async (table: UserType, scuccess: () => void, error: (err_msg: string) => void) => {
    try {
        const baseUrl = import.meta.env.ROBOT_BASE_URL
        const res = await http(RequestHttpEnum.POST)<undefined>(`${baseUrl}/user/update`, {}, table)
        if (res.code != ResultEnum.REQUEST_SUCCESS) error(errorMsg(res))
        else scuccess()

    } catch (err_msg) {
        if (error) error(String(err_msg))
    }
}

export const UserProgressApi = async (progress_id: number, scuccess: (progress: number) => void, error: (err_msg: string) => void) => {
    try {
        const baseUrl = import.meta.env.ROBOT_BASE_URL
        const res = await http(RequestHttpEnum.GET)<number>(`${baseUrl}/user/progress`, { progress_id })
        if (res.code != ResultEnum.REQUEST_SUCCESS) error(errorMsg(res))
        else scuccess(res.data)

    } catch (err_msg) {
        if (error) error(String(err_msg))
    }
}

export const UserDisabledRobotApi = async (disabled: boolean, scuccess: () => void, error: (err_msg: string) => void) => {
    try {
        const baseUrl = import.meta.env.ROBOT_BASE_URL
        const res = await http(RequestHttpEnum.GET)<undefined>(`${baseUrl}/user/disabled_robot`, { disabled, time: store.now })
        if (res.code != ResultEnum.REQUEST_SUCCESS) error(errorMsg(res))
        else scuccess()

    } catch (err_msg) {
        if (error) error(String(err_msg))
    }
}

export const UserStatusApi = async (scuccess: (status: { [user_id: string]: UserStatusType }) => void, error: (err_msg: string) => void) => {
    try {
        const baseUrl = import.meta.env.ROBOT_BASE_URL
        const res = await http(RequestHttpEnum.GET)<{ [user_id: string]: UserStatusType }>(`${baseUrl}/user/status`, {})
        if (res.code != ResultEnum.REQUEST_SUCCESS) error(errorMsg(res))
        else scuccess(res.data)

    } catch (err_msg) {
        if (error) error(String(err_msg))
    }
}
