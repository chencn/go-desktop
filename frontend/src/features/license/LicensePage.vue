<!-- 文件职责：授权模式开启时显示设备码和授权码激活入口。 -->
<script setup lang="ts">
import { computed, ref } from 'vue'
import { AlertCircle, CheckCircle2, Clipboard, KeyRound } from '@lucide/vue'
import { useAppStore } from '../../stores/app'

// appStore 负责授权状态读取、激活请求和错误展示。
const appStore = useAppStore()
// licenseKey 保存用户输入的授权码字符串，提交前只做空白裁剪。
const licenseKey = ref('')
// copied 标记最近一次设备码复制结果，用于短暂展示反馈。
const copied = ref(false)

// licenseStatus 当前后端授权状态；为空时使用初始状态兜底。
const licenseStatus = computed(() => appStore.licenseStatus)
// deviceCode 当前设备短码，用于发给授权签发脚本。
const deviceCode = computed(() => licenseStatus.value?.deviceCode ?? '')
// statusMessage 优先展示后端明确错误，其次展示授权状态消息。
const statusMessage = computed(() => appStore.licenseError || licenseStatus.value?.lastError || licenseStatus.value?.message || '需要授权')
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

// submitLicense 调用后端激活授权码；错误由 store 写入 licenseError。
async function submitLicense() {
  if (!canSubmit.value) return
  try {
    await appStore.activateLicenseKey(licenseKey.value.trim())
  } catch {
    // 授权错误已写入 store，页面只负责展示。
  }
}
</script>

<template>
  <main class="license-shell">
    <section class="license-panel" aria-labelledby="license-title">
      <div class="license-heading">
        <span class="license-icon" aria-hidden="true">
          <KeyRound :size="22" />
        </span>
        <div>
          <p class="license-kicker">授权校验</p>
          <h1 id="license-title">需要授权</h1>
        </div>
      </div>

      <div class="license-status" :class="{ 'is-error': !!appStore.licenseError || !!licenseStatus?.lastError }">
        <AlertCircle v-if="appStore.licenseError || licenseStatus?.lastError" :size="18" aria-hidden="true" />
        <CheckCircle2 v-else :size="18" aria-hidden="true" />
        <span>{{ statusMessage }}</span>
      </div>

      <form class="license-form" @submit.prevent="submitLicense">
        <UiField class="license-field">
          <UiLabel for="license-device-code">设备码</UiLabel>
          <div class="license-copy-row">
            <UiInput id="license-device-code" :model-value="deviceCode" readonly />
            <UiTooltip :content="copied ? '已复制' : '复制设备码'">
              <UiButton class="license-icon-button" type="button" variant="outline" size="icon" :disabled="!deviceCode" @click="copyDeviceCode">
                <Clipboard :size="16" aria-hidden="true" />
                <span class="sr-only">复制设备码</span>
              </UiButton>
            </UiTooltip>
          </div>
        </UiField>

        <UiField class="license-field">
          <UiLabel for="license-key">授权码</UiLabel>
          <textarea
            id="license-key"
            v-model="licenseKey"
            class="license-textarea"
            autocomplete="off"
            spellcheck="false"
            rows="5"
            placeholder="GD1-..."
          />
        </UiField>

        <UiButton class="license-submit" type="submit" :disabled="!canSubmit">
          <KeyRound :size="16" aria-hidden="true" />
          <span>{{ appStore.licenseLoading ? '正在激活' : '激活授权' }}</span>
        </UiButton>
      </form>
    </section>
  </main>
</template>

<style scoped src="./LicensePage.css"></style>
