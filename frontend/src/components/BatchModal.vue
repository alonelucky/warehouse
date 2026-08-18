<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { FileSpreadsheet, RefreshCw, X } from 'lucide-vue-next'
import * as XLSX from 'xlsx'
import type { BatchItemInput, MovementType, Product } from '../types'

const props = defineProps<{ products: Product[]; open: boolean }>()

const emit = defineEmits<{
  close: []
  submit: [items: BatchItemInput[]]
  reselect: []
}>()

interface PreviewRow {
  row: number
  productName: string
  spec: string
  type: MovementType
  typeText: string
  qty: string
  unitPrice: string
  location: string
  counterparty: string
  note: string
  status: 'ok' | 'new' | 'error'
  error: string
}

const TYPE_TEXT: Record<MovementType, string> = { in: '入库', out: '出库', adjust: '盘点' }

const rows = ref<PreviewRow[]>([])
const fileName = ref('')
const err = ref('')

watch(
  () => props.open,
  (v) => {
    if (v) reset()
  },
)

function reset() {
  rows.value = []
  fileName.value = ''
  err.value = ''
}

function typeMap(t: string): MovementType | '' {
  switch (t.trim()) {
    case '入库':
    case 'in':
      return 'in'
    case '出库':
    case 'out':
      return 'out'
    case '盘点':
    case 'adjust':
      return 'adjust'
    default:
      return ''
  }
}

function parseFile(file: File) {
  fileName.value = file.name
  const reader = new FileReader()
  reader.onload = () => {
    try {
      const wb = XLSX.read(new Uint8Array(reader.result as ArrayBuffer), { type: 'array' })
      const ws = wb.Sheets[wb.SheetNames[0]]
      const raw = XLSX.utils.sheet_to_json<Record<string, unknown>>(ws, { defval: '' })
      rows.value = raw.map((r, i) => buildRow(r, i))
      err.value = ''
    } catch {
      rows.value = []
      err.value = '文件解析失败,请使用模板格式的 .xlsx 文件'
    }
  }
  reader.readAsArrayBuffer(file)
}

defineExpose({ parseFile })

function buildRow(r: Record<string, unknown>, idx: number): PreviewRow {
  const productName = String(r['商品名称'] ?? '').trim()
  const spec = String(r['规格'] ?? '').trim()
  const t = typeMap(String(r['类型'] ?? ''))
  const qty = Number(r['数量'])
  const price = Number(r['单价(元)'])
  const row: PreviewRow = {
    row: idx + 2,
    productName,
    spec,
    type: t || 'in',
    typeText: t ? TYPE_TEXT[t] : '?',
    qty: String(r['数量'] ?? ''),
    unitPrice: String(r['单价(元)'] ?? ''),
    location: String(r['货位'] ?? '').trim(),
    counterparty: String(r['供应商/去向'] ?? '').trim(),
    note: String(r['备注'] ?? '').trim(),
    status: 'ok',
    error: '',
  }
  const errors: string[] = []
  if (!productName) errors.push('商品名称为空')
  if (!t) {
    errors.push('类型需为 入库/出库/盘点')
  } else if (t === 'adjust' ? !(Number.isFinite(qty) && qty >= 0) : !(Number.isFinite(qty) && qty > 0)) {
    errors.push('数量无效')
  }
  if (Number.isFinite(price) && price < 0) errors.push('单价无效')
  const found = props.products.find((p) => p.name === productName && p.spec === spec)
  if (errors.length) {
    row.status = 'error'
    row.error = errors.join('; ')
  } else if (!found) {
    if (t === 'in') {
      row.status = 'new'
      row.error = '新商品,提交后自动创建'
    } else {
      row.status = 'error'
      row.error = '商品不存在'
    }
  }
  return row
}

const okRows = computed(() => rows.value.filter((r) => r.status !== 'error'))
const errorsCount = computed(() => rows.value.filter((r) => r.status === 'error').length)

function submitBatch() {
  if (!okRows.value.length) {
    err.value = '没有可提交的行,请先修正错误行'
    return
  }
  emit(
    'submit',
    okRows.value.map((r) => ({
      productName: r.productName,
      spec: r.spec,
      type: r.type,
      qty: Number(r.qty),
      unitPriceCents: Math.round((Number(r.unitPrice) || 0) * 100),
      location: r.location,
      counterparty: r.counterparty,
      note: r.note,
    })),
  )
}
</script>

<template>
  <div v-if="open" class="overlay" role="dialog" aria-modal="true">
    <div class="modal batch-modal">
      <div class="modal-head">
        <FileSpreadsheet :size="18" />
        <h3>批量导入</h3>
        <button class="icon-btn" aria-label="关闭" @click="emit('close')">
          <X :size="18" />
        </button>
      </div>

      <div class="form">
        <div class="toolbar batch-toolbar">
          <button class="btn primary" @click="emit('reselect')">
            <RefreshCw :size="16" />
            重新选择
          </button>
          <span v-if="fileName" class="muted">{{ fileName }}</span>
        </div>

        <p class="hint">模板列:商品名称、规格、类型(入库/出库/盘点)、数量、单价(元)、货位、供应商/去向、备注</p>
        <p v-if="err" class="form-err">{{ err }}</p>

        <div v-if="rows.length" class="batch-table">
          <table>
            <thead>
              <tr>
                <th>行</th>
                <th>商品</th>
                <th>类型</th>
                <th class="num">数量</th>
                <th class="num">单价</th>
                <th>货位</th>
                <th>供应商/去向</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in rows" :key="row.row" :class="{ 'row-err': row.status === 'error' }">
                <td>{{ row.row }}</td>
                <td>
                  {{ row.productName || '–' }}
                  <span v-if="row.spec" class="muted"> ({{ row.spec }})</span>
                </td>
                <td>{{ row.typeText }}</td>
                <td class="num">{{ row.qty }}</td>
                <td class="num">{{ row.unitPrice }}</td>
                <td>{{ row.location || '–' }}</td>
                <td>{{ row.counterparty || '–' }}</td>
                <td>
                  <span
                    :class="[
                      'badge status-badge',
                      row.status === 'error' ? 'out' : row.status === 'new' ? 'adjust' : 'in',
                    ]"
                  >
                    {{ row.status === 'ok' ? 'OK' : row.status === 'new' ? '新建' : '错误' }}
                  </span>
                  <span v-if="row.error" class="row-err-text">{{ row.error }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-if="errorsCount" class="form-err">
          {{ errorsCount }} 行有错误,已剔除,提交其余 {{ okRows.length }} 行
        </div>

        <div class="modal-actions">
          <button type="button" class="btn ghost" @click="emit('close')">取消</button>
          <button type="button" class="btn primary" :disabled="!okRows.length" @click="submitBatch">
            提交 {{ okRows.length }} 条
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
