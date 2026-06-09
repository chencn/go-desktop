<script setup lang="ts">
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

defineProps<{
  // content 是图标按钮的可访问解释文案，触发元素由默认 slot 提供。
  content: string
}>()
</script>

<template>
  <TooltipProvider>
    <Tooltip>
      <TooltipTrigger as-child>
        <slot />
      </TooltipTrigger>
      <TooltipContent>
        {{ content }}
      </TooltipContent>
    </Tooltip>
  </TooltipProvider>
</template>
