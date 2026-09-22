// ==========================================================================
// Termino Optimized Terminal Client Management (xterm.js + WebSocket)
// ==========================================================================

window.terminals = window.terminals || {};

/**
 * Returns current terminal theme colors read from CSS variables.
 */
function getTerminalThemeFromCSS() {
    const style = getComputedStyle(document.body);
    const bg = style.getPropertyValue('--terminal-bg').trim() || '#121212';
    const fg = style.getPropertyValue('--terminal-fg').trim() || '#e0e0e0';
    const cursor = style.getPropertyValue('--terminal-cursor').trim() || '#e0e0e0';
    const selection = style.getPropertyValue('--terminal-selection').trim();
    const mode = style.getPropertyValue('--theme-mode').trim();
    const isLight = mode === 'light' || document.body.classList.contains('theme-light');

    if (isLight) {
        return {
            background: bg,
            foreground: fg,
            cursor: cursor,
            cursorAccent: bg,
            selectionBackground: selection || 'rgba(0, 0, 0, 0.18)',
            black: '#000000',
            red: '#cd3131',
            green: '#008000',
            yellow: '#946600',
            blue: '#0451a5',
            magenta: '#bc05bc',
            cyan: '#0598bc',
            white: '#555555',
            brightBlack: '#666666',
            brightRed: '#cd3131',
            brightGreen: '#14ce14',
            brightYellow: '#b5ba00',
            brightBlue: '#0451a5',
            brightMagenta: '#bc05bc',
            brightCyan: '#0598bc',
            brightWhite: '#1f1f1f',
        };
    }

    return {
        background: bg,
        foreground: fg,
        cursor: cursor,
        cursorAccent: bg,
        selectionBackground: selection || 'rgba(255, 255, 255, 0.25)',
        black: '#000000',
        red: '#cd0000',
        green: '#00cd00',
        yellow: '#cdcd00',
        blue: '#0000ee',
        magenta: '#cd00cd',
        cyan: '#00cdcd',
        white: '#e5e5e5',
        brightBlack: '#7f7f7f',
        brightRed: '#ff0000',
        brightGreen: '#00ff00',
        brightYellow: '#ffff00',
        brightBlue: '#5c5cff',
        brightMagenta: '#ff00ff',
        brightCyan: '#00ffff',
        brightWhite: '#ffffff',
    };
}

/**
 * Computes optimal font size for narrow viewports.
 */
function getOptimalFontSize() {
    const base = (window.APP_CONFIG && window.APP_CONFIG.fontSize) ? window.APP_CONFIG.fontSize : 14;
    if (window.innerWidth <= 400) {
        return 11;
    } else if (window.innerWidth <= 600) {
        return 12;
    }
    return base;
}

/**
 * Initializes an xterm.js instance and binds it to the server WebSocket.
 */
function initTerminal(sessionId) {
    if (window.terminals[sessionId]) {
        return window.terminals[sessionId];
    }

    const container = document.getElementById(`term-container-${sessionId}`);
    if (!container) {
        return null;
    }

    const fontSize = getOptimalFontSize();
    const scrollback = (window.APP_CONFIG && window.APP_CONFIG.scrollback) ? window.APP_CONFIG.scrollback : 5000;

    // Instantiate xterm with lightweight options
    const term = new Terminal({
        cursorBlink: false, // Disabling cursor blink saves idle CPU rendering cycles
        cursorStyle: 'block',
        fontSize: fontSize,
        fontFamily: '"Cascadia Code", "Fira Code", monospace, "Courier New"',
        scrollback: scrollback,
        theme: getTerminalThemeFromCSS(),
        allowProposedApi: true,
        fastScrollModifier: 'alt',
    });

    // Fit addon for dimension calculation
    let fitAddon = null;
    if (typeof FitAddon !== 'undefined' && FitAddon.FitAddon) {
        fitAddon = new FitAddon.FitAddon();
        term.loadAddon(fitAddon);
    }

    term.open(container);

    // Mobile touch tap-to-focus: tapping anywhere in the container summons the soft keyboard
    container.addEventListener('touchend', (e) => {
        if (!e.defaultPrevented && e.changedTouches && e.changedTouches.length === 1) {
            term.focus();
        }
    }, { passive: true });

    // Update centered title dynamically with running binary or process
    term.onTitleChange((newTitle) => {
        const titleEl = document.getElementById(`term-title-${sessionId}`);
        if (titleEl && newTitle && newTitle.trim()) {
            titleEl.textContent = newTitle.trim();
        }
    });

    // Connect WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/terminal/${sessionId}`;
    const ws = new WebSocket(wsUrl);
    ws.binaryType = 'arraybuffer';

    let lastCols = 0;
    let lastRows = 0;
    let resizeRafId = null;

    // Dynamic responsive fit: computes available container geometry and resizes xterm & backend PTY
    function fitTerminal(force = false) {
        if (!term || !container) return;

        // Dynamically adjust font size if viewport width changed
        const currentOptimal = getOptimalFontSize();
        if (term.options.fontSize !== currentOptimal) {
            term.options.fontSize = currentOptimal;
        }

        // Try standard fit addon
        if (fitAddon) {
            try { fitAddon.fit(); } catch (e) {}
        }

        // Explicit fallback geometry check: guarantees xterm never overflows narrow viewports
        const rect = container.getBoundingClientRect();
        if (rect.width > 0 && rect.height > 0) {
            const dims = (term._core && term._core._renderService && term._core._renderService.dimensions)
                ? term._core._renderService.dimensions.css.cell
                : null;
            const charW = (dims && dims.width > 0) ? dims.width : (currentOptimal * 0.60);
            const charH = (dims && dims.height > 0) ? dims.height : (currentOptimal * 1.25);

            const padX = 8;
            const padY = 6;
            const availW = Math.max(0, rect.width - padX);
            const availH = Math.max(0, rect.height - padY);

            const expectedCols = Math.max(10, Math.floor(availW / charW));
            const expectedRows = Math.max(3, Math.floor(availH / charH));

            if (term.cols !== expectedCols || term.rows !== expectedRows) {
                term.resize(expectedCols, expectedRows);
            }
        }

        // Notify backend PTY of new dimensions
        if ((term.cols !== lastCols || term.rows !== lastRows || force) && ws.readyState === WebSocket.OPEN) {
            lastCols = term.cols;
            lastRows = term.rows;
            ws.send(JSON.stringify({
                type: 'resize',
                cols: term.cols,
                rows: term.rows
            }));
        }
    }

    // Optimized resize: throttled via requestAnimationFrame
    function sendResize(force = false) {
        if (resizeRafId) {
            cancelAnimationFrame(resizeRafId);
            resizeRafId = null;
        }

        resizeRafId = requestAnimationFrame(() => {
            resizeRafId = null;
            fitTerminal(force);
        });
    }

    ws.onopen = () => {
        fitTerminal(true);
        setTimeout(() => fitTerminal(true), 60);
        setTimeout(() => fitTerminal(true), 250);
        term.focus();
    };

    // Keystroke input: encode and send
    const encoder = new TextEncoder();
    term.onData((data) => {
        if (ws.readyState === WebSocket.OPEN) {
            ws.send(encoder.encode(data));
        }
    });

    // PTY output -> WebSocket -> xterm.js
    ws.onmessage = (event) => {
        if (event.data instanceof ArrayBuffer) {
            term.write(new Uint8Array(event.data));
        } else {
            term.write(event.data);
        }
    };

    ws.onclose = () => {
        term.write('\r\n\x1b[33;1m[Terminal disconnected]\x1b[0m\r\n');
    };

    // ResizeObserver throttled via sendResize
    const resizeObserver = new ResizeObserver(() => {
        sendResize(false);
    });
    resizeObserver.observe(container);

    const sessionObj = {
        term: term,
        ws: ws,
        fitAddon: fitAddon,
        resizeObserver: resizeObserver,
        fit: (force) => sendResize(force),
        focus: () => term.focus(),
        destroy: () => {
            if (resizeRafId) {
                cancelAnimationFrame(resizeRafId);
                resizeRafId = null;
            }
            try { resizeObserver.disconnect(); } catch (e) {}
            try { ws.close(); } catch (e) {}
            try { term.dispose(); } catch (e) {}
            delete window.terminals[sessionId];
        }
    };

    window.terminals[sessionId] = sessionObj;
    return sessionObj;
}

/**
 * Updates color themes across all running terminal instances.
 */
function updateAllTerminalThemes() {
    const newTheme = getTerminalThemeFromCSS();
    Object.values(window.terminals).forEach(session => {
        if (session.term && session.term.options) {
            session.term.options.theme = newTheme;
        }
    });
}
