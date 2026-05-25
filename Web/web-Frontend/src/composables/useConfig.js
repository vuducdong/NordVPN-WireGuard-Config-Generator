import { ref } from 'vue'
import { storage } from '@/services/storageService'
import { Validators } from '@/utils/utils'
import { createZipArchive } from '@/utils/zip'

const KEY = 'wg_gen_settings'
const DEF = { dns: '103.86.96.100', endpoint: 'hostname', keepalive: 25 }

function buildWireGuardConfig(privateKey, dns, publicKey, endpoint, keepalive) {
  return `[Interface]
PrivateKey=${privateKey || ""}
Address=10.5.0.2/16
DNS=${dns}

[Peer]
PublicKey=${publicKey}
AllowedIPs=0.0.0.0/0,::/0
Endpoint=${endpoint}:51820
PersistentKeepalive=${keepalive}`;
}

function buildBatchFilePath(batchCountry, batchCity, s) {
  if (batchCity !== "") {
    return s.fileName
  }
  if (batchCountry === "") {
    return `${s.country}/${s.city}/${s.fileName}`
  }
  return `${s.city}/${s.fileName}`
}

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

  const saveBlob = (blob, name) => {
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = name
    a.click()
    URL.revokeObjectURL(url)
  }

  const dl = s => {
    const endpoint = settings.value.endpoint === 'station' ? s.ip : s.endpoint
    const configText = buildWireGuardConfig(
      privKey.value,
      settings.value.dns,
      s.publicKey,
      endpoint,
      settings.value.keepalive
    )
    const blob = new Blob([configText], { type: 'application/x-wireguard-config' })
    saveBlob(blob, s.fileName)
  }

  const dlBatch = (servers, filters = {}) => {
    const targetCountry = filters.country || ''
    const targetCity = filters.city || ''

    const list = servers.value
    if (!list || list.length === 0) throw new Error("No configurations found")

    let archiveName = "NordVPN_All"
    if (targetCountry !== "") {
      archiveName = `NordVPN_${targetCountry.replace(/[^a-zA-Z0-9-_]/g, "_")}`
      if (targetCity !== "") {
        archiveName += `_${targetCity.replace(/[^a-zA-Z0-9-_]/g, "_")}`
      }
    }

    const entries = list.map(s => {
      const endpoint = settings.value.endpoint === 'station' ? s.ip : s.endpoint
      const configText = buildWireGuardConfig(
        privKey.value,
        settings.value.dns,
        s.publicKey,
        endpoint,
        settings.value.keepalive
      )
      const zipPath = buildBatchFilePath(targetCountry, targetCity, s)
      return {
        name: zipPath,
        data: new TextEncoder().encode(configText)
      }
    })
    
    const zipData = createZipArchive(entries)
    const blob = new Blob([zipData], { type: 'application/zip' })
    saveBlob(blob, `${archiveName}.zip`)
  }

  const copy = s => {
    const endpoint = settings.value.endpoint === 'station' ? s.ip : s.endpoint
    const configText = buildWireGuardConfig(
      privKey.value,
      settings.value.dns,
      s.publicKey,
      endpoint,
      settings.value.keepalive
    )
    return navigator.clipboard.writeText(configText)
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
    copy
  }
}