import { GetQueryParams } from "@/utils/text"

export let ymlink_q2_win_server_url = undefined
export const ymlink_q2_win_api_axiospre = '/api'

if (window.location.hostname == 'localhost') {

    const params = GetQueryParams()

    if (params['url']) ymlink_q2_win_server_url = `http://${params['url']}`
    else ymlink_q2_win_server_url = `http://localhost:8080`


} else {
    ymlink_q2_win_server_url = window.location.origin
}
