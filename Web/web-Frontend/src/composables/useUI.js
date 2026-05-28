import { reactive, toRefs, watch } from 'vue'
import { storage } from '@/services/storageService'

let instance = null

export function useUI() {
  if (instance) return instance

  const state = reactive({
    panel: false,
    topBtn: false,
    headerHeight: 0,
    showIp: storage.get('showIp') === true,
    modals: { custom: false, key: false, qr: false },
    qrUrl: '',
    server: null
  })

  watch(() => state.showIp, v => storage.set('showIp', v))

  const close = () => state.panel = false
  const open = m => { close(); Object.keys(state.modals).forEach(k => state.modals[k] = k === m) }

  instance = {
    ...toRefs(state),
    close,
    toggle: () => state.panel = !state.panel,
    top: () => window.scrollTo({ top: 0, behavior: 'smooth' }),
    openCustom: () => open('custom'),
    openKey: () => open('key'),
    showQR: async (s, fn) => {
      state.server = s
      const previousUrl = state.qrUrl
      try {
        const nextUrl = URL.createObjectURL(await fn())
        if (previousUrl) URL.revokeObjectURL(previousUrl)
        state.qrUrl = nextUrl
        open('qr')
      } catch (e) {
        state.modals.qr = false
        throw e
      }
    }
  }

  return instance
}