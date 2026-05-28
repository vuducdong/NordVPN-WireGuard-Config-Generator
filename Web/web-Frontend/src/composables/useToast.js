import { ref } from 'vue'

const TIME = 2000
const MAX = 100

let instance = null

export function useToast() {
  if (instance) return instance

  const toast = ref(null)
  let timer = null

  const show = (msg, type = 'success') => {
    if (!msg) return
    const m = (msg instanceof Error ? msg.message : String(msg)).split('\n')[0].slice(0, MAX)
    clearTimeout(timer)
    toast.value = { message: m, type: ['success', 'error'].includes(type) ? type : 'success' }
    timer = setTimeout(() => { toast.value = null }, TIME)
  }

  instance = { toast, show }
  return instance
}