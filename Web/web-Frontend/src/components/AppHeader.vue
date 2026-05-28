<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useServers } from '@/composables/useServers'
import { useConfig } from '@/composables/useConfig'
import { useUI } from '@/composables/useUI'
import { useToast } from '@/composables/useToast'
import Icon from '@/components/Icon.vue'

const srv = useServers()
const cfg = useConfig()
const ui = useUI()
const notif = useToast()

const headerRef = ref(null)
const dlLoading = ref(false)
let ro = null

const dlLabel = computed(() => dlLoading.value ? 'Processing...' : srv.fCity.value ? 'Download City' : srv.fCountry.value ? 'Download Country' : srv.fGroup.value ? 'Download Group' : 'Download All')

const dlBatch = async () => {
  if (dlLoading.value) return
  dlLoading.value = true
  notif.show('Compressing...', 'success')
  
  await new Promise(r => setTimeout(r, 50))
  
  try {
    cfg.dlBatch(srv.filtered.value, { group: srv.groupName.value, country: srv.fCountry.value, city: srv.fCity.value })
    notif.show('Download started', 'success')
  } catch (e) {
    notif.show(e.message || 'Batch download failed', 'error')
  } finally {
    dlLoading.value = false
  }
}

onMounted(() => {
  ro = new ResizeObserver(() => { ui.headerHeight.value = headerRef.value?.offsetHeight || 0 })
  if (headerRef.value) ro.observe(headerRef.value)
})

onBeforeUnmount(() => {
  ro?.disconnect()
})
</script>

<template>
  <header ref="headerRef" class="sticky top-0 z-50 bg-vscode-header border-b border-vscode-active">
    <div class="flex flex-col sm:flex-row sm:items-center gap-2 p-2">
      <nav class="flex items-center gap-2 flex-1 min-w-0">
        <button @click="ui.toggle" class="shrink-0 p-2 flex items-center justify-center rounded hover:bg-nord-bg-hover"><Icon name="menu" class="w-5 h-5" /></button>
        <div class="flex-1 flex gap-2 min-w-0" @click="ui.close">
          <select v-model="srv.fGroup.value" class="w-full truncate bg-vscode-bg border border-vscode-active rounded px-2 py-1.5 text-sm sm:w-40">
            <option value="">All Groups</option>
            <option v-for="g in srv.groups" :key="g.id" :value="g.id">{{ g.name }}</option>
          </select>
          <select v-model="srv.fCountry.value" :disabled="srv.countries.value.length === 0" class="w-full truncate bg-vscode-bg border border-vscode-active rounded px-2 py-1.5 text-sm sm:w-50 disabled:opacity-50">
            <option value="">All Countries</option>
            <option v-for="c in srv.countries.value" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
          <div v-if="srv.fCountry.value" class="w-full sm:w-50 min-w-0">
            <select v-model="srv.fCity.value" :disabled="srv.cities.value.length < 2" class="w-full truncate bg-vscode-bg border border-vscode-active rounded px-2 py-1.5 text-sm disabled:opacity-50">
              <option v-if="srv.cities.value.length > 1" value="">All Cities</option>
              <option v-for="c in srv.cities.value" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
          </div>
        </div>
      </nav>
      <div class="sm:pl-0 pl-11">
        <div class="flex flex-wrap items-center justify-end gap-2 text-xs" @click="ui.close">
          <button @click="dlBatch" :disabled="dlLoading" class="w-full sm:w-auto flex items-center justify-center gap-1.5 px-3 py-1.5 rounded bg-nord-button-primary text-white font-semibold hover:bg-nord-button-primary-hover disabled:opacity-50 disabled:cursor-not-allowed transition-colors">
            <div v-if="dlLoading" class="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" />
            <Icon v-else name="archive" class="w-4 h-4" />
            <span class="whitespace-nowrap">{{ dlLabel }}</span>
          </button>
          <button @click="srv.toggleSort('load')" class="flex-1 sm:flex-none flex items-center justify-center gap-1 sm:min-w-20 px-2 sm:px-3 py-1.5 rounded border font-semibold transition-colors" :class="srv.sortKey.value === 'load' ? 'bg-nord-bg-active border-vscode-accent text-white' : 'border-vscode-active hover:bg-nord-bg-hover'">
            <span>Load</span><Icon v-if="srv.sortKey.value === 'load'" :name="srv.sortOrd.value === 'asc' ? 'sortAsc' : 'sortDesc'" class="w-4 h-4" />
          </button>
          <button @click="srv.toggleSort('name')" class="flex-1 sm:flex-none flex items-center justify-center gap-1 sm:min-w-20 px-2 sm:px-3 py-1.5 rounded border font-semibold transition-colors" :class="srv.sortKey.value === 'name' ? 'bg-nord-bg-active border-vscode-accent text-white' : 'border-vscode-active hover:bg-nord-bg-hover'">
            <span>A-Z</span><Icon v-if="srv.sortKey.value === 'name'" :name="srv.sortOrd.value === 'asc' ? 'sortAsc' : 'sortDesc'" class="w-4 h-4" />
          </button>
          <div class="px-3 py-1.5 rounded bg-vscode-bg/50 border border-vscode-active/50"><span class="text-xs text-nord-text-secondary font-semibold">{{ srv.total }}</span></div>
        </div>
      </div>
    </div>
  </header>
</template>