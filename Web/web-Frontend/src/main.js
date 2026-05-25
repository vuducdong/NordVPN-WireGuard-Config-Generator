import { createApp } from 'vue'
import '@/style.css'
import App from '@/App.vue'
import { storage } from '@/services/storageService'

createApp(App).mount('#app')

setTimeout(() => {
  storage.clean()

  if ('serviceWorker' in navigator) {
    navigator.serviceWorker.getRegistrations().then(registrations => {
      for (const registration of registrations) {
        registration.unregister()
      }
    })
  }

  if ('caches' in window) {
    caches.keys().then(keys => {
      for (const key of keys) {
        caches.delete(key)
      }
    })
  }
}, 0)