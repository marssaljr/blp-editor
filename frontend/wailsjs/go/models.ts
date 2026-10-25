export namespace main {
	
	export class FileInfo {
	    version: number;
	    format: number;
	    formatName: string;
	    width: number;
	    height: number;
	    mipmapCount: number;
	    hasAlpha: boolean;
	    fileSize: number;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.format = source["format"];
	        this.formatName = source["formatName"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.mipmapCount = source["mipmapCount"];
	        this.hasAlpha = source["hasAlpha"];
	        this.fileSize = source["fileSize"];
	    }
	}

}

export namespace mipmap {
	
	export class MipmapInfo {
	    Level: number;
	    Width: number;
	    Height: number;
	    Size: number;
	
	    static createFrom(source: any = {}) {
	        return new MipmapInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Level = source["Level"];
	        this.Width = source["Width"];
	        this.Height = source["Height"];
	        this.Size = source["Size"];
	    }
	}

}

