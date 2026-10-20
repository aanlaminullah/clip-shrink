<script setup>
import { computed } from 'vue'
import {
  CheckCircle,
  FolderOpen,
  Play,
  ArrowRight,
  TrendingDown,
  Sparkles,
  Share2
} from 'lucide-vue-next'
import { useCompressionStore } from '../stores/compression'

const props = defineProps({
  file: {
    type: Object,
    required: true,
  }
})

const store = useCompressionStore()

const result = computed(() => props.file.result)

function formatBytes(bytes) {
  if (!bytes || bytes <= 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}
</script>

<template>
  <div
    v-if="result"
    class="glass-panel p-5 rounded-2xl border border-emerald-500/30 bg-gradient-to-b from-emerald-950/20 to-slate-900/60 shadow-xl space-y-4"
  >
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2.5">
        <div class="w-8 h-8 rounded-lg bg-emerald-500/20 border border-emerald-500/30 text-emerald-400 flex items-center justify-center">
          <CheckCircle :size="18" />
        </div>
        <div>
          <h4 class="text-sm font-bold text-slate-100 truncate max-w-sm">
            {{ file.fileName }}
          </h4>
          <span class="text-[11px] text-emerald-400 font-medium">Berhasil Dikompresi</span>
        </div>
      </div>

      <!-- Savings Pill -->
      <div class="flex items-center gap-1.5 px-3 py-1 rounded-full bg-emerald-500/20 border border-emerald-500/30 text-emerald-300 font-mono font-bold text-xs">
        <TrendingDown :size="14" />
        Hemat {{ result.savingsPercent.toFixed(1) }}%
      </div>
    </div>

    <!-- Size Comparison Banner -->
    <div class="p-3 rounded-xl bg-slate-900/80 border border-slate-700/50 flex items-center justify-between">
      <div class="text-left">
        <div class="text-[10px] uppercase font-semibold text-slate-500">Ukuran Awal</div>
        <div class="text-sm font-mono font-semibold text-slate-300">
          {{ formatBytes(result.originalSize) }}
        </div>
      </div>

      <div class="flex items-center gap-2 text-slate-500">
        <ArrowRight :size="16" class="text-indigo-400" />
      </div>

      <div class="text-left">
        <div class="text-[10px] uppercase font-semibold text-slate-500">Ukuran Akhir</div>
        <div class="text-sm font-mono font-bold text-emerald-400">
          {{ formatBytes(result.compressedSize) }}
        </div>
      </div>

      <div class="text-right pl-4 border-l border-slate-700/60">
        <div class="text-[10px] uppercase font-semibold text-slate-500">Ruang Dihemat</div>
        <div class="text-sm font-mono font-bold text-indigo-400">
          {{ formatBytes(result.savedBytes) }}
        </div>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2 pt-1">
      <button
        type="button"
        @click="store.openFile(result.outputPath)"
        class="flex-1 py-2 px-3 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold flex items-center justify-center gap-2 shadow-md shadow-indigo-600/30 transition-all active:scale-95 cursor-pointer"
      >
        <Play :size="14" class="fill-current" />
        Putar Video
      </button>

      <button
        type="button"
        @click="store.openFolder(result.outputPath)"
        class="flex-1 py-2 px-3 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold flex items-center justify-center gap-2 border border-slate-700 transition-all active:scale-95 cursor-pointer"
      >
        <FolderOpen :size="14" />
        Buka di Explorer
      </button>
    </div>
  </div>
</template>
