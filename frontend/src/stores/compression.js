import { defineStore } from 'pinia'
import {
  SelectFile,
  SelectMultipleFiles,
  SelectOutputFolder,
  GetVideoMetadata,
  StartCompression,
  CancelCompression,
  OpenFolder,
  OpenFile,
  CheckBinaries
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff, OnFileDrop, OnFileDropOff } from '../../wailsjs/runtime/runtime'

// Model perhitungan matematis estimasi ukuran file encoding FFmpeg secara dinamis
export function calculateEstimatedSize(file, preset = 'youtube', customConfig = null) {
  if (!file || !file.fileSize || file.fileSize <= 0) {
    return { size: 0, savedBytes: 0, savingsPercent: 0, bitrate: 0 }
  }

  const duration = file.metadata?.duration || file.totalDuration || 0
  const origSize = file.fileSize

  // Fallback jika durasi belum terdeteksi dari metadata
  if (duration <= 0) {
    let ratio = 0.40
    if (preset === 'hevc') ratio = 0.25
    else if (preset === 'custom') {
      const crf = customConfig?.crf || 23
      const crfFactor = Math.pow(2, (23 - crf) / 6)
      const isHevc = customConfig?.customCodec?.includes('hevc') || customConfig?.customCodec?.includes('265')
      ratio = 0.40 * (isHevc ? 0.6 : 1.0) * crfFactor
    }
    ratio = Math.max(0.05, Math.min(0.95, ratio))
    const estSize = Math.round(origSize * ratio)
    const saved = Math.max(0, origSize - estSize)
    const pct = parseFloat(((saved / origSize) * 100).toFixed(1))
    return { size: estSize, savedBytes: saved, savingsPercent: pct, bitrate: 0 }
  }

  const origWidth = file.metadata?.width || 1920
  const origHeight = file.metadata?.height || 1080
  const fps = file.metadata?.fps && file.metadata.fps > 0 ? file.metadata.fps : 30

  // 1. Ekstraksi konfigurasi preset / kustomisasi
  let codec = 'h264'
  let crf = 23
  let targetRes = 'original'

  if (preset === 'youtube') {
    codec = 'h264'
    crf = 23
    targetRes = 'original'
  } else if (preset === 'hevc') {
    codec = 'hevc'
    crf = 28
    targetRes = 'original'
  } else if (preset === 'custom') {
    codec = customConfig?.customCodec?.includes('hevc') || customConfig?.customCodec?.includes('265') ? 'hevc' : 'h264'
    crf = customConfig?.crf || 23
    targetRes = customConfig?.resolution || 'original'
  }

  // 2. Hitung dimensi target sesuai pilihan downscale resolusi
  let targetHeight = origHeight
  if (targetRes === '1080p') targetHeight = Math.min(origHeight, 1080)
  else if (targetRes === '720p') targetHeight = Math.min(origHeight, 720)
  else if (targetRes === '480p') targetHeight = Math.min(origHeight, 480)

  const targetWidth = Math.round(origWidth * (targetHeight / origHeight))

  // 3. Model baseline bitrate video (bps)
  // Baseline 1080p 30fps pada CRF 23 H.264 adalah ~3,800,000 bps
  const baselinePixels = 1920 * 1080
  const targetPixels = targetWidth * targetHeight
  const pixelRatio = targetPixels / baselinePixels
  const fpsRatio = Math.max(0.7, Math.min(1.5, fps / 30))

  // Skala resolusi dengan exponent 0.75 (spatial redundancy scaling)
  const resScaleFactor = Math.pow(pixelRatio, 0.75) * fpsRatio
  let baseVideoBitrate = 3800000 * resScaleFactor

  // 4. Efisiensi Codec (HEVC lebih hemat ~40% untuk level visual serupa)
  if (codec === 'hevc') {
    baseVideoBitrate *= 0.60
  }

  // 5. Pengaruh CRF: setiap pergeseran +6 CRF memotong bitrate ~50%
  const crfFactor = Math.pow(2, (23 - crf) / 6)
  const targetVideoBitrate = baseVideoBitrate * crfFactor

  // 6. Audio Bitrate (128 kbps AAC)
  const audioBitrate = 128000

  // 7. Estimasi total ukuran dalam bytes (+1.5% overhead kontainer MP4)
  const totalBitrate = targetVideoBitrate + audioBitrate
  let estimatedBytes = Math.round(((totalBitrate * duration) / 8) * 1.015)

  // Batas minimum masuk akal (header kontainer + audio)
  const minBytes = Math.round(duration * 16000)
  estimatedBytes = Math.max(minBytes, estimatedBytes)

  // Batas maksimum: kompresi CRF tidak akan melebihi 95% ukuran asli
  if (estimatedBytes >= origSize) {
    estimatedBytes = Math.round(origSize * 0.95)
  }

  const savedBytes = Math.max(0, origSize - estimatedBytes)
  const savingsPercent = parseFloat(((savedBytes / origSize) * 100).toFixed(1))

  return {
    size: estimatedBytes,
    savedBytes,
    savingsPercent,
    bitrate: totalBitrate,
  }
}

export const useCompressionStore = defineStore('compression', {
  state: () => ({
    files: [],
    activeFileId: null,
    isProcessing: false,
    selectedPreset: 'youtube', // 'youtube', 'hevc', 'custom'
    customConfig: {
      crf: 23,
      resolution: 'original', // 'original', '1080p', '720p', '480p'
      customCodec: 'h264',
      useHardwareAcc: false,
    },
    customOutputFolder: '',
    removeCompressSuffix: false, // Opsi untuk menghilangkan penambahan teks _compress pada nama file hasil
    binaryInfo: null,
    toast: null,
  }),

  getters: {
    totalFiles: (state) => state.files.length,
    pendingFiles: (state) => state.files.filter(f => f.status === 'idle'),
    completedFiles: (state) => state.files.filter(f => f.status === 'completed'),
    activeFile: (state) => state.files.find(f => f.id === state.activeFileId),
    overallProgress: (state) => {
      if (state.files.length === 0) return 0
      const sum = state.files.reduce((acc, f) => {
        if (f.status === 'completed') return acc + 100
        if (f.status === 'processing') return acc + (f.progress || 0)
        return acc
      }, 0)
      return Math.round(sum / state.files.length)
    },
    totalOriginalSize: (state) => state.files.reduce((acc, f) => acc + (f.fileSize || 0), 0),
    totalSavedBytes: (state) => state.files.reduce((acc, f) => {
      return acc + (f.result ? f.result.savedBytes : 0)
    }, 0),
    overallSavingsPercent: (state) => {
      const orig = state.files.reduce((acc, f) => f.result ? acc + f.result.originalSize : acc, 0)
      const saved = state.files.reduce((acc, f) => f.result ? acc + f.result.savedBytes : acc, 0)
      if (orig <= 0) return 0
      return Math.max(0, ((saved / orig) * 100).toFixed(1))
    },
    // Mengambil direktori folder dari file path
    getFileDirectory: () => (filePath) => {
      if (!filePath) return ''
      const lastSlash = Math.max(filePath.lastIndexOf('/'), filePath.lastIndexOf('\\'))
      return lastSlash >= 0 ? filePath.substring(0, lastSlash) : ''
    },
    // Mengecek apakah folder tujuan penyimpanan sama dengan folder file asli
    isDestinationSameAsOriginal: (state) => (file) => {
      if (!file || !file.filePath) return true
      const lastSlash = Math.max(file.filePath.lastIndexOf('/'), file.filePath.lastIndexOf('\\'))
      const fileDir = (lastSlash >= 0 ? file.filePath.substring(0, lastSlash) : '')
        .replace(/[/\\]+/g, '/')
        .replace(/\/+$/, '')
        .toLowerCase()

      // Jika folder custom belum dipilih, maka tujuan adalah folder yang sama dengan file asli
      if (!state.customOutputFolder) {
        return true
      }

      const customDir = state.customOutputFolder
        .replace(/[/\\]+/g, '/')
        .replace(/\/+$/, '')
        .toLowerCase()

      return fileDir === customDir
    },
    // Mengetahui apakah ada risiko penimpaan file (replace) pada antrean
    hasOverwriteRisk: (state) => {
      if (!state.removeCompressSuffix) return false
      return state.files.filter(f => f.status === 'idle').some(file => {
        const lastSlash = Math.max(file.filePath.lastIndexOf('/'), file.filePath.lastIndexOf('\\'))
        const fileDir = (lastSlash >= 0 ? file.filePath.substring(0, lastSlash) : '')
          .replace(/[/\\]+/g, '/')
          .replace(/\/+$/, '')
          .toLowerCase()

        if (!state.customOutputFolder) {
          return true
        }

        const customDir = state.customOutputFolder
          .replace(/[/\\]+/g, '/')
          .replace(/\/+$/, '')
          .toLowerCase()

        return fileDir === customDir
      })
    },
    // Daftar file yang berpotensi tertimpa
    overwriteRiskFiles: (state) => {
      if (!state.removeCompressSuffix) return []
      return state.files.filter(f => f.status === 'idle').filter(file => {
        const lastSlash = Math.max(file.filePath.lastIndexOf('/'), file.filePath.lastIndexOf('\\'))
        const fileDir = (lastSlash >= 0 ? file.filePath.substring(0, lastSlash) : '')
          .replace(/[/\\]+/g, '/')
          .replace(/\/+$/, '')
          .toLowerCase()

        if (!state.customOutputFolder) {
          return true
        }

        const customDir = state.customOutputFolder
          .replace(/[/\\]+/g, '/')
          .replace(/\/+$/, '')
          .toLowerCase()

        return fileDir === customDir
      })
    },
    // Menghitung path tujuan lengkap untuk berkas
    computeOutputFilePath: (state) => (file) => {
      if (!file || !file.filePath) return ''
      const lastSlash = Math.max(file.filePath.lastIndexOf('/'), file.filePath.lastIndexOf('\\'))
      const fileDir = lastSlash >= 0 ? file.filePath.substring(0, lastSlash) : ''
      const targetDir = state.customOutputFolder || fileDir
      const normalizedTarget = targetDir.replace(/\//g, '\\').replace(/\\+$/, '')
      const baseName = file.fileName.replace(/\.[^/.]+$/, '')
      const suffix = state.removeCompressSuffix ? '' : '_compressed'
      return `${normalizedTarget}\\${baseName}${suffix}.mp4`
    },

    // Estimasi ukuran dinamis untuk satu file sesuai preset dan konfigurasi saat ini
    estimateForFile: (state) => (file, presetOverride = null, configOverride = null) => {
      const preset = presetOverride || state.selectedPreset
      const config = configOverride || state.customConfig
      return calculateEstimatedSize(file, preset, config)
    },

    // Total perkiraan ukuran akhir (bytes) seluruh berkas yang masih dalam antrean
    totalEstimatedSize: (state) => {
      const pending = state.files.filter(f => f.status === 'idle')
      return pending.reduce((acc, f) => {
        const est = calculateEstimatedSize(f, state.selectedPreset, state.customConfig)
        return acc + est.size
      }, 0)
    },

    // Total perkiraan penghematan (bytes) seluruh berkas yang masih dalam antrean
    totalEstimatedSavedBytes: (state) => {
      const pending = state.files.filter(f => f.status === 'idle')
      const orig = pending.reduce((acc, f) => acc + (f.fileSize || 0), 0)
      const est = pending.reduce((acc, f) => {
        return acc + calculateEstimatedSize(f, state.selectedPreset, state.customConfig).size
      }, 0)
      return Math.max(0, orig - est)
    },

    // Persentase perkiraan penghematan total antrean
    totalEstimatedSavingsPercent: (state) => {
      const pending = state.files.filter(f => f.status === 'idle')
      const orig = pending.reduce((acc, f) => acc + (f.fileSize || 0), 0)
      if (orig <= 0) return 0
      const est = pending.reduce((acc, f) => {
        return acc + calculateEstimatedSize(f, state.selectedPreset, state.customConfig).size
      }, 0)
      const saved = Math.max(0, orig - est)
      return Math.max(0, parseFloat(((saved / orig) * 100).toFixed(1)))
    },

    // Estimasi total ukuran untuk masing-masing preset (YouTube, HEVC, Custom)
    presetEstimates: (state) => {
      const pending = state.files.filter(f => f.status === 'idle')
      const orig = pending.reduce((acc, f) => acc + (f.fileSize || 0), 0)

      const calculateForPreset = (presetId) => {
        if (pending.length === 0 || orig <= 0) return { size: 0, savings: 0, savedBytes: 0 }
        const estTotal = pending.reduce((acc, f) => {
          return acc + calculateEstimatedSize(f, presetId, state.customConfig).size
        }, 0)
        const saved = Math.max(0, orig - estTotal)
        const pct = Math.max(0, parseFloat(((saved / orig) * 100).toFixed(1)))
        return { size: estTotal, savings: pct, savedBytes: saved }
      }

      return {
        youtube: calculateForPreset('youtube'),
        hevc: calculateForPreset('hevc'),
        custom: calculateForPreset('custom'),
      }
    }
  },

  actions: {
    async init() {
      try {
        const bin = await CheckBinaries()
        this.binaryInfo = bin
      } catch (err) {
        this.showToast('Gagal memuat binary FFmpeg: ' + err, 'error')
      }

      // Listen to compression progress emitted from Go backend
      EventsOff('compression:progress')
      EventsOn('compression:progress', (data) => {
        if (!data || !this.activeFileId) return
        const file = this.files.find(f => f.id === this.activeFileId)
        if (!file) return

        if (data.status === 'running') {
          file.status = 'processing'
          file.progress = Math.min(99.9, Math.max(0, data.progress))
          file.currentTime = data.currentTime
          file.totalDuration = data.totalDuration
          file.fps = data.fps || 0
          file.speed = data.speed || ''
          
          // Calculate ETA
          if (data.speed && data.totalDuration && data.currentTime) {
            const speedMultiplier = parseFloat(data.speed.replace('x', ''))
            if (speedMultiplier > 0) {
              const remainingVideoSeconds = data.totalDuration - data.currentTime
              file.etaSeconds = Math.max(0, Math.round(remainingVideoSeconds / speedMultiplier))
            }
          }
        } else if (data.status === 'completed') {
          file.progress = 100
        } else if (data.status === 'cancelled') {
          file.status = 'cancelled'
        } else if (data.status === 'error') {
          file.status = 'error'
          file.error = data.message || 'Kompresi gagal'
        }
      })

      // Listen to Wails runtime file drop (both on drop targets and globally in window)
      try {
        OnFileDropOff()
      } catch (e) {}

      try {
        OnFileDrop(async (x, y, paths) => {
          if (Array.isArray(paths) && paths.length > 0) {
            for (const p of paths) {
              await this.addFileByPath(p)
            }
          }
        }, false)
      } catch (e) {
        console.warn('Wails OnFileDrop error:', e)
      }

      // Listen to native file drop events from Wails Go backend
      EventsOff('file:dropped')
      EventsOn('file:dropped', async (paths) => {
        if (Array.isArray(paths) && paths.length > 0) {
          for (const p of paths) {
            await this.addFileByPath(p)
          }
        }
      })
    },

    showToast(message, type = 'info') {
      this.toast = { message, type }
      setTimeout(() => {
        if (this.toast && this.toast.message === message) {
          this.toast = null
        }
      }, 4000)
    },

    async addFileByPath(filePath) {
      if (!filePath) return
      const cleanPath = filePath.trim()

      // Validate that filePath is an absolute path
      if (!cleanPath.includes('/') && !cleanPath.includes('\\')) {
        this.showToast(`Path tidak valid: '${cleanPath}'. Silakan pilih berkas via tombol Pilih Video`, 'error')
        return
      }

      // Prevent duplicates (case-insensitive on Windows)
      if (this.files.some(f => f.filePath.toLowerCase() === cleanPath.toLowerCase())) {
        this.showToast('Berkas sudah ada di daftar antrean', 'info')
        return
      }

      const id = 'vid_' + Date.now() + '_' + Math.random().toString(36).substring(2, 7)
      const baseName = cleanPath.split(/[/\\]/).pop()

      const item = {
        id,
        filePath: cleanPath,
        fileName: baseName,
        fileSize: 0,
        metadata: null,
        status: 'idle', // 'idle' | 'processing' | 'completed' | 'cancelled' | 'error'
        progress: 0,
        currentTime: 0,
        totalDuration: 0,
        speed: '',
        fps: 0,
        etaSeconds: 0,
        result: null,
        error: null,
        loadingMeta: true,
      }

      this.files.push(item)

      // Fetch metadata immediately and update reactive object in this.files
      try {
        const meta = await GetVideoMetadata(cleanPath)
        const target = this.files.find(f => f.id === id)
        if (target) {
          target.metadata = meta
          target.fileSize = meta?.fileSize || 0
          target.totalDuration = meta?.duration || 0
          target.loadingMeta = false
        }
      } catch (err) {
        const target = this.files.find(f => f.id === id)
        if (target) {
          target.loadingMeta = false
          target.error = 'Gagal membaca metadata: ' + err
          target.status = 'error'
        }
        this.showToast('Gagal membaca info video: ' + baseName, 'error')
      }
    },

    async pickFileNative() {
      try {
        const path = await SelectFile()
        if (path) {
          await this.addFileByPath(path)
        }
      } catch (err) {
        this.showToast('Error memilih berkas: ' + err, 'error')
      }
    },

    async pickMultipleFilesNative() {
      try {
        const paths = await SelectMultipleFiles()
        if (paths && paths.length > 0) {
          for (const path of paths) {
            await this.addFileByPath(path)
          }
        }
      } catch (err) {
        this.showToast('Error memilih berkas: ' + err, 'error')
      }
    },

    async pickOutputFolderNative() {
      try {
        const folder = await SelectOutputFolder()
        if (folder) {
          this.customOutputFolder = folder
        }
      } catch (err) {
        this.showToast('Error memilih folder: ' + err, 'error')
      }
    },

    clearOutputFolder() {
      this.customOutputFolder = ''
    },

    removeFile(id) {
      if (this.activeFileId === id && this.isProcessing) {
        this.cancelActiveCompression()
      }
      this.files = this.files.filter(f => f.id !== id)
    },

    clearAll() {
      if (this.isProcessing) {
        this.cancelActiveCompression()
      }
      this.files = []
      this.activeFileId = null
    },

    clearCompleted() {
      this.files = this.files.filter(f => f.status !== 'completed')
    },

    async startAll() {
      if (this.isProcessing) return
      this.isProcessing = true

      for (const file of this.files) {
        if (file.status === 'completed') continue
        if (!this.isProcessing) break // In case cancelled

        this.activeFileId = file.id
        file.status = 'processing'
        file.progress = 0
        file.error = null

        // Generate output path based on destination folder and removeCompressSuffix setting
        const outputPath = this.computeOutputFilePath(file)

        const config = {
          inputPath: file.filePath,
          outputPath: outputPath,
          preset: this.selectedPreset,
          customCodec: this.customConfig.customCodec,
          crf: parseInt(this.customConfig.crf) || 23,
          resolution: this.customConfig.resolution,
          useHardwareAcc: Boolean(this.customConfig.useHardwareAcc),
          totalDuration: file.metadata?.duration || file.totalDuration || 0,
        }

        try {
          const res = await StartCompression(config)
          file.status = 'completed'
          file.progress = 100
          file.result = res
        } catch (err) {
          if (file.status !== 'cancelled') {
            file.status = 'error'
            file.error = String(err)
            this.showToast(`Gagal mengompresi ${file.fileName}: ${err}`, 'error')
          }
        }
      }

      this.isProcessing = false
      this.activeFileId = null
    },

    async cancelActiveCompression() {
      try {
        await CancelCompression()
      } catch (err) {
        console.error('Cancel error', err)
      }
      if (this.activeFileId) {
        const file = this.files.find(f => f.id === this.activeFileId)
        if (file) {
          file.status = 'cancelled'
        }
      }
      this.isProcessing = false
      this.activeFileId = null
      this.showToast('Kompresi dibatalkan oleh pengguna', 'info')
    },

    async openFolder(targetPath) {
      try {
        await OpenFolder(targetPath)
      } catch (err) {
        this.showToast('Gagal membuka folder: ' + err, 'error')
      }
    },

    async openFile(targetPath) {
      try {
        await OpenFile(targetPath)
      } catch (err) {
        this.showToast('Gagal memutar video: ' + err, 'error')
      }
    }
  }
})
