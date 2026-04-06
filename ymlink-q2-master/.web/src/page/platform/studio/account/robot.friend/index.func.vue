<template>

    <n-button size="small" @click="table.export" :loading="table.loading">导出历史库</n-button>

    <div class="func-place-2" />

    <n-button size="small" type="info" @click="friend_notice.view">好友请求</n-button>

    <div class="func-place-2" />

    <n-select class="group" size="small" :value="'批量操作'" :consistent-menu-width="false" :options="[
        { label: '导入拉粉任务', value: 'groupchatpull' },
        { label: '发送消息', value: 'send_message' },
        { label: '分组发送消息', value: 'send_messages' },
    ]" style="width: 128px;" @update:value="(v: string) => {
        switch (v) {
            case 'groupchatpull': return table_group_groupchatpull.view()
            case 'send_message': return send_message.view()
            case 'send_messages': return send_messages.view()
        }
    }" />

    <group-groupchatpull />
    <group-send-message />
    <group-send-messages />
    <friend-notice-modal />

</template>

<script setup lang="ts">
import { table, table_group_groupchatpull } from '.';
import GroupGroupchatpull from './index.func.group.groupchatpull.vue';
import GroupSendMessage from './index.func.group.send.message.vue';
import GroupSendMessages from './index.func.group.send.messages.vue';
import FriendNoticeModal from './index.func.friend.notice.vue';
import { friend_notice } from './index.func.friend.notice';
import { _table as send_message } from './index.func.group.send.message';
import { _table as send_messages } from './index.func.group.send.messages';

</script>

<style lang="scss" scoped>
.group {
    :deep() {
        * {
            text-align: center !important;
        }

        .n-base-select-option {
            text-align: center;
        }

        .n-base-select-option__content {
            width: 100%;
            text-align: center;
        }
    }
}
</style>
