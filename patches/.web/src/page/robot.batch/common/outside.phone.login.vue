<template>
    <n-modal :show="batch.OutSidePhoneLogin.show" @mask-click="batch.OutSidePhoneLogin.close">
        <div>
            <div class="create modal" v-if="batch.OutSidePhoneLogin.show">

                <vue-qrcode class="qrcode" :value="qrcode_value" :color="{ dark: '#000000ff', light: '#ffffffff' }"
                    :type="'image/png'" />

            </div>
        </div>
    </n-modal>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ROBOT_LOGIN_MODE_ENUM } from '@/types/type.robot.e';
import { batch } from '..';

const qrcode_value = computed(() => {
    // [PATCH] 使用 window.location.origin 补全完整地址
    // 原始代码用 import.meta.env.VITE_API_URL，production 模式下是 '/api'（相对路径），
    // 导致二维码里没有 IP 和端口，手机扫码后无法访问
    const baseUrl = import.meta.env.VITE_API_URL
    const fullBase = baseUrl.startsWith('http') ? baseUrl : (window.location.origin + baseUrl)
    return `{"api":"${fullBase}/robot/login?key=${batch.object.key.value}&mode=${ROBOT_LOGIN_MODE_ENUM.PASSWORD}"}`
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
