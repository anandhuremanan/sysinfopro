# Microsoft Store listing: SysInfo Pro 3.2.2

Copy each section into the matching field in Partner Center. Limits are Partner Center's.

Positioning for this release: a plain-spoken doctor for Windows PCs, made for developer machines, that works without administrator rights. Lead with diagnoses, not scores. Keep "cleaner", "boost", "optimize", "junk" and "health score" out of the text and keywords: they put the app in the scareware category.

## Properties

- **Category:** Utilities & tools
- **Privacy policy URL:** https://sysinfopro.imanandhu.in/privacy.html
- **Website:** https://sysinfopro.imanandhu.in
- **Support contact:** https://github.com/anandhuremanan/sysinfopro/issues

## Product name

SysInfo Pro

## Short description (up to 1,000 characters)

Find out why your PC is slow, what is filling your drive and what is wrong with your developer setup, in plain words, with what to try next. Every tool works without administrator rights, so it runs on a locked-down work laptop too.

## Description (up to 10,000 characters)

Find out why your PC is slow, what is filling your drive and what is wrong with your developer setup, in plain words, with what to try next.

Git projects shows the work that exists only on this PC. Dev doctor shows which python, node or java really runs when you type it. WSL & Docker shows the true size of each virtual disk. Every tool works without administrator rights.

No account, no analytics, no scare tactics: only the numbers Windows and your drives report. Nothing is changed until you confirm, and changes to your PATH can be undone.

Answers, not scores
• Why so slow: watches the PC for a few seconds and says what is holding it back right now, such as a Defender scan, Windows Update, search indexing, WSL holding memory, or a processor slowed by heat, and what to try.
• Network doctor: checks each link in turn (this PC, the router, name lookup, the internet, and any site you type) and says where the connection breaks. Spots sign-in pages, proxies and hosts-file redirects.
• Dev doctor: for Python, Node.js, Java, Git, Go, .NET and Rust, which copy runs in a new terminal, which others are installed, and settings that point builds elsewhere: the Microsoft Store python placeholder, nvm or pyenv having no effect, JAVA_HOME or GOROOT pointing at a different version.
• Security check: antivirus, firewall, updates, BitLocker, Secure Boot and more, in one read-only pass.

Made for developer machines
• Git projects: finds every git project on a drive and lists commits never pushed (by branch), stashes, uncommitted changes and projects with no remote. Read-only; nothing is fetched.
• Dev caches: npm, pnpm, pip, NuGet, Gradle, Maven, Cargo and Go caches, plus old node_modules, Rust and .NET build folders and Python environments, with when each was last used.
• WSL & Docker: the real size of each virtual disk, and removal of Docker data it no longer uses.
• PATH: broken and duplicate entries removed from your own PATH in one click, with undo.
• Ports and File locks: which program is using a port, or holding a file you cannot delete.

Your drive
• Disk usage: the largest folders and files on any drive, measured in the background while you do other things.
• Disk growth: watch a drive or folder and see what is growing, and how fast.
• Duplicates: files with exactly the same contents, checked byte for byte. Extra copies go to the Recycle Bin and one copy is always kept.
• Drive health as the drive reports it (wear, temperature, hours), and temporary files cleared with live progress.

Your PC
• Battery: real capacity from Windows' own battery report, how it has changed week by week, and how long a charge lasts now compared with new.
• Hand over: before you sell or return a PC, a checklist of signed-in accounts, browsers, OneDrive, saved Wi-Fi and sign-ins, developer keys, encryption and the licence type. Nothing is changed.
• Specs: a full inventory with a shareable spec card, and a one-page report you can send to IT or a forum, with serial numbers, names and addresses left out.
• What changed, Startup, Software updates through winget, Speed test and a short Stress test.

Always at hand
Shrink the app to a small widget that stays on top of every app (Ctrl+Alt+W), get an alert before a drive fills up, and press Ctrl+K to jump to any tool. Light, dark and pure-black AMOLED themes with any accent colour.

## What's new in this version (up to 1,500 characters)

SysInfo Pro 3.2.2
• Move off C: move big caches (npm, pip, Go, NuGet, Gradle and more), WSL disks and browser data to another drive without breaking them, and move them back any time.
• File shredder: overwrite files before deleting them, and wipe a drive's free space. It tells you honestly what overwriting can and cannot promise on SSDs.
• Clearing the Go module cache and other read-only caches is more than twice as fast.

Also in 3.2.1
• Choose which drive SysInfo Pro keeps its files on (Settings > General).
• A short welcome and an optional guided tour for first-time users.

Also in 3.2
• Git projects: work that exists only on this PC, such as unpushed commits and stashes.
• Why so slow: what is holding the PC back right now, in plain words.
• Network doctor: where the connection breaks, one link at a time.
• Dev doctor: which python, node or java really runs.
• Duplicates, Battery history and a Hand over checklist before you sell a PC.
• Health score worked out from real readings, with a screen that explains every point.

## Product features (up to 20, 200 characters each)

1. Why so slow: what is holding the PC back right now, and what to try
2. Git projects: unpushed commits, stashes and projects with no remote, across a whole drive
3. Dev doctor: which python, node, java, git or go really runs, and what shadows it
4. Network doctor: where the connection breaks, from the router to a single site
5. Every tool works without administrator rights
6. See the real size of WSL and Docker disks, and clean up Docker
7. Free npm, pip, NuGet, Gradle, Maven, Cargo and Go caches and old build folders, with last-used dates
8. Fix broken and duplicate PATH entries, with undo
9. Find identical files, checked byte for byte; one copy is always kept
10. Battery capacity history from Windows' own battery report
11. Hand over: a checklist before you sell or return a PC
12. Find the largest folders and files on any drive, in the background
13. Move off C: move big caches, WSL disks and browser data to another drive, and back
14. See which program is using a port or holding a file open
15. Read-only security check: antivirus, firewall, updates, BitLocker and more
16. Real drive health: wear, temperature and hours
17. One-page report to share, with personal details left out
18. Floating widget that stays on top, and alerts before a drive fills up
19. File shredder: overwrite files before deleting them, and wipe free space
20. No account, no analytics, no ads

## Keywords (up to 7)

developer tools, git, WSL, disk space, PATH, network troubleshooter, system information

## Copyright and trademark info

© 2026 Probably Fine Labs

## Screenshots

Make a new set for 3.2 in `build/store-screenshots/` (1920 × 1080), in this order: Why so slow, Git projects, Dev doctor, WSL & Docker, Network doctor, Hand over, home screen, widget. Use sample data where a real screen would show personal paths, names or serial numbers.

## Submission options: restricted capabilities

Partner Center asks why each restricted capability is needed. Paste:

runFullTrust: SysInfo Pro is a desktop system utility. It reads hardware, disk, process, network and security information, runs read-only commands such as git, powercfg and netsh, ends processes the user chooses, and deletes or recycles files and folders the user selects. These require a full-trust desktop process.

unvirtualizedResources: SysInfo Pro lets the user remove broken and duplicate entries from their own PATH environment variable, stored in HKCU\Environment. Registry write virtualization must be off so the change reaches the user's real PATH; otherwise the edit would only apply inside the app's container and have no effect. The app writes no other registry values, only after the user confirms, and keeps a copy so the change can be undone.

## Notes for certification

No account or sign-in is needed; every feature works on first launch.

Actions that change the system only happen when the user asks, after a confirmation dialog: ending a process (Processes, Ports, File locks, Why so slow); deleting folders (Dev caches and Disk & health), limited to known cache folders, temporary files, browser caches and project build folders recognised by their project files; moving duplicate files to the Recycle Bin (at least one copy of each is always kept); removing entries from the user's own PATH (with undo); removing unused Docker data (docker system prune, volumes kept); and updating apps through winget.

Read-only tools: Git projects runs git status and similar read-only git commands in projects it finds, and never fetches or changes them; Battery runs powercfg /batteryreport; Hand over reads account, browser and network settings and never reads passwords or key contents.

Low-space alerts appear as Windows notifications and inside the app. A global shortcut (Ctrl+Alt+W by default, changeable or off in Settings) switches to the widget.

Network use: the speed test contacts Cloudflare, or Measurement Lab if Cloudflare is unavailable; Network doctor pings the user's own router, looks up www.microsoft.com and fetches Windows' connectivity check (www.msftconnecttest.com), plus any site the user types; the Software tool fetches latest version numbers from nodejs.org, go.dev, endoflife.date and GitHub, and its All apps list runs Windows Package Manager (winget); a connection check contacts 8.8.8.8. The app does not check for or download its own updates.

Some security checks report "Unknown" on accounts that cannot read a setting; this is expected and not an error.
