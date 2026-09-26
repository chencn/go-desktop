<!--
  文件职责：渲染设计稿 .license-card 的授权卡片（设备码复制 + 授权码激活）。
  说明：关于页与未授权闸门页共用本组件，激活逻辑一律走 appStore.activateLicenseKey。
-->
<script setup lang="ts">
import { computed, ref } from 'vue'
import { Copy, KeyRound } from '@lucide/vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()
// licenseKey 保存用户输入的授权码，提交前只做空白裁剪。
const licenseKey = ref('')
// copied 标记最近一次设备码复制结果，用于短暂切换按钮文案。
const copied = ref(false)

// licenseStatus 来自启动阶段 GetLicenseStatus；后端未启用授权时字段为默认值。
const licenseStatus = computed(() => appStore.licenseStatus)
// deviceCode 当前设备短码，用于发给授权签发脚本。
const deviceCode = computed(() => licenseStatus.value?.deviceCode ?? '')
// authorized 决定状态药丸的配色与文案。
const authorized = computed(() => Boolean(licenseStatus.value?.authorized))
// statusLabel 是卡片右上角的授权结论。
const statusLabel = computed(() => (authorized.value ? '已激活商用授权' : '未激活商用授权'))
// errorMessage 只在激活失败时出现，避免用整块版面承载一次性错误。
const errorMessage = computed(() => appStore.licenseError || licenseStatus.value?.lastError || '')
// canSubmit 控制激活按钮，避免提交空授权码或重复提交。
const canSubmit = computed(() => licenseKey.value.trim() !== '' && !appStore.licenseLoading)

// copyDeviceCode 把设备码写入剪贴板；剪贴板不可用时保持静默。
async function copyDeviceCode() {
  if (!deviceCode.value || typeof navigator === 'undefined' || !navigator.clipboard) return
  await navigator.clipboard.writeText(deviceCode.value)
  copied.value = true
  window.setTimeout(() => {
    copied.value = false
  }, 1600)
}

// submitLicense 调用后端 ActivateLicense；store 会去除空白并在授权成功后重新 initialise。
async function submitLicense() {
  if (!canSubmit.value) return
  try {
    await appStore.activateLicenseKey(licenseKey.value.trim())
  } catch {
    // 授权错误已写入 store，卡片只负责展示。
  }
}
</script>

<template>
  <section class="license-card" aria-labelledby="license-card-title">
    <div class="license-card-head">
      <div class="license-card-heading">
        <span class="license-card-kicker">授权校验</span>
        <h3 id="license-card-title" class="license-card-title">
          <span class="sq-icon-badge size-sm teal" aria-hidden="true">
            <KeyRound :size="13" :stroke-width="2.2" />
          </span>
          商业许可与设备激活
        </h3>
      </div>
      <span class="badge-apple" :class="authorized ? 'ok' : 'error'">{{ statusLabel }}</span>
    </div>

    <p v-if="errorMessage" class="license-error">{{ errorMessage }}</p>

    <form class="license-field" @submit.prevent="submitLicense">
      <label for="license-device-code">设备码 (用于授权签发脚本):</label>
      <div class="input-with-append">
        <el-input
          id="license-device-code"
          class="apple-field is-tall is-mono is-appended license-device-field"
          :model-value="deviceCode"
          readonly
        />
        <el-button class="btn-apple is-append" :disabled="!deviceCode" @click="copyDeviceCode">
          <Copy :size="14" aria-hidden="true" />
          <span>{{ copied ? '已复制' : '复制设备码' }}</span>
        </el-button>
      </div>

      <label class="license-key-label" for="license-key">授权码:</label>
      <el-input
        id="license-key"
        v-model="licenseKey"
        class="apple-field is-mono"
        type="textarea"
        :rows="4"
        autocomplete="off"
        spellcheck="false"
        placeholder="GD1-..."
      />

      <div class="license-actions">
        <el-button class="btn-apple" type="primary" native-type="submit" :disabled="!canSubmit">
          <KeyRound :size="15" aria-hidden="true" />
          <span>{{ appStore.licenseLoading ? '正在激活...' : '激活授权' }}</span>
        </el-button>
      </div>
    </form>
  </section>
</template>

<style scoped src="./LicenseCard.css"></style>
