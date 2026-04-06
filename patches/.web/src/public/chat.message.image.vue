<template>
    <div class="loading center" v-if="self.loading" :style="self.style">
        <tool-loading />
    </div>
    <n-text class="error" v-else-if="self.error" type="error">
        {{ self.error }}
    </n-text>
    <div class="image center" v-else>
        <n-image :style="self.style"
            :src="`/file/message/${robot_id}/${self.index_node.FileUuid}`" />
    </div>
</template>

<script lang="ts" setup>
import { store } from '@/store';
import { computed, onMounted, reactive, watch } from 'vue';
import ToolLoading from './tool.loading.vue';

const props = defineProps<{
    robot_id: string
    data: string
}>()

const self = reactive({
    info: <ProtobufEleMsgInfoType>undefined,
    index_node: <ProtobufEleMsgInfoType['msgInfoBody'][0]['indexNode']>undefined,
    picture_info: <ProtobufEleMsgInfoType['msgInfoBody'][0]['pictureInfo']>undefined,
    style: computed(() => {
        let max_width = 140
        let width = self.index_node?.Info?.Width
        let height = self.index_node?.Info?.Height
        if (width > max_width) {
            width = max_width
            height = height * (max_width / self.index_node?.Info?.Width)
        }
        return { width: `${width}px`, height: `${height}px` }
    }),
    loading: true,
    error: undefined,
    time: store.now,
    update: async () => {
        self.loading = true
        self.info = JSON.parse(props.data)

        await (async () => {
            for (const item of self.info.msgInfoBody) {
                if (item.indexNode) self.index_node = item.indexNode
                if (item.pictureInfo) self.picture_info = item.pictureInfo
            }

            if (!self.index_node || !self.picture_info) {

                self.error = `照片数据异常`

                window.$notification.error({
                    title: self.error,
                    content: props.robot_id,
                    meta: props.data,
                    keepAliveOnHover: true
                })

                return
            }

            await store.robot.MessageImage(props.robot_id, self.index_node.FileUuid, self.picture_info, () => {
                self.error = undefined
            }, (err_msg) => {
                self.error = err_msg
            })
        })()

        self.loading = false
    },
})

watch(() => [props.robot_id, props.data], async () => {
    await self.update()
})

onMounted(() => {
    self.update()
})

</script>


<style lang="scss" scoped>
.loading {
    padding: 8px;
    min-width: 100px;
    min-height: 100px;
    padding: 8px;
    transform: scale(0.7);
}

.error {
    white-space: pre-wrap !important;
    word-break: break-word;
    display: inline-block;
    padding: 10px 12px;
}

.image {
    overflow: hidden;
    padding: 8px;

    .n-image {
        border-radius: 8px;
        height: 100%;
    }
}
</style>
