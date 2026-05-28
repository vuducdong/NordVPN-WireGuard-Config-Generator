<script setup>
import { useUI } from '@/composables/useUI'
import Icon from '@/components/Icon.vue'

const ui = useUI()
</script>

<template>
  <aside class="fixed inset-y-0 left-0 w-1/2 sm:w-63 bg-vscode-header border-r border-vscode-active z-40 transition-transform flex flex-col" :class="ui.panel.value ? 'translate-x-0' : '-translate-x-full'">
    <div :style="{ height: ui.headerHeight.value + 'px' }" class="shrink-0 bg-vscode-header transition-[height]" />
    <div class="flex-1 overflow-y-auto p-4 space-y-3">
      <button @click="ui.openCustom" class="w-full px-2 sm:px-4 py-2 rounded border border-vscode-active bg-nord-bg-overlay/20 text-xs sm:text-sm hover:bg-nord-bg-hover"><div class="flex items-center gap-1.5 sm:gap-2"><Icon name="settings" class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-vscode-accent" /><span>Customize</span></div></button>
      <button @click="ui.openKey" class="w-full px-2 sm:px-4 py-2 rounded border border-vscode-active bg-nord-bg-overlay/20 text-xs sm:text-sm hover:bg-nord-bg-hover"><div class="flex items-center gap-1.5 sm:gap-2"><Icon name="key" class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-vscode-accent" /><span>Generate Key</span></div></button>
      <label class="flex items-center justify-between cursor-pointer group mt-4 select-none">
        <span class="text-xs sm:text-sm font-medium text-nord-text-primary">Show IP</span>
        <button type="button" @click="ui.showIp.value = !ui.showIp.value" class="relative w-8 h-4 sm:w-10 sm:h-5 rounded-full transition-colors" :class="ui.showIp.value ? 'bg-nord-button-primary' : 'bg-nord-button-secondary'">
          <span class="absolute left-0.5 top-0.5 w-3 h-3 sm:w-4 sm:h-4 rounded-full bg-white transition-transform" :class="ui.showIp.value ? 'translate-x-4 sm:translate-x-4.5' : ''" />
        </button>
      </label>
    </div>
    <div class="p-4 space-y-3 border-t border-vscode-active bg-nord-bg-overlay/20">
      <a href="https://github.com/mustafachyi/NordVPN-WireGuard-Config-Generator" target="_blank" rel="noopener noreferrer" class="block w-full px-2 sm:px-4 py-2 rounded border border-vscode-active bg-nord-bg-overlay/20 text-xs sm:text-sm hover:bg-nord-bg-hover"><div class="flex items-center gap-1.5 sm:gap-2"><Icon name="github" class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-vscode-accent" /><span>Star on GitHub</span></div></a>
      <a href="https://refer-nordvpn.com/MXIVDoJGpKT" target="_blank" rel="noopener noreferrer" class="block w-full px-2 sm:px-4 py-2 rounded border border-vscode-active bg-nord-bg-overlay/20 text-xs sm:text-sm hover:bg-nord-bg-hover"><div class="flex items-center gap-1.5 sm:gap-2"><Icon name="externalLink" class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-vscode-accent" /><span>Get NordVPN</span></div></a>
    </div>
  </aside>
</template>