<script setup>
import { onMounted, onBeforeUnmount } from 'vue'
import { useServers } from '@/composables/useServers'
import { useConfig } from '@/composables/useConfig'
import { useUI } from '@/composables/useUI'
import { useToast } from '@/composables/useToast'
import AppHeader from '@/components/AppHeader.vue'
import AppSidebar from '@/components/AppSidebar.vue'
import ServerGrid from '@/components/ServerGrid.vue'
import Toast from '@/components/Toast.vue'
import Icon from '@/components/Icon.vue'
import ConfigCustomizer from '@/components/ConfigCustomizer.vue'
import KeyGenerator from '@/components/KeyGenerator.vue'
import QrModal from '@/components/QrModal.vue'

const srv = useServers()
const cfg = useConfig()
const ui = useUI()
const notif = useToast()

let ticking = false

const onScroll = () => {
  if (!ticking) {
    window.requestAnimationFrame(() => {
      ui.topBtn.value = window.scrollY > 500
      ticking = false
    })
    ticking = true
  }
}

onMounted(async () => {
  window.scrollTo(0, 0)
  cfg.load()
  await srv.init()
  window.addEventListener('scroll', onScroll, { passive: true })
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
})
</script>

<template>
  <Toast v-if="notif.toast.value" :msg="notif.toast.value.message" :type="notif.toast.value.type" @close="notif.toast.value = null" />

  <QrModal v-if="ui.modals.value.qr" />
  <KeyGenerator v-if="ui.modals.value.key" />
  <ConfigCustomizer v-if="ui.modals.value.custom" />

  <div v-show="!ui.modals.value.key && !ui.modals.value.custom" class="min-h-screen bg-vscode-bg text-vscode-text">
    <AppHeader />

    <div class="fixed inset-0 bg-nord-bg-overlay/30 z-30 transition-opacity" :class="ui.panel.value ? 'opacity-100' : 'opacity-0 pointer-events-none'" @click="ui.close" />
    <AppSidebar />

    <ServerGrid />

    <button v-show="ui.topBtn.value" @click="ui.top" class="fixed bottom-4 right-4 p-2 rounded-full bg-vscode-header/90 border border-vscode-accent z-50 hover:bg-vscode-header"><Icon name="arrowUp" class="w-5 h-5" /></button>
  </div>
</template>