#!/usr/bin/env bash
# ==============================================================================
# Termino - Vendor Asset Pull Script
# Downloads the latest stable production versions of:
#   - htmx.min.js        (from htmx.org)
#   - xterm.js           (from @xterm/xterm)
#   - xterm.css          (from @xterm/xterm)
#   - xterm-addon-fit.js (from @xterm/addon-fit)
# ==============================================================================

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
VENDOR_DIR="${REPO_ROOT}/web/vendor"

# Temporary directory for atomic download
TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'termino-vendor')"

cleanup() {
    local exit_code=$?
    if [[ -d "${TMP_DIR:-}" ]]; then
        rm -rf "${TMP_DIR}"
    fi
    if [[ ${exit_code} -ne 0 ]]; then
        echo -e "\n[!] Vendor pull failed with exit code ${exit_code}." >&2
    fi
}
trap cleanup EXIT ERR INT TERM

# Ensure curl is available
if ! command -v curl >/dev/null 2>&1; then
    echo "[!] Error: 'curl' is required but not installed." >&2
    exit 1
fi

mkdir -p "${VENDOR_DIR}"

log_info() {
    echo " [>] $*"
}

log_success() {
    echo " [+] $*"
}

log_warn() {
    echo " [!] $*" >&2
}

# Download an asset with primary + fallback CDN and minimum size check
fetch_vendor_asset() {
    local filename="$1"
    local primary_url="$2"
    local fallback_url="$3"
    local min_bytes="$4"
    local dest_tmp="${TMP_DIR}/${filename}"
    local dest_final="${VENDOR_DIR}/${filename}"

    log_info "Downloading ${filename}..."

    # Attempt primary URL
    if curl -fsSL --connect-timeout 10 --max-time 60 "${primary_url}" -o "${dest_tmp}"; then
        log_info "  Fetched from primary CDN"
    else
        log_warn "  Primary CDN failed, trying fallback: ${fallback_url}"
        if curl -fsSL --connect-timeout 10 --max-time 60 "${fallback_url}" -o "${dest_tmp}"; then
            log_info "  Fetched from fallback CDN"
        else
            log_warn "  Failed to download ${filename} from both CDNs."
            return 1
        fi
    fi

    # Validate file existence and minimum size
    if [[ ! -s "${dest_tmp}" ]]; then
        log_warn "  Downloaded file ${filename} is empty."
        return 1
    fi

    local size
    size=$(wc -c < "${dest_tmp}")
    if [[ ${size} -lt ${min_bytes} ]]; then
        log_warn "  Downloaded file ${filename} size (${size} bytes) is under minimum threshold (${min_bytes} bytes)."
        return 1
    fi

    # Atomic move to vendor directory
    mv -f "${dest_tmp}" "${dest_final}"
    local readable_size
    readable_size=$(ls -lh "${dest_final}" | awk '{print $5}')
    log_success "Updated ${filename} (${readable_size})"
}

echo "============================================================="
echo " Termino Vendor Asset Puller"
echo " Destination: ${VENDOR_DIR}"
echo "============================================================="

# 1. HTMX (htmx.min.js)
fetch_vendor_asset \
    "htmx.min.js" \
    "https://unpkg.com/htmx.org@latest/dist/htmx.min.js" \
    "https://cdn.jsdelivr.net/npm/htmx.org@latest/dist/htmx.min.js" \
    20000

# 2. Xterm.js Core (xterm.js)
fetch_vendor_asset \
    "xterm.js" \
    "https://unpkg.com/@xterm/xterm@latest/lib/xterm.js" \
    "https://cdn.jsdelivr.net/npm/@xterm/xterm@latest/lib/xterm.js" \
    100000

# 3. Xterm.js CSS (xterm.css)
fetch_vendor_asset \
    "xterm.css" \
    "https://unpkg.com/@xterm/xterm@latest/css/xterm.css" \
    "https://cdn.jsdelivr.net/npm/@xterm/xterm@latest/css/xterm.css" \
    2000

# 4. Xterm.js Fit Addon (xterm-addon-fit.js)
fetch_vendor_asset \
    "xterm-addon-fit.js" \
    "https://unpkg.com/@xterm/addon-fit@latest/lib/addon-fit.js" \
    "https://cdn.jsdelivr.net/npm/@xterm/addon-fit@latest/lib/addon-fit.js" \
    1000

echo "============================================================="
echo " All vendor assets updated successfully in:"
echo " ${VENDOR_DIR}"
ls -lh "${VENDOR_DIR}"
echo "============================================================="
