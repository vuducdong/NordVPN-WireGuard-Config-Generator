<script setup>
import { ref, onMounted, onBeforeUnmount, nextTick, watch, computed } from 'vue'
import { useServers } from '@/composables/useServers'
import { useConfig } from '@/composables/useConfig'
import { useUI } from '@/composables/useUI'
import { useToast } from '@/composables/useToast'
import ServerCard from '@/components/ServerCard.vue'
import Icon from '@/components/Icon.vue'

const srv = useServers()
const cfg = useConfig()
const ui = useUI()
const notif = useToast()

const sentinel = ref(null)
let obs = null

const emptyMsg = computed(() => {
  if (srv.error.value) return srv.error.value
  if (srv.fCountry.value || srv.fGroup.value) return 'No servers match criteria.'
  return 'No servers loaded.'
})

const dl = s => {
  try {
    cfg.dl(s)
    notif.show('Downloaded', 'success')
  } catch (e) {
    notif.show(e.message || 'Download failed', 'error')
  }
}

const cp = async s => {
  try {
    await cfg.copy(s)
    notif.show('Copied', 'success')
  } catch (e) {
    notif.show(e.message || 'Copy failed', 'error')
  }
}

const qr = s => {
  ui.showQR(s, () => cfg.getQrBlob(s)).catch(e => notif.show(e.message || 'QR generation failed', 'error'))
}

const observe = async () => {
  await nextTick()
  obs?.disconnect()
  if (sentinel.value) obs?.observe(sentinel.value)
}

onMounted(() => {
  obs = new IntersectionObserver(e => {
    if (e[0].isIntersecting) srv.loadMore()
  }, { rootMargin: '200px' })
  observe()
})

onBeforeUnmount(() => {
  obs?.disconnect()
})

watch([srv.fGroup, srv.fCountry, srv.fCity], observe)
</script>

<template>
  <main class="container mx-auto px-4 py-6">
    <div v-if="srv.visible.value.length > 0" class="server-grid grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2 mx-auto group/grid" :class="{ 'show-ips': ui.showIp.value }">
      <ServerCard v-for="s in srv.visible.value" v-memo="[s]" :key="s.name" :s="s" @download="dl(s)" @copy="cp(s)" @show-qr="qr(s)" @copy-ip="notif.show('IP copied', 'success')" />
    </div>
    <div v-else-if="!srv.loading.value" class="text-center py-20">
      <Icon name="error" class="w-12 h-12 mx-auto text-nord-text-secondary/50 mb-4" />
      <p class="text-nord-text-secondary font-medium">{{ emptyMsg }}</p>
    </div>
    <div ref="sentinel" class="h-10" />
    <div v-if="srv.loading.value" class="flex justify-center py-4"><div class="w-6 h-6 border-2 border-vscode-accent border-t-transparent rounded-full animate-spin" /></div>
  </main>
</template>