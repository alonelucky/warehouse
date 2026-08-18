export type MovementType = 'in' | 'out' | 'adjust'

export interface Product {
  id: number
  name: string
  spec: string
  unit: string
  category: string
  stock: number
  avgPriceCents: number
  locations: string
  createdAt: string
  updatedAt: string
}

export interface ProductInput {
  name: string
  spec: string
  unit: string
  category: string
}

export interface Movement {
  id: number
  type: MovementType
  productId: number
  productName: string
  qty: number
  stockBefore: number
  stockAfter: number
  targetQty?: number
  locations: string
  batchRefs: string
  counterparty: string
  note: string
  createdAt: string
}

export interface MovementInput {
  type: MovementType
  productId: number
  qty: number
  counterparty: string
  note: string
  unitPriceCents: number
  location: string
}

export interface BatchItemInput {
  productName: string
  spec: string
  type: MovementType
  qty: number
  unitPriceCents: number
  location: string
  counterparty: string
  note: string
}

export interface Location {
  id: number
  name: string
  qtyLeft: number
  createdAt: string
}

export interface Batch {
  id: number
  productId: number
  productName: string
  qty: number
  qtyLeft: number
  unitPriceCents: number
  location: string
  supplier: string
  note: string
  createdAt: string
}

export interface AuditEntry {
  id: number
  ts: string
  operator: string
  action: string
  entity: string
  entityId?: number
  detail: Record<string, unknown>
}

export interface Stats {
  skuCount: number
  totalUnits: number
  inventoryValue: number
  inToday: number
  outToday: number
  inTotal: number
  outTotal: number
  movementCount: number
}
