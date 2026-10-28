import type { Component } from 'svelte';
import { getTool, type ToolId } from '$lib/tools';
import BatteryReport from '$lib/components/advanced/BatteryReport.svelte';
import CpuView from '$lib/components/advanced/CpuView.svelte';
import DevCacheCleaner from '$lib/components/advanced/DevCacheCleaner.svelte';
import DevDoctor from '$lib/components/advanced/DevDoctor.svelte';
import DiskAnalyzer from '$lib/components/advanced/DiskAnalyzer.svelte';
import DiskCleaner from '$lib/components/advanced/DiskCleaner.svelte';
import DuplicateFinder from '$lib/components/advanced/DuplicateFinder.svelte';
import EnvPathInspector from '$lib/components/advanced/EnvPathInspector.svelte';
import FileLocks from '$lib/components/advanced/FileLocks.svelte';
import FileShredder from '$lib/components/advanced/FileShredder.svelte';
import GitSweep from '$lib/components/advanced/GitSweep.svelte';
import GrowthWatch from '$lib/components/advanced/GrowthWatch.svelte';
import Handover from '$lib/components/advanced/Handover.svelte';
import HardwareSpecs from '$lib/components/advanced/HardwareSpecs.svelte';
import MoveOffC from '$lib/components/advanced/MoveOffC.svelte';
import NetworkDoctor from '$lib/components/advanced/NetworkDoctor.svelte';
import PortInspector from '$lib/components/advanced/PortInspector.svelte';
import ScoreView from '$lib/components/advanced/ScoreView.svelte';
import SecurityCheck from '$lib/components/advanced/SecurityCheck.svelte';
import SlowNow from '$lib/components/advanced/SlowNow.svelte';
import SoftwareChecker from '$lib/components/advanced/SoftwareChecker.svelte';
import StartupManager from '$lib/components/advanced/StartupManager.svelte';
import StressBenchmark from '$lib/components/advanced/StressBenchmark.svelte';
import VirtualDisks from '$lib/components/advanced/VirtualDisks.svelte';
import WhatChanged from '$lib/components/advanced/WhatChanged.svelte';
import ProcessView from '$lib/components/basic/ProcessView.svelte';
import SpeedAnimation from '$lib/components/basic/SpeedAnimation.svelte';

// Every screen below the home and tools pages, in one place. Adding a tool means
// one entry in tools.ts and one here.

// the screens that are not tools in the tools list
export type ExtraScreen = 'processes' | 'score' | 'cpu';
export type ScreenId = ToolId | ExtraScreen;
// every place the app can show
export type View = 'home' | 'tools' | ScreenId;

// what every screen may receive; each takes only what it needs
export type ScreenProps = {
	systemInfo: any;
	onOpen: (view: string) => void;
	// Disk usage: the drive an alert asked for, and a call to say it has been used
	drive?: string;
	onDriveUsed?: () => void;
};

export const screens: Record<ScreenId, Component<any>> = {
	specs: HardwareSpecs,
	security: SecurityCheck,
	slow: SlowNow,
	battery: BatteryReport,
	handover: Handover,
	changes: WhatChanged,
	stress: StressBenchmark,
	locks: FileLocks,
	move: MoveOffC,
	shred: FileShredder,
	health: DiskCleaner,
	space: DiskAnalyzer,
	growth: GrowthWatch,
	startup: StartupManager,
	software: SoftwareChecker,
	dupes: DuplicateFinder,
	devcache: DevCacheCleaner,
	wsl: VirtualDisks,
	gitsweep: GitSweep,
	devdoctor: DevDoctor,
	env: EnvPathInspector,
	speed: SpeedAnimation,
	netdoctor: NetworkDoctor,
	ports: PortInspector,
	processes: ProcessView,
	score: ScoreView,
	cpu: CpuView
};

const EXTRA_TITLES: Record<ExtraScreen | 'home' | 'tools', string> = {
	home: 'Home',
	tools: 'All tools',
	processes: 'Processes',
	score: 'Health score',
	cpu: 'Processor'
};

export function viewTitle(view: View): string {
	return view in EXTRA_TITLES
		? EXTRA_TITLES[view as keyof typeof EXTRA_TITLES]
		: getTool(view as ToolId).name;
}

export function isView(value: string): value is View {
	return value === 'home' || value === 'tools' || value in screens;
}

export { pushView, popView } from '$lib/nav';
