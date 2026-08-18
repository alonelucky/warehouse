import type {
  AuditEntry,
  Batch,
  BatchItemInput,
  Location,
  Movement,
  MovementInput,
  Product,
  ProductInput,
  Stats,
} from './types'

interface Envelope<T> {
  code: number
  msg: string
  data: T
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  const body = await res.json().catch(() => null)
  if (!body || body.code !== 0) {
    const msg = body && typeof body.msg === 'string' ? body.msg : `请求失败 (${res.status})`
    throw new Error(msg)
  }
  return (body as Envelope<T>).data
}

export const api = {
  meta: () => request<{ operator: string }>('/api/meta'),
  stats: () => request<Stats>('/api/stats'),
  products: (q = '') => request<Product[]>(`/api/products${q ? `?q=${encodeURIComponent(q)}` : ''}`),
  createProduct: (inp: ProductInput) =>
    request<Product>('/api/products', { method: 'POST', body: JSON.stringify(inp) }),
  updateProduct: (id: number, inp: ProductInput) =>
    request<Product>(`/api/products/${id}`, { method: 'PUT', body: JSON.stringify(inp) }),
  movements: (opts: { type?: string; q?: string } = {}) => {
    const p = new URLSearchParams()
    if (opts.type) p.set('type', opts.type)
    if (opts.q) p.set('q', opts.q)
    const qs = p.toString()
    return request<Movement[]>(`/api/movements${qs ? `?${qs}` : ''}`)
  },
  addMovement: (inp: MovementInput) =>
    request<Movement>('/api/movements', { method: 'POST', body: JSON.stringify(inp) }),
  batchMovements: (items: BatchItemInput[]) =>
    request<{ count: number }>('/api/movements/batch', { method: 'POST', body: JSON.stringify({ items }) }),
  locations: () => request<Location[]>('/api/locations'),
  batches: (opts: { productId?: number; q?: string } = {}) => {
    const p = new URLSearchParams()
    if (opts.productId) p.set('productId', String(opts.productId))
    if (opts.q) p.set('q', opts.q)
    const qs = p.toString()
    return request<Batch[]>(`/api/batches${qs ? `?${qs}` : ''}`)
  },
  audit: (opts: { action?: string; q?: string } = {}) => {
    const p = new URLSearchParams()
    if (opts.action) p.set('action', opts.action)
    if (opts.q) p.set('q', opts.q)
    const qs = p.toString()
    return request<AuditEntry[]>(`/api/audit${qs ? `?${qs}` : ''}`)
  },
}
