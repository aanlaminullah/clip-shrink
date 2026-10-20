<script setup>
import { computed } from 'vue'
import {
  FileVideo,
  Clock,
  HardDrive,
  Tv,
  X,
  CheckCircle2,
  AlertCircle,
  Loader2,
  PlayCircle,
  FolderOpen,
  Sparkles,
  ArrowRight
} from 'lucide-vue-next'
import { useCompressionStore } from '../stores/compression'

const props = defineProps({
  file: {
    type: Object,
    required: true,
  }
})

const store = useCompressionStore()

const estimatedData = computed(() => {
  return store.estimateForFile(props.file)
})

function formatBytes(bytes) {
  if (!bytes || bytes <= 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}

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

const resolutionBadge = computed(() => {
  if (!props.file.metadata) return ''
  const { width, height } = props.file.metadata
  if (!width || !height) return ''
  if (height >= 2160 || width >= 3840) return '4K UHD'
  if (height >= 1440 || width >= 2560) return '1440p QHD'
  if (height >= 1080 || width >= 1920) return '1080p FHD'
  if (height >= 720 || width >= 1280) return '720p HD'
  return `${width}x${height}`
})
</script>

<template>
  <div
    :class="[
      'relative p-4 rounded-2xl border transition-all duration-300',
      file.status === 'processing'
        ? 'bg-slate-800/90 border-indigo-500/60 shadow-[0_0_25px_rgba(99,102,241,0.2)]'
        : file.status === 'completed'
        ? 'bg-slate-800/70 border-emerald-500/40'
        : file.status === 'error'
        ? 'bg-slate-800/70 border-rose-500/40'
        : 'bg-slate-800/50 border-slate-700/60 hover:border-slate-600'
    ]"
  >
    <div class="flex items-start justify-between gap-3">
      <!-- Icon & Main Details -->
      <div class="flex items-start gap-3.5 min-w-0 flex-1">
        <div
          :class="[
            'w-12 h-12 rounded-xl flex items-center justify-center shrink-0 border transition-colors',
            file.status === 'completed'
              ? 'bg-emerald-500/15 border-emerald-500/30 text-emerald-400'
              : file.status === 'processing'
              ? 'bg-indigo-500/20 border-indigo-500/40 text-indigo-400 animate-pulse-subtle'
              : file.status === 'error'
              ? 'bg-rose-500/15 border-rose-500/30 text-rose-400'
              : 'bg-slate-700/40 border-slate-600/40 text-slate-300'
          ]"
        >
          <FileVideo :size="24" />
        </div>

        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2 mb-1 flex-wrap">
            <h4 class="text-sm font-semibold text-slate-100 truncate max-w-md" :title="file.filePath">
              {{ file.fileName }}
            </h4>

            <!-- Status Badges -->
            <span
              v-if="file.status === 'processing'"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-indigo-500/20 text-indigo-300 border border-indigo-500/30"
            >
              <Loader2 :size="10" class="animate-spin" />
              Mengompresi {{ file.progress.toFixed(0) }}%
            </span>

            <span
              v-else-if="file.status === 'completed'"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-500/20 text-emerald-300 border border-emerald-500/30"
            >
              <CheckCircle2 :size="10" />
              Hemat {{ file.result?.savingsPercent ? file.result.savingsPercent.toFixed(1) + '%' : 'Selesai' }}
            </span>

            <span
              v-else-if="file.status === 'error'"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-rose-500/20 text-rose-300 border border-rose-500/30"
            >
              <AlertCircle :size="10" />
              Gagal
            </span>

            <span
              v-else-if="file.status === 'cancelled'"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-amber-500/20 text-amber-300 border border-amber-500/30"
            >
              Dibatalkan
            </span>

            <!-- Overwrite notice when suffix is disabled -->
            <span
              v-else-if="file.status === 'idle' && store.removeCompressSuffix && store.isDestinationSameAsOriginal(file)"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-amber-500/20 text-amber-300 border border-amber-500/30"
              title="File asli akan ditimpa (replace) saat kompresi"
            >
              Menimpa File Asli
            </span>
            <span
              v-else-if="file.status === 'idle' && store.removeCompressSuffix && !store.isDestinationSameAsOriginal(file)"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-500/15 text-emerald-300 border border-emerald-500/30"
              title="Disimpan ke folder tujuan baru tanpa menimpa"
            >
              Folder Baru
            </span>
          </div>

          <!-- Metadata Tags -->
          <div v-if="file.metadata" class="flex flex-wrap items-center gap-3 text-xs text-slate-400">
            <span class="inline-flex items-center gap-1">
              <HardDrive :size="13" class="text-slate-500" />
              {{ formatBytes(file.fileSize) }}
            </span>

            <span class="inline-flex items-center gap-1">
              <Clock :size="13" class="text-slate-500" />
              {{ formatDuration(file.metadata.duration) }}
            </span>

            <span v-if="resolutionBadge" class="inline-flex items-center gap-1">
              <Tv :size="13" class="text-slate-500" />
              {{ resolutionBadge }}
            </span>

            <span v-if="file.metadata.codec" class="px-1.5 py-0.2 rounded bg-slate-700/50 text-[10px] font-mono text-slate-300 uppercase">
              {{ file.metadata.codec }}
            </span>

            <span v-if="file.metadata.fps" class="text-[11px] text-slate-400 font-mono">
              {{ file.metadata.fps }} fps
            </span>
          </div>

          <div v-else-if="file.loadingMeta" class="text-xs text-slate-400 flex items-center gap-1.5">
            <Loader2 :size="12" class="animate-spin text-indigo-400" />
            Membaca metadata video...
          </div>

          <div v-else-if="file.error" class="text-xs text-rose-400 mt-1">
            {{ file.error }}
          </div>

          <!-- Real-Time Size Estimation Bar (Dinamis Berbasis Kustomisasi) -->
          <div
            v-if="file.status === 'idle' && file.metadata"
            class="mt-2.5 p-2 rounded-xl bg-slate-900/60 border border-slate-700/60 flex items-center justify-between gap-3 flex-wrap"
          >
            <div class="flex items-center gap-2 text-xs">
              <span class="flex items-center gap-1 text-slate-400 text-[11px] font-medium">
                <Sparkles :size="12" class="text-indigo-400" />
                Estimasi Hasil:
              </span>
              <div class="flex items-center gap-1.5 font-mono">
                <span class="text-slate-400 line-through text-[11px]">{{ formatBytes(file.fileSize) }}</span>
                <ArrowRight :size="11" class="text-slate-500" />
                <span class="font-bold text-indigo-300">~{{ formatBytes(estimatedData.size) }}</span>
              </div>
            </div>

            <div class="flex items-center gap-2">
              <span
                :class="[
                  'px-2 py-0.5 rounded-lg text-[10px] font-bold font-mono border',
                  estimatedData.savingsPercent > 0
                    ? 'bg-emerald-500/15 border-emerald-500/30 text-emerald-300'
                    : 'bg-slate-700/50 border-slate-600/50 text-slate-300'
                ]"
              >
                Hemat ~{{ estimatedData.savingsPercent }}% (~{{ formatBytes(estimatedData.savedBytes) }})
              </span>
            </div>
          </div>

          <!-- Output path preview when idle -->
          <div v-if="file.status === 'idle'" class="mt-2 text-[11px] text-slate-400 flex items-center gap-1.5 truncate">
            <span class="text-slate-500 shrink-0">Hasil:</span>
            <span class="font-mono text-slate-300 truncate" :title="store.computeOutputFilePath(file)">
              {{ store.computeOutputFilePath(file) }}
            </span>
          </div>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center gap-1 shrink-0">
        <!-- Post-completion quick actions -->
        <template v-if="file.status === 'completed' && file.result">
          <button
            type="button"
            @click="store.openFile(file.result.outputPath)"
            title="Putar Hasil Kompresi"
            class="p-2 rounded-lg text-slate-400 hover:text-emerald-400 hover:bg-slate-700/60 transition-colors cursor-pointer"
          >
            <PlayCircle :size="18" />
          </button>
          <button
            type="button"
            @click="store.openFolder(file.result.outputPath)"
            title="Buka Folder di Explorer"
            class="p-2 rounded-lg text-slate-400 hover:text-indigo-400 hover:bg-slate-700/60 transition-colors cursor-pointer"
          >
            <FolderOpen :size="18" />
          </button>
        </template>

        <!-- Remove file button -->
        <button
          type="button"
          @click="store.removeFile(file.id)"
          :disabled="file.status === 'processing'"
          title="Hapus dari antrean"
          class="p-2 rounded-lg text-slate-400 hover:text-rose-400 hover:bg-slate-700/60 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer"
        >
          <X :size="18" />
        </button>
      </div>
    </div>

    <!-- Active inline progress if processing -->
    <div v-if="file.status === 'processing'" class="mt-3 pt-3 border-t border-slate-700/60">
      <div class="flex items-center justify-between text-xs font-mono text-slate-300 mb-1.5">
        <span class="text-indigo-400 font-semibold">{{ file.progress.toFixed(1) }}%</span>
        <div class="flex items-center gap-3 text-slate-400">
          <span v-if="file.speed">Kecepatan: {{ file.speed }}</span>
          <span v-if="file.fps">FPS: {{ file.fps.toFixed(0) }}</span>
          <span v-if="file.etaSeconds > 0">Sisa waktu: ~{{ formatDuration(file.etaSeconds) }}</span>
        </div>
      </div>

      <div class="h-1.5 w-full bg-slate-700/60 rounded-full overflow-hidden">
        <div
          class="h-full bg-gradient-to-r from-indigo-500 via-cyan-400 to-indigo-500 rounded-full transition-all duration-200"
          :style="{ width: `${file.progress}%` }"
        ></div>
      </div>
    </div>
  </div>
</template>
