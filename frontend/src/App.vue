<script setup>
import { onMounted, computed, ref } from 'vue'
import {
  Film,
  Play,
  Trash2,
  FolderOpen,
  FolderCheck,
  CheckCircle2,
  AlertTriangle,
  Info,
  Check,
  Zap,
  TrendingDown,
  Layers,
  Sparkles,
  RefreshCw
} from 'lucide-vue-next'
import { useCompressionStore } from './stores/compression'
import DropZone from './components/DropZone.vue'
import VideoCard from './components/VideoCard.vue'
import PresetSelector from './components/PresetSelector.vue'
import ProgressBar from './components/ProgressBar.vue'
import ResultCard from './components/ResultCard.vue'

const store = useCompressionStore()
const activeTab = ref('queue') // 'queue' | 'results'
const showOverwriteWarningModal = ref(false)

onMounted(() => {
  store.init()

  // Prevent default window navigation on drag and drop so file drops are always caught
  window.addEventListener('dragover', (e) => {
    e.preventDefault()
  })
  window.addEventListener('drop', (e) => {
    e.preventDefault()
  })
})

function handleStartClick() {
  if (store.hasOverwriteRisk) {
    showOverwriteWarningModal.value = true
  } else {
    store.startAll()
  }
}

function confirmOverwriteAndStart() {
  showOverwriteWarningModal.value = false
  store.startAll()
}

const hasFiles = computed(() => store.files.length > 0)
const hasCompleted = computed(() => store.completedFiles.length > 0)

function formatBytes(bytes) {
  if (!bytes || bytes <= 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}
</script>

<template>
  <div class="min-h-screen bg-[#090d16] text-slate-100 flex flex-col justify-between selection:bg-indigo-500 selection:text-white">
    <!-- Top Navigation Header -->
    <header class="sticky top-0 z-40 bg-[#090d16]/80 backdrop-blur-md border-b border-slate-800/80 px-6 py-3.5">
      <div class="max-w-5xl mx-auto flex items-center justify-between">
        <!-- Logo -->
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-gradient-to-tr from-indigo-600 to-cyan-500 p-0.5 shadow-lg shadow-indigo-600/30">
            <div class="w-full h-full bg-[#090d16] rounded-[10px] flex items-center justify-center text-indigo-400">
              <Film :size="20" />
            </div>
          </div>
          <div>
            <div class="flex items-center gap-2">
              <h1 class="text-base font-extrabold tracking-tight text-white">ClipShrink</h1>
              <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-indigo-500/15 border border-indigo-500/30 text-indigo-300 font-semibold">
                v1.0 Windows
              </span>
            </div>
            <p class="text-[11px] text-slate-400">Kompresor Video Berkecepatan Tinggi Offline</p>
          </div>
        </div>

        <!-- Engine Status & Quick Actions -->
        <div class="flex items-center gap-3">
          <!-- Binary Status -->
          <div
            v-if="store.binaryInfo"
            class="hidden sm:flex items-center gap-1.5 px-3 py-1 rounded-full bg-slate-800/80 border border-slate-700/60 text-xs font-mono text-emerald-400"
            title="Engine FFmpeg siap digunakan"
          >
            <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
            <span>FFmpeg Ready</span>
          </div>

          <!-- Clear queue button -->
          <button
            v-if="hasFiles"
            type="button"
            @click="store.clearAll"
            :disabled="store.isProcessing"
            class="px-3 py-1.5 rounded-xl bg-slate-800/80 hover:bg-slate-800 text-slate-400 hover:text-rose-400 border border-slate-700/60 text-xs font-medium flex items-center gap-1.5 transition-all disabled:opacity-40 cursor-pointer"
          >
            <Trash2 :size="13" />
            <span>Kosongkan</span>
          </button>
        </div>
      </div>
    </header>

    <!-- Main Container -->
    <main class="max-w-5xl w-full mx-auto p-6 space-y-6 flex-1">
      <!-- Drag & Drop Zone -->
      <DropZone />

      <!-- Preset Selector Options -->
      <PresetSelector />

      <!-- Output Folder & Naming Configuration -->
      <div class="p-4 rounded-2xl bg-slate-800/40 border border-slate-700/60 space-y-3.5 text-xs">
        <!-- Destination Folder Row -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-3 border-b border-slate-700/50">
          <div class="flex items-center gap-2.5 min-w-0">
            <div class="w-8 h-8 rounded-xl bg-indigo-500/15 border border-indigo-500/30 flex items-center justify-center text-indigo-400 shrink-0">
              <FolderOpen :size="16" />
            </div>
            <div class="min-w-0">
              <div class="text-[11px] text-slate-400 font-medium">Folder Tujuan Penyimpanan:</div>
              <div class="truncate">
                <span v-if="store.customOutputFolder" class="font-mono text-slate-200 font-semibold text-xs">
                  {{ store.customOutputFolder }}
                </span>
                <span v-else class="text-slate-400 italic">
                  Folder yang sama dengan berkas video asli
                </span>
              </div>
            </div>
          </div>

          <div class="flex items-center gap-2 shrink-0 self-end sm:self-auto">
            <button
              v-if="store.customOutputFolder"
              type="button"
              @click="store.clearOutputFolder"
              class="px-2.5 py-1.5 rounded-lg text-slate-400 hover:text-rose-400 hover:bg-slate-700/40 text-[11px] font-medium transition-all cursor-pointer"
            >
              Reset ke Asal
            </button>
            <button
              type="button"
              @click="store.pickOutputFolderNative"
              class="px-3 py-1.5 rounded-xl bg-slate-700/70 hover:bg-slate-700 text-slate-200 font-semibold border border-slate-600/60 hover:border-indigo-500/50 transition-all cursor-pointer flex items-center gap-1.5 shadow-sm"
            >
              <FolderCheck :size="14" class="text-indigo-400" />
              <span>Ubah Folder</span>
            </button>
          </div>
        </div>

        <!-- File Naming Options Row -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <label class="relative inline-flex items-center cursor-pointer select-none">
              <input
                type="checkbox"
                v-model="store.removeCompressSuffix"
                class="sr-only peer"
              />
              <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-600"></div>
            </label>
            <div>
              <div class="text-xs font-semibold text-slate-200">
                Hilangkan sufiks <code class="px-1.5 py-0.5 rounded bg-slate-700 text-indigo-300 font-mono text-[11px]">_compress</code> pada nama file
              </div>
              <div class="text-[11px] text-slate-400">
                <template v-if="store.removeCompressSuffix">
                  Nama file hasil akan sama persis dengan nama video aslinya.
                </template>
                <template v-else>
                  Menambahkan akhiran <code class="text-slate-300 font-mono">_compressed.mp4</code> agar tidak menimpa file asli.
                </template>
              </div>
            </div>
          </div>

          <!-- Filename Preview Pill -->
          <div class="px-2.5 py-1 rounded-lg bg-slate-900/60 border border-slate-700/60 text-[11px] font-mono text-slate-400 shrink-0">
            Contoh: <span :class="store.removeCompressSuffix ? 'text-amber-300 font-semibold' : 'text-indigo-300'">video{{ store.removeCompressSuffix ? '' : '_compressed' }}.mp4</span>
          </div>
        </div>

        <!-- Dynamic Destination Warning if same folder and suffix removed -->
        <transition
          enter-active-class="transition duration-200 ease-out"
          enter-from-class="opacity-0 -translate-y-1"
          enter-to-class="opacity-100 translate-y-0"
          leave-active-class="transition duration-150 ease-in"
          leave-from-class="opacity-100 translate-y-0"
          leave-to-class="opacity-0 -translate-y-1"
        >
          <div
            v-if="store.removeCompressSuffix && store.hasOverwriteRisk"
            class="p-3 rounded-xl bg-amber-500/15 border border-amber-500/40 text-amber-200 text-xs flex items-start gap-2.5"
          >
            <AlertTriangle :size="16" class="text-amber-400 shrink-0 mt-0.5" />
            <div class="space-y-0.5">
              <span class="font-bold text-amber-300">Peringatan: Berpotensi Menimpa File Asli (Replace)</span>
              <p class="text-amber-200/90 leading-relaxed text-[11px]">
                Folder tujuan penyimpanan sama dengan folder file asli dan opsi sufiks <code class="px-1 py-0.2 rounded bg-amber-500/20 text-amber-300 font-mono">_compress</code> dihilangkan. Berkas video asli berpotensi tertimpa (replace) secara permanen.
              </p>
            </div>
          </div>

          <div
            v-else-if="store.removeCompressSuffix && !store.hasOverwriteRisk && store.customOutputFolder"
            class="p-2.5 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 text-xs flex items-center gap-2"
          >
            <CheckCircle2 :size="15" class="text-emerald-400 shrink-0" />
            <span class="text-[11px]">Folder tujuan berbeda dengan folder file asli. Berkas asli aman dan proses kompresi akan berjalan tanpa peringatan penimpaan.</span>
          </div>
        </transition>
      </div>

      <!-- Real-time Active Progress Bar -->
      <ProgressBar />

      <!-- Queue Header and List -->
      <div v-if="hasFiles" class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2 flex-wrap">
            <Layers :size="16" class="text-indigo-400" />
            <h3 class="text-sm font-bold uppercase tracking-wider text-slate-300">
              Antrean Berkas ({{ store.totalFiles }})
            </h3>
            <span class="text-xs text-slate-500 font-mono">
              Total: {{ formatBytes(store.totalOriginalSize) }}
            </span>
            <span
              v-if="store.pendingFiles.length > 0 && store.totalEstimatedSize > 0"
              class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-mono bg-indigo-500/15 border border-indigo-500/30 text-indigo-300"
              title="Estimasi total ukuran akhir berdasarkan konfigurasi yang dipilih"
            >
              <Sparkles :size="11" class="text-indigo-400" />
              <span>Estimasi: ~{{ formatBytes(store.totalEstimatedSize) }}</span>
              <span class="text-emerald-400 font-bold">(-{{ store.totalEstimatedSavingsPercent }}%)</span>
            </span>
          </div>

          <!-- Tab Selector if completed files exist -->
          <div v-if="hasCompleted" class="flex items-center gap-1 p-1 bg-slate-800/80 rounded-xl border border-slate-700 text-xs">
            <button
              @click="activeTab = 'queue'"
              :class="[
                'px-2.5 py-1 rounded-lg font-medium transition-all cursor-pointer',
                activeTab === 'queue' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-slate-200'
              ]"
            >
              Semua ({{ store.totalFiles }})
            </button>
            <button
              @click="activeTab = 'results'"
              :class="[
                'px-2.5 py-1 rounded-lg font-medium transition-all cursor-pointer',
                activeTab === 'results' ? 'bg-emerald-600 text-white' : 'text-slate-400 hover:text-slate-200'
              ]"
            >
              Hasil ({{ store.completedFiles.length }})
            </button>
          </div>
        </div>

        <!-- Queue view -->
        <div v-if="activeTab === 'queue'" class="space-y-2.5">
          <VideoCard
            v-for="file in store.files"
            :key="file.id"
            :file="file"
          />
        </div>

        <!-- Results view -->
        <div v-else class="space-y-3">
          <ResultCard
            v-for="file in store.completedFiles"
            :key="'res_' + file.id"
            :file="file"
          />
        </div>
      </div>
    </main>

    <!-- Bottom Sticky Action Bar -->
    <footer v-if="hasFiles" class="sticky bottom-0 z-40 bg-[#090d16]/90 backdrop-blur-md border-t border-slate-800/80 px-6 py-4">
      <div class="max-w-5xl mx-auto flex items-center justify-between gap-4">
        <div class="flex items-center gap-4 text-xs text-slate-400 flex-wrap">
          <div>
            <span class="font-bold text-slate-200">{{ store.pendingFiles.length }}</span> belum diproses
          </div>
          <div
            v-if="store.pendingFiles.length > 0 && store.totalEstimatedSize > 0 && !store.isProcessing"
            class="flex items-center gap-1.5 text-indigo-300 font-semibold font-mono"
            title="Estimasi ukuran akhir seluruh video yang belum diproses"
          >
            <Sparkles :size="14" class="text-indigo-400" />
            <span>Estimasi Hasil: ~{{ formatBytes(store.totalEstimatedSize) }}</span>
            <span class="text-emerald-400">(-{{ store.totalEstimatedSavingsPercent }}%)</span>
          </div>
          <div v-if="store.completedFiles.length > 0" class="flex items-center gap-1.5 text-emerald-400 font-semibold font-mono">
            <TrendingDown :size="14" />
            Total hemat: {{ formatBytes(store.totalSavedBytes) }} ({{ store.overallSavingsPercent }}%)
          </div>
        </div>

        <!-- Action Button -->
        <div class="flex items-center gap-3">
          <button
            v-if="!store.isProcessing"
            type="button"
            @click="handleStartClick"
            :disabled="store.pendingFiles.length === 0"
            class="px-6 py-3 rounded-2xl bg-gradient-to-r from-indigo-600 via-indigo-500 to-cyan-500 hover:from-indigo-500 hover:to-cyan-400 text-white text-sm font-bold shadow-lg shadow-indigo-600/30 hover:shadow-indigo-500/40 active:scale-95 transition-all disabled:opacity-40 disabled:cursor-not-allowed flex items-center gap-2 cursor-pointer"
          >
            <Play :size="16" class="fill-current" />
            <span>Mulai Kompresi ({{ store.pendingFiles.length }} Video)</span>
          </button>

          <button
            v-else
            type="button"
            @click="store.cancelActiveCompression"
            class="px-6 py-3 rounded-2xl bg-rose-600 hover:bg-rose-500 text-white text-sm font-bold shadow-lg shadow-rose-600/30 transition-all active:scale-95 flex items-center gap-2 cursor-pointer"
          >
            <span>Hentikan Proses</span>
          </button>
        </div>
      </div>
    </footer>

    <!-- Overwrite Warning Confirmation Modal -->
    <transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="showOverwriteWarningModal"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-md"
      >
        <div class="max-w-md w-full bg-[#0d1322] border border-amber-500/50 rounded-3xl p-6 shadow-2xl shadow-amber-950/60 space-y-4">
          <div class="flex items-center gap-3">
            <div class="w-11 h-11 rounded-2xl bg-amber-500/20 border border-amber-500/40 flex items-center justify-center text-amber-400 shrink-0">
              <AlertTriangle :size="24" />
            </div>
            <div>
              <h3 class="text-base font-bold text-slate-100">Peringatan: Penimpaan File Asli</h3>
              <p class="text-xs text-amber-400 font-medium">Potensi replace berkas video sumber</p>
            </div>
          </div>

          <div class="space-y-2.5 text-xs text-slate-300 leading-relaxed">
            <p>
              Folder tujuan penyimpanan <strong class="text-amber-300">sama dengan folder file asli</strong> dan opsi sufiks <code class="px-1.5 py-0.5 rounded bg-slate-800 text-amber-300 font-mono">_compress</code> dinonaktifkan.
            </p>
            <p>
              Proses kompresi ini berpotensi <strong class="text-rose-400 font-semibold">menimpa (replace) berkas asli</strong> untuk <span class="font-bold text-white">{{ store.overwriteRiskFiles.length }} video</span> berikut:
            </p>

            <div class="max-h-32 overflow-y-auto space-y-1 p-2.5 rounded-xl bg-slate-950/70 border border-slate-800 text-[11px] font-mono text-slate-300">
              <div v-for="f in store.overwriteRiskFiles" :key="f.id" class="truncate flex items-center gap-2">
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400 shrink-0"></span>
                <span class="truncate">{{ f.fileName }}</span>
              </div>
            </div>

            <p class="text-slate-400 text-[11px]">
              Pastikan Anda telah memiliki cadangan (backup) jika masih memerlukan berkas asli sebelum melanjutkan proses ini.
            </p>
          </div>

          <div class="flex items-center justify-end gap-2.5 pt-2">
            <button
              type="button"
              @click="showOverwriteWarningModal = false"
              class="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-semibold transition-all cursor-pointer"
            >
              Batal
            </button>
            <button
              type="button"
              @click="confirmOverwriteAndStart"
              class="px-4 py-2 rounded-xl bg-gradient-to-r from-amber-600 to-rose-600 hover:from-amber-500 hover:to-rose-500 text-white text-xs font-bold shadow-lg shadow-amber-600/30 transition-all cursor-pointer flex items-center gap-1.5"
            >
              <AlertTriangle :size="14" />
              <span>Ya, Lanjutkan & Timpa File</span>
            </button>
          </div>
        </div>
      </div>
    </transition>

    <!-- Toast Notification Overlay -->
    <transition
      enter-active-class="transform ease-out duration-300 transition"
      enter-from-class="translate-y-4 opacity-0"
      enter-to-class="translate-y-0 opacity-100"
      leave-active-class="transition ease-in duration-200"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="store.toast"
        :class="[
          'fixed bottom-20 right-6 z-50 p-4 rounded-2xl shadow-2xl border flex items-center gap-3 max-w-md text-xs font-medium backdrop-blur-lg',
          store.toast.type === 'error'
            ? 'bg-rose-950/90 border-rose-500/40 text-rose-200'
            : store.toast.type === 'success'
            ? 'bg-emerald-950/90 border-emerald-500/40 text-emerald-200'
            : 'bg-slate-900/90 border-slate-700/60 text-slate-200'
        ]"
      >
        <AlertTriangle v-if="store.toast.type === 'error'" :size="18" class="text-rose-400 shrink-0" />
        <CheckCircle2 v-else-if="store.toast.type === 'success'" :size="18" class="text-emerald-400 shrink-0" />
        <Info v-else :size="18" class="text-indigo-400 shrink-0" />
        <span>{{ store.toast.message }}</span>
      </div>
    </transition>
  </div>
</template>
