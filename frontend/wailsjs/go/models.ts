export namespace main {
	
	export class ActionResult {
	    ok: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	    }
	}
	export class StatusItem {
	    name: string;
	    ok: boolean;
	    detail: string;
	    remediation: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ok = source["ok"];
	        this.detail = source["detail"];
	        this.remediation = source["remediation"];
	    }
	}
	export class SystemStatus {
	    java8: StatusItem;
	    tokenService: StatusItem;
	    tokenReader: StatusItem;
	    certificate: StatusItem;
	    browserPkcs11: StatusItem;
	    packages: StatusItem[];
	    allOk: boolean;
	    checkedAt: string;
	    platformWarning?: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.java8 = this.convertValues(source["java8"], StatusItem);
	        this.tokenService = this.convertValues(source["tokenService"], StatusItem);
	        this.tokenReader = this.convertValues(source["tokenReader"], StatusItem);
	        this.certificate = this.convertValues(source["certificate"], StatusItem);
	        this.browserPkcs11 = this.convertValues(source["browserPkcs11"], StatusItem);
	        this.packages = this.convertValues(source["packages"], StatusItem);
	        this.allOk = source["allOk"];
	        this.checkedAt = source["checkedAt"];
	        this.platformWarning = source["platformWarning"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

