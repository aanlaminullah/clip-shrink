<script setup>
import { computed } from 'vue'
import {
  Activity,
  Gauge,
  Clock,
  Square,
  AlertTriangle,
  FileVideo
} from 'lucide-vue-next'
import { useCompressionStore } from '../stores/compression'

const store = useCompressionStore()

const activeFile = computed(() => store.activeFile)

function formatDuration(sec) {
  if (!sec || sec <= 0) return '00:00'
  const m = Math.floor(sec / 60)
  const s = Math.floor(sec % 60)
  const h = Math.floor(m / 60)
  if (h > 0) {
    const remM = m % 60
    return `${h}:${remM.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  }
  return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
}
</script>

<template>
  <div
    v-if="store.isProcessing && activeFile"
    class="glass-panel p-5 rounded-2xl border border-indigo-500/40 shadow-2xl space-y-4 animate-in fade-in slide-in-from-bottom-2 duration-300"
  >
    <div class="flex items-center justify-between gap-4">
      <div class="flex items-center gap-3 min-w-0">
        <div class="w-10 h-10 rounded-xl bg-indigo-500/20 text-indigo-400 flex items-center justify-center shrink-0 border border-indigo-500/30">
          <Activity :size="20" class="animate-pulse" />
        </div>
        <div class="min-w-0">
          <div class="text-xs font-semibold uppercase tracking-wider text-indigo-400 flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-indigo-400 animate-ping"></span>
            Mengompresi Berkas
          </div>
          <h4 class="text-sm font-bold text-slate-100 truncate max-w-sm" :title="activeFile.fileName">
            {{ activeFile.fileName }}
          </h4>
        </div>
      </div>

      <!-- Cancel Button -->
      <button
        type="button"
        @click="store.cancelActiveCompression"
        class="inline-flex items-center gap-2 px-3.5 py-2 rounded-xl bg-rose-500/15 hover:bg-rose-500/25 text-rose-300 hover:text-rose-200 border border-rose-500/30 font-semibold text-xs transition-all active:scale-95 cursor-pointer"
      >
        <Square :size="13" class="fill-current" />
        Batalkan
      </button>
    </div>

    <!-- Main Progress Bar -->
    <div class="space-y-1.5">
      <div class="flex items-center justify-between text-xs font-mono">
        <span class="font-bold text-indigo-300 text-sm">
          {{ activeFile.progress.toFixed(1) }}%
        </span>
        <span class="text-slate-400">
          {{ formatDuration(activeFile.currentTime) }} / {{ formatDuration(activeFile.totalDuration) }}
        </span>
      </div>

      <div class="h-3 w-full bg-slate-800 rounded-full overflow-hidden p-0.5 border border-slate-700/60 shadow-inner">
        <div
          class="h-full bg-gradient-to-r from-indigo-500 via-cyan-400 to-indigo-500 rounded-full transition-all duration-200 relative overflow-hidden"
          :style="{ width: `${activeFile.progress}%` }"
        >
          <!-- Shimmer effect -->
          <div class="absolute inset-0 bg-gradient-to-r from-transparent via-white/25 to-transparent animate-shimmer"></div>
        </div>
      </div>
    </div>

    <!-- Stats row -->
    <div class="grid grid-cols-3 gap-2 pt-1 border-t border-slate-700/50 text-xs">
      <div class="flex items-center gap-2 p-2 rounded-xl bg-slate-800/50 border border-slate-700/40">
        <Gauge :size="15" class="text-indigo-400 shrink-0" />
        <div class="min-w-0">
          <div class="text-[10px] text-slate-400 uppercase">Kecepatan</div>
          <div class="font-mono font-semibold text-slate-200 truncate">
            {{ activeFile.speed || 'Menghitung...' }}
          </div>
        </div>
      </div>

      <div class="flex items-center gap-2 p-2 rounded-xl bg-slate-800/50 border border-slate-700/40">
        <Activity :size="15" class="text-cyan-400 shrink-0" />
        <div class="min-w-0">
          <div class="text-[10px] text-slate-400 uppercase">Render FPS</div>
          <div class="font-mono font-semibold text-slate-200 truncate">
            {{ activeFile.fps ? activeFile.fps.toFixed(0) + ' fps' : '-' }}
          </div>
        </div>
      </div>

      <div class="flex items-center gap-2 p-2 rounded-xl bg-slate-800/50 border border-slate-700/40">
        <Clock :size="15" class="text-amber-400 shrink-0" />
        <div class="min-w-0">
          <div class="text-[10px] text-slate-400 uppercase">Sisa Waktu</div>
          <div class="font-mono font-semibold text-slate-200 truncate">
            {{ activeFile.etaSeconds > 0 ? '~' + formatDuration(activeFile.etaSeconds) : 'Menghitung...' }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
