import axios, { AxiosResponse, AxiosRequestConfig, InternalAxiosRequestConfig, Axios } from 'axios'
import { store } from '@/store'
import { PATH_ENUM } from '@/types/type.base.e'

export enum ResultEnum {
    REQUEST_SUCCESS = 200, // 请求成功
    REQUEST_CREATED = 201, // 创建成功
    REQUEST_BAD = 400, // 请求参数有误或格式不正确
    REQUEST_UNAUTHORIZED = 401, // 未通过用户验证
    REQUEST_FORBIDDEN = 403, // 拒绝访问
    REQUEST_NOT_FOUND = 404, // 资源不存在
    REQUEST_METHOD_NOT_ALLOWED = 405, // 方法不允许
    REQUEST_SERVER_ERROR = 500, // 服务器异常
}

export interface MyResponseType<T> {
    code: ResultEnum
    data: T
    msg: string
}

export interface MyRequestInstance extends Axios {
    <T = any>(config: AxiosRequestConfig): Promise<MyResponseType<T>>
}

export const errorMsg = (res: MyResponseType<any>) => {

    if (res.msg != "") return res.msg

    switch (res.code) {
        case ResultEnum.REQUEST_SUCCESS: return "请求成功"
        case ResultEnum.REQUEST_CREATED: return "创建成功"
        case ResultEnum.REQUEST_BAD: return "请求参数有误或格式不正确"
        case ResultEnum.REQUEST_UNAUTHORIZED: return "未通过用户验证"
        case ResultEnum.REQUEST_FORBIDDEN: return "拒绝访问"
        case ResultEnum.REQUEST_NOT_FOUND: return "资源不存在"
        case ResultEnum.REQUEST_METHOD_NOT_ALLOWED: return "方法不允许"
        case ResultEnum.REQUEST_SERVER_ERROR: return "服务器异常"
        default: return "fail"
    }
}

const axiosInstance = axios.create({
    baseURL: '/',
    timeout: 10 * 60 * 1000,
}) as unknown as MyRequestInstance

axiosInstance.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {

        switch (store.view.path) {
            case PATH_ENUM.LOGIN:
            case PATH_ENUM.PLATFORM:
                config.headers['Authorization'] = 'Bearer ' + (store.user.info ? store.user.info.token : '')
                break
            case PATH_ENUM.ROBOT_BATCH:
                break
            case PATH_ENUM.CUSTSERVICE:
                config.headers['Authorization'] = 'Bearer ' + (store.custservice.info ? store.custservice.info.token : '')
                break
        }

        return config
    },
    (err: AxiosRequestConfig) => {
        Promise.reject(err)
    }
)

// 响应拦截器
axiosInstance.interceptors.response.use(
    (res: AxiosResponse) => {

        switch (res.data?.code) {
            case ResultEnum.REQUEST_UNAUTHORIZED:
                window.$message.error(res.data.msg, { closable: true, duration: 5000 })
                store.user.info = undefined
                break
            case ResultEnum.REQUEST_BAD:
                window.$message.error(res.data?.msg)
                break
            case ResultEnum.REQUEST_SERVER_ERROR:
                window.$notification.error({
                    title: res.data?.msg,
                    content: res.config.url,
                    meta: res.data?.data,
                    keepAliveOnHover: true
                })
                break
        }

        return Promise.resolve(res.data)
    },
    (err: AxiosResponse) => {
        Promise.reject(err)
    }
)

export default axiosInstance
