<template>
    <n-modal :show="table_slider_login.show" @mask-click="table_slider_login.close">
        <div>
            <div class="create modal" v-if="table_slider_login.show">

                <div class="middle" :style="{
                    width: `${store.view.screen.height * 0.7 * (375 / 667)}px`,
                    height: `${store.view.screen.height * 0.7}px`,
                }">
                    <div class="loading">
                        <tool-loading></tool-loading>
                    </div>
                    <div class="page">
                        <iframe :src="table_slider_login.value" ref="iframe" frameborder="0" allowfullscreen
                            width="100%" height="100%" />
                    </div>
                </div>
            </div>
        </div>
    </n-modal>
</template>

<script setup lang="ts">
import { table_slider_login } from '.';
import { store } from '@/store';
import { ref, watch } from 'vue';
import toolLoading from '@/public/tool.loading.vue';

</script>

<style lang="scss" scoped>
.create {
    padding: 15px;
    display: flex;
    flex-direction: column;

    .top {
        display: flex;
        margin-bottom: 10px;
    }

    .middle {
        position: relative;

        .page {
            position: absolute;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
        }

        .loading {
            position: absolute;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
        }
    }
}
</style>
