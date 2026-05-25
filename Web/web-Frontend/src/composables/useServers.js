import { shallowRef, computed, watch, markRaw } from 'vue'
import { formatName } from '@/utils/utils'
import { api } from '@/services/apiService'

const INC = 24

export function useServers() {
  const all = shallowRef([])
  const loading = shallowRef(false)
  const error = shallowRef('')
  const sortKey = shallowRef('name')
  const sortOrd = shallowRef('asc')
  const fCountry = shallowRef('')
  const fCity = shallowRef('')
  const limit = shallowRef(INC)

  const countries = shallowRef([])
  const cityMap = shallowRef({})

  const filtered = computed(() => {
    const c = fCountry.value
    const t = fCity.value
    let list = all.value

    if (c) list = list.filter(s => s.country === c)
    if (t) list = list.filter(s => s.city === t)

    const k = sortKey.value
    const m = sortOrd.value === 'asc' ? 1 : -1

    return [...list].sort((a, b) => {
      if (k === 'load') {
        const d = a.load - b.load
        if (d !== 0) return d * m
      }
      return (a.dName > b.dName ? 1 : -1) * m
    })
  })

  const visible = computed(() => filtered.value.slice(0, limit.value))
  const total = computed(() => filtered.value.length)
  const currentCities = computed(() => cityMap.value[fCountry.value] || [])

  const reset = () => {
    limit.value = INC
    window.scrollTo(0, 0)
  }

  watch(fCountry, () => {
    const l = cityMap.value[fCountry.value] || []
    fCity.value = l.length === 1 ? l[0].id : ''
    reset()
  })

  watch([fCity, sortKey, sortOrd], reset)

  const processServerData = (k, l) => {
    const list = []
    const cList = []
    const cMap = {}
    const fmtCache = new Map()

    const getFmt = s => {
      if (fmtCache.has(s)) return fmtCache.get(s)
      const v = formatName(s)
      fmtCache.set(s, v)
      return v
    }

    for (const countryData of l) {
      const cn = countryData[0]
      const lowCode = countryData[1]
      const cities = countryData[2]

      const dCountry = getFmt(cn)
      cList.push({ id: cn, name: dCountry })

      const cityList = []

      for (const cityData of cities) {
        const ci = cityData[0]
        const servers = cityData[1]
        
        const dCity = getFmt(ci)
        cityList.push({ id: ci, name: dCity })

        for (const t of servers) {
          const num = t[0]
          const load = t[1]
          const ipNumeric = t[2]
          const keyIdx = t[3]
          const hName = t[4] || ""
          const dedup = t[5] || ""

          const ip = [
            (ipNumeric >>> 24) & 255,
            (ipNumeric >>> 16) & 255,
            (ipNumeric >>> 8) & 255,
            ipNumeric & 255
          ].join('.')

          const prefix = lowCode === "gb" ? "uk" : lowCode;
          const endpoint = hName || `${prefix}${num}.nordvpn.com`;
          const publicKey = k[keyIdx]

          const fileName = `${lowCode}${num}${dedup}.conf`
          const dName = `${dCountry} ${num}${dedup ? ` (${dedup.replace('_', '')})` : ''}`

          list.push(markRaw({
            name: fileName,
            load,
            ip,
            publicKey,
            endpoint,
            fileName,
            country: cn,
            city: ci,
            dName,
            dCountry,
            dCity
          }))
        }
      }
      cMap[cn] = cityList
    }

    all.value = list
    countries.value = cList
    cityMap.value = cMap
  }

  const init = async () => {
    loading.value = true
    error.value = ''

    try {
      if (window.__SERVER_LIST__ && window.__SERVER_LIST__.k && window.__SERVER_LIST__.l) {
        const { k, l } = window.__SERVER_LIST__
        processServerData(k, l)
        loading.value = false
        return
      }

      const { k, l } = await api.getServers()
      processServerData(k, l)
    } catch (e) {
      error.value = e.message || 'Failed to load servers'
      console.error(e)
    } finally {
      loading.value = false
    }
  }

  return {
    filtered,
    visible,
    loading,
    error,
    sortKey,
    sortOrd,
    fCountry,
    fCity,
    countries,
    cities: currentCities,
    total,
    loadMore: () => { if (!loading.value && limit.value < total.value) limit.value += INC },
    toggleSort: k => {
      if (sortKey.value === k) sortOrd.value = sortOrd.value === 'asc' ? 'desc' : 'asc'
      else { sortKey.value = k; sortOrd.value = 'asc' }
    },
    init
  }
}