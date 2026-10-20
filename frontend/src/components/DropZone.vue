<script setup>
import { ref } from 'vue'
import { UploadCloud, FolderPlus, Film } from 'lucide-vue-next'
import { useCompressionStore } from '../stores/compression'

const store = useCompressionStore()
const isDragging = ref(false)

function onDragOver(e) {
  e.preventDefault()
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'copy'
  }
  isDragging.value = true
}

function onDragLeave(e) {
  e.preventDefault()
  isDragging.value = false
}

async function onDrop(e) {
  e.preventDefault()
  isDragging.value = false

  // Fallback for environments where HTML5 File API includes the path property
  const files = e.dataTransfer?.files
  if (files && files.length > 0) {
    for (let i = 0; i < files.length; i++) {
      const file = files[i]
      if (file.path) {
        await store.addFileByPath(file.path)
      }
    }
  }
}
</script>

<template>
  <div
    style="--wails-drop-target: drop"
    @dragover="onDragOver"
    @dragleave="onDragLeave"
    @drop="onDrop"
    :class="[
      'relative group rounded-3xl p-8 transition-all duration-300 text-center flex flex-col items-center justify-center cursor-pointer border-2 border-dashed',
      isDragging
        ? 'border-indigo-400 bg-indigo-500/10 shadow-[0_0_35px_rgba(99,102,241,0.25)] scale-[1.01]'
        : 'border-slate-700/80 hover:border-indigo-500/50 bg-slate-800/40 hover:bg-slate-800/60'
    ]"
    @click="store.pickMultipleFilesNative"
  >
    <!-- Background Glow -->
    <div class="absolute inset-0 bg-gradient-to-b from-indigo-500/5 to-transparent rounded-3xl pointer-events-none"></div>

    <div
      :class="[
        'w-16 h-16 rounded-2xl flex items-center justify-center mb-4 transition-transform duration-300 shadow-xl',
        isDragging
          ? 'bg-indigo-500 text-white scale-110 rotate-3'
          : 'bg-gradient-to-tr from-indigo-600/30 to-cyan-500/30 text-indigo-400 group-hover:scale-105 group-hover:text-indigo-300 border border-indigo-500/20'
      ]"
    >
      <UploadCloud :size="32" class="transition-transform group-hover:-translate-y-0.5 duration-200" />
    </div>

    <h3 class="text-lg font-bold text-slate-100 tracking-tight mb-1">
      Tarik & Letakkan Berkas Video Di Sini
    </h3>
    <p class="text-sm text-slate-400 max-w-sm mb-5">
      atau klik untuk memilih berkas dari komputer Anda (mendukung kompresi multi-berkas / batch)
    </p>

    <!-- Action buttons inside dropzone -->
    <div class="flex items-center gap-3" @click.stop>
      <button
        type="button"
        @click="store.pickMultipleFilesNative"
        class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-lg shadow-indigo-600/30 hover:shadow-indigo-500/40 active:scale-95 transition-all cursor-pointer"
      >
        <Film :size="15" />
        Pilih Video
      </button>

      <button
        type="button"
        @click="store.pickMultipleFilesNative"
        class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-slate-700/70 hover:bg-slate-700 text-slate-200 text-xs font-medium border border-slate-600/50 active:scale-95 transition-all cursor-pointer"
      >
        <FolderPlus :size="15" />
        Tambah Batch
      </button>
    </div>

    <!-- Supported formats tags -->
    <div class="mt-6 flex flex-wrap items-center justify-center gap-1.5 text-[11px] font-mono text-slate-400">
      <span class="text-slate-500 font-sans mr-1">Format didukung:</span>
      <span class="px-2 py-0.5 rounded-md bg-slate-800/80 border border-slate-700 text-slate-300">.MP4</span>
      <span class="px-2 py-0.5 rounded-md bg-slate-800/80 border border-slate-700 text-slate-300">.MOV</span>
      <span class="px-2 py-0.5 rounded-md bg-slate-800/80 border border-slate-700 text-slate-300">.MKV</span>
      <span class="px-2 py-0.5 rounded-md bg-slate-800/80 border border-slate-700 text-slate-300">.AVI</span>
      <span class="px-2 py-0.5 rounded-md bg-slate-800/80 border border-slate-700 text-slate-300">.WEBM</span>
    </div>
  </div>
</template>
