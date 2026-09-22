// ==========================================================================
// Termino Optimized Desktop & Window Manager
// ==========================================================================

const desktop = {
    maxZIndex: 100,
    activeWindowId: null,

    init() {
        this.setupEventListeners();
        this.restoreSavedTheme();
        this.scanAndInitWindows();
    },

    // ---------------- Window Focus & Stacking ----------------
    bringToFront(id) {
        if (this.activeWindowId === id) return;

        const win = document.getElementById(`terminal-${id}`);
        if (!win) return;

        this.maxZIndex += 1;
        win.style.zIndex = this.maxZIndex;
        this.activeWindowId = id;

        // Update active class on windows
        const prevActive = document.querySelector('.retro-window.active-window');
        if (prevActive && prevActive !== win) {
            prevActive.classList.remove('active-window');
        }
        win.classList.add('active-window');

        // Focus the active terminal session
        if (window.terminals && window.terminals[id]) {
            window.terminals[id].focus();
        }
    },

    // ---------------- Window Maximize / Restore ----------------
    toggleMaximize(id) {
        const win = document.getElementById(`terminal-${id}`);
        if (!win) return;

        this.bringToFront(id);

        if (win.classList.contains('maximized')) {
            // Restore original geometry
            win.classList.remove('maximized');
            win.style.left = win.dataset.prevLeft || '50px';
            win.style.top = win.dataset.prevTop || '40px';
            win.style.width = win.dataset.prevWidth || '720px';
            win.style.height = win.dataset.prevHeight || '440px';
            const maxBtn = win.querySelector('.win-max');
            if (maxBtn) maxBtn.textContent = '^';
        } else {
            // Store geometry and maximize
            win.dataset.prevLeft = win.style.left;
            win.dataset.prevTop = win.style.top;
            win.dataset.prevWidth = win.style.width;
            win.dataset.prevHeight = win.style.height;

            win.classList.add('maximized');
            const maxBtn = win.querySelector('.win-max');
            if (maxBtn) maxBtn.textContent = 'v';
        }

        if (window.terminals && window.terminals[id]) {
            requestAnimationFrame(() => window.terminals[id].fit(true));
        }
    },

    // ---------------- Window Dragging (Touch + Mouse Optimized via RAF) ----------------
    startDrag(e, id) {
        if (e.target.closest('.win-btn')) return;

        const win = document.getElementById(`terminal-${id}`);
        if (!win || win.classList.contains('maximized')) return;

        this.bringToFront(id);

        const isTouch = !!(e.touches && e.touches.length > 0);
        const startX = isTouch ? e.touches[0].clientX : e.clientX;
        const startY = isTouch ? e.touches[0].clientY : e.clientY;
        const rect = win.getBoundingClientRect();
        const initialLeft = rect.left;
        const initialTop = rect.top;

        let rafId = null;
        let pendingX = startX;
        let pendingY = startY;

        const onMove = (moveEvent) => {
            const touch = (moveEvent.touches && moveEvent.touches.length > 0) ? moveEvent.touches[0] : moveEvent;
            pendingX = touch.clientX;
            pendingY = touch.clientY;

            if (!rafId) {
                rafId = requestAnimationFrame(() => {
                    rafId = null;
                    const dx = pendingX - startX;
                    const dy = pendingY - startY;

                    const newLeft = Math.max(0, Math.min(window.innerWidth - 60, initialLeft + dx));
                    const newTop = Math.max(0, Math.min(window.innerHeight - 36, initialTop + dy));

                    win.style.left = `${newLeft}px`;
                    win.style.top = `${newTop}px`;
                });
            }
        };

        const onEnd = () => {
            if (rafId) {
                cancelAnimationFrame(rafId);
                rafId = null;
            }
            document.removeEventListener('mousemove', onMove);
            document.removeEventListener('mouseup', onEnd);
            document.removeEventListener('touchmove', onMove);
            document.removeEventListener('touchend', onEnd);
            document.removeEventListener('touchcancel', onEnd);
        };

        if (isTouch) {
            document.addEventListener('touchmove', onMove, { passive: false });
            document.addEventListener('touchend', onEnd);
            document.addEventListener('touchcancel', onEnd);
        } else {
            document.addEventListener('mousemove', onMove, { passive: true });
            document.addEventListener('mouseup', onEnd);
        }
        if (e.cancelable) e.preventDefault();
    },

    // ---------------- Window Resizing (Touch + Mouse Optimized via RAF) ----------------
    startResize(e, id) {
        const win = document.getElementById(`terminal-${id}`);
        if (!win || win.classList.contains('maximized')) return;

        this.bringToFront(id);

        const isTouch = !!(e.touches && e.touches.length > 0);
        const startX = isTouch ? e.touches[0].clientX : e.clientX;
        const startY = isTouch ? e.touches[0].clientY : e.clientY;
        const startWidth = parseInt(win.style.width || win.offsetWidth, 10);
        const startHeight = parseInt(win.style.height || win.offsetHeight, 10);

        let rafId = null;
        let pendingX = startX;
        let pendingY = startY;

        const onMove = (moveEvent) => {
            const touch = (moveEvent.touches && moveEvent.touches.length > 0) ? moveEvent.touches[0] : moveEvent;
            pendingX = touch.clientX;
            pendingY = touch.clientY;

            if (!rafId) {
                rafId = requestAnimationFrame(() => {
                    rafId = null;
                    const minW = window.innerWidth <= 768 ? 240 : 320;
                    const minH = window.innerWidth <= 768 ? 160 : 200;
                    const newWidth = Math.max(minW, startWidth + (pendingX - startX));
                    const newHeight = Math.max(minH, startHeight + (pendingY - startY));

                    win.style.width = `${newWidth}px`;
                    win.style.height = `${newHeight}px`;

                    if (window.terminals && window.terminals[id]) {
                        window.terminals[id].fit(false);
                    }
                });
            }
        };

        const onEnd = () => {
            if (rafId) {
                cancelAnimationFrame(rafId);
                rafId = null;
            }
            document.removeEventListener('mousemove', onMove);
            document.removeEventListener('mouseup', onEnd);
            document.removeEventListener('touchmove', onMove);
            document.removeEventListener('touchend', onEnd);
            document.removeEventListener('touchcancel', onEnd);
            if (window.terminals && window.terminals[id]) {
                window.terminals[id].fit(true);
            }
        };

        if (isTouch) {
            document.addEventListener('touchmove', onMove, { passive: false });
            document.addEventListener('touchend', onEnd);
            document.addEventListener('touchcancel', onEnd);
        } else {
            document.addEventListener('mousemove', onMove, { passive: true });
            document.addEventListener('mouseup', onEnd);
        }
        if (e.cancelable) e.preventDefault();
    },

    // ---------------- Cascade & Tile Layouts ----------------
    cascadeWindows() {
        const windows = Array.from(document.querySelectorAll('.retro-window'));
        windows.forEach((win, index) => {
            if (win.classList.contains('maximized')) {
                const id = win.getAttribute('data-session-id');
                this.toggleMaximize(id);
            }
            const offset = (index % 12) * 28;
            win.style.left = `${30 + offset}px`;
            win.style.top = `${25 + offset}px`;
            win.style.width = '720px';
            win.style.height = '440px';

            const id = win.getAttribute('data-session-id');
            if (window.terminals && window.terminals[id]) {
                window.terminals[id].fit(true);
            }
        });
        if (windows.length > 0) {
            const lastId = windows[windows.length - 1].getAttribute('data-session-id');
            if (lastId) this.bringToFront(lastId);
        }
    },

    tileWindows() {
        const windows = Array.from(document.querySelectorAll('.retro-window'));
        const count = windows.length;
        if (count === 0) return;

        const desktopWidth = window.innerWidth;
        const desktopHeight = window.innerHeight;

        let cols = 1;
        let rows = 1;
        if (count === 2) {
            cols = 2; rows = 1;
        } else if (count <= 4) {
            cols = 2; rows = 2;
        } else if (count <= 6) {
            cols = 3; rows = 2;
        } else {
            cols = Math.ceil(Math.sqrt(count));
            rows = Math.ceil(count / cols);
        }

        const tileWidth = Math.floor(desktopWidth / cols);
        const tileHeight = Math.floor(desktopHeight / rows);

        windows.forEach((win, i) => {
            if (win.classList.contains('maximized')) {
                win.classList.remove('maximized');
            }
            const c = i % cols;
            const r = Math.floor(i / cols);

            win.style.left = `${c * tileWidth}px`;
            win.style.top = `${r * tileHeight}px`;
            win.style.width = `${tileWidth}px`;
            win.style.height = `${tileHeight}px`;

            const id = win.getAttribute('data-session-id');
            if (window.terminals && window.terminals[id]) {
                window.terminals[id].fit(true);
            }
        });
    },

    // ---------------- Context Menu (Right Click) ----------------
    openContextMenu(x, y) {
        const menu = document.getElementById('context-menu');
        if (!menu) return;

        menu.style.left = `${x}px`;
        menu.style.top = `${y}px`;
        menu.classList.add('open');

        const rect = menu.getBoundingClientRect();
        if (rect.right > window.innerWidth) {
            menu.style.left = `${window.innerWidth - rect.width - 4}px`;
        }
        if (rect.bottom > window.innerHeight) {
            menu.style.top = `${window.innerHeight - rect.height - 4}px`;
        }
    },

    closeContextMenu() {
        const menu = document.getElementById('context-menu');
        if (menu) menu.classList.remove('open');
    },

    // ---------------- Themes & About Modal ----------------
    getCurrentTheme() {
        const bodyClass = document.body.className || '';
        const match = bodyClass.match(/theme-([a-zA-Z0-9_-]+)/);
        if (match && match[1]) {
            return match[1];
        }
        return localStorage.getItem('termino-theme') || (window.APP_CONFIG && window.APP_CONFIG.theme) || 'classic';
    },

    isLightTheme(themeName) {
        const t = themeName || this.getCurrentTheme();
        return t === 'light';
    },

    setThemeMode(mode) {
        if (mode === 'light') {
            this.selectTheme('light');
        } else {
            const savedDark = localStorage.getItem('termino-last-dark') || 'classic';
            this.selectTheme(savedDark === 'light' ? 'classic' : savedDark);
        }
    },

    selectTheme(themeName) {
        let canonicalTheme = themeName || 'classic';
        if (canonicalTheme === 'dark' || canonicalTheme === 'default') {
            canonicalTheme = 'classic';
        }

        document.body.className = `theme-${canonicalTheme}`;
        localStorage.setItem('termino-theme', canonicalTheme);

        if (this.isLightTheme(canonicalTheme)) {
            localStorage.setItem('termino-last-light', canonicalTheme);
        } else {
            localStorage.setItem('termino-last-dark', canonicalTheme);
        }

        const sel = document.getElementById('theme-select');
        if (sel && sel.value !== canonicalTheme) {
            sel.value = canonicalTheme;
        }

        this.updateThemeModeButtons(canonicalTheme);
        this.updateMetaThemeColor();

        if (typeof updateAllTerminalThemes === 'function') {
            updateAllTerminalThemes();
        }
    },

    updateThemeModeButtons(themeName) {
        const isLight = this.isLightTheme(themeName || this.getCurrentTheme());
        const btnDark = document.getElementById('mode-btn-dark');
        const btnLight = document.getElementById('mode-btn-light');
        if (btnDark && btnLight) {
            if (isLight) {
                btnDark.classList.remove('active');
                btnLight.classList.add('active');
            } else {
                btnLight.classList.remove('active');
                btnDark.classList.add('active');
            }
        }
    },

    updateMetaThemeColor() {
        const style = getComputedStyle(document.body);
        const desktopBg = style.getPropertyValue('--desktop').trim();
        if (desktopBg) {
            let meta = document.getElementById('meta-theme-color');
            if (!meta) {
                meta = document.querySelector('meta[name="theme-color"]');
            }
            if (meta) {
                meta.setAttribute('content', desktopBg);
            }
        }
    },

    restoreSavedTheme() {
        const saved = localStorage.getItem('termino-theme') || localStorage.getItem('retroterm-theme');
        if (saved) {
            this.selectTheme(saved);
            return;
        }

        const configTheme = (window.APP_CONFIG && window.APP_CONFIG.theme);
        if (configTheme && configTheme !== 'default') {
            this.selectTheme(configTheme);
            return;
        }

        if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
            this.selectTheme('light');
        } else {
            this.selectTheme('classic');
        }
    },

    openAboutDialog() {
        this.closeContextMenu();
        const dlg = document.getElementById('about-dialog');
        if (dlg) {
            dlg.classList.add('open');
            const current = this.getCurrentTheme();
            const sel = document.getElementById('theme-select');
            if (sel) {
                sel.value = current;
            }
            this.updateThemeModeButtons(current);
        }
    },

    closeAboutDialog() {
        const dlg = document.getElementById('about-dialog');
        if (dlg) dlg.classList.remove('open');
    },

    // ---------------- Scanner for Dynamic Windows ----------------
    scanAndInitWindows() {
        const isMobile = window.innerWidth <= 768;
        document.querySelectorAll('.retro-window').forEach(win => {
            const sessionId = win.getAttribute('data-session-id');
            if (sessionId && (!window.terminals || !window.terminals[sessionId])) {
                // Ensure window geometry never overflows viewport
                this.clampWindowToViewport(win);

                // On mobile / narrow viewports, auto-maximize new windows to fit phone screen
                if (isMobile && !win.classList.contains('maximized')) {
                    this.toggleMaximize(sessionId);
                }
                if (typeof initTerminal === 'function') {
                    initTerminal(sessionId);
                    this.bringToFront(sessionId);
                }
            }
        });
    },

    clampWindowToViewport(win) {
        const viewW = window.innerWidth;
        const viewH = window.innerHeight;
        if (viewW <= 768) {
            win.style.left = '0px';
            win.style.top = '0px';
            win.style.width = '100%';
            win.style.height = '100%';
            return;
        }
        const currentW = parseInt(win.style.width, 10) || win.offsetWidth || 720;
        const currentH = parseInt(win.style.height, 10) || win.offsetHeight || 440;
        const currentLeft = parseInt(win.style.left, 10) || 0;
        const currentTop = parseInt(win.style.top, 10) || 0;

        const maxW = Math.min(currentW, viewW - 20);
        const maxH = Math.min(currentH, viewH - 20);
        const safeLeft = Math.max(0, Math.min(currentLeft, viewW - maxW));
        const safeTop = Math.max(0, Math.min(currentTop, viewH - maxH));

        win.style.width = `${maxW}px`;
        win.style.height = `${maxH}px`;
        win.style.left = `${safeLeft}px`;
        win.style.top = `${safeTop}px`;
    },

    // ---------------- Event Listeners ----------------
    setupEventListeners() {
        // Right-click context menu for desktop
        document.addEventListener('contextmenu', (e) => {
            const selection = window.getSelection();
            if (selection && selection.toString().length > 0 && e.target.closest('.terminal-screen')) {
                return;
            }
            e.preventDefault();
            this.openContextMenu(e.clientX, e.clientY);
        });

        // Touch events on desktop background: long-press (500ms) or double-tap (<350ms) opens menu
        let longPressTimer = null;
        let lastTapTime = 0;

        document.addEventListener('touchstart', (e) => {
            if (e.target.id === 'desktop' || e.target === document.body) {
                const touch = e.touches[0];
                const x = touch.clientX;
                const y = touch.clientY;

                longPressTimer = setTimeout(() => {
                    this.openContextMenu(x, y);
                }, 500);

                const now = Date.now();
                if (now - lastTapTime < 350) {
                    clearTimeout(longPressTimer);
                    this.openContextMenu(x, y);
                }
                lastTapTime = now;
            }
        }, { passive: true });

        document.addEventListener('touchmove', () => {
            if (longPressTimer) {
                clearTimeout(longPressTimer);
                longPressTimer = null;
            }
        }, { passive: true });

        document.addEventListener('touchend', () => {
            if (longPressTimer) {
                clearTimeout(longPressTimer);
                longPressTimer = null;
            }
        }, { passive: true });

        // Close context menu when tapping/clicking outside
        document.addEventListener('click', (e) => {
            if (!e.target.closest('#context-menu')) {
                this.closeContextMenu();
            }
        });

        document.addEventListener('touchstart', (e) => {
            if (!e.target.closest('#context-menu')) {
                this.closeContextMenu();
            }
        }, { passive: true });

        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeContextMenu();
                this.closeAboutDialog();
            }
        });

        // Click or tap on window brings to front
        document.addEventListener('mousedown', (e) => {
            const win = e.target.closest('.retro-window');
            if (win) {
                const id = win.getAttribute('data-session-id');
                if (id) this.bringToFront(id);
            }
        });

        // HTMX after-swap: initialize newly inserted windows
        document.body.addEventListener('htmx:afterSwap', () => {
            requestAnimationFrame(() => {
                this.scanAndInitWindows();
            });
        });

        // MutationObserver to clean up deleted terminal instances
        const observer = new MutationObserver((mutations) => {
            mutations.forEach(m => {
                m.removedNodes.forEach(node => {
                    if (node.nodeType === 1 && node.classList && node.classList.contains('retro-window')) {
                        const id = node.getAttribute('data-session-id');
                        if (id && window.terminals && window.terminals[id]) {
                            window.terminals[id].destroy();
                        }
                    }
                });
            });
        });

        observer.observe(document.getElementById('desktop') || document.body, {
            childList: true,
            subtree: true,
        });

        // Window resize: debounced reclamp and re-fit for active terminals
        let resizeTimer = null;
        const handleViewportResize = () => {
            if (resizeTimer) clearTimeout(resizeTimer);
            resizeTimer = setTimeout(() => {
                document.querySelectorAll('.retro-window').forEach(win => {
                    desktop.clampWindowToViewport(win);
                });
                if (window.terminals) {
                    Object.values(window.terminals).forEach(session => {
                        if (session.fit) session.fit(true);
                    });
                }
            }, 60);
        };

        window.addEventListener('resize', handleViewportResize);
        window.addEventListener('orientationchange', handleViewportResize);

        // Visual Viewport resize (handles mobile soft keyboard open/close seamlessly)
        if (window.visualViewport) {
            window.visualViewport.addEventListener('resize', () => {
                const activeMax = document.querySelector('.retro-window.maximized');
                if (activeMax) {
                    activeMax.style.height = `${window.visualViewport.height}px`;
                    const id = activeMax.getAttribute('data-session-id');
                    if (window.terminals && window.terminals[id]) {
                        window.terminals[id].fit(true);
                    }
                }
            });
        }
    }
};

document.addEventListener('DOMContentLoaded', () => {
    desktop.init();
});
