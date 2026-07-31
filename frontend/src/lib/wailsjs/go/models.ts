export namespace diagnostics {
	
	export class Recommendation {
	    id: string;
	    severity: string;
	    title: string;
	    description: string;
	    action_label?: string;
	
	    static createFrom(source: any = {}) {
	        return new Recommendation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.severity = source["severity"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.action_label = source["action_label"];
	    }
	}
	export class FactorScore {
	    name: string;
	    score: number;
	    status: string;
	    details: string;
	
	    static createFrom(source: any = {}) {
	        return new FactorScore(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.score = source["score"];
	        this.status = source["status"];
	        this.details = source["details"];
	    }
	}
	export class DiagnosticReport {
	    overall_score: number;
	    status_text: string;
	    factors: FactorScore[];
	    recommendations: Recommendation[];
	
	    static createFrom(source: any = {}) {
	        return new DiagnosticReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.overall_score = source["overall_score"];
	        this.status_text = source["status_text"];
	        this.factors = this.convertValues(source["factors"], FactorScore);
	        this.recommendations = this.convertValues(source["recommendations"], Recommendation);
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
	
	export class MemoryVelocitySample {
	    timestamp: string;
	    used_mb: number;
	    velocity_mbps: number;
	    status_text: string;
	    estimated_oom_sec: number;
	
	    static createFrom(source: any = {}) {
	        return new MemoryVelocitySample(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.used_mb = source["used_mb"];
	        this.velocity_mbps = source["velocity_mbps"];
	        this.status_text = source["status_text"];
	        this.estimated_oom_sec = source["estimated_oom_sec"];
	    }
	}

}

export namespace functions {
	
	export class CPUStats {
	    usage: number;
	    timestamp: number;
	    frequency_ghz: number;
	    is_throttling: boolean;
	    sustained_load: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CPUStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.usage = source["usage"];
	        this.timestamp = source["timestamp"];
	        this.frequency_ghz = source["frequency_ghz"];
	        this.is_throttling = source["is_throttling"];
	        this.sustained_load = source["sustained_load"];
	    }
	}
	export class PermissionStatus {
	    IsElevated: boolean;
	    RequiresElevation: boolean;
	    UnaccessiblePaths: string[];
	    ElevationCommand: string;
	    CanCleanUserFiles: boolean;
	    CanCleanSystemFiles: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PermissionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.IsElevated = source["IsElevated"];
	        this.RequiresElevation = source["RequiresElevation"];
	        this.UnaccessiblePaths = source["UnaccessiblePaths"];
	        this.ElevationCommand = source["ElevationCommand"];
	        this.CanCleanUserFiles = source["CanCleanUserFiles"];
	        this.CanCleanSystemFiles = source["CanCleanSystemFiles"];
	    }
	}
	export class CleanerResult {
	    Files: Record<string, Array<FileInfo>>;
	    TotalSize: number;
	    Permissions: PermissionStatus;
	
	    static createFrom(source: any = {}) {
	        return new CleanerResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Files = this.convertValues(source["Files"], Array<FileInfo>, true);
	        this.TotalSize = source["TotalSize"];
	        this.Permissions = this.convertValues(source["Permissions"], PermissionStatus);
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
	export class FileInfo {
	    Path: string;
	    Size: number;
	    Name: string;
	    Location: string;
	    NeedsElevation: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Path = source["Path"];
	        this.Size = source["Size"];
	        this.Name = source["Name"];
	        this.Location = source["Location"];
	        this.NeedsElevation = source["NeedsElevation"];
	    }
	}
	export class MemoryStats {
	    totalRAMMB: number;
	    freeRAMMB: number;
	    usedRAMMB: number;
	    usagePercent: number;
	    timestamp: number;
	    swapTotalMB: number;
	    swapUsedMB: number;
	    pressureLevel: string;
	
	    static createFrom(source: any = {}) {
	        return new MemoryStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalRAMMB = source["totalRAMMB"];
	        this.freeRAMMB = source["freeRAMMB"];
	        this.usedRAMMB = source["usedRAMMB"];
	        this.usagePercent = source["usagePercent"];
	        this.timestamp = source["timestamp"];
	        this.swapTotalMB = source["swapTotalMB"];
	        this.swapUsedMB = source["swapUsedMB"];
	        this.pressureLevel = source["pressureLevel"];
	    }
	}

}

export namespace hardware {
	
	export class DriveHealthInfo {
	    device: string;
	    mount_point: string;
	    fs_type: string;
	    total_gb: number;
	    free_gb: number;
	    used_gb: number;
	    usage_percent: number;
	    health_status: string;
	    estimated_life_pct: number;
	    estimated_life_years: number;
	
	    static createFrom(source: any = {}) {
	        return new DriveHealthInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device = source["device"];
	        this.mount_point = source["mount_point"];
	        this.fs_type = source["fs_type"];
	        this.total_gb = source["total_gb"];
	        this.free_gb = source["free_gb"];
	        this.used_gb = source["used_gb"];
	        this.usage_percent = source["usage_percent"];
	        this.health_status = source["health_status"];
	        this.estimated_life_pct = source["estimated_life_pct"];
	        this.estimated_life_years = source["estimated_life_years"];
	    }
	}
	export class StressResult {
	    duration_sec: number;
	    ops_completed: number;
	    thermal_score: number;
	    baseline_load_pct: number;
	    peak_load_pct: number;
	    status_text: string;
	
	    static createFrom(source: any = {}) {
	        return new StressResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.duration_sec = source["duration_sec"];
	        this.ops_completed = source["ops_completed"];
	        this.thermal_score = source["thermal_score"];
	        this.baseline_load_pct = source["baseline_load_pct"];
	        this.peak_load_pct = source["peak_load_pct"];
	        this.status_text = source["status_text"];
	    }
	}

}

export namespace network {
	
	export class SocketItem {
	    pid: number;
	    process_name: string;
	    protocol: string;
	    local_port: number;
	    local_ip: string;
	    remote_ip: string;
	    remote_port: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new SocketItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.process_name = source["process_name"];
	        this.protocol = source["protocol"];
	        this.local_port = source["local_port"];
	        this.local_ip = source["local_ip"];
	        this.remote_ip = source["remote_ip"];
	        this.remote_port = source["remote_port"];
	        this.status = source["status"];
	    }
	}

}

export namespace process {
	
	export class ProcessItem {
	    pid: number;
	    name: string;
	    cpu_percent: number;
	    memory_mb: number;
	    memory_percent: number;
	    energy_impact: number;
	    energy_impact_label: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.name = source["name"];
	        this.cpu_percent = source["cpu_percent"];
	        this.memory_mb = source["memory_mb"];
	        this.memory_percent = source["memory_percent"];
	        this.energy_impact = source["energy_impact"];
	        this.energy_impact_label = source["energy_impact_label"];
	    }
	}

}

export namespace system {
	
	export class EnvVariable {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new EnvVariable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class PathEntry {
	    path: string;
	    is_valid: boolean;
	    is_duplicate: boolean;
	    status_text: string;
	
	    static createFrom(source: any = {}) {
	        return new PathEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.is_valid = source["is_valid"];
	        this.is_duplicate = source["is_duplicate"];
	        this.status_text = source["status_text"];
	    }
	}
	export class EnvPathAnalysis {
	    total_entries: number;
	    valid_entries: number;
	    invalid_entries: number;
	    duplicates_count: number;
	    path_entries: PathEntry[];
	    key_variables: EnvVariable[];
	
	    static createFrom(source: any = {}) {
	        return new EnvPathAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_entries = source["total_entries"];
	        this.valid_entries = source["valid_entries"];
	        this.invalid_entries = source["invalid_entries"];
	        this.duplicates_count = source["duplicates_count"];
	        this.path_entries = this.convertValues(source["path_entries"], PathEntry);
	        this.key_variables = this.convertValues(source["key_variables"], EnvVariable);
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
	
	
	export class SoftwareItem {
	    name: string;
	    current_version: string;
	    latest_version: string;
	    is_outdated: boolean;
	    release_date: string;
	    official_download: string;
	
	    static createFrom(source: any = {}) {
	        return new SoftwareItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.current_version = source["current_version"];
	        this.latest_version = source["latest_version"];
	        this.is_outdated = source["is_outdated"];
	        this.release_date = source["release_date"];
	        this.official_download = source["official_download"];
	    }
	}
	export class StartupItem {
	    name: string;
	    command: string;
	    location: string;
	    impact: string;
	    impact_score: number;
	    is_enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StartupItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.command = source["command"];
	        this.location = source["location"];
	        this.impact = source["impact"];
	        this.impact_score = source["impact_score"];
	        this.is_enabled = source["is_enabled"];
	    }
	}

}

