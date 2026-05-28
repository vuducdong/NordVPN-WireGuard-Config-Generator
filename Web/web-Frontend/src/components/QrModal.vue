<script setup>
import { useUI } from '@/composables/useUI'
import Icon from '@/components/Icon.vue'

const ui = useUI()
</script>

<template>
  <div class="fixed inset-0 z-100 flex items-center justify-center p-4" @click="ui.modals.value.qr = false">
    <div class="fixed inset-0 bg-nord-bg-overlay" />
    <div class="relative bg-vscode-bg rounded-lg border border-vscode-active overflow-hidden max-w-sm w-full" @click.stop>
      <div class="bg-nord-bg-overlay-light px-3 py-1.5 text-xs font-medium border-b border-vscode-active flex items-center justify-between text-nord-text-primary">
        <span>{{ ui.server.value?.dName }}</span>
        <button @click="ui.modals.value.qr = false" class="p-1 rounded hover:bg-nord-bg-hover"><Icon name="close" class="w-3.5 h-3.5" /></button>
      </div>
      <div class="p-6 flex justify-center">
        <img :src="ui.qrUrl.value" class="w-50 h-50 rounded">
      </div>
    </div>
  </div>
</template>