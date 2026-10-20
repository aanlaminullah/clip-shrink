<script setup>
import { computed } from 'vue'
import {
  Youtube,
  Zap,
  Sliders,
  Cpu,
  Sparkles,
  Check
} from 'lucide-vue-next'
import { useCompressionStore } from '../stores/compression'

const store = useCompressionStore()

const presets = computed(() => [
  {
    id: 'youtube',
    name: 'YouTube Standard',
    codec: store.customConfig.useHardwareAcc ? 'H.264 (NVENC RTX/GTX)' : 'H.264 (libx264)',
    badge: 'Disarankan',
    badgeColor: 'bg-indigo-500/20 text-indigo-300 border-indigo-500/30',
    icon: Youtube,
    description: 'Kualitas optimal dengan kompatibilitas 100% di semua perangkat & media player.',
    rateInfo: store.customConfig.useHardwareAcc ? 'NVENC p4 • AAC 128k • Cepat' : 'CRF 23 • AAC 128k • FastStart'
  },
  {
    id: 'hevc',
    name: 'Maximum Savings',
    codec: store.customConfig.useHardwareAcc ? 'HEVC (NVENC RTX/GTX)' : 'HEVC / H.265',
    badge: 'Ukuran Terkecil',
    badgeColor: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30',
    icon: Zap,
    description: 'Penyusutan ukuran maksimal hingga 50-70% lebih kecil pada ketajaman visual serupa.',
    rateInfo: store.customConfig.useHardwareAcc ? 'NVENC HEVC • AAC 128k' : 'CRF 28 • AAC 128k • Efisiensi'
  },
  {
    id: 'custom',
    name: 'Custom Mode',
    codec: 'Kustom CRF & Resolusi',
    badge: 'Lanjutan',
    badgeColor: 'bg-purple-500/20 text-purple-300 border-purple-500/30',
    icon: Sliders,
    description: 'Atur slider CRF manual dan opsi downscaling resolusi secara fleksibel.',
    rateInfo: 'CRF 18–32 • Resolusi Downscale'
  }
])

const crfQualityLabel = computed(() => {
  const v = store.customConfig.crf
  if (v <= 20) return 'Kualitas Sangat Tinggi (Ukuran lebih besar)'
  if (v <= 24) return 'Keseimbangan Ideal (Standar)'
  if (v <= 28) return 'Ukuran Sangat Hemat'
  return 'Kompresi Agresif (Paling Kecil)'
})

const customBitrateEstimate = computed(() => {
  const crf = store.customConfig.crf || 23
  const isHevc = store.customConfig.customCodec?.includes('hevc') || store.customConfig.customCodec?.includes('265')
  const res = store.customConfig.resolution || 'original'

  let baseBps = 3800000
  if (res === '1080p') baseBps *= 1.0
  else if (res === '720p') baseBps *= 0.55
  else if (res === '480p') baseBps *= 0.30

  if (isHevc) baseBps *= 0.60

  const crfFactor = Math.pow(2, (23 - crf) / 6)
  const totalBps = baseBps * crfFactor + 128000
  const mbps = (totalBps / 1000000).toFixed(1)
  return `${mbps} Mbps`
})

function formatBytes(bytes) {
  if (!bytes || bytes <= 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <Sparkles :size="16" class="text-indigo-400" />
        <h3 class="text-sm font-bold uppercase tracking-wider text-slate-300">
          Pilihan Preset Kompresi
        </h3>
      </div>
      <span class="text-xs text-slate-500">Pilih algoritma encoding yang sesuai</span>
    </div>

    <!-- Cards Grid -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <div
        v-for="preset in presets"
        :key="preset.id"
        @click="store.selectedPreset = preset.id"
        :class="[
          'relative p-4 rounded-2xl border cursor-pointer transition-all duration-200 flex flex-col justify-between text-left',
          store.selectedPreset === preset.id
            ? 'bg-gradient-to-b from-indigo-950/40 to-slate-800/90 border-indigo-500 shadow-[0_0_20px_rgba(99,102,241,0.25)] ring-1 ring-indigo-500/50'
            : 'bg-slate-800/40 border-slate-700/60 hover:border-slate-600 hover:bg-slate-800/60'
        ]"
      >
        <div>
          <!-- Header -->
          <div class="flex items-start justify-between gap-2 mb-2.5">
            <div
              :class="[
                'w-9 h-9 rounded-xl flex items-center justify-center transition-colors',
                store.selectedPreset === preset.id
                  ? 'bg-indigo-600 text-white shadow-md shadow-indigo-600/40'
                  : 'bg-slate-700/50 text-slate-400'
              ]"
            >
              <component :is="preset.icon" :size="18" />
            </div>

            <span
              :class="[
                'text-[10px] font-semibold px-2 py-0.5 rounded-full border',
                preset.badgeColor
              ]"
            >
              {{ preset.badge }}
            </span>
          </div>

          <!-- Titles -->
          <h4 class="text-sm font-bold text-slate-100 mb-0.5">
            {{ preset.name }}
          </h4>
          <div class="text-[11px] font-mono text-indigo-400 mb-2">
            {{ preset.codec }}
          </div>

          <p class="text-xs text-slate-400 leading-relaxed mb-2.5">
            {{ preset.description }}
          </p>

          <!-- Real-Time Estimated Size for this preset if files exist in queue -->
          <div
            v-if="store.pendingFiles.length > 0 && store.presetEstimates[preset.id].size > 0"
            class="mb-3 px-2.5 py-1.5 rounded-xl bg-slate-900/70 border border-slate-700/60 flex items-center justify-between text-[11px] font-mono"
          >
            <span class="text-slate-400">Estimasi Total:</span>
            <span class="text-emerald-400 font-bold">
              ~{{ formatBytes(store.presetEstimates[preset.id].size) }} (-{{ store.presetEstimates[preset.id].savings }}%)
            </span>
          </div>
        </div>

        <div class="pt-2 border-t border-slate-700/50 text-[11px] font-mono text-slate-400 flex items-center justify-between">
          <span>{{ preset.rateInfo }}</span>
          <div
            :class="[
              'w-4 h-4 rounded-full border flex items-center justify-center',
              store.selectedPreset === preset.id
                ? 'border-indigo-500 bg-indigo-500'
                : 'border-slate-600 bg-transparent'
            ]"
          >
            <div v-if="store.selectedPreset === preset.id" class="w-1.5 h-1.5 rounded-full bg-white"></div>
          </div>
        </div>
      </div>
    </div>

    <!-- Akselerasi Kartu Grafis / VGA Eksternal RTX / GTX (Always Visible for All Presets) -->
    <div
      :class="[
        'p-4 rounded-2xl border transition-all duration-300 flex items-center justify-between gap-4',
        store.customConfig.useHardwareAcc
          ? 'bg-gradient-to-r from-emerald-950/40 via-slate-800/80 to-slate-800/80 border-emerald-500/50 shadow-[0_0_20px_rgba(16,185,129,0.15)] ring-1 ring-emerald-500/30'
          : 'bg-slate-800/40 border-slate-700/60 hover:border-slate-600'
      ]"
    >
      <div class="flex items-center gap-3.5 min-w-0">
        <div
          :class="[
            'w-11 h-11 rounded-xl flex items-center justify-center shrink-0 border transition-colors',
            store.customConfig.useHardwareAcc
              ? 'bg-emerald-500/20 border-emerald-500/40 text-emerald-400 shadow-md shadow-emerald-500/20'
              : 'bg-slate-700/50 border-slate-600/50 text-slate-400'
          ]"
        >
          <Cpu :size="22" />
        </div>
        <div class="min-w-0">
          <div class="flex items-center gap-2 mb-0.5 flex-wrap">
            <span class="text-xs font-bold text-slate-100">
              Akselerasi Kartu Grafis Eksternal (NVIDIA NVENC RTX / GTX)
            </span>
            <span
              v-if="store.customConfig.useHardwareAcc"
              class="inline-flex items-center gap-1 px-2 py-0.2 rounded-full text-[10px] font-mono font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-500/30"
            >
              <Zap :size="10" />
              NVENC AKTIF (Super Cepat)
            </span>
            <span
              v-else
              class="inline-flex items-center gap-1 px-2 py-0.2 rounded-full text-[10px] font-mono text-slate-400 bg-slate-700/40 border border-slate-600/40"
            >
              Mode CPU (Standar)
            </span>
          </div>
          <p class="text-[11px] text-slate-400 leading-relaxed">
            Gunakan GPU dedicated NVIDIA RTX / GTX untuk mempercepat rendering & kompresi video hingga 10x-20x lebih cepat pada preset apa pun.
          </p>
        </div>
      </div>

      <label class="relative inline-flex items-center cursor-pointer shrink-0">
        <input
          type="checkbox"
          v-model="store.customConfig.useHardwareAcc"
          class="sr-only peer"
        />
        <div class="w-12 h-6 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-600"></div>
      </label>
    </div>

    <!-- Custom Mode Expandable Options -->
    <div
      v-if="store.selectedPreset === 'custom'"
      class="p-5 rounded-2xl bg-slate-800/60 border border-indigo-500/30 space-y-5 animate-in fade-in duration-200"
    >
      <div class="flex items-center justify-between pb-3 border-b border-slate-700/60">
        <h4 class="text-xs font-bold uppercase tracking-wider text-indigo-300 flex items-center gap-1.5">
          <Sliders :size="14" />
          Kustomisasi Parameter Encoding
        </h4>
        <div class="flex items-center gap-2">
          <span v-if="store.pendingFiles.length > 0" class="text-xs text-emerald-400 font-mono font-semibold">
            Estimasi Custom: ~{{ formatBytes(store.presetEstimates['custom'].size) }} (-{{ store.presetEstimates['custom'].savings }}%)
          </span>
          <span v-else class="text-xs text-slate-400 font-mono">Preset: Custom</span>
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- CRF Slider -->
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="text-xs font-semibold text-slate-300">
              Konstan Kualitas (CRF): <span class="font-mono text-indigo-400 font-bold">{{ store.customConfig.crf }}</span>
            </label>
            <span class="text-[11px] text-emerald-400 font-mono font-semibold">
              Bitrate ~{{ customBitrateEstimate }}
            </span>
          </div>

          <input
            type="range"
            min="18"
            max="32"
            step="1"
            v-model.number="store.customConfig.crf"
            class="w-full h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-indigo-500"
          />

          <div class="flex items-center justify-between text-[11px] text-slate-400">
            <div>Profil: <span class="text-slate-300 font-medium">{{ crfQualityLabel }}</span></div>
            <div v-if="store.pendingFiles.length > 0" class="text-emerald-400 font-mono font-medium">
              Antrean: ~{{ formatBytes(store.presetEstimates['custom'].size) }}
            </div>
          </div>
        </div>

        <!-- Resolution Downscale -->
        <div class="space-y-2">
          <label class="text-xs font-semibold text-slate-300">
            Target Resolusi Video:
          </label>
          <div class="grid grid-cols-4 gap-1.5">
            <button
              v-for="res in ['original', '1080p', '720p', '480p']"
              :key="res"
              type="button"
              @click="store.customConfig.resolution = res"
              :class="[
                'py-1.5 px-2 rounded-xl text-xs font-semibold uppercase tracking-wider border transition-all cursor-pointer',
                store.customConfig.resolution === res
                  ? 'bg-indigo-600 text-white border-indigo-500 shadow-md'
                  : 'bg-slate-700/50 text-slate-300 border-slate-600/60 hover:bg-slate-700'
              ]"
            >
              {{ res === 'original' ? 'Asli' : res }}
            </button>
          </div>
          <p class="text-[11px] text-slate-400">
            Resolusi otomatis di-downscale jika ukuran asal lebih besar dari target.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
