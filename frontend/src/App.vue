<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  ArrowDownToLine,
  ArrowUpFromLine,
  Boxes,
  ClipboardList,
  Download,
  History,
  Layers,
  Pencil,
  Plus,
  Scale,
  Search,
  Upload,
} from 'lucide-vue-next'
import * as XLSX from 'xlsx'
import BatchModal from './components/BatchModal.vue'
import MovementModal from './components/MovementModal.vue'
import ProductModal from './components/ProductModal.vue'
import { api } from './api'
import type {
  AuditEntry,
  Batch,
  BatchItemInput,
  Location,
  Movement,
  MovementInput,
  MovementType,
  Product,
  ProductInput,
  Stats,
} from './types'

type Tab = 'stock' | 'batches' | 'movements' | 'audit'

const tab = ref<Tab>('stock')
const operator = ref('admin')
const stats = ref<Stats | null>(null)
const products = ref<Product[]>([])
const batches = ref<Batch[]>([])
const locations = ref<Location[]>([])
const movements = ref<Movement[]>([])
const audits = ref<AuditEntry[]>([])

const q = ref('')
const bQ = ref('')
const mType = ref('')
const mQ = ref('')
const aAction = ref('')
const aQ = ref('')
const errMsg = ref('')
const toastTimer = ref<number | undefined>()

const movModal = ref<{ open: boolean; type: MovementType; product?: Product }>({
  open: false,
  type: 'in',
})
const prodModal = ref<{ open: boolean; edit?: Product }>({ open: false })
const batchModal = ref(false)
const batchModalRef = ref<InstanceType<typeof BatchModal>>()
const excelInput = ref<HTMLInputElement>()

function pickExcel() {
  excelInput.value?.click()
}

async function onExcelFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  await batchModalRef.value?.parseFile(file)
  batchModal.value = true
  input.value = ''
}

const filteredProducts = computed(() => {
  const s = q.value.trim().toLowerCase()
  if (!s) return products.value
  return products.value.filter((p) =>
    [p.name, p.spec, p.category, p.unit].some((v) => v.toLowerCase().includes(s)),
  )
})

const actionLabels: Record<string, string> = {
  'product.create': '新建商品',
  'product.update': '修改商品',
  'stock.in': '入库',
  'stock.out': '出库',
  'stock.adjust': '盘点调整',
}

const auditSummary = computed(() => {
  const s = aQ.value.trim().toLowerCase()
  if (!s) return audits.value
  return audits.value.filter((a) =>
    [a.action, a.entity, a.operator, JSON.stringify(a.detail)].some((v) => v.toLowerCase().includes(s)),
  )
})

async function loadState() {
  const [p, b, loc, m, a, st, meta] = await Promise.all([
    api.products(),
    api.batches({ q: bQ.value }),
    api.locations(),
    api.movements({ type: mType.value, q: mQ.value }),
    api.audit({ action: aAction.value, q: aQ.value }),
    api.stats(),
    api.meta(),
  ])
  products.value = p
  batches.value = b
  locations.value = loc
  movements.value = m
  audits.value = a
  stats.value = st
  operator.value = meta.operator
}

function changeBatches() {
  api
    .batches({ q: bQ.value })
    .then((b) => (batches.value = b))
    .catch(fail)
}

function fail(e: unknown) {
  errMsg.value = e instanceof Error ? e.message : '操作失败'
  window.clearTimeout(toastTimer.value)
  toastTimer.value = window.setTimeout(() => (errMsg.value = ''), 4000)
}

function openMovement(type: MovementType, product?: Product) {
  if (!products.value.length) {
    errMsg.value = '请先新建商品'
    window.setTimeout(() => (errMsg.value = ''), 3000)
    return
  }
  movModal.value = { open: true, type, product }
}

function exportExcel() {
  const wb = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(
    wb,
    XLSX.utils.json_to_sheet(
      products.value.map((p) => ({
        '商品': p.name,
        '规格': p.spec,
        '分类': p.category,
        '单位': p.unit,
        '当前库存': p.stock,
        '参考单价': (p.avgPriceCents / 100).toFixed(2),
        '货位': p.locations,
      })),
    ),
    '库存',
  )
  XLSX.utils.book_append_sheet(
    wb,
    XLSX.utils.json_to_sheet(
      batches.value.map((b) => ({
        '时间': b.createdAt,
        '商品': b.productName,
        '入库量': b.qty,
        '剩余': b.qtyLeft,
        '单价': (b.unitPriceCents / 100).toFixed(2),
        '货位': b.location,
        '供应商': b.supplier,
        '备注': b.note,
      })),
    ),
    '批次',
  )
  XLSX.utils.book_append_sheet(
    wb,
    XLSX.utils.json_to_sheet(
      movements.value.map((m) => ({
        '时间': m.createdAt,
        '类型': { in: '入库', out: '出库', adjust: '盘点' }[m.type],
        '商品': m.productName,
        '数量': m.qty,
        '库存前后': `${m.stockBefore}→${m.stockAfter}`,
        '货位': m.locations,
        '批次': m.batchRefs,
        '对方': m.counterparty,
        '备注': m.note,
      })),
    ),
    '流水',
  )
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  XLSX.writeFile(wb, `仓库台账_${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}.xlsx`)
}

function downloadTemplate() {
  const HEADERS = ['商品名称', '规格', '类型', '数量', '单价(元)', '货位', '供应商/去向', '备注']
  const wb = XLSX.utils.book_new()
  const ws = XLSX.utils.aoa_to_sheet([
    HEADERS,
    ['扳手', '10寸', '入库', 100, 12.5, 'A-01', '供应商甲', '首批采购'],
  ])
  ws['!cols'] = HEADERS.map((h) => ({ wch: Math.max(10, h.length * 2) }))
  XLSX.utils.book_append_sheet(wb, ws, '出入库')
  XLSX.writeFile(wb, '出入库模板.xlsx')
}

async function submitBatch(items: BatchItemInput[]) {
  try {
    const res = await api.batchMovements(items)
    batchModal.value = false
    await loadState()
    errMsg.value = `批量完成,共 ${res.count} 条`
    window.clearTimeout(toastTimer.value)
    toastTimer.value = window.setTimeout(() => (errMsg.value = ''), 4000)
  } catch (e) {
    fail(e)
  }
}

async function submitMovement(inp: MovementInput) {
  try {
    await api.addMovement(inp)
    movModal.value.open = false
    tab.value = 'stock'
    await loadState()
  } catch (e) {
    fail(e)
  }
}

async function submitProduct(inp: ProductInput) {
  try {
    if (prodModal.value.edit) {
      await api.updateProduct(prodModal.value.edit.id, inp)
    } else {
      await api.createProduct(inp)
    }
    prodModal.value.open = false
    await loadState()
  } catch (e) {
    fail(e)
  }
}

function changeMovements() {
  api
    .movements({ type: mType.value, q: mQ.value })
    .then((m) => (movements.value = m))
    .catch(fail)
}

function changeAudit() {
  api
    .audit({ action: aAction.value, q: aQ.value })
    .then((a) => (audits.value = a))
    .catch(fail)
}

function fmtDetail(detail: Record<string, unknown>): string {
  return Object.entries(detail)
    .map(([k, v]) => `${k}: ${typeof v === 'object' ? JSON.stringify(v) : v}`)
    .join(' · ')
}

function fmtMoney(cents: number): string {
  return (cents / 100).toFixed(2)
}

onMounted(() => {
  loadState().catch(fail)
})
</script>

<template>
  <div class="app">
    <header class="topbar">
      <div class="brand">
        <Boxes :size="22" />
        <div>
          <h1>仓库台账</h1>
          <p>单仓进销存 · 全程留痕</p>
        </div>
      </div>

      <nav class="tabs" role="tablist">
        <button
          v-for="t in ([
            ['stock', '库存', Boxes],
            ['batches', '批次', Layers],
            ['movements', '流水', ClipboardList],
            ['audit', '操作日志', History],
          ] as const)"
          :key="t[0]"
          :class="['tab', { active: tab === t[0] }]"
          role="tab"
          :aria-selected="tab === t[0]"
          @click="tab = t[0]"
        >
          <component :is="t[2]" :size="16" />
          {{ t[1] }}
        </button>
      </nav>
      <button class="btn primary" @click="openMovement('in')">
        <ArrowDownToLine :size="16" />
        入库
      </button>
      <button class="btn" @click="openMovement('out')">
        <ArrowUpFromLine :size="16" />
        出库
      </button>
      <span class="operator" :title="`操作留痕记录人`">{{ operator }}</span>
    </header>

    <main class="main">
      <section v-if="tab === 'stock'" class="view">
        <div class="stats">
          <div class="stat">
            <span>商品种类</span>
            <strong>{{ stats?.skuCount ?? '–' }}</strong>
            <em>累计 SKU</em>
          </div>
          <div class="stat">
            <span>当前库存</span>
            <strong>{{ stats?.totalUnits ?? '–' }}</strong>
            <em>总值 ¥{{ stats ? fmtMoney(stats.inventoryValue) : '–' }}</em>
          </div>
          <div class="stat pos">
            <span>今日入库</span>
            <strong>{{ stats?.inToday ?? '–' }}</strong>
            <em>累计 {{ stats?.inTotal ?? 0 }}</em>
          </div>
          <div class="stat warn">
            <span>今日出库</span>
            <strong>{{ stats?.outToday ?? '–' }}</strong>
            <em>累计 {{ stats?.outTotal ?? 0 }}</em>
          </div>
        </div>

        <div class="panel">
          <div class="toolbar">
            <div class="search">
              <Search :size="15" />
              <input v-model="q" type="search" placeholder="搜索名称 / 规格 / 分类" />
            </div>
            <button class="btn primary" @click="openMovement('in')">
              <ArrowDownToLine :size="16" />
              入库
            </button>
            <button class="btn" @click="openMovement('out')">
              <ArrowUpFromLine :size="16" />
              出库
            </button>
            <button class="btn" @click="openMovement('adjust')">
              <Scale :size="16" />
              盘点
            </button>
            <button class="btn ghost" @click="prodModal = { open: true }">
              <Plus :size="16" />
              新建商品
            </button>
            <button class="btn" @click="pickExcel">
              <Upload :size="16" />
              导入 Excel
            </button>
            <button class="btn" @click="downloadTemplate">
              <Download :size="16" />
              出入库模板
            </button>
            <input
              ref="excelInput"
              type="file"
              accept=".xlsx,.xls"
              hidden
              @change="onExcelFile"
            />
            <button class="btn" @click="exportExcel">
              <Download :size="16" />
              导出 Excel
            </button>
          </div>

          <table>
            <thead>
              <tr>
                <th>商品</th>
                <th>规格</th>
                <th>分类</th>
                <th class="num">参考单价</th>
                <th>货位(余量)</th>
                <th class="num">当前库存</th>
                <th>更新时间</th>
                <th class="ops">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in filteredProducts" :key="p.id">
                <td>
                  <span class="name">{{ p.name }}</span>
                  <span class="unit">{{ p.unit }}</span>
                </td>
                <td class="muted">{{ p.spec || '–' }}</td>
                <td><span class="chip">{{ p.category || '未分类' }}</span></td>
                <td class="num">{{ p.avgPriceCents ? `¥${fmtMoney(p.avgPriceCents)}` : '–' }}</td>
                <td class="muted">{{ p.locations || '–' }}</td>
                <td class="num"><strong :class="{ low: p.stock === 0 }">{{ p.stock }}</strong></td>
                <td class="muted">{{ p.updatedAt }}</td>
                <td class="ops">
                  <button class="icon-btn" title="入库" @click="openMovement('in', p)">
                    <ArrowDownToLine :size="16" />
                  </button>
                  <button class="icon-btn" title="出库" @click="openMovement('out', p)">
                    <ArrowUpFromLine :size="16" />
                  </button>
                  <button class="icon-btn" title="盘点" @click="openMovement('adjust', p)">
                    <Scale :size="16" />
                  </button>
                  <button class="icon-btn" title="编辑" @click="prodModal = { open: true, edit: p }">
                    <Pencil :size="16" />
                  </button>
                </td>
              </tr>
              <tr v-if="!filteredProducts.length">
                <td colspan="8" class="empty">暂无商品,先新建一个</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-else-if="tab === 'batches'" class="view">
        <div class="panel">
          <div class="toolbar">
            <div class="search">
              <Search :size="15" />
              <input v-model="bQ" type="search" placeholder="搜索商品 / 货位 / 供应商" @input="changeBatches" />
            </div>
            <button class="btn primary" @click="openMovement('in')">
              <ArrowDownToLine :size="16" />
              入库
            </button>
          </div>

          <table>
            <thead>
              <tr>
                <th>时间</th>
                <th>商品</th>
                <th class="num">入库量</th>
                <th class="num">剩余</th>
                <th class="num">单价</th>
                <th>货位</th>
                <th>供应商</th>
                <th>备注</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="b in batches" :key="b.id">
                <td class="muted">{{ b.createdAt }}</td>
                <td>{{ b.productName }}</td>
                <td class="num">{{ b.qty }}</td>
                <td :class="['num', { warn: b.qtyLeft === 0 }]">
                  <strong>{{ b.qtyLeft }}</strong>
                </td>
                <td class="num">{{ b.unitPriceCents ? `¥${fmtMoney(b.unitPriceCents)}` : '–' }}</td>
                <td>{{ b.location || '–' }}</td>
                <td>{{ b.supplier || '–' }}</td>
                <td class="muted note">{{ b.note || '–' }}</td>
              </tr>
              <tr v-if="!batches.length">
                <td colspan="8" class="empty">暂无批次,先入库一批</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-else-if="tab === 'movements'" class="view">
        <div class="panel">
          <div class="toolbar">
            <select v-model="mType" class="filter" @change="changeMovements">
              <option value="">全部类型</option>
              <option value="in">入库</option>
              <option value="out">出库</option>
              <option value="adjust">盘点调整</option>
            </select>
            <div class="search">
              <Search :size="15" />
              <input v-model="mQ" type="search" placeholder="搜索商品 / 对方 / 备注" @input="changeMovements" />
            </div>
          </div>

          <table>
            <thead>
              <tr>
                <th>时间</th>
                <th>类型</th>
                <th>商品</th>
                <th class="num">变化</th>
                <th class="num">库存前后</th>
                <th>货位</th>
                <th>批次</th>
                <th>对方 / 去向</th>
                <th>备注</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="m in movements" :key="m.id">
                <td class="muted">{{ m.createdAt }}</td>
                <td>
                  <span :class="['badge', m.type]">
                    {{ { in: '入库', out: '出库', adjust: '盘点' }[m.type] }}
                  </span>
                </td>
                <td>{{ m.productName }}</td>
                <td :class="['num', m.qty > 0 ? 'pos' : m.qty < 0 ? 'warn' : '']">
                  <strong>{{ m.qty > 0 ? '+' : '' }}{{ m.qty }}</strong>
                </td>
                <td class="num muted">{{ m.stockBefore }} → {{ m.stockAfter }}</td>
                <td class="muted">{{ m.locations || '–' }}</td>
                <td class="muted">{{ m.batchRefs ? '#' + m.batchRefs.replace(/x/g, '×') : '–' }}</td>
                <td>{{ m.counterparty || '–' }}</td>
                <td class="muted note">{{ m.note || '–' }}</td>
              </tr>
              <tr v-if="!movements.length">
                <td colspan="9" class="empty">暂无流水记录</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-else-if="tab === 'audit'" class="view">
        <div class="panel">
          <div class="toolbar">
            <select v-model="aAction" class="filter" @change="changeAudit">
              <option value="">全部动作</option>
              <option v-for="(label, act) in actionLabels" :key="act" :value="act">
                {{ label }}
              </option>
            </select>
            <div class="search">
              <Search :size="15" />
              <input v-model="aQ" type="search" placeholder="搜索日志内容" @input="changeAudit" />
            </div>
            <span class="note">日志只读,不可修改</span>
          </div>

          <table>
            <thead>
              <tr>
                <th>时间</th>
                <th>操作人</th>
                <th>动作</th>
                <th>明细</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in auditSummary" :key="a.id">
                <td class="muted">{{ a.ts }}</td>
                <td>{{ a.operator }}</td>
                <td>
                  <span :class="['badge', 'act-' + a.action.replace('.', '-')]">
                    {{ actionLabels[a.action] ?? a.action }}
                  </span>
                </td>
                <td class="muted detail">{{ fmtDetail(a.detail) }}</td>
              </tr>
              <tr v-if="!auditSummary.length">
                <td colspan="4" class="empty">暂无操作记录</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

    </main>

    <MovementModal
      :products="products"
      :locations="locations"
      :batches="batches"
      :open="movModal.open"
      :type="movModal.type"
      :product="movModal.product"
      @close="movModal.open = false"
      @submit="submitMovement"
    />
    <ProductModal
      :open="prodModal.open"
      :edit="prodModal.edit"
      @close="prodModal.open = false"
      @submit="submitProduct"
    />
    <BatchModal
      ref="batchModalRef"
      :products="products"
      :open="batchModal"
      @close="batchModal = false"
      @submit="submitBatch"
      @reselect="pickExcel"
    />

    <transition name="toast">
      <div v-if="errMsg" class="toast">{{ errMsg }}</div>
    </transition>
  </div>
</template>
