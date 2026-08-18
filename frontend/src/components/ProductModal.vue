<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { Package, X } from 'lucide-vue-next'
import type { Product, ProductInput } from '../types'

const props = defineProps<{ open: boolean; edit?: Product }>()

const emit = defineEmits<{
  close: []
  submit: [payload: ProductInput]
}>()

const form = reactive({ name: '', spec: '', unit: '件', category: '' })
const err = ref('')

watch(
  () => [props.open, props.edit?.id],
  () => {
    if (!props.open) return
    err.value = ''
    form.name = props.edit?.name ?? ''
    form.spec = props.edit?.spec ?? ''
    form.unit = props.edit?.unit ?? '件'
    form.category = props.edit?.category ?? ''
  },
)

function submit() {
  if (!form.name.trim()) {
    err.value = '商品名称必填'
    return
  }
  emit('submit', {
    name: form.name.trim(),
    spec: form.spec.trim(),
    unit: form.unit.trim() || '件',
    category: form.category.trim(),
  })
}
</script>

<template>
  <div v-if="open" class="overlay" role="dialog" aria-modal="true">
    <div class="modal">
      <div class="modal-head">
        <Package :size="18" />
        <h3>{{ edit ? '编辑商品' : '新建商品' }}</h3>
        <button class="icon-btn" aria-label="关闭" @click="emit('close')">
          <X :size="18" />
        </button>
      </div>

      <form class="form" @submit.prevent="submit">
        <label>
          <span>名称</span>
          <input v-model="form.name" type="text" placeholder="例如:螺丝钉" />
        </label>
        <div class="form-row">
          <label>
            <span>规格</span>
            <input v-model="form.spec" type="text" placeholder="例如:M4x12" />
          </label>
          <label>
            <span>单位</span>
            <input v-model="form.unit" type="text" placeholder="件" />
          </label>
        </div>
        <label>
          <span>分类</span>
          <input v-model="form.category" type="text" placeholder="例如:五金" />
        </label>

        <p v-if="err" class="form-err">{{ err }}</p>

        <div class="modal-actions">
          <button type="button" class="btn ghost" @click="emit('close')">取消</button>
          <button type="submit" class="btn primary">{{ edit ? '保存' : '创建' }}</button>
        </div>
      </form>
    </div>
  </div>
</template>
