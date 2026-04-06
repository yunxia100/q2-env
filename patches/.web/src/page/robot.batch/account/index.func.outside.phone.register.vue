<template>
    <n-modal :show="table_outside_phone_register.show" @mask-click="table_outside_phone_register.close">
        <div>
            <div class="create modal" v-if="table_outside_phone_register.show">
                <vue-qrcode class="qrcode" :value="qrcode_value" :color="{ dark: '#000000ff', light: '#ffffffff' }"
                    :type="'image/png'" />

            </div>
        </div>
    </n-modal>
</template>

<script setup lang="ts">
import { store } from '@/store';
import { table_outside_phone_register } from '.';
import { computed } from 'vue';
import { ROBOT_LOGIN_MODE_ENUM, ROBOT_REGISTER_MODE_ENUM, ROBOT_REGISTER_MODE_NAME } from '@/types/type.robot.e';
import { table } from '.';
import { batch } from '..';

const qrcode_value = computed(() => {
    // [PATCH] 使用 window.location.origin 动态获取当前服务器地址
    // 原始代码硬编码了 8.217.115.54，换服务器后二维码地址就错了
    const origin = window.location.origin
    return `{"api":"${origin}/api/robot/register?key=${batch.object.key.value}&mode=${ROBOT_REGISTER_MODE_ENUM.PHONE_CHECK}"}`
})

</script>

<style lang="scss" scoped>
.create {
    padding: 15px;
    display: flex;
    flex-direction: column;
    width: 300px;
    height: 300px;

    .qrcode {
        height: 100%;
        width: 100%;
    }
}
</style>
