<template>
    <n-modal :show="friend_notice.show" @mask-click="friend_notice.close">
        <div>
            <div class="create modal" v-if="friend_notice.show">
                <div class="top">
                    <n-button type="primary" text>好友请求</n-button>
                    <div class="func-place" />
                    <div class="modal-close" @click="friend_notice.close">
                        <icon-close />
                    </div>
                </div>

                <div class="middle">

                    <n-input-group style="margin-bottom: 10px;">
                        <n-button size="small">选择机器人</n-button>
                        <n-select size="small" v-model:value="friend_notice.robot_id" :options="robotOptions"
                            filterable placeholder="请选择机器人" style="min-width: 200px;"
                            :consistent-menu-width="false" />
                        <n-button size="small" type="primary" @click="friend_notice.fetch"
                            :loading="friend_notice.loading" :disabled="!friend_notice.robot_id">
                            查询
                        </n-button>
                    </n-input-group>

                    <n-scrollbar style="max-height: 500px;" v-if="friend_notice.list.length > 0">
                        <tool-table :empty="false" :loading="false">
                            <template v-slot:head>
                                <tr>
                                    <th style="min-width: 60px;">序号</th>
                                    <th style="min-width: 100px;">QQ号</th>
                                    <th style="min-width: 80px;">昵称</th>
                                    <th style="min-width: 40px;">性别</th>
                                    <th style="min-width: 40px;">年龄</th>
                                    <th style="min-width: 120px;">验证消息</th>
                                    <th style="min-width: 100px;">来源</th>
                                    <th style="min-width: 120px;">时间</th>
                                    <th style="min-width: 80px;">操作</th>
                                </tr>
                            </template>
                            <template v-slot:body>
                                <tr v-for="(item, index) in friend_notice.list" :key="item.req_uin">
                                    <td>{{ index + 1 }}</td>
                                    <td>
                                        <n-text type="primary">{{ item.req_uin }}</n-text>
                                    </td>
                                    <td>{{ item.nick || '-' }}</td>
                                    <td>{{ item.gender == 0 ? '男' : item.gender == 1 ? '女' : '-' }}</td>
                                    <td>{{ item.age || '-' }}</td>
                                    <td>
                                        <n-text depth="3">{{ item.msg_additional || '-' }}</n-text>
                                    </td>
                                    <td>
                                        <n-text depth="3">{{ item.msg_source || item.msg_detail || '-' }}</n-text>
                                    </td>
                                    <td>
                                        <n-text depth="3">{{ formatTime(item.msg_time) }}</n-text>
                                    </td>
                                    <td>
                                        <n-button size="tiny" type="success"
                                            :loading="friend_notice.passing[item.req_uin]"
                                            :disabled="friend_notice.passed[item.req_uin]"
                                            @click="friend_notice.pass(item)">
                                            {{ friend_notice.passed[item.req_uin] ? '已通过' : '通过' }}
                                        </n-button>
                                    </td>
                                </tr>
                            </template>
                        </tool-table>
                    </n-scrollbar>

                    <n-empty v-else-if="!friend_notice.loading && friend_notice.fetched"
                        description="暂无好友请求" style="padding: 40px 0;" />

                </div>
            </div>
        </div>
    </n-modal>
</template>

<script setup lang="ts">
import IconClose from '@/assets/icon/icon.close.vue'
import { friend_notice, robotOptions, formatTime } from './index.func.friend.notice'
</script>

<style lang="scss" scoped>
.create {
    padding: 15px;
    display: flex;
    flex-direction: column;
    min-width: 800px;

    .top {
        display: flex;
        margin-bottom: 10px;
    }

    .middle {
        flex: 1;
    }
}
</style>
