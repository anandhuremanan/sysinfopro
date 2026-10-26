export namespace alerts {
	
	export class Settings {
	    enabled: boolean;
	    min_free_gb: number;
	    drop_gb: number;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.min_free_gb = source["min_free_gb"];
	        this.drop_gb = source["drop_gb"];
	    }
	}

}

export namespace cleaner {
	
	export class Category {
	    name: string;
	    size_bytes: number;
	    items: number;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.size_bytes = source["size_bytes"];
	        this.items = source["items"];
	    }
	}
	export class ScanResult {
	    categories: Category[];
	    total_bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.categories = this.convertValues(source["categories"], Category);
	        this.total_bytes = source["total_bytes"];
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
	export class Summary {
	    freed_bytes: number;
	    files_deleted: number;
	    skipped: number;
	    stopped: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.freed_bytes = source["freed_bytes"];
	        this.files_deleted = source["files_deleted"];
	        this.skipped = source["skipped"];
	        this.stopped = source["stopped"];
	    }
	}

}

export namespace devcache {
	
	export class DeleteResult {
	    freed_bytes: number;
	    deleted: string[];
	    failed: string[];
	
	    static createFrom(source: any = {}) {
	        return new DeleteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.freed_bytes = source["freed_bytes"];
	        this.deleted = source["deleted"];
	        this.failed = source["failed"];
	    }
	}
	export class Item {
	    id: string;
	    name: string;
	    kind: string;
	    path: string;
	    size_bytes: number;
	    files: number;
	    note: string;
	    deletable: boolean;
	    label: string;
	    folder: string;
	    last_used: string;
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.path = source["path"];
	        this.size_bytes = source["size_bytes"];
	        this.files = source["files"];
	        this.note = source["note"];
	        this.deletable = source["deletable"];
	        this.label = source["label"];
	        this.folder = source["folder"];
	        this.last_used = source["last_used"];
	    }
	}

}

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

export namespace dupes {
	
	export class File {
	    path: string;
	    modified: number;
	
	    static createFrom(source: any = {}) {
	        return new File(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.modified = source["modified"];
	    }
	}
	export class Group {
	    id: string;
	    size: number;
	    files: File[];
	    wasted: number;
	    risky: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Group(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.size = source["size"];
	        this.files = this.convertValues(source["files"], File);
	        this.wasted = source["wasted"];
	        this.risky = source["risky"];
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
	export class Place {
	    label: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new Place(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.path = source["path"];
	    }
	}
	export class Result {
	    root: string;
	    groups: Group[];
	    files: number;
	    wasted: number;
	    cloud_skipped: number;
	    unreadable: number;
	    truncated: boolean;
	    duration_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.groups = this.convertValues(source["groups"], Group);
	        this.files = source["files"];
	        this.wasted = source["wasted"];
	        this.cloud_skipped = source["cloud_skipped"];
	        this.unreadable = source["unreadable"];
	        this.truncated = source["truncated"];
	        this.duration_ms = source["duration_ms"];
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
	
	export class BIOSSpec {
	    vendor: string;
	    version: string;
	    date: string;
	    serial: string;
	
	    static createFrom(source: any = {}) {
	        return new BIOSSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vendor = source["vendor"];
	        this.version = source["version"];
	        this.date = source["date"];
	        this.serial = source["serial"];
	    }
	}
	export class BoardSpec {
	    manufacturer: string;
	    product: string;
	
	    static createFrom(source: any = {}) {
	        return new BoardSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manufacturer = source["manufacturer"];
	        this.product = source["product"];
	    }
	}
	export class CPUSpec {
	    name: string;
	    cores: number;
	    threads: number;
	    max_mhz: number;
	    socket: string;
	
	    static createFrom(source: any = {}) {
	        return new CPUSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.cores = source["cores"];
	        this.threads = source["threads"];
	        this.max_mhz = source["max_mhz"];
	        this.socket = source["socket"];
	    }
	}
	export class DiskSpec {
	    model: string;
	    size_bytes: number;
	    interface: string;
	    media: string;
	
	    static createFrom(source: any = {}) {
	        return new DiskSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.size_bytes = source["size_bytes"];
	        this.interface = source["interface"];
	        this.media = source["media"];
	    }
	}
	export class DriveHealth {
	    drive: string;
	    model: string;
	    type: string;
	    status: string;
	    reason: string;
	    nvme?: storage.NVMeHealth;
	    life_left_pct: number;
	
	    static createFrom(source: any = {}) {
	        return new DriveHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.drive = source["drive"];
	        this.model = source["model"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.reason = source["reason"];
	        this.nvme = this.convertValues(source["nvme"], storage.NVMeHealth);
	        this.life_left_pct = source["life_left_pct"];
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
	export class GPUSpec {
	    name: string;
	    driver: string;
	    vram_bytes: number;
	    width: number;
	    height: number;
	    refresh_hz: number;
	
	    static createFrom(source: any = {}) {
	        return new GPUSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.driver = source["driver"];
	        this.vram_bytes = source["vram_bytes"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.refresh_hz = source["refresh_hz"];
	    }
	}
	export class MemoryModuleSpec {
	    slot: string;
	    size_bytes: number;
	    speed_mhz: number;
	    manufacturer: string;
	    part: string;
	
	    static createFrom(source: any = {}) {
	        return new MemoryModuleSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.size_bytes = source["size_bytes"];
	        this.speed_mhz = source["speed_mhz"];
	        this.manufacturer = source["manufacturer"];
	        this.part = source["part"];
	    }
	}
	export class OSSpec {
	    name: string;
	    version: string;
	    build: string;
	    arch: string;
	    installed: string;
	    last_boot: string;
	
	    static createFrom(source: any = {}) {
	        return new OSSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.build = source["build"];
	        this.arch = source["arch"];
	        this.installed = source["installed"];
	        this.last_boot = source["last_boot"];
	    }
	}
	export class SystemSpec {
	    manufacturer: string;
	    model: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manufacturer = source["manufacturer"];
	        this.model = source["model"];
	        this.type = source["type"];
	    }
	}
	export class HardwareSpecs {
	    system: SystemSpec;
	    bios: BIOSSpec;
	    board: BoardSpec;
	    os: OSSpec;
	    cpus: CPUSpec[];
	    gpus: GPUSpec[];
	    memory: MemoryModuleSpec[];
	    memory_slots: number;
	    disks: DiskSpec[];
	
	    static createFrom(source: any = {}) {
	        return new HardwareSpecs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.system = this.convertValues(source["system"], SystemSpec);
	        this.bios = this.convertValues(source["bios"], BIOSSpec);
	        this.board = this.convertValues(source["board"], BoardSpec);
	        this.os = this.convertValues(source["os"], OSSpec);
	        this.cpus = this.convertValues(source["cpus"], CPUSpec);
	        this.gpus = this.convertValues(source["gpus"], GPUSpec);
	        this.memory = this.convertValues(source["memory"], MemoryModuleSpec);
	        this.memory_slots = source["memory_slots"];
	        this.disks = this.convertValues(source["disks"], DiskSpec);
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

export namespace history {
	
	export class CategoryDiff {
	    id: string;
	    title: string;
	    added: string[];
	    removed: string[];
	    changed: string[];
	
	    static createFrom(source: any = {}) {
	        return new CategoryDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.added = source["added"];
	        this.removed = source["removed"];
	        this.changed = source["changed"];
	    }
	}
	export class Changes {
	    has_baseline: boolean;
	    since: string;
	    total: number;
	    categories: CategoryDiff[];
	
	    static createFrom(source: any = {}) {
	        return new Changes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.has_baseline = source["has_baseline"];
	        this.since = source["since"];
	        this.total = source["total"];
	        this.categories = this.convertValues(source["categories"], CategoryDiff);
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

export namespace hotkey {
	
	export class Settings {
	    enabled: boolean;
	    combo: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.combo = source["combo"];
	        this.error = source["error"];
	    }
	}

}

export namespace locks {
	
	export class LockingProcess {
	    pid: number;
	    name: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new LockingProcess(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	    }
	}
	export class Result {
	    path: string;
	    is_dir: boolean;
	    files_checked: number;
	    truncated: boolean;
	    processes: LockingProcess[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.is_dir = source["is_dir"];
	        this.files_checked = source["files_checked"];
	        this.truncated = source["truncated"];
	        this.processes = this.convertValues(source["processes"], LockingProcess);
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

export namespace main {
	
	export class DuplicateRemoval {
	    removed: number;
	    freed: number;
	    failed: string[];
	
	    static createFrom(source: any = {}) {
	        return new DuplicateRemoval(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.removed = source["removed"];
	        this.freed = source["freed"];
	        this.failed = source["failed"];
	    }
	}
	export class WatchFolder {
	    label: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new WatchFolder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.path = source["path"];
	    }
	}

}

export namespace netdoctor {
	
	export class Adapter {
	    name: string;
	    kind: string;
	    ipv4: string;
	    gateway: string;
	    dns: string[];
	
	    static createFrom(source: any = {}) {
	        return new Adapter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.ipv4 = source["ipv4"];
	        this.gateway = source["gateway"];
	        this.dns = source["dns"];
	    }
	}
	export class HostsEntry {
	    ip: string;
	    names: string[];
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new HostsEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.names = source["names"];
	        this.kind = source["kind"];
	    }
	}
	export class Proxy {
	    enabled: boolean;
	    server: string;
	    auto_config: string;
	    bypass: string;
	    winhttp: string;
	
	    static createFrom(source: any = {}) {
	        return new Proxy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.server = source["server"];
	        this.auto_config = source["auto_config"];
	        this.bypass = source["bypass"];
	        this.winhttp = source["winhttp"];
	    }
	}
	export class Step {
	    id: string;
	    name: string;
	    status: string;
	    detail: string;
	    fix: string;
	    ms: number;
	
	    static createFrom(source: any = {}) {
	        return new Step(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.detail = source["detail"];
	        this.fix = source["fix"];
	        this.ms = source["ms"];
	    }
	}
	export class Report {
	    steps: Step[];
	    adapter?: Adapter;
	    proxy: Proxy;
	    hosts: HostsEntry[];
	    summary: string;
	    healthy: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steps = this.convertValues(source["steps"], Step);
	        this.adapter = this.convertValues(source["adapter"], Adapter);
	        this.proxy = this.convertValues(source["proxy"], Proxy);
	        this.hosts = this.convertValues(source["hosts"], HostsEntry);
	        this.summary = source["summary"];
	        this.healthy = source["healthy"];
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
	    command_line: string;
	    working_dir: string;
	    project_name: string;
	    project_root: string;
	    git_branch: string;
	    started_at: number;
	
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
	        this.command_line = source["command_line"];
	        this.working_dir = source["working_dir"];
	        this.project_name = source["project_name"];
	        this.project_root = source["project_root"];
	        this.git_branch = source["git_branch"];
	        this.started_at = source["started_at"];
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

export namespace reminder {
	
	export class Settings {
	    enabled: boolean;
	    projects_root: string;
	    last_check: number;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.projects_root = source["projects_root"];
	        this.last_check = source["last_check"];
	    }
	}

}

export namespace security {
	
	export class Check {
	    id: string;
	    title: string;
	    status: string;
	    detail: string;
	    advice: string;
	
	    static createFrom(source: any = {}) {
	        return new Check(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.status = source["status"];
	        this.detail = source["detail"];
	        this.advice = source["advice"];
	    }
	}
	export class Report {
	    checks: Check[];
	    passed: number;
	    warned: number;
	    failed: number;
	    unknown: number;
	
	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.checks = this.convertValues(source["checks"], Check);
	        this.passed = source["passed"];
	        this.warned = source["warned"];
	        this.failed = source["failed"];
	        this.unknown = source["unknown"];
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

export namespace storage {
	
	export class DriveInfo {
	    path: string;
	    fs_type: string;
	    total_bytes: number;
	    free_bytes: number;
	    used_bytes: number;
	    used_pct: number;
	
	    static createFrom(source: any = {}) {
	        return new DriveInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.fs_type = source["fs_type"];
	        this.total_bytes = source["total_bytes"];
	        this.free_bytes = source["free_bytes"];
	        this.used_bytes = source["used_bytes"];
	        this.used_pct = source["used_pct"];
	    }
	}
	export class UsageEntry {
	    name: string;
	    path: string;
	    is_dir: boolean;
	    size_bytes: number;
	    files: number;
	
	    static createFrom(source: any = {}) {
	        return new UsageEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.is_dir = source["is_dir"];
	        this.size_bytes = source["size_bytes"];
	        this.files = source["files"];
	    }
	}
	export class FolderUsage {
	    path: string;
	    total_bytes: number;
	    files: number;
	    skipped: number;
	    entries: UsageEntry[];
	    largest_files: UsageEntry[];
	    duration_ms: number;
	    cancelled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FolderUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.total_bytes = source["total_bytes"];
	        this.files = source["files"];
	        this.skipped = source["skipped"];
	        this.entries = this.convertValues(source["entries"], UsageEntry);
	        this.largest_files = this.convertValues(source["largest_files"], UsageEntry);
	        this.duration_ms = source["duration_ms"];
	        this.cancelled = source["cancelled"];
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
	export class NVMeHealth {
	    percent_used: number;
	    available_spare: number;
	    spare_threshold: number;
	    temperature_c: number;
	    power_on_hours: number;
	    media_errors: number;
	    critical_warning: number;
	    tb_written: number;
	
	    static createFrom(source: any = {}) {
	        return new NVMeHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.percent_used = source["percent_used"];
	        this.available_spare = source["available_spare"];
	        this.spare_threshold = source["spare_threshold"];
	        this.temperature_c = source["temperature_c"];
	        this.power_on_hours = source["power_on_hours"];
	        this.media_errors = source["media_errors"];
	        this.critical_warning = source["critical_warning"];
	        this.tb_written = source["tb_written"];
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
	export class ToolIssue {
	    level: string;
	    text: string;
	    fix: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.text = source["text"];
	        this.fix = source["fix"];
	    }
	}
	export class ToolCopy {
	    path: string;
	    scope: string;
	    version: string;
	    active: boolean;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolCopy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.scope = source["scope"];
	        this.version = source["version"];
	        this.active = source["active"];
	        this.note = source["note"];
	    }
	}
	export class DevTool {
	    id: string;
	    name: string;
	    command: string;
	    status: string;
	    version: string;
	    copies: ToolCopy[];
	    issues: ToolIssue[];
	    vars: EnvVariable[];
	
	    static createFrom(source: any = {}) {
	        return new DevTool(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.command = source["command"];
	        this.status = source["status"];
	        this.version = source["version"];
	        this.copies = this.convertValues(source["copies"], ToolCopy);
	        this.issues = this.convertValues(source["issues"], ToolIssue);
	        this.vars = this.convertValues(source["vars"], EnvVariable);
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
	export class PathEntry {
	    path: string;
	    raw: string;
	    scope: string;
	    index: number;
	    is_valid: boolean;
	    is_duplicate: boolean;
	    status_text: string;
	    fixable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PathEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.raw = source["raw"];
	        this.scope = source["scope"];
	        this.index = source["index"];
	        this.is_valid = source["is_valid"];
	        this.is_duplicate = source["is_duplicate"];
	        this.status_text = source["status_text"];
	        this.fixable = source["fixable"];
	    }
	}
	export class EnvPathAnalysis {
	    total_entries: number;
	    valid_entries: number;
	    invalid_entries: number;
	    duplicates_count: number;
	    path_entries: PathEntry[];
	    key_variables: EnvVariable[];
	    user_path_raw: string;
	    can_undo: boolean;
	    editable: boolean;
	
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
	        this.user_path_raw = source["user_path_raw"];
	        this.can_undo = source["can_undo"];
	        this.editable = source["editable"];
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

export namespace vdisk {
	
	export class Disk {
	    name: string;
	    kind: string;
	    path: string;
	    default: boolean;
	    version: number;
	    size_bytes: number;
	    on_disk_bytes: number;
	    sparse: boolean;
	    missing: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Disk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.path = source["path"];
	        this.default = source["default"];
	        this.version = source["version"];
	        this.size_bytes = source["size_bytes"];
	        this.on_disk_bytes = source["on_disk_bytes"];
	        this.sparse = source["sparse"];
	        this.missing = source["missing"];
	    }
	}
	export class DockerRow {
	    type: string;
	    count: string;
	    size: string;
	    reclaimable: string;
	
	    static createFrom(source: any = {}) {
	        return new DockerRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.count = source["count"];
	        this.size = source["size"];
	        this.reclaimable = source["reclaimable"];
	    }
	}
	export class DockerUsage {
	    installed: boolean;
	    running: boolean;
	    rows: DockerRow[];
	
	    static createFrom(source: any = {}) {
	        return new DockerUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.running = source["running"];
	        this.rows = this.convertValues(source["rows"], DockerRow);
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

export namespace winget {
	
	export class Package {
	    name: string;
	    id: string;
	    version: string;
	    available: string;
	    source: string;
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Package(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.id = source["id"];
	        this.version = source["version"];
	        this.available = source["available"];
	        this.source = source["source"];
	        this.truncated = source["truncated"];
	    }
	}
	export class Result {
	    status: string;
	    message: string;
	    packages: Package[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.packages = this.convertValues(source["packages"], Package);
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

