<script setup>
import { ref, onMounted, reactive, nextTick, watch } from 'vue'
import { AgGridVue } from 'ag-grid-vue3'
import { ModuleRegistry, AllCommunityModule } from 'ag-grid-community'

import 'ag-grid-community/styles/ag-grid.css'
import 'ag-grid-community/styles/ag-theme-quartz.css'
import { OpenCSV, SaveCSV, GetRows, CheckUpdate, GetVersion, SelectFile, LoadCSV } from '../wailsjs/go/main/App'
import { FolderOpen, Save, Search, Plus, Trash2, Info, FileWarning, ExternalLink, RefreshCw } from 'lucide-vue-next'
import { BrowserOpenURL } from '../wailsjs/runtime/runtime'

// 注册所有社区版模块 (AG Grid v35 推荐方式)
ModuleRegistry.registerModules([ AllCommunityModule ])

// --- State ---
const gridApi = ref(null)
const rowData = ref([])
const columnDefs = ref([])
const filePath = ref('')
const totalRows = ref(0)
const searchText = ref('')
const isLargeFile = ref(false)
const isLoading = ref(false)
const loadingMessage = ref('')
const currentRawHeaders = ref([])
const gridKey = ref(0)
const isDirty = ref(false)
const showAbout = ref(false)
const isCheckingUpdate = ref(false)
const appVersion = ref('v0.0.0')

const LARGE_FILE_THRESHOLD = 50000 

const gridOptions = reactive({
  defaultColDef: {
    sortable: true,
    filter: true,
    resizable: true,
    editable: true,
    width: 120,
    minWidth: 60,
  },
  animateRows: false,
  pagination: true,
  paginationPageSize: 100,
  paginationPageSizeSelector: [100, 500, 1000],
  rowHeight: 28,
  headerHeight: 32,
  theme: 'legacy',
  rowSelection: { 
    mode: 'multiRow',
    checkboxes: false,
    headerCheckbox: false,
    enableClickSelection: true
  },
  onCellValueChanged: () => { isDirty.value = true },
  enableCellTextSelection: true, // 允许鼠标选中单元格文字以供复制
  ensureDomOrder: true,          // 配合文字选中，确保 DOM 顺序
})

// --- Logic ---
onMounted(async () => {
  try {
    appVersion.value = await GetVersion()
  } catch (e) { console.error("Failed to get version:", e) }
})

const onGridReady = (params) => {
  gridApi.value = params.api
  if (isLargeFile.value && filePath.value) {
    params.api.setGridOption('datasource', createDatasource(currentRawHeaders.value))
  }
}

const checkUpdate = async () => {
  isCheckingUpdate.value = true
  try {
    const res = await CheckUpdate()
    isCheckingUpdate.value = false
    if (res.error) {
      alert("检查失败: " + res.error)
      return
    }
    if (res.hasUpdate) {
      if (confirm(`发现新版本 ${res.latestVer}！\n\n更新说明：\n${res.desc || '无'}\n\n是否前往下载页面？`)) {
        BrowserOpenURL(res.downloadURL)
      }
    } else {
      alert(`当前已是最新版本 (${res.currentVer})`)
    }
  } catch (e) {
    isCheckingUpdate.value = false
    alert("检查出错: " + e)
  }
}

const openFile = async () => {
  if (isDirty.value && !confirm("当前文件有未保存的修改，确定要直接打开新文件吗？")) return
  try {
    const path = await SelectFile()
    if (!path) return

    searchText.value = ''
    isLoading.value = true
    loadingMessage.value = "正在载入并索引文件..."

    // 给 UI 一个渲染加载状态的机会
    await new Promise(resolve => setTimeout(resolve, 50))

    const meta = await LoadCSV(path)
    if (meta.error) {
      isLoading.value = false
      alert(meta.error)
      return
    }

    filePath.value = ''
    isDirty.value = false
    await nextTick()
    filePath.value = meta.filePath
    totalRows.value = meta.total
    currentRawHeaders.value = meta.headers
    isLargeFile.value = meta.total > LARGE_FILE_THRESHOLD
    let headers = [...meta.headers]
    const minCols = 20
    if (headers.length < minCols) {
      for (let i = headers.length; i < minCols; i++) headers.push(`__pad_${i}`)
    }
    const baseCols = headers.map(h => ({
      headerName: h.startsWith('__pad_') ? '' : h,
      field: h,
      editable: !isLargeFile.value, 
    }))
    columnDefs.value = [
      {
        headerName: '#',
        valueGetter: (params) => params.node.rowIndex + 1,
        width: 60,
        pinned: 'left',
        suppressMovable: true,
        sortable: false,
        filter: false,
        resizable: false,
        editable: false,
        cellStyle: { color: '#94a3b8', 'text-align': 'center', 'font-size': '11px', 'background-color': '#f8fafc' }
      },
      ...baseCols
    ]
    gridKey.value++
    if (!isLargeFile.value) {
      loadingMessage.value = "正在解析数据..."
      const response = await GetRows(0, meta.total, "")
      rowData.value = response.rows.map(row => {
        const obj = {}
        headers.forEach((h, i) => obj[h] = (i < meta.headers.length) ? (row[i] || '') : '')
        return obj
      })
    } else {
      rowData.value = null
    }
    isLoading.value = false
  } catch (err) { 
    isLoading.value = false
    alert("打开失败: " + err)
    console.error(err) 
  }
}

const createDatasource = (headers) => ({
  getRows: async (params) => {
    try {
      const response = await GetRows(params.startRow, params.endRow - params.startRow, searchText.value)
      if (response.error) { params.failCallback(); return }
      const rows = response.rows.map(row => {
        const obj = {}
        columnDefs.value.forEach(col => {
          const idx = headers.indexOf(col.field)
          obj[col.field] = idx !== -1 ? (row[idx] || '') : ''
        })
        return obj
      })
      params.successCallback(rows, response.total)
    } catch (e) { params.failCallback() }
  }
})

let searchTimeout = null
watch(searchText, (newVal) => {
  if (!isLargeFile.value || !gridApi.value) return
  
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    gridApi.value.setGridOption('datasource', createDatasource(currentRawHeaders.value))
  }, 500)
})

const enableEditMode = async () => {
  if (!confirm("确定要加载全量数据以启用编辑吗？")) return
  isLoading.value = true
  loadingMessage.value = "正在拉取全量数据..."
  setTimeout(async () => {
    const response = await GetRows(0, totalRows.value, "")
    rowData.value = response.rows.map(row => {
      const obj = {}
      columnDefs.value.forEach((col, idx) => obj[col.field] = idx < row.length ? row[idx] : '')
      return obj
    })
    isLargeFile.value = false
    columnDefs.value = columnDefs.value.map(c => ({ ...c, editable: !!c.field }))
    gridKey.value++
    isLoading.value = false
  }, 50)
}

const saveFile = async () => {
  if (isLargeFile.value || !filePath.value) return
  const validCols = columnDefs.value.filter(c => c.field && (c.headerName !== '' || (rowData.value && rowData.value.some(r => r[c.field]))))
  const headers = validCols.map(c => c.headerName || 'Column')
  const rows = rowData.value.map(r => validCols.map(c => r[c.field]))
  const err = await SaveCSV(filePath.value, headers, rows)
  if (!err) isDirty.value = false
  alert(err || '保存成功！')
}

const addRow = () => { 
  if (!isLargeFile.value) {
    rowData.value = [{}, ...rowData.value]
    isDirty.value = true
  }
}
const removeSelected = () => {
  if (isLargeFile.value || !gridApi.value) return
  const selected = gridApi.value.getSelectedRows()
  if (selected.length === 0) return
  rowData.value = rowData.value.filter(r => !selected.includes(r))
  isDirty.value = true
}
</script>

<template>
  <div class="ecsv-app">
    <div class="ecsv-header">
      <div class="ecsv-nav-left">
        <span class="ecsv-title">EasyCSV</span>
        <span v-if="isLargeFile" class="ecsv-badge">只读模式</span>
        <i class="ecsv-v-divider" style="margin: 0 8px;"></i>
        <button class="ecsv-btn-ui" @click="openFile">
          <FolderOpen :size="14" /> <span>打开</span>
        </button>
        <button class="ecsv-btn-ui" :class="{ 'ecsv-btn-dirty': isDirty }" @click="saveFile" v-if="!isLargeFile" :disabled="!filePath">
          <Save :size="14" /> <span>保存</span>
        </button>
        <i class="ecsv-v-divider" v-if="!isLargeFile"></i>
        <button class="ecsv-btn-ui ecsv-icon-btn" @click="addRow" v-if="!isLargeFile" :disabled="!filePath">
          <Plus :size="14" />
        </button>
        <button class="ecsv-btn-ui ecsv-icon-btn ecsv-danger" @click="removeSelected" v-if="!isLargeFile" :disabled="!filePath">
          <Trash2 :size="14" />
        </button>
        <button class="ecsv-btn-ui ecsv-warning" @click="enableEditMode" v-if="isLargeFile">
          <FileWarning :size="14" /> <span>启用编辑</span>
        </button>
      </div>
      <div class="ecsv-nav-right">
        <div class="ecsv-search-input">
          <Search :size="12" class="ecsv-search-icon" />
          <input 
            v-model="searchText" 
            placeholder="搜索..." 
          />
        </div>
        <i class="ecsv-v-divider"></i>
        <button class="ecsv-btn-ui ecsv-icon-btn" @click="showAbout = true" title="软件信息">
          <Info :size="14" />
        </button>
      </div>
    </div>
    <div class="ecsv-main">
      <div v-if="isLoading" class="ecsv-loader">
        <div class="ecsv-spin"></div>
        <p>{{ loadingMessage }}</p>
      </div>
      <div v-if="filePath" class="ecsv-grid">
        <ag-grid-vue
          :key="gridKey"
          style="width: 100%; height: 100%;"
          class="ag-theme-quartz"
          :columnDefs="columnDefs"
          :rowData="rowData"
          :rowModelType="isLargeFile ? 'infinite' : 'clientSide'"
          :gridOptions="gridOptions"
          :quickFilterText="searchText"
          @grid-ready="onGridReady"
        />
      </div>
      <div v-else class="ecsv-welcome">
        <div class="ecsv-welcome-box">
          <FolderOpen :size="48" color="#94a3b8" />
          <p>请打开一个 CSV 文件进行操作</p>
          <button class="ecsv-btn-primary" @click="openFile">立即打开</button>
        </div>
      </div>
      <div v-if="showAbout" class="ecsv-modal-overlay" @click.self="showAbout = false">
        <div class="ecsv-modal">
          <div class="ecsv-modal-header">
            <h3>关于 EasyCSV</h3>
            <button @click="showAbout = false" class="ecsv-modal-close">&times;</button>
          </div>
          <div class="ecsv-modal-body">
            <div class="ecsv-about-content">
              <img src="./assets/images/logo-universal.png" class="ecsv-about-logo" alt="Logo" />
              <h4>EasyCSV <span class="ecsv-ver">{{ appVersion }}</span></h4>
              <p class="ecsv-about-desc">一个简洁、高效且支持超大文件的 CSV 查看与编辑器。</p>
              <div class="ecsv-info-grid">
                <span class="ecsv-info-label">作者：</span>
                <span class="ecsv-info-val">WangTwoThree</span>
                <span class="ecsv-info-label">GitHub：</span>
                <a href="https://github.com/TwoThreeWang/EasyCSV" target="_blank" class="ecsv-link">
                  TwoThreeWang/EasyCSV <ExternalLink :size="12" />
                </a>
                <span class="ecsv-info-label">技术栈：</span>
                <span class="ecsv-info-val">Wails, Go, Vue 3, AG Grid</span>
                <span class="ecsv-info-label">更新日期：</span>
                <span class="ecsv-info-val">2026-01-05</span>
              </div>
            </div>
          </div>
          <div class="ecsv-modal-footer">
            <button class="ecsv-btn-ui" style="margin-right: 8px;" @click="checkUpdate" :disabled="isCheckingUpdate">
               <RefreshCw :size="12" :class="{ 'ecsv-spin': isCheckingUpdate }" v-if="isCheckingUpdate" />
               <span v-else>检查更新</span>
            </button>
            <button class="ecsv-btn-primary" @click="showAbout = false" style="margin:0;">确定</button>
          </div>
        </div>
      </div>
    </div>
    <div class="ecsv-footer">
      <div class="ecsv-status-left">
        <Info :size="12" />
        <span class="ecsv-path">
          {{ filePath || '准备就绪' }}
          <span v-if="isDirty" style="color: #ea580c; font-weight: bold; margin-left: 4px;">(未保存)</span>
        </span>
      </div>
      <div class="ecsv-status-right" v-if="totalRows > 0">{{ totalRows }} 行数据</div>
    </div>
  </div>
</template>

<style>
.ecsv-app { display: flex; flex-direction: column; height: 100vh; width: 100vw; background: #f8fafc; overflow: hidden; }
.ecsv-header { display: flex; align-items: center; justify-content: space-between; height: 42px; background: #ffffff; border-bottom: 1px solid #cbd5e1; padding: 0 12px; flex-shrink: 0; }
.ecsv-nav-left { display: flex; align-items: center; gap: 6px; }
.ecsv-title { font-weight: 800; font-size: 15px; color: #2563eb; }
.ecsv-badge { font-size: 10px; background: #f59e0b; color: white; padding: 1px 6px; border-radius: 4px; margin-left: 8px; font-weight: bold; }
.ecsv-nav-right { display: flex; align-items: center; min-width: 180px; justify-content: flex-end; }
.ecsv-btn-ui { display: inline-flex; align-items: center; justify-content: center; height: 28px; padding: 0 10px; background: #ffffff; border: 1px solid #94a3b8; border-radius: 4px; color: #334155; font-size: 12px; cursor: pointer; gap: 5px; transition: all 0.1s ease; }
.ecsv-btn-ui:hover:not(:disabled) { background: #f1f5f9; border-color: #2563eb; color: #2563eb; }
.ecsv-btn-ui:disabled { opacity: 0.4; cursor: not-allowed; border-color: #e2e8f0; }
.ecsv-btn-dirty { background-color: #eff6ff !important; border-color: #3b82f6 !important; color: #1d4ed8 !important; box-shadow: 0 0 0 1px rgba(59, 130, 246, 0.2); }
.ecsv-btn-dirty:hover { background-color: #dbeafe !important; }
.ecsv-icon-btn { padding: 0 8px; }
.ecsv-danger:hover:not(:disabled) { color: #dc2626 !important; border-color: #dc2626 !important; background: #fef2f2 !important; }
.ecsv-warning { color: #d97706; border-color: #f59e0b; }
.ecsv-v-divider { width: 1px; height: 18px; background: #e2e8f0; margin: 0 4px; }
.ecsv-search-input { position: relative; width: 160px; }
.ecsv-search-input input { width: 100%; height: 28px; padding: 0 8px 0 28px; border: 1px solid #94a3b8; border-radius: 4px; font-size: 12px; }
.ecsv-search-icon { position: absolute; left: 8px; top: 8px; color: #64748b; }
.ecsv-search-tip {
  position: absolute;
  top: 32px;
  right: 0;
  background: #fff7ed;
  color: #9a3412;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid #ffedd5;
  white-space: nowrap;
  pointer-events: none;
  z-index: 10;
  box-shadow: 0 2px 4px rgba(0,0,0,0.05);
}

.ecsv-main { flex: 1; position: relative; background: #f1f5f9; overflow: hidden; }
.ecsv-grid { width: 100%; height: 100%; }
.ecsv-welcome { display: flex; height: 100%; align-items: center; justify-content: center; }
.ecsv-welcome-box { text-align: center; color: #64748b; }
.ecsv-btn-primary { background: #2563eb; color: white; border: none; padding: 8px 24px; border-radius: 6px; font-weight: 600; cursor: pointer; margin-top: 12px; }
.ecsv-footer { height: 26px; background: #ffffff; border-top: 1px solid #cbd5e1; display: flex; align-items: center; padding: 0 12px; font-size: 11px; color: #475569; justify-content: space-between; flex-shrink: 0; }
.ecsv-status-left { display: flex; align-items: center; gap: 6px; }
.ecsv-path { max-width: 600px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.ecsv-loader { position: absolute; inset: 0; background: rgba(255,255,255,0.8); z-index: 100; display: flex; flex-direction: column; align-items: center; justify-content: center; }
.ecsv-spin { width: 28px; height: 28px; border: 3px solid #e2e8f0; border-top-color: #2563eb; border-radius: 50%; animation: e-spin 0.8s linear infinite; margin-bottom: 8px; }
@keyframes e-spin { 100% { transform: rotate(360deg); } }
.ecsv-modal-overlay { position: absolute; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000; backdrop-filter: blur(2px); }
.ecsv-modal { background: white; width: 320px; border-radius: 12px; box-shadow: 0 20px 25px -5px rgba(0,0,0,0.1), 0 10px 10px -5px rgba(0,0,0,0.04); overflow: hidden; animation: modal-in 0.2s ease-out; }
@keyframes modal-in { from { transform: scale(0.95); opacity: 0; } to { transform: scale(1); opacity: 1; } }
.ecsv-modal-header { padding: 12px 16px; border-bottom: 1px solid #f1f5f9; display: flex; align-items: center; justify-content: space-between; }
.ecsv-modal-header h3 { margin: 0; font-size: 14px; color: #1e293b; }
.ecsv-modal-close { border: none; background: none; font-size: 20px; cursor: pointer; color: #94a3b8; }
.ecsv-modal-close:hover { color: #475569; }
.ecsv-modal-body { padding: 20px 24px; }
.ecsv-about-content { text-align: center; }
.ecsv-about-logo { width: 64px; height: 64px; margin-bottom: 12px; }
.ecsv-about-content h4 { margin: 0 0 4px; font-size: 18px; color: #2563eb; }
.ecsv-ver { font-size: 10px; background: #e2e8f0; color: #475569; padding: 2px 6px; border-radius: 10px; vertical-align: middle; }
.ecsv-about-desc { font-size: 12px; color: #64748b; margin-bottom: 20px; line-height: 1.5; }
.ecsv-info-grid { display: grid; grid-template-columns: 70px 1fr; text-align: left; gap: 8px; font-size: 12px; }
.ecsv-info-label { color: #94a3b8; }
.ecsv-info-val { color: #334155; }
.ecsv-link { color: #2563eb; text-decoration: none; display: inline-flex; align-items: center; gap: 4px; }
.ecsv-link:hover { text-decoration: underline; }
.ecsv-modal-footer { padding: 12px 16px; border-top: 1px solid #f1f5f9; display: flex; justify-content: flex-end; }
.ag-theme-quartz { --ag-font-size: 12px; --ag-grid-size: 4px; --ag-odd-row-background-color: #f1f5f9; }
.ag-theme-quartz .ag-cell, .ag-theme-quartz .ag-header-cell { border-right: 1px solid #e2e8f0 !important; }
</style>