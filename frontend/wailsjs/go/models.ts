export namespace backend {
	
	export class CompressionConfig {
	    inputPath: string;
	    outputPath?: string;
	    preset: string;
	    customCodec?: string;
	    crf?: number;
	    resolution?: string;
	    useHardwareAcc: boolean;
	    totalDuration: number;
	
	    static createFrom(source: any = {}) {
	        return new CompressionConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputPath = source["inputPath"];
	        this.outputPath = source["outputPath"];
	        this.preset = source["preset"];
	        this.customCodec = source["customCodec"];
	        this.crf = source["crf"];
	        this.resolution = source["resolution"];
	        this.useHardwareAcc = source["useHardwareAcc"];
	        this.totalDuration = source["totalDuration"];
	    }
	}
	export class CompressionResult {
	    inputPath: string;
	    outputPath: string;
	    originalSize: number;
	    compressedSize: number;
	    savedBytes: number;
	    savingsPercent: number;
	    durationSeconds: number;
	
	    static createFrom(source: any = {}) {
	        return new CompressionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputPath = source["inputPath"];
	        this.outputPath = source["outputPath"];
	        this.originalSize = source["originalSize"];
	        this.compressedSize = source["compressedSize"];
	        this.savedBytes = source["savedBytes"];
	        this.savingsPercent = source["savingsPercent"];
	        this.durationSeconds = source["durationSeconds"];
	    }
	}
	export class FFmpegPaths {
	    ffmpeg: string;
	    ffprobe: string;
	
	    static createFrom(source: any = {}) {
	        return new FFmpegPaths(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ffmpeg = source["ffmpeg"];
	        this.ffprobe = source["ffprobe"];
	    }
	}
	export class VideoMetadata {
	    filePath: string;
	    fileName: string;
	    fileSize: number;
	    duration: number;
	    width: number;
	    height: number;
	    bitrate: number;
	    codec: string;
	    fps: number;
	    audioCodec?: string;
	    audioBitrate?: number;
	
	    static createFrom(source: any = {}) {
	        return new VideoMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.fileName = source["fileName"];
	        this.fileSize = source["fileSize"];
	        this.duration = source["duration"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.bitrate = source["bitrate"];
	        this.codec = source["codec"];
	        this.fps = source["fps"];
	        this.audioCodec = source["audioCodec"];
	        this.audioBitrate = source["audioBitrate"];
	    }
	}

}

