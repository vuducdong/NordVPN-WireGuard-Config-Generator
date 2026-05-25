const BASE = import.meta.env.VITE_API_BASE || '/api'
const TIMEOUT = 60000

async function req(end, opt = {}) {
  const c = new AbortController()
  const id = setTimeout(() => c.abort(), TIMEOUT)
  try {
    const isGet = !opt.method || opt.method.toUpperCase() === 'GET'
    const headers = { ...opt.headers }
    if (!isGet) headers['Content-Type'] = 'application/json'

    const r = await fetch(`${BASE}${end}`, {
      ...opt,
      headers,
      signal: c.signal
    })

    if (!r.ok) {
      let m = `HTTP ${r.status}`
      try {
        const d = await r.json()
        if (d?.error) m = d.error
      } catch {}
      const e = new Error(m)
      e.status = r.status
      throw e
    }

    const t = (r.headers.get('content-type') || '').toLowerCase()
    return t.includes('application/json') ? r.json() : r.text()
  } catch (e) {
    throw e.name === 'AbortError' ? new Error('Request timeout') : e
  } finally {
    clearTimeout(id)
  }
}

export const api = {
  getServers: () => req('/servers'),
  genKey: token => req('/key', { method: 'POST', body: JSON.stringify({ token }) })
}