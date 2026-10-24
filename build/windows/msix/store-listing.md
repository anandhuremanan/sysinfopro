# Microsoft Store listing: SysInfo Pro 3.1.0

Copy each section into the matching field in Partner Center. Limits are Partner Center's.

## Properties

- **Category:** Utilities & tools
- **Privacy policy URL:** https://sysinfopro.imanandhu.in/privacy.html
- **Website:** https://sysinfopro.imanandhu.in
- **Support contact:** https://github.com/anandhuremanan/sysinfopro/issues

## Product name

SysInfo Pro

## Short description (up to 1,000 characters)

See what your PC is doing and fix what is in the way. SysInfo Pro shows health, storage, ports, PATH and developer caches in one small window, with a floating widget that stays on top of every app.

## Description (up to 10,000 characters)

SysInfo Pro is a system tool for people who use Windows every day, and especially for people who write code on it. It shows how your PC is doing at a glance, and gives you one-click tools for the problems that usually need three different programs.

The home screen shows a health score, CPU, memory, storage and battery, updating live. Below them are your favourite tools; every tool is one click away.

Tools for developers
• Ports: see which project and git branch opened each listening port, how long it has been running, and release a stuck port in one click.
• Dev caches: measure your npm, pnpm, pip, NuGet, Gradle, Maven, Cargo and Go caches, find old node_modules, Rust and .NET build folders and Python environments, and free everything unused for months in one click.
• WSL and Docker: see how much space each virtual disk really takes, remove Docker data it no longer uses, and follow safe steps to shrink a disk.
• File locks: pick a file or folder that will not delete or move, see every program holding it open, and end the one in the way.
• PATH: find broken and duplicate PATH entries and remove them from your own PATH in one click, with undo.

Tools for your disk
• Disk usage: find the largest folders and files on any drive.
• Disk growth: watch a folder and see which parts are growing, and how fast.
• Disk health and cleanup: your drive's real health, wear and temperature, and temporary files and browser caches cleared with live progress.

Tools for your system
• What changed: compare startup programs, installed software, listening ports and PATH with the last time you looked.
• Security check: antivirus, firewall, Windows updates, BitLocker, Secure Boot, User Account Control and more, in one read-only pass.
• Hardware specs: a full inventory down to each memory module, with a one-click report.
• Processes: sort running programs by memory, CPU or how fast their memory is growing, to catch a leak early.
• Software: check developer tools against their latest releases, and update your apps through Windows Package Manager (winget).
• Startup programs, internet speed test, battery health and a ten-second stress test.
• Press Ctrl+K to jump to any tool by typing.

A widget that stays on top
Shrink the app to a small floating window showing CPU, memory, battery and the space left on each drive. It stays above every app, even through Show desktop, and an arrow appears the moment a drive starts filling up. Press Ctrl+Alt+W to switch to it from any app.

Alerts before a drive fills up
Get a notification when a drive drops below the free space you choose, or loses gigabytes in minutes. An optional monthly reminder tells you how much you could free.

Private by design
No account, no analytics, no advertising. Everything the app reads stays on your PC. Light and dark themes follow Windows, or pick light, dark or pure-black AMOLED in Settings, with any accent colour you like.

## What's new in this version (up to 1,500 characters)

SysInfo Pro 3.1
• Low-space alerts: a notification when a drive runs low or fills up fast.
• Dev caches now finds Rust, .NET, Next.js and other build folders and Python environments, not just node_modules.
• New WSL & Docker tool: the real size of each virtual disk, and Docker cleanup.
• Update your apps through winget from the Software tool.
• Fix broken and duplicate PATH entries in one click, with undo.
• Real drive health: wear, temperature and hours, as the drive reports them.
• Temporary file cleanup shows live progress and can be stopped.
• Ctrl+K quick search, and Ctrl+Alt+W to switch to the widget.
• AMOLED theme and accent colours, including neon shades for gaming setups.
• Optional monthly cleanup reminder.

## Product features (up to 20, 200 characters each)

1. Health score with live CPU, memory, storage and battery
2. Alerts when a drive runs low or fills up fast
3. See which project and git branch opened each port
4. Free space from npm, pnpm, pip, NuGet, Gradle, Maven, Cargo and Go caches
5. Free old node_modules, Rust and .NET build folders and Python environments in one click
6. See the real size of WSL and Docker disks, and clean up Docker
7. Find the program locking a file or folder
8. Find the largest folders and files on any drive
9. Watch a folder to see what is growing, and how fast
10. See what changed: startup programs, software, ports and PATH
11. Read-only security check: antivirus, firewall, updates, BitLocker and more
12. Real drive health: wear, temperature and hours
13. Update your apps through winget
14. Fix broken and duplicate PATH entries, with undo
15. Floating widget that stays on top, with a keyboard shortcut
16. Sort processes by memory growth to catch leaks
17. Full hardware inventory with an exportable report
18. Ctrl+K quick search across every tool
19. Light, dark and AMOLED themes with your choice of accent colour
20. No account, no analytics, no ads

## Keywords (up to 7)

disk space, developer tools, cache cleaner, node_modules, WSL, port checker, system monitor

## Copyright and trademark info

© 2026 Probably Fine Labs

## Screenshots

Upload the eight files in `build/store-screenshots/` in their numbered order (1920 × 1080). They use sample data where a real screen would show personal paths.

## Submission options: restricted capabilities

Partner Center asks why each restricted capability is needed. Paste:

runFullTrust: SysInfo Pro is a desktop system utility. It reads hardware, disk, process, network and security information, ends processes the user chooses, and deletes cache folders the user selects. These require a full-trust desktop process.

unvirtualizedResources: SysInfo Pro lets the user remove broken and duplicate entries from their own PATH environment variable, stored in HKCU\Environment. Registry write virtualization must be off so the change reaches the user's real PATH; otherwise the edit would only apply inside the app's container and have no effect. The app writes no other registry values, only after the user confirms, and keeps a copy so the change can be undone.

## Notes for certification

No account or sign-in is needed; every feature works on first launch.

Actions that change the system only happen when the user asks, after a confirmation dialog: ending a process (Processes, Ports, File locks); deleting folders (Dev caches and Disk & health), limited to known cache folders, temporary files, browser caches and project build folders recognised by their project files; removing entries from the user's own PATH (with undo); removing unused Docker data (docker system prune, volumes kept); and updating apps through winget.

Low-space alerts appear as Windows notifications and inside the app. A global shortcut (Ctrl+Alt+W by default, changeable or off in Settings) switches to the widget.

Network use: the speed test contacts Cloudflare, or Measurement Lab if Cloudflare is unavailable; the Software tool fetches latest version numbers from nodejs.org, go.dev, endoflife.date and GitHub, and its All apps list runs Windows Package Manager (winget); a connection check contacts 8.8.8.8. The app does not check for or download its own updates.

Some security checks report "Unknown" on accounts that cannot read a setting; this is expected and not an error.
