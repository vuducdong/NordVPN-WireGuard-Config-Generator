import { ref } from 'vue'
import { api } from '@/services/apiService'
import { storage } from '@/services/storageService'
import { Validators } from '@/utils/utils'
import { createZipArchive } from '@/utils/zip'

const KEY = 'wg_gen_settings'
const DEF = { dns: '103.86.96.100', endpoint: 'hostname', keepalive: 25 }

export function useConfig() {
  const privKey = ref('')
  const settings = ref({ ...DEF })

  const load = () => {
    const s = storage.get(KEY)
    if (s && Validators.DNS.valid(s.dns) && Validators.Keepalive.valid(s.keepalive)) {
      settings.value = {
        dns: s.dns ?? DEF.dns,
        endpoint: s.endpoint ?? DEF.endpoint,
        keepalive: s.keepalive ?? DEF.keepalive
      }
    }
  }

  const save = s => {
    const next = {
      dns: s.dns ?? settings.value.dns,
      endpoint: s.endpoint ?? settings.value.endpoint,
      keepalive: s.keepalive ?? settings.value.keepalive
    }
    
    if (Validators.DNS.valid(next.dns) && Validators.Keepalive.valid(next.keepalive)) {
      storage.set(KEY, next)
      settings.value = next
    }
  }

  const make = s => ({
    name: s.name,
    dns: settings.value.dns,
    endpoint: settings.value.endpoint,
    keepalive: settings.value.keepalive
  })

  const saveBlob = (blob, name) => {
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = name
    a.click()
    URL.revokeObjectURL(url)
  }

  const dl = async s => {
    if (privKey.value) {
      const payload = { ...make(s), mode: "client" }
      const { filename, template } = await api.genConfig(payload)
      const finalConfig = template.replace("__CLIENT_PK__", privKey.value)
      const blob = new Blob([finalConfig], { type: 'application/x-wireguard-config' })
      saveBlob(blob, filename)
    } else {
      const payload = { ...make(s), mode: "server" }
      const { blob, name } = await api.dlConfig(payload)
      saveBlob(blob, name || `${s.name}.conf`)
    }
  }

  const dlBatch = async (filters = {}) => {
    const payload = {
      dns: settings.value.dns,
      endpoint: settings.value.endpoint,
      keepalive: settings.value.keepalive,
      country: filters.country || '',
      city: filters.city || ''
    }

    if (privKey.value) {
      payload.mode = "client"
      const { archiveName, templates } = await api.dlBatch(payload)
      const entries = templates.map(t => {
        const finalConfig = t.template.replace("__CLIENT_PK__", privKey.value)
        return {
          name: t.name,
          data: new TextEncoder().encode(finalConfig)
        }
      })
      const zipData = createZipArchive(entries)
      const blob = new Blob([zipData], { type: 'application/zip' })
      saveBlob(blob, `${archiveName}.zip`)
    } else {
      payload.mode = "server"
      const { blob, name } = await api.dlBatch(payload)
      saveBlob(blob, name)
    }
  }

  return {
    privKey,
    settings,
    defaults: DEF,
    load,
    save,
    setKey: k => { if (Validators.Key.valid(k)) privKey.value = k; else throw new Error(Validators.Key.err) },
    dl,
    dlBatch,
    copy: async s => {
      if (privKey.value) {
        const payload = { ...make(s), mode: "client" }
        const { template } = await api.genConfig(payload)
        const finalConfig = template.replace("__CLIENT_PK__", privKey.value)
        return navigator.clipboard.writeText(finalConfig)
      } else {
        const { template } = await api.genConfig({ ...make(s), mode: "server" })
        return navigator.clipboard.writeText(template)
      }
    },
    make
  }
}