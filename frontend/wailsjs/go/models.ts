export namespace main {
	
	export class CSVDataMetadata {
	    filePath: string;
	    headers: string[];
	    total: number;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CSVDataMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.headers = source["headers"];
	        this.total = source["total"];
	        this.error = source["error"];
	    }
	}
	export class CSVRowsResponse {
	    rows: string[][];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CSVRowsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rows = source["rows"];
	        this.error = source["error"];
	    }
	}
	export class UpdateInfo {
	    hasUpdate: boolean;
	    latestVer: string;
	    currentVer: string;
	    downloadURL: string;
	    desc: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasUpdate = source["hasUpdate"];
	        this.latestVer = source["latestVer"];
	        this.currentVer = source["currentVer"];
	        this.downloadURL = source["downloadURL"];
	        this.desc = source["desc"];
	        this.error = source["error"];
	    }
	}

}

