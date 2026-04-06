<template>
    <div class="loading center" v-if="self.loading">
        <tool-loading />
    </div>
    <n-text class="error" v-else-if="self.error" type="error">
        {{ self.error }}
    </n-text>
    <div class="image center" v-else>
        <n-image height="140" alt="图片" :src="self.baseUrl + '/api/debug/proxy?target_url=' + self.url">
            <template #error>
                <n-button>失败</n-button>
            </template>
        </n-image>
    </div>
</template>

<script lang="ts" setup>
import { store } from '@/store';
import { onMounted, reactive, watch } from 'vue';
import ToolLoading from './tool.loading.vue';

const props = defineProps<{
    robot_id: string
    data: string
}>()

const self = reactive({
    baseUrl: '',
    info: <ProtobufNotOnlineImageType>undefined,
    url: '',
    loading: true,
    error: undefined,
    time: store.now,
    update: async () => {
        self.loading = true
        self.info = JSON.parse(props.data)

        const splits = self.info?.resId.replaceAll('/', '').split('-')
        if (splits.length == 0) {
            self.error = `照片数据异常`
            return
        }

        self.url = `https://gchat.qpic.cn/offpic_new/` + splits[0] + self.info?.resId + `/0?term=2&is_origin=0`

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
    position: relative;
    overflow: hidden;
    padding: 8px;

    .n-image {
        border-radius: 8px;
        height: 100%;
    }
}
</style>