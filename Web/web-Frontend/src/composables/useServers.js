import { shallowRef, computed, watch, markRaw } from 'vue'
import { formatName } from '@/utils/utils'
import { api } from '@/services/apiService'

const INC = 24
const GROUPS = [
  { id: 1, name: 'Standard' },
  { id: 2, name: 'P2P' },
  { id: 4, name: 'Dedicated IP' },
  { id: 8, name: 'Onion Over VPN' },
  { id: 16, name: 'Double VPN' }
]

let instance = null

export function useServers() {
  if (instance) return instance

  const all = shallowRef([])
  const loading = shallowRef(false)
  const error = shallowRef('')
  const sortKey = shallowRef('name')
  const sortOrd = shallowRef('asc')
  const fGroup = shallowRef('')
  const fCountry = shallowRef('')
  const fCity = shallowRef('')
  const limit = shallowRef(INC)

  const availableCountries = computed(() => {
    const g = parseInt(fGroup.value, 10)
    const map = new Map()
    for (const s of all.value) {
      if (!g || (s.groupMask & g)) {
        if (!map.has(s.country)) map.set(s.country, { id: s.country, name: s.dCountry })
      }
    }
    return Array.from(map.values()).sort((a, b) => a.name.localeCompare(b.name))
  })

  const availableCities = computed(() => {
    if (!fCountry.value) return []
    const g = parseInt(fGroup.value, 10)
    const map = new Map()
    for (const s of all.value) {
      if (s.country === fCountry.value && (!g || (s.groupMask & g))) {
        if (!map.has(s.city)) map.set(s.city, { id: s.city, name: s.dCity })
      }
    }
    return Array.from(map.values()).sort((a, b) => a.name.localeCompare(b.name))
  })

  watch(availableCountries, (cList) => {
    if (fCountry.value && !cList.some(c => c.id === fCountry.value)) {
      fCountry.value = ''
    }
  })

  watch(availableCities, (cList) => {
    if (cList.length === 1) {
      fCity.value = cList[0].id
    } else if (fCity.value && !cList.some(c => c.id === fCity.value)) {
      fCity.value = ''
    }
  })

  const filtered = computed(() => {
    const g = parseInt(fGroup.value, 10)
    const c = fCountry.value
    const t = fCity.value
    let list = all.value

    if (g) list = list.filter(s => s.groupMask & g)
    if (c) list = list.filter(s => s.country === c)
    if (t) list = list.filter(s => s.city === t)

    const k = sortKey.value
    const m = sortOrd.value === 'asc' ? 1 : -1

    return list.slice().sort((a, b) => {
      if (k === 'load') {
        const d = a.load - b.load
        if (d !== 0) return d * m
      }
      return (a.dName > b.dName ? 1 : -1) * m
    })
  })

  const visible = computed(() => filtered.value.slice(0, limit.value))
  const total = computed(() => filtered.value.length)
  const groupName = computed(() => GROUPS.find(g => g.id === parseInt(fGroup.value, 10))?.name || '')

  const reset = () => {
    limit.value = INC
    window.scrollTo(0, 0)
  }

  watch([fGroup, fCountry, fCity, sortKey, sortOrd], reset)

  const processServerData = (payload) => {
    const [keysStr, rawCountries] = payload
    const keys = []
    for (let i = 0; i < keysStr.length; i += 43) {
      keys.push(keysStr.slice(i, i + 43) + "=")
    }

    const list = []
    const fmtCache = new Map()

    const getFmt = s => {
      if (fmtCache.has(s)) return fmtCache.get(s)
      const v = formatName(s)
      fmtCache.set(s, v)
      return v
    }

    for (const countryData of rawCountries) {
      const [cName, lowCode, cities] = countryData
      const dCountry = getFmt(cName)
      const prefix = lowCode === "gb" ? "uk" : lowCode

      for (const cityData of cities) {
        const ciName = cityData[0]
        const defKey = cityData[1]
        const defGrp = cityData[2]
        const dCity = getFmt(ciName)

        const lastEl = cityData[cityData.length - 1]
        const hasExceptions = Array.isArray(lastEl)
        const exceptions = hasExceptions ? lastEl : []
        const len = hasExceptions ? cityData.length - 1 : cityData.length

        let lastIp = 0
        let lastNum = 0
        let excIdx = 0

        for (let i = 3; i < len; i += 2) {
          const packed = cityData[i]
          const dIp = cityData[i + 1]

          lastIp += dIp
          const ip = [
            (lastIp >>> 24) & 255,
            (lastIp >>> 16) & 255,
            (lastIp >>> 8) & 255,
            lastIp & 255
          ].join('.')

          const load = packed & 0x7F
          const isExc = packed < 0

          let hName = "", dedup = "", keyIdx = defKey, grp = defGrp
          let numberStr
          let isNumeric = true

          if (isExc) {
            const exc = exceptions[excIdx++]
            const idVal = exc[0]
            if (typeof idVal === 'number') {
              lastNum = idVal
              numberStr = String(idVal)
            } else {
              numberStr = idVal
              isNumeric = false
            }

            if (exc.length > 1 && exc[1] !== -1) keyIdx = exc[1]
            if (exc.length > 2 && exc[2] !== -1) grp = exc[2]
            if (exc.length > 3 && exc[3]) hName = exc[3]
            if (exc.length > 4 && exc[4]) dedup = exc[4]
          } else {
            const dNum = packed >> 7
            lastNum += dNum
            numberStr = String(lastNum)
          }

          const baseHost = isNumeric ? `${prefix}${numberStr}` : numberStr
          const endpoint = hName || `${baseHost}.nordvpn.com`

          const baseFile = isNumeric ? `${prefix}${numberStr}` : numberStr
          const fileName = `${baseFile}${dedup}.conf`
          
          const dName = `${dCountry} ${numberStr}${dedup ? ` (${dedup.replace('_', '')})` : ''}`
          const publicKey = keys[keyIdx]

          list.push(markRaw({
            name: fileName,
            load,
            ip,
            publicKey,
            endpoint,
            fileName,
            country: cName,
            city: ciName,
            dName,
            dCountry,
            dCity,
            groupMask: grp
          }))
        }
      }
    }

    all.value = list
  }

  const init = async () => {
    loading.value = true
    error.value = ''

    try {
      if (window.__SERVER_LIST__ && Array.isArray(window.__SERVER_LIST__)) {
        processServerData(window.__SERVER_LIST__)
        loading.value = false
        return
      }

      const payload = await api.getServers()
      processServerData(payload)
    } catch (e) {
      error.value = e.message || 'Failed to load servers'
      console.error(e)
    } finally {
      loading.value = false
    }
  }

  instance = {
    filtered,
    visible,
    loading,
    error,
    sortKey,
    sortOrd,
    fGroup,
    fCountry,
    fCity,
    groups: GROUPS,
    countries: availableCountries,
    cities: availableCities,
    total,
    groupName,
    loadMore: () => { if (!loading.value && limit.value < total.value) limit.value += INC },
    toggleSort: k => {
      if (sortKey.value === k) sortOrd.value = sortOrd.value === 'asc' ? 'desc' : 'asc'
      else { sortKey.value = k; sortOrd.value = 'asc' }
    },
    init
  }

  return instance
}