<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ArrowDownToLine, ArrowUpFromLine, Scale, X } from 'lucide-vue-next'
import type { Batch, Location, MovementInput, MovementType, Product } from '../types'

const props = defineProps<{
  products: Product[]
  locations: Location[]
  batches: Batch[]
  open: boolean
  type: MovementType
  product?: Product
}>()

const emit = defineEmits<{
  close: []
  submit: [payload: MovementInput]
}>()

const form = reactive({ productId: 0, qty: 1, counterparty: '', note: '', unitPrice: '', location: '' })
const err = ref('')

const meta = computed(() => {
  switch (props.type) {
    case 'in':
      return { title: '入库', icon: ArrowDownToLine, counter: '供应商' }
    case 'out':
      return { title: '出库', icon: ArrowUpFromLine, counter: '领用 / 去向' }
    default:
      return { title: '盘点调整', icon: Scale, counter: '' }
  }
})

const selected = computed(() => props.products.find((p) => p.id === form.productId))

const locationAvail = computed(() => {
  if (!form.location || !form.productId || props.type === 'in') return null
  return props.batches
    .filter((b) => b.productId === form.productId && b.location === form.location && b.qtyLeft > 0)
    .reduce((sum, b) => sum + b.qtyLeft, 0)
})

const selectableLocations = computed(() => props.locations.filter((l) => l.qtyLeft > 0))

interface FifoItem {
  id: number
  createdAt: string
  unitPriceCents: number
  location: string
  qty: number
}

const fifoPreview = computed<FifoItem[]>(() => {
  if (props.type !== 'out' || !form.productId) return []
  const qty = Number(form.qty)
  if (!Number.isFinite(qty) || qty <= 0) return []
  let remain = qty
  const out: FifoItem[] = []
  const list = props.batches
    .filter(
      (b) =>
        b.productId === form.productId &&
        (!form.location || b.location === form.location) &&
        b.qtyLeft > 0,
    )
    .sort((a, b) => a.id - b.id)
  for (const b of list) {
    if (remain <= 0) break
    const take = Math.min(remain, b.qtyLeft)
    out.push({
      id: b.id,
      createdAt: b.createdAt,
      unitPriceCents: b.unitPriceCents,
      location: b.location,
      qty: take,
    })
    remain -= take
  }
  return out
})

function fmtMoney(cents: number): string {
  return (cents / 100).toFixed(2)
}

watch(
  () => [props.open, props.type, props.product?.id],
  () => {
    if (!props.open) return
    err.value = ''
    form.productId = props.product?.id ?? props.products[0]?.id ?? 0
    form.qty = props.type === 'adjust' ? (selected.value?.stock ?? 0) : 1
    form.counterparty = ''
    form.note = ''
    form.unitPrice = ''
    form.location = ''
  },
)

watch(
  () => form.location,
  () => {
    if (props.type === 'adjust' && form.location && locationAvail.value !== null) {
      form.qty = locationAvail.value
    }
  },
)

function submit() {
  const qty = Number(form.qty)
  const price = Number(form.unitPrice)
  if (!form.productId) {
    err.value = '请选择商品'
    return
  }
  if (!Number.isFinite(qty) || qty < 0 || (props.type !== 'adjust' && qty < 1)) {
    err.value = props.type === 'adjust' ? '盘点数量不能为负' : '数量必须大于 0'
    return
  }
  if (props.type === 'out' && selected.value && qty > selected.value.stock) {
    err.value = `库存不足,当前 ${selected.value.stock}`
    return
  }
  if (props.type === 'out' && form.location && locationAvail.value !== null && qty > locationAvail.value) {
    err.value = `该货位库存不足,可用 ${locationAvail.value}`
    return
  }
  if (props.type === 'in' && form.unitPrice !== '' && (!Number.isFinite(price) || price < 0)) {
    err.value = '单价无效'
    return
  }
  emit('submit', {
    type: props.type,
    productId: form.productId,
    qty,
    counterparty: props.type === 'adjust' ? '' : form.counterparty.trim(),
    note: form.note.trim(),
    unitPriceCents: props.type === 'in' ? Math.round(price * 100) : 0,
    location: form.location.trim(),
  })
}
</script>

<template>
  <div v-if="open" class="overlay" role="dialog" aria-modal="true">
    <div class="modal">
      <div class="modal-head">
        <component :is="meta.icon" :size="18" />
        <h3>{{ meta.title }}</h3>
        <button class="icon-btn" aria-label="关闭" @click="emit('close')">
          <X :size="18" />
        </button>
      </div>

      <form class="form" @submit.prevent="submit">
        <label>
          <span>商品</span>
          <select v-model.number="form.productId">
            <option v-for="p in products" :key="p.id" :value="p.id">
              {{ p.name }}{{ p.spec ? ` (${p.spec})` : '' }} — 库存 {{ p.stock }} {{ p.unit }}
            </option>
          </select>
        </label>

        <div class="form-row">
          <label>
            <span>{{ props.type === 'adjust' ? '盘点后数量' : '数量' }}</span>
            <input v-model.number="form.qty" type="number" min="0" step="1" />
          </label>
          <div v-if="selected" class="stock-hint">
            当前库存:{{ selected.stock }} {{ selected.unit }}
          </div>
        </div>

        <label v-if="meta.counter">
          <span>{{ meta.counter }}</span>
          <input v-model="form.counterparty" type="text" :placeholder="`例如:${meta.counter}`" />
        </label>

        <div v-if="props.type === 'in'" class="form-row">
          <label>
            <span>单价(元)</span>
            <input v-model="form.unitPrice" type="number" min="0" step="0.01" placeholder="0.00" />
          </label>
          <label>
            <span>货位</span>
            <input v-model="form.location" type="text" list="location-options" placeholder="例如:A-01" />
            <datalist id="location-options">
              <option v-for="loc in locations" :key="loc.id" :value="loc.name" />
            </datalist>
          </label>
        </div>

        <label v-if="props.type === 'out' || props.type === 'adjust'">
          <span>货位</span>
          <select v-model="form.location">
            <option value="">
              {{ props.type === 'adjust' ? '全部货位(整件商品)' : '自动(所有货位)' }}
            </option>
            <option v-for="loc in selectableLocations" :key="loc.id" :value="loc.name">
              {{ loc.name }} ({{ loc.qtyLeft }})
            </option>
          </select>
          <span v-if="locationAvail !== null" class="stock-hint">
            {{ props.type === 'out' ? `该货位可用 ${locationAvail}` : `当前货位 ${locationAvail} 件` }}
          </span>
        </label>

        <div v-if="props.type === 'out' && fifoPreview.length" class="batch-preview">
          <div class="preview-title">扣减批次(先进先出)</div>
          <div v-for="p in fifoPreview" :key="p.id" class="preview-row">
            <span class="muted">{{ p.createdAt }}</span>
            <strong>#{{ p.id }} {{ p.location || '未设货位' }} × {{ p.qty }}</strong>
            <span class="muted">{{ p.unitPriceCents ? `¥${fmtMoney(p.unitPriceCents)}` : '–' }}</span>
          </div>
        </div>

        <label>
          <span>备注</span>
          <input
            v-model="form.note"
            type="text"
            :placeholder="props.type === 'adjust' ? '盘点原因 / 说明' : '说明(可留空)'"
          />
        </label>

        <p v-if="err" class="form-err">{{ err }}</p>

        <div class="modal-actions">
          <button type="button" class="btn ghost" @click="emit('close')">取消</button>
          <button type="submit" class="btn primary">{{ meta.title }}</button>
        </div>
      </form>
    </div>
  </div>
</template>
