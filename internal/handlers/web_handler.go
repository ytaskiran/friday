package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"friday/internal/models"
	"friday/internal/whatsapp"
)

// WebHandler handles all web page rendering.
//
type WebHandler struct {
	draftRepo  *models.DraftRepository
	attrRepo   *models.AttributeRepository
	waClient   *whatsapp.Client
}

func NewWebHandler(draftRepo *models.DraftRepository, attrRepo *models.AttributeRepository, waClient *whatsapp.Client) *WebHandler {
	return &WebHandler{
		draftRepo: draftRepo,
		attrRepo:  attrRepo,
		waClient:  waClient,
	}
}

const sharedHead = `
<script src="https://cdn.tailwindcss.com"></script>
<script>
tailwind.config = {
    theme: {
        extend: {
            colors: {
                whatsapp: {
                    50: '#e8f8ef',
                    100: '#d1f1df',
                    500: '#25D366',
                    600: '#1ebe5d',
                    700: '#128C7E'
                }
            }
        }
    }
}
</script>
<style>
    @keyframes spin { to { transform: rotate(360deg); } }
    @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
    @keyframes slideIn { from { transform: translateX(100%); opacity: 0; } to { transform: translateX(0); opacity: 1; } }
    @keyframes slideOut { from { transform: translateX(0); opacity: 1; } to { transform: translateX(100%); opacity: 0; } }
    .animate-spin { animation: spin 1s linear infinite; }
    .animate-pulse-slow { animation: pulse 2s ease-in-out infinite; }
    .toast-enter { animation: slideIn 0.3s ease-out; }
    .toast-exit { animation: slideOut 0.2s ease-in forwards; }
</style>
`

const toastScript = `
const Toast = {
    container: null,
    init() {
        if (!this.container) {
            this.container = document.createElement('div');
            this.container.className = 'fixed top-4 right-4 z-50 flex flex-col gap-3 max-w-sm';
            document.body.appendChild(this.container);
        }
    },
    show(message, type = 'info', duration = 4000) {
        this.init();
        const colors = {
            success: 'border-green-500 bg-green-50',
            error: 'border-red-500 bg-red-50',
            warning: 'border-amber-500 bg-amber-50',
            info: 'border-blue-500 bg-blue-50'
        };
        const icons = {
            success: '<svg class="w-5 h-5 text-green-500" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/></svg>',
            error: '<svg class="w-5 h-5 text-red-500" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/></svg>',
            warning: '<svg class="w-5 h-5 text-amber-500" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/></svg>',
            info: '<svg class="w-5 h-5 text-blue-500" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"/></svg>'
        };
        const toast = document.createElement('div');
        toast.className = 'flex items-start gap-3 p-4 bg-white rounded-lg shadow-lg border-l-4 ' + colors[type] + ' toast-enter';
        toast.innerHTML = icons[type] + '<p class="text-sm text-gray-700 flex-1">' + message + '</p>' +
            '<button onclick="Toast.dismiss(this.parentElement)" class="text-gray-400 hover:text-gray-600"><svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd"/></svg></button>';
        this.container.appendChild(toast);
        if (duration > 0) setTimeout(() => this.dismiss(toast), duration);
        return toast;
    },
    dismiss(toast) {
        if (toast && toast.parentElement) {
            toast.classList.remove('toast-enter');
            toast.classList.add('toast-exit');
            setTimeout(() => toast.remove(), 200);
        }
    },
    success(msg, dur) { return this.show(msg, 'success', dur); },
    error(msg, dur) { return this.show(msg, 'error', dur); },
    warning(msg, dur) { return this.show(msg, 'warning', dur); },
    info(msg, dur) { return this.show(msg, 'info', dur); }
};
`

func (h *WebHandler) HandleLandingPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Friday - WhatsApp API</title>
    ` + sharedHead + `
</head>
<body class="min-h-screen bg-gray-50">
    <!-- Hero Section -->
    <div class="bg-gradient-to-br from-whatsapp-500 to-whatsapp-700 text-white">
        <div class="max-w-4xl mx-auto px-4 py-16 sm:py-24 text-center">
            <h1 class="text-4xl sm:text-5xl font-bold tracking-tight mb-4">Friday</h1>
            <p class="text-lg sm:text-xl text-white/90">WhatsApp API Server for Developers</p>
        </div>
    </div>

    <!-- Main Content -->
    <div class="max-w-2xl mx-auto px-4 -mt-12">
        <!-- Status Card -->
        <div class="bg-white rounded-xl shadow-xl p-8 mb-8">
            <div class="text-center">
                <!-- Status Icon -->
                <div id="status-icon" class="w-16 h-16 rounded-full bg-whatsapp-50 flex items-center justify-center mx-auto mb-4">
                    <svg class="w-8 h-8 text-whatsapp-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                    </svg>
                </div>

                <h2 id="status-title" class="text-xl font-semibold text-gray-900 mb-2">Ready to Connect</h2>
                <p id="status-message" class="text-gray-500 mb-8">Connect your WhatsApp account to start sending messages via API.</p>

                <!-- Action Button -->
                <button id="action-btn" onclick="handleAction()"
                    class="inline-flex items-center justify-center gap-2 px-6 py-3 bg-whatsapp-500 text-white font-medium rounded-lg hover:bg-whatsapp-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
                    <span id="btn-text">Connect WhatsApp</span>
                    <svg id="btn-spinner" class="w-5 h-5 animate-spin hidden" fill="none" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                </button>
            </div>
        </div>

        <!-- Features Grid -->
        <div class="grid grid-cols-2 gap-4 mb-8">
            <div class="bg-white rounded-lg p-4 shadow-sm border border-gray-100">
                <div class="w-10 h-10 rounded-lg bg-whatsapp-50 flex items-center justify-center mb-3">
                    <svg class="w-5 h-5 text-whatsapp-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z"/>
                    </svg>
                </div>
                <h3 class="font-medium text-gray-900 text-sm">QR Code Auth</h3>
                <p class="text-xs text-gray-500 mt-1">Scan once, stay connected</p>
            </div>

            <div class="bg-white rounded-lg p-4 shadow-sm border border-gray-100">
                <div class="w-10 h-10 rounded-lg bg-whatsapp-50 flex items-center justify-center mb-3">
                    <svg class="w-5 h-5 text-whatsapp-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                    </svg>
                </div>
                <h3 class="font-medium text-gray-900 text-sm">REST API</h3>
                <p class="text-xs text-gray-500 mt-1">Simple HTTP endpoints</p>
            </div>

            <div class="bg-white rounded-lg p-4 shadow-sm border border-gray-100">
                <div class="w-10 h-10 rounded-lg bg-whatsapp-50 flex items-center justify-center mb-3">
                    <svg class="w-5 h-5 text-whatsapp-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                    </svg>
                </div>
                <h3 class="font-medium text-gray-900 text-sm">Contact Sync</h3>
                <p class="text-xs text-gray-500 mt-1">Access your contacts</p>
            </div>

            <div class="bg-white rounded-lg p-4 shadow-sm border border-gray-100">
                <div class="w-10 h-10 rounded-lg bg-whatsapp-50 flex items-center justify-center mb-3">
                    <svg class="w-5 h-5 text-whatsapp-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/>
                    </svg>
                </div>
                <h3 class="font-medium text-gray-900 text-sm">Secure</h3>
                <p class="text-xs text-gray-500 mt-1">Local session storage</p>
            </div>
        </div>

        <!-- Footer -->
        <div class="text-center text-xs text-gray-400 pb-8">
            <p>Use responsibly. Not affiliated with WhatsApp Inc.</p>
        </div>
    </div>

    <script>
    ` + toastScript + `

    let isConnecting = false;
    let isConnected = false;
    let statusInterval;

    async function checkStatus() {
        try {
            const response = await fetch('/api/whatsapp/status');
            const data = await response.json();

            if (data.connected) {
                isConnected = true;
                document.getElementById('status-title').textContent = 'Connected';
                document.getElementById('status-message').textContent = 'WhatsApp is ready. Access the dashboard to send messages.';
                document.getElementById('btn-text').textContent = 'Go to Dashboard';
                document.getElementById('status-icon').innerHTML = '<svg class="w-8 h-8 text-whatsapp-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>';
            }
        } catch (error) {
            console.error('Status check failed:', error);
        }
    }

    async function handleAction() {
        if (isConnected) {
            window.location.href = '/dashboard';
            return;
        }

        const btn = document.getElementById('action-btn');
        const btnText = document.getElementById('btn-text');
        const spinner = document.getElementById('btn-spinner');

        btn.disabled = true;
        btnText.textContent = 'Connecting...';
        spinner.classList.remove('hidden');

        try {
            const response = await fetch('/api/whatsapp/connect', { method: 'POST' });
            const data = await response.json();

            if (data.success) {
                // Check if already connected (existing session restored immediately)
                if (data.message && data.message.includes('Already connected')) {
                    Toast.success('Connected! Going to dashboard...');
                    setTimeout(() => window.location.href = '/dashboard', 1000);
                    return;
                }

                // Connection initiated - poll status to check session restoration
                btnText.textContent = 'Checking session...';

                let attempts = 0;
                const maxAttemptsWithSession = 10; // 15 seconds for session restoration
                const maxAttemptsNoSession = 3;    // 4.5 seconds if no session

                const checkConnection = async () => {
                    attempts++;
                    try {
                        const statusResp = await fetch('/api/whatsapp/status');
                        const statusData = await statusResp.json();

                        if (statusData.connected) {
                            // Fully connected! Go to dashboard
                            Toast.success('Connected! Going to dashboard...');
                            window.location.href = '/dashboard';
                            return;
                        }

                        const maxAttempts = statusData.has_session ? maxAttemptsWithSession : maxAttemptsNoSession;

                        if (statusData.has_session) {
                            if (statusData.connecting) {
                                btnText.textContent = 'Restoring session...';
                            } else {
                                btnText.textContent = 'Session found, connecting...';
                            }
                        }

                        if (attempts < maxAttempts) {
                            setTimeout(checkConnection, 1500);
                        } else {
                            if (statusData.has_session) {
                                // Session exists but couldn't connect - might be stale
                                Toast.warning('Session may be stale. Try disconnecting from your phone first.');
                                btn.disabled = false;
                                btnText.textContent = 'Try Again';
                                spinner.classList.add('hidden');
                            } else {
                                // No session - need QR code
                                Toast.info('QR code required. Redirecting...');
                                window.location.href = '/qr-scan';
                            }
                        }
                    } catch (e) {
                        console.log('Status check failed:', e);
                        if (attempts < 5) {
                            setTimeout(checkConnection, 1500);
                        }
                    }
                };

                // Start checking after a short delay
                setTimeout(checkConnection, 1500);
            } else {
                throw new Error(data.message || 'Connection failed');
            }
        } catch (error) {
            Toast.error('Failed to connect: ' + error.message);
            btn.disabled = false;
            btnText.textContent = 'Connect WhatsApp';
            spinner.classList.add('hidden');
        }
    }

    // Check initial status
    checkStatus();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func (h *WebHandler) HandleQRScanPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Scan QR Code - Friday</title>
    ` + sharedHead + `
</head>
<body class="min-h-screen bg-gray-50">
    <!-- Header -->
    <div class="bg-white border-b border-gray-200">
        <div class="max-w-2xl mx-auto px-4 py-4 flex items-center justify-between">
            <a href="/" class="text-gray-500 hover:text-gray-700 flex items-center gap-2">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"/>
                </svg>
                <span class="text-sm">Back</span>
            </a>
            <h1 class="font-semibold text-gray-900">Scan QR Code</h1>
            <div class="w-16"></div>
        </div>
    </div>

    <!-- Main Content -->
    <div class="max-w-md mx-auto px-4 py-8">
        <!-- Timer -->
        <div id="timer-container" class="mb-6">
            <div class="flex items-center justify-center gap-4">
                <!-- Circular Progress -->
                <div class="relative w-16 h-16">
                    <svg class="w-16 h-16 transform -rotate-90">
                        <circle cx="32" cy="32" r="28" fill="none" stroke="#e5e7eb" stroke-width="4"/>
                        <circle id="progress-ring" cx="32" cy="32" r="28" fill="none" stroke="#25D366" stroke-width="4"
                            stroke-dasharray="175.93" stroke-dashoffset="0" stroke-linecap="round" class="transition-all duration-1000"/>
                    </svg>
                    <span id="countdown" class="absolute inset-0 flex items-center justify-center text-lg font-semibold text-gray-700">60</span>
                </div>
                <div>
                    <p id="timer-label" class="text-sm font-medium text-gray-900">QR Code Active</p>
                    <p class="text-xs text-gray-500">Scan with WhatsApp</p>
                </div>
            </div>
        </div>

        <!-- QR Code Card -->
        <div class="bg-white rounded-xl shadow-lg p-6 mb-6">
            <div id="qr-container" class="relative flex items-center justify-center min-h-[280px]">
                <img id="qr-image" src="/api/whatsapp/qr.png" alt="WhatsApp QR Code"
                     class="w-64 h-64 rounded-lg transition-opacity duration-300"
                     onerror="handleQRError()" onload="handleQRLoad()">

                <!-- Loading Overlay -->
                <div id="qr-loading" class="absolute inset-0 flex items-center justify-center bg-white/80 hidden">
                    <div class="w-8 h-8 border-2 border-gray-200 border-t-whatsapp-500 rounded-full animate-spin"></div>
                </div>
            </div>

            <button onclick="refreshQR()" class="w-full mt-4 px-4 py-2 text-sm font-medium text-whatsapp-600 bg-whatsapp-50 rounded-lg hover:bg-whatsapp-100 transition-colors flex items-center justify-center gap-2">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                </svg>
                Refresh QR Code
            </button>
        </div>

        <!-- Status Message -->
        <div id="status-container" class="hidden mb-6">
            <div id="status-box" class="rounded-lg p-4 flex items-center gap-3">
                <div id="status-icon"></div>
                <p id="status-text" class="text-sm font-medium"></p>
            </div>
        </div>

        <!-- Instructions -->
        <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
            <h3 class="font-medium text-gray-900 mb-4">How to scan</h3>
            <ol class="space-y-3">
                <li class="flex items-start gap-3">
                    <span class="flex-shrink-0 w-6 h-6 rounded-full bg-whatsapp-50 text-whatsapp-600 text-xs font-semibold flex items-center justify-center">1</span>
                    <span class="text-sm text-gray-600">Open <strong>WhatsApp</strong> on your phone</span>
                </li>
                <li class="flex items-start gap-3">
                    <span class="flex-shrink-0 w-6 h-6 rounded-full bg-whatsapp-50 text-whatsapp-600 text-xs font-semibold flex items-center justify-center">2</span>
                    <span class="text-sm text-gray-600">Go to <strong>Settings</strong> → <strong>Linked Devices</strong></span>
                </li>
                <li class="flex items-start gap-3">
                    <span class="flex-shrink-0 w-6 h-6 rounded-full bg-whatsapp-50 text-whatsapp-600 text-xs font-semibold flex items-center justify-center">3</span>
                    <span class="text-sm text-gray-600">Tap <strong>Link a Device</strong> and scan this code</span>
                </li>
            </ol>
        </div>
    </div>

    <script>
    ` + toastScript + `

    let countdown = 60;
    let countdownInterval;
    let statusCheckInterval;
    const circumference = 2 * Math.PI * 28;

    function updateProgress() {
        const offset = circumference - (countdown / 60) * circumference;
        const ring = document.getElementById('progress-ring');
        ring.style.strokeDashoffset = offset;

        // Change color based on time
        if (countdown <= 10) {
            ring.style.stroke = '#ef4444';
        } else if (countdown <= 20) {
            ring.style.stroke = '#f59e0b';
        } else {
            ring.style.stroke = '#25D366';
        }
    }

    function startCountdown() {
        countdown = 60;
        updateProgress();

        clearInterval(countdownInterval);
        countdownInterval = setInterval(() => {
            countdown--;
            document.getElementById('countdown').textContent = countdown;
            updateProgress();

            if (countdown <= 10) {
                document.getElementById('timer-label').textContent = 'Expiring soon';
            }

            if (countdown <= 0) {
                clearInterval(countdownInterval);
                document.getElementById('timer-label').textContent = 'Expired';
                Toast.warning('QR Code expired. Refreshing...');
                setTimeout(() => {
                    fetch('/api/whatsapp/connect', { method: 'POST' }).then(() => refreshQR());
                }, 1000);
            }
        }, 1000);
    }

    function refreshQR() {
        const img = document.getElementById('qr-image');
        const loading = document.getElementById('qr-loading');

        img.style.opacity = '0.3';
        loading.classList.remove('hidden');

        fetch('/api/whatsapp/connect', { method: 'POST' })
            .then(() => {
                const timestamp = new Date().getTime();
                img.src = '/api/whatsapp/qr.png?' + timestamp;
                startCountdown();
                document.getElementById('timer-label').textContent = 'QR Code Active';
            })
            .catch(error => {
                Toast.error('Failed to refresh QR code');
                img.style.opacity = '1';
                loading.classList.add('hidden');
            });
    }

    function handleQRLoad() {
        document.getElementById('qr-image').style.opacity = '1';
        document.getElementById('qr-loading').classList.add('hidden');
    }

    async function handleQRError() {
        // First check if we're already connected (existing session)
        try {
            const statusResp = await fetch('/api/whatsapp/status');
            const statusData = await statusResp.json();

            if (statusData.connected) {
                showStatus('Connected! Redirecting to dashboard...', 'success');
                document.getElementById('timer-container').classList.add('hidden');
                Toast.success('WhatsApp connected successfully!');
                setTimeout(() => window.location.href = '/dashboard', 1500);
                return;
            }
        } catch (e) {
            console.log('Status check failed:', e);
        }

        // Not connected - try to generate QR code
        showStatus('Generating QR code...', 'loading');
        setTimeout(async () => {
            try {
                await fetch('/api/whatsapp/connect', { method: 'POST' });
                // Small delay before reload to let QR generate
                setTimeout(() => location.reload(), 500);
            } catch (e) {
                showStatus('Failed to connect. Please try again.', 'error');
            }
        }, 2000);
    }

    function showStatus(message, type) {
        const container = document.getElementById('status-container');
        const box = document.getElementById('status-box');
        const icon = document.getElementById('status-icon');
        const text = document.getElementById('status-text');

        container.classList.remove('hidden');
        text.textContent = message;

        const configs = {
            loading: { bg: 'bg-blue-50', iconHtml: '<div class="w-5 h-5 border-2 border-blue-200 border-t-blue-500 rounded-full animate-spin"></div>', textColor: 'text-blue-700' },
            success: { bg: 'bg-green-50', iconHtml: '<svg class="w-5 h-5 text-green-500" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/></svg>', textColor: 'text-green-700' },
            error: { bg: 'bg-red-50', iconHtml: '<svg class="w-5 h-5 text-red-500" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/></svg>', textColor: 'text-red-700' }
        };

        const config = configs[type] || configs.loading;
        box.className = 'rounded-lg p-4 flex items-center gap-3 ' + config.bg;
        icon.innerHTML = config.iconHtml;
        text.className = 'text-sm font-medium ' + config.textColor;
    }

    function startStatusMonitoring() {
        statusCheckInterval = setInterval(async () => {
            try {
                const response = await fetch('/api/whatsapp/status');
                const data = await response.json();

                if (data.connected) {
                    clearInterval(statusCheckInterval);
                    clearInterval(countdownInterval);

                    showStatus('Connected! Redirecting to dashboard...', 'success');
                    document.getElementById('timer-container').classList.add('hidden');
                    document.getElementById('qr-container').classList.add('hidden');

                    Toast.success('WhatsApp connected successfully!');
                    setTimeout(() => window.location.href = '/dashboard', 2000);
                }
            } catch (error) {
                console.log('Status check failed:', error);
            }
        }, 2000);
    }

    async function initialize() {
        // Immediately check if already connected
        try {
            const response = await fetch('/api/whatsapp/status');
            const data = await response.json();

            if (data.connected) {
                showStatus('Already connected! Redirecting to dashboard...', 'success');
                document.getElementById('timer-container').classList.add('hidden');
                document.getElementById('qr-container').classList.add('hidden');
                Toast.success('WhatsApp is already connected!');
                setTimeout(() => window.location.href = '/dashboard', 1500);
                return; // Don't start countdown or monitoring
            }
        } catch (e) {
            console.log('Initial status check failed:', e);
        }

        // Not connected - initiate connection to generate QR code
        // This is essential when arriving from a disconnect/session clear
        try {
            await fetch('/api/whatsapp/connect', { method: 'POST' });
        } catch (e) {
            console.log('Connect call failed:', e);
        }

        // Start normal QR flow
        startCountdown();
        startStatusMonitoring();
    }

    initialize();

    // Cleanup
    window.onbeforeunload = function() {
        clearInterval(countdownInterval);
        clearInterval(statusCheckInterval);
    };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func (h *WebHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dashboard - Friday</title>
    ` + sharedHead + `
</head>
<body class="min-h-screen bg-gray-50">
    ` + navComponent + `

    <!-- Main Content -->
    <main class="max-w-6xl mx-auto px-4 py-8">
        <div class="grid lg:grid-cols-3 gap-6">
            <!-- Left Column: Send Message -->
            <div class="lg:col-span-2 space-y-6">
                <!-- Send Message Card -->
                <div class="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
                    <div class="px-6 py-4 border-b border-gray-100">
                        <h2 class="font-semibold text-gray-900">Send Message</h2>
                    </div>
                    <div class="p-6">
                        <div class="space-y-4">
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1.5">Recipient</label>
                                <div class="flex gap-2">
                                    <input type="text" id="recipient" placeholder="Phone number or contact name"
                                        class="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-whatsapp-500 focus:border-whatsapp-500 outline-none transition-shadow">
                                    <button onclick="showContactPicker()" class="px-3 py-2 text-gray-600 bg-gray-100 hover:bg-gray-200 rounded-lg transition-colors" title="Select from contacts">
                                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                                        </svg>
                                    </button>
                                </div>
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1.5">Message</label>
                                <textarea id="message" rows="4" placeholder="Type your message..."
                                    class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-whatsapp-500 focus:border-whatsapp-500 outline-none transition-shadow resize-none">Hello from Friday!</textarea>
                            </div>
                            <button onclick="sendMessage()" id="send-btn"
                                class="w-full px-4 py-2.5 bg-whatsapp-500 text-white font-medium rounded-lg hover:bg-whatsapp-600 transition-colors flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed">
                                <span id="send-btn-text">Send Message</span>
                                <svg id="send-btn-spinner" class="w-4 h-4 animate-spin hidden" fill="none" viewBox="0 0 24 24">
                                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                </svg>
                            </button>
                        </div>

                        <!-- Response -->
                        <div id="response-container" class="hidden mt-4">
                            <div class="text-xs font-medium text-gray-500 mb-1.5">Response</div>
                            <pre id="response" class="bg-gray-900 text-gray-100 rounded-lg p-4 text-xs overflow-x-auto"></pre>
                        </div>
                    </div>
                </div>

                <!-- API Reference -->
                <div class="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
                    <button onclick="toggleApiRef()" class="w-full px-6 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors">
                        <h2 class="font-semibold text-gray-900">API Reference</h2>
                        <svg id="api-chevron" class="w-5 h-5 text-gray-400 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
                        </svg>
                    </button>
                    <div id="api-content" class="hidden border-t border-gray-100">
                        <div class="p-6 space-y-3">
                            <div class="flex items-start gap-3 p-3 bg-gray-50 rounded-lg">
                                <span class="px-2 py-0.5 text-xs font-semibold bg-blue-100 text-blue-700 rounded">GET</span>
                                <div>
                                    <code class="text-sm font-medium text-gray-900">/api/whatsapp/status</code>
                                    <p class="text-xs text-gray-500 mt-0.5">Get connection status</p>
                                </div>
                            </div>
                            <div class="flex items-start gap-3 p-3 bg-gray-50 rounded-lg">
                                <span class="px-2 py-0.5 text-xs font-semibold bg-green-100 text-green-700 rounded">POST</span>
                                <div>
                                    <code class="text-sm font-medium text-gray-900">/api/whatsapp/send</code>
                                    <p class="text-xs text-gray-500 mt-0.5">Send a message. Body: {"recipient": "...", "message": "..."}</p>
                                </div>
                            </div>
                            <div class="flex items-start gap-3 p-3 bg-gray-50 rounded-lg">
                                <span class="px-2 py-0.5 text-xs font-semibold bg-blue-100 text-blue-700 rounded">GET</span>
                                <div>
                                    <code class="text-sm font-medium text-gray-900">/api/contacts</code>
                                    <p class="text-xs text-gray-500 mt-0.5">List all contacts</p>
                                </div>
                            </div>
                            <div class="flex items-start gap-3 p-3 bg-gray-50 rounded-lg">
                                <span class="px-2 py-0.5 text-xs font-semibold bg-blue-100 text-blue-700 rounded">GET</span>
                                <div>
                                    <code class="text-sm font-medium text-gray-900">/api/contacts/search?q=</code>
                                    <p class="text-xs text-gray-500 mt-0.5">Search contacts by name or phone</p>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Right Column: Contacts -->
            <div class="space-y-6">
                <div class="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
                    <div class="px-6 py-4 border-b border-gray-100">
                        <h2 class="font-semibold text-gray-900">Contacts</h2>
                    </div>
                    <div class="p-4">
                        <div class="relative mb-4">
                            <input type="text" id="contact-search" placeholder="Search contacts..."
                                class="w-full pl-9 pr-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-whatsapp-500 focus:border-whatsapp-500 outline-none transition-shadow"
                                onkeyup="searchContacts(event)">
                            <svg class="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                            </svg>
                        </div>

                        <div id="contacts-list" class="space-y-1 max-h-96 overflow-y-auto">
                            <div class="text-center py-8 text-gray-500">
                                <svg class="w-12 h-12 mx-auto mb-3 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                                </svg>
                                <p class="text-sm">No contacts loaded</p>
                                <button onclick="loadContacts()" class="mt-2 text-sm text-whatsapp-600 hover:text-whatsapp-700 font-medium">Load Contacts</button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </main>

    <script>
    ` + toastScript + `

    let allContacts = [];
    let apiRefOpen = false;

    function toggleApiRef() {
        apiRefOpen = !apiRefOpen;
        document.getElementById('api-content').classList.toggle('hidden', !apiRefOpen);
        document.getElementById('api-chevron').style.transform = apiRefOpen ? 'rotate(180deg)' : '';
    }

    async function loadContacts() {
        const container = document.getElementById('contacts-list');
        container.innerHTML = '<div class="flex justify-center py-8"><div class="w-6 h-6 border-2 border-gray-200 border-t-whatsapp-500 rounded-full animate-spin"></div></div>';

        try {
            const response = await fetch('/api/contacts');
            const data = await response.json();

            if (data.success) {
                allContacts = data.contacts || [];
                displayContacts(allContacts);
            } else {
                // Show friendly message based on error
                if (data.message && data.message.includes('not connected')) {
                    container.innerHTML = '<div class="text-center py-8 text-gray-500">' +
                        '<svg class="w-12 h-12 mx-auto mb-3 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">' +
                        '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 5.636a9 9 0 010 12.728m-3.536-3.536a4 4 0 010-5.656m-7.072 7.072a9 9 0 010-12.728m3.536 3.536a4 4 0 010 5.656"/>' +
                        '</svg>' +
                        '<p class="text-sm">Connect WhatsApp to view contacts</p>' +
                        '<a href="/" class="mt-2 inline-block text-sm text-whatsapp-600 hover:text-whatsapp-700 font-medium">Go to Connect</a>' +
                        '</div>';
                } else {
                    container.innerHTML = '<p class="text-center py-4 text-red-500 text-sm">Error loading contacts</p>';
                    Toast.error('Failed to load contacts');
                }
            }
        } catch (error) {
            container.innerHTML = '<p class="text-center py-4 text-red-500 text-sm">Error: ' + error.message + '</p>';
            Toast.error('Failed to load contacts');
        }
    }

    function displayContacts(contacts) {
        const container = document.getElementById('contacts-list');

        if (contacts.length === 0) {
            container.innerHTML = '<p class="text-center py-4 text-gray-500 text-sm">No contacts found</p>';
            return;
        }

        container.innerHTML = contacts.map(contact => {
            const name = contact.name || contact.phone;
            const initials = name.charAt(0).toUpperCase();
            const jid = contact.jid || '';
            return '<div class="flex items-center gap-2 p-2 rounded-lg hover:bg-gray-50 transition-colors">' +
                '<button onclick="selectContact(\'' + (contact.name || contact.phone).replace(/'/g, "\\'") + '\')" ' +
                'class="flex-1 flex items-center gap-3 text-left">' +
                '<div class="w-9 h-9 rounded-full bg-whatsapp-50 text-whatsapp-600 flex items-center justify-center text-sm font-medium">' + initials + '</div>' +
                '<div class="flex-1 min-w-0">' +
                '<p class="text-sm font-medium text-gray-900 truncate">' + name + '</p>' +
                '<p class="text-xs text-gray-500">' + contact.phone + '</p>' +
                '</div></button>' +
                '<a href="/contact/' + encodeURIComponent(jid) + '" ' +
                'class="p-1.5 text-gray-400 hover:text-whatsapp-600 hover:bg-whatsapp-50 rounded-lg transition-colors" ' +
                'title="View contact & attributes">' +
                '<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">' +
                '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>' +
                '</svg></a></div>';
        }).join('');
    }

    function searchContacts(event) {
        const query = event.target.value.toLowerCase().trim();

        if (!query) {
            displayContacts(allContacts);
            return;
        }

        const filtered = allContacts.filter(c =>
            (c.name && c.name.toLowerCase().includes(query)) ||
            c.phone.includes(query)
        );
        displayContacts(filtered);
    }

    function selectContact(identifier) {
        document.getElementById('recipient').value = identifier;
        Toast.success('Selected: ' + identifier);
    }

    function showContactPicker() {
        if (allContacts.length === 0) {
            loadContacts();
        }
        document.getElementById('contact-search').focus();
    }

    async function sendMessage() {
        const recipient = document.getElementById('recipient').value.trim();
        const message = document.getElementById('message').value.trim();
        const btn = document.getElementById('send-btn');
        const btnText = document.getElementById('send-btn-text');
        const spinner = document.getElementById('send-btn-spinner');

        if (!recipient || !message) {
            Toast.warning('Please enter recipient and message');
            return;
        }

        btn.disabled = true;
        btnText.textContent = 'Sending...';
        spinner.classList.remove('hidden');

        try {
            const response = await fetch('/api/whatsapp/send', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ recipient, message })
            });

            const data = await response.json();

            document.getElementById('response-container').classList.remove('hidden');
            const responseEl = document.getElementById('response');
            responseEl.textContent = JSON.stringify(data, null, 2);
            responseEl.className = 'rounded-lg p-4 text-xs overflow-x-auto ' +
                (data.success ? 'bg-gray-900 text-green-400' : 'bg-gray-900 text-red-400');

            if (data.success) {
                Toast.success('Message sent successfully!');
            } else {
                Toast.error('Failed to send: ' + data.message);
            }
        } catch (error) {
            Toast.error('Error: ' + error.message);
        } finally {
            btn.disabled = false;
            btnText.textContent = 'Send Message';
            spinner.classList.add('hidden');
        }
    }

    // Load contacts on page load
    loadContacts();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// Navigation component for consistent header across pages
const navComponent = `
<nav class="bg-white shadow-sm border-b border-gray-100 sticky top-0 z-50">
    <div class="max-w-6xl mx-auto px-4">
        <div class="flex justify-between h-14">
            <div class="flex items-center gap-6">
                <a href="/" class="flex items-center gap-2">
                    <div class="w-8 h-8 bg-gradient-to-br from-whatsapp-500 to-whatsapp-700 rounded-lg flex items-center justify-center">
                        <svg class="w-5 h-5 text-white" fill="currentColor" viewBox="0 0 24 24">
                            <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/>
                        </svg>
                    </div>
                    <span class="font-semibold text-gray-900">Friday</span>
                </a>
                <div class="flex items-center gap-1">
                    <a href="/dashboard" class="nav-link px-3 py-2 rounded-lg text-sm font-medium text-gray-600 hover:text-whatsapp-600 hover:bg-whatsapp-50 transition-colors">Dashboard</a>
                    <a href="/contacts" class="nav-link px-3 py-2 rounded-lg text-sm font-medium text-gray-600 hover:text-whatsapp-600 hover:bg-whatsapp-50 transition-colors">Contacts</a>
                    <a href="/groups" class="nav-link px-3 py-2 rounded-lg text-sm font-medium text-gray-600 hover:text-whatsapp-600 hover:bg-whatsapp-50 transition-colors">Groups</a>
                    <a href="/drafts" class="nav-link px-3 py-2 rounded-lg text-sm font-medium text-gray-600 hover:text-whatsapp-600 hover:bg-whatsapp-50 transition-colors">Drafts</a>
                    <a href="/send" class="nav-link px-3 py-2 rounded-lg text-sm font-medium text-gray-600 hover:text-whatsapp-600 hover:bg-whatsapp-50 transition-colors">Send</a>
                    <a href="/batch-runs" class="nav-link px-3 py-2 rounded-lg text-sm font-medium text-gray-600 hover:text-whatsapp-600 hover:bg-whatsapp-50 transition-colors">Batches</a>
                </div>
            </div>
            <div class="flex items-center gap-2">
                <button onclick="refreshStatus()" id="status-indicator" class="flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-sm hover:bg-gray-100 transition-colors" title="Click to refresh status">
                    <span class="w-2 h-2 rounded-full bg-gray-300"></span>
                    <span class="text-gray-500">Checking...</span>
                </button>
                <button onclick="disconnectWhatsApp()" class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors" title="Disconnect WhatsApp">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
                    </svg>
                </button>
            </div>
        </div>
    </div>
</nav>
<script>
// Highlight current nav link
document.querySelectorAll('.nav-link').forEach(link => {
    if (link.getAttribute('href') === window.location.pathname) {
        link.classList.add('text-whatsapp-600', 'bg-whatsapp-50');
        link.classList.remove('text-gray-600');
    }
});

// Track connection state to detect transitions
let wasConnected = null;
let consecutiveDisconnects = 0;

// Update status indicator with auto-redirect on disconnect
async function refreshStatus() {
    const indicator = document.getElementById('status-indicator');

    try {
        const response = await fetch('/api/whatsapp/status');
        const data = await response.json();

        if (data.connected) {
            indicator.innerHTML = '<span class="w-2 h-2 rounded-full bg-green-500"></span><span class="text-green-600">Connected</span>';
            wasConnected = true;
            consecutiveDisconnects = 0;
        } else {
            indicator.innerHTML = '<span class="w-2 h-2 rounded-full bg-red-500"></span><span class="text-red-600">Disconnected</span>';
            consecutiveDisconnects++;

            if (wasConnected === true && consecutiveDisconnects >= 2) {
                indicator.innerHTML = '<span class="w-2 h-2 rounded-full bg-yellow-500 animate-pulse"></span><span class="text-yellow-600">Reconnecting...</span>';
                if (typeof Toast !== 'undefined') Toast.warning('Connection lost. Redirecting...');
                setTimeout(() => {
                    window.location.href = '/';
                }, 1500);
                return;
            }

            wasConnected = false;
        }
    } catch (e) {
        indicator.innerHTML = '<span class="w-2 h-2 rounded-full bg-gray-300"></span><span class="text-gray-500">Unknown</span>';
        consecutiveDisconnects++;

        // Network error could mean server restart - treat as disconnect
        if (wasConnected === true && consecutiveDisconnects >= 2) {
            if (typeof Toast !== 'undefined') Toast.warning('Server connection lost. Redirecting...');
            setTimeout(() => {
                window.location.href = '/';
            }, 1500);
        }
    }
}

async function disconnectWhatsApp() {
    if (!confirm('Are you sure you want to disconnect WhatsApp? You will need to scan a new QR code to reconnect.')) return;

    const indicator = document.getElementById('status-indicator');
    indicator.innerHTML = '<span class="w-2 h-2 rounded-full bg-yellow-500 animate-pulse"></span><span class="text-yellow-600">Disconnecting...</span>';

    try {
        const response = await fetch('/api/whatsapp/disconnect', { method: 'POST' });
        const data = await response.json();
        if (data.success) {
            if (typeof Toast !== 'undefined') Toast.success('Disconnected successfully');
            // Redirect to landing page - user can click Connect to get new QR
            setTimeout(() => {
                window.location.href = '/';
            }, 1000);
        } else {
            if (typeof Toast !== 'undefined') Toast.error(data.message || 'Failed to disconnect');
            refreshStatus();
        }
    } catch (e) {
        if (typeof Toast !== 'undefined') Toast.error('Failed to disconnect');
        refreshStatus();
    }
}

// Initial status check
refreshStatus();

setInterval(refreshStatus, 5000);
</script>
`

// HandleDraftsPage renders the drafts management page
func (h *WebHandler) HandleDraftsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Drafts - Friday</title>
    ` + sharedHead + `
</head>
<body class="min-h-screen bg-gray-50">
    ` + navComponent + `
    <script>` + toastScript + `</script>

    <!-- Main Content -->
    <main class="max-w-6xl mx-auto px-4 py-8">
        <!-- Page Header -->
        <div class="flex items-center justify-between mb-6">
            <div>
                <h1 class="text-2xl font-bold text-gray-900">Drafts</h1>
                <p class="text-gray-500 mt-1">Manage your message templates</p>
            </div>
            <button onclick="showCreateModal()" class="inline-flex items-center gap-2 px-4 py-2 bg-whatsapp-500 text-white text-sm font-medium rounded-lg hover:bg-whatsapp-600 transition-colors">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
                </svg>
                New Draft
            </button>
        </div>
        <!-- Placeholder hint -->
        <div class="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
            <div class="flex gap-3">
                <svg class="w-5 h-5 text-blue-500 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"/>
                </svg>
                <div>
                    <h3 class="font-medium text-blue-800">Using Placeholders</h3>
                    <p class="text-sm text-blue-700 mt-1">
                        Use <code class="bg-blue-100 px-1 rounded">{{name}}</code> syntax in your drafts.
                        Built-in: <code class="bg-blue-100 px-1 rounded">{{name}}</code>, <code class="bg-blue-100 px-1 rounded">{{phone}}</code>, <code class="bg-blue-100 px-1 rounded">{{first_name}}</code>, <code class="bg-blue-100 px-1 rounded">{{full_name}}</code>.
                        Custom: Any attribute you add to a contact.
                    </p>
                </div>
            </div>
        </div>

        <!-- Drafts Grid -->
        <div id="drafts-container" class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            <div class="animate-pulse bg-white rounded-xl p-6 shadow-sm border border-gray-100">
                <div class="h-4 bg-gray-200 rounded w-3/4 mb-4"></div>
                <div class="h-3 bg-gray-200 rounded w-full mb-2"></div>
                <div class="h-3 bg-gray-200 rounded w-5/6"></div>
            </div>
        </div>

        <!-- Empty state (hidden by default) -->
        <div id="empty-state" class="hidden text-center py-12">
            <svg class="w-16 h-16 mx-auto text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            <h3 class="text-lg font-medium text-gray-900 mb-1">No drafts yet</h3>
            <p class="text-gray-500 mb-4">Create your first message template to get started</p>
            <button onclick="showCreateModal()" class="inline-flex items-center gap-2 px-4 py-2 bg-whatsapp-500 text-white rounded-lg hover:bg-whatsapp-600 transition-colors">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
                </svg>
                Create Draft
            </button>
        </div>
    </main>

    <!-- Create/Edit Modal -->
    <div id="draft-modal" class="hidden fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
        <div class="bg-white rounded-xl shadow-xl max-w-lg w-full max-h-[90vh] overflow-y-auto">
            <div class="p-6 border-b border-gray-100">
                <h2 id="modal-title" class="text-lg font-semibold text-gray-900">New Draft</h2>
            </div>
            <form id="draft-form" class="p-6 space-y-4">
                <input type="hidden" id="draft-id" value="">
                <div>
                    <label for="draft-title" class="block text-sm font-medium text-gray-700 mb-1">Title</label>
                    <input type="text" id="draft-title" required
                        class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-whatsapp-500 focus:border-whatsapp-500"
                        placeholder="e.g., Welcome Message">
                </div>
                <div>
                    <label for="draft-content" class="block text-sm font-medium text-gray-700 mb-1">Message Content</label>
                    <textarea id="draft-content" rows="6" required
                        class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-whatsapp-500 focus:border-whatsapp-500 font-mono text-sm"
                        placeholder="Hello {{name}}, welcome to our service!"></textarea>
                </div>
                <div id="placeholders-preview" class="text-sm text-gray-500"></div>
            </form>
            <div class="p-6 border-t border-gray-100 flex justify-end gap-3">
                <button type="button" onclick="hideModal()" class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors">
                    Cancel
                </button>
                <button type="submit" form="draft-form" class="px-4 py-2 bg-whatsapp-500 text-white rounded-lg hover:bg-whatsapp-600 transition-colors">
                    Save Draft
                </button>
            </div>
        </div>
    </div>

    <script>
    let drafts = [];

    async function loadDrafts() {
        try {
            const response = await fetch('/api/drafts');
            const data = await response.json();
            if (data.success) {
                drafts = data.drafts;
                renderDrafts();
            } else {
                Toast.error('Failed to load drafts: ' + data.message);
            }
        } catch (error) {
            Toast.error('Failed to load drafts');
        }
    }

    function renderDrafts() {
        const container = document.getElementById('drafts-container');
        const emptyState = document.getElementById('empty-state');

        if (drafts.length === 0) {
            container.classList.add('hidden');
            emptyState.classList.remove('hidden');
            return;
        }

        emptyState.classList.add('hidden');
        container.classList.remove('hidden');

        container.innerHTML = drafts.map(draft => {
            const placeholders = extractPlaceholders(draft.content);
            const preview = draft.content.length > 100 ? draft.content.substring(0, 100) + '...' : draft.content;
            return ` + "`" + `
                <div class="bg-white rounded-xl p-6 shadow-sm border border-gray-100 hover:border-whatsapp-200 transition-colors">
                    <div class="flex justify-between items-start mb-3">
                        <h3 class="font-semibold text-gray-900">${escapeHtml(draft.title)}</h3>
                        <div class="flex gap-1">
                            <button onclick="editDraft(${draft.id})" class="p-1.5 text-gray-400 hover:text-whatsapp-600 hover:bg-whatsapp-50 rounded transition-colors" title="Edit">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                                </svg>
                            </button>
                            <button onclick="deleteDraft(${draft.id})" class="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors" title="Delete">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                                </svg>
                            </button>
                        </div>
                    </div>
                    <p class="text-sm text-gray-600 mb-3 whitespace-pre-wrap">${escapeHtml(preview)}</p>
                    ${placeholders.length > 0 ? ` + "`" + `
                        <div class="flex flex-wrap gap-1.5 mb-3">
                            ${placeholders.map(p => ` + "`" + `<span class="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs rounded-full">{{${p}}}</span>` + "`" + `).join('')}
                        </div>
                    ` + "`" + ` : ''}
                    <a href="/send?draft=${draft.id}" class="inline-flex items-center gap-1.5 text-sm text-whatsapp-600 hover:text-whatsapp-700 font-medium">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/>
                        </svg>
                        Use this draft
                    </a>
                </div>
            ` + "`" + `;
        }).join('');
    }

    function extractPlaceholders(content) {
        const matches = content.match(/\{\{(\w+)\}\}/g) || [];
        const unique = [...new Set(matches.map(m => m.slice(2, -2)))];
        return unique;
    }

    function showCreateModal() {
        document.getElementById('modal-title').textContent = 'New Draft';
        document.getElementById('draft-id').value = '';
        document.getElementById('draft-title').value = '';
        document.getElementById('draft-content').value = '';
        document.getElementById('placeholders-preview').innerHTML = '';
        document.getElementById('draft-modal').classList.remove('hidden');
    }

    function editDraft(id) {
        const draft = drafts.find(d => d.id === id);
        if (!draft) return;

        document.getElementById('modal-title').textContent = 'Edit Draft';
        document.getElementById('draft-id').value = draft.id;
        document.getElementById('draft-title').value = draft.title;
        document.getElementById('draft-content').value = draft.content;
        updatePlaceholdersPreview();
        document.getElementById('draft-modal').classList.remove('hidden');
    }

    function hideModal() {
        document.getElementById('draft-modal').classList.add('hidden');
    }

    async function deleteDraft(id) {
        if (!confirm('Delete this draft?')) return;

        try {
            const response = await fetch('/api/drafts/' + id, { method: 'DELETE' });
            const data = await response.json();
            if (data.success) {
                Toast.success('Draft deleted');
                loadDrafts();
            } else {
                Toast.error('Failed to delete: ' + data.message);
            }
        } catch (error) {
            Toast.error('Failed to delete draft');
        }
    }

    document.getElementById('draft-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const id = document.getElementById('draft-id').value;
        const title = document.getElementById('draft-title').value.trim();
        const content = document.getElementById('draft-content').value;

        if (!title || !content) {
            Toast.error('Title and content are required');
            return;
        }

        try {
            const isEdit = id !== '';
            const response = await fetch('/api/drafts' + (isEdit ? '/' + id : ''), {
                method: isEdit ? 'PUT' : 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title, content })
            });
            const data = await response.json();
            if (data.success) {
                Toast.success(isEdit ? 'Draft updated' : 'Draft created');
                hideModal();
                loadDrafts();
            } else {
                Toast.error('Failed to save: ' + data.message);
            }
        } catch (error) {
            Toast.error('Failed to save draft');
        }
    });

    // Live placeholder detection
    document.getElementById('draft-content').addEventListener('input', updatePlaceholdersPreview);

    function updatePlaceholdersPreview() {
        const content = document.getElementById('draft-content').value;
        const placeholders = extractPlaceholders(content);
        const preview = document.getElementById('placeholders-preview');

        if (placeholders.length > 0) {
            preview.innerHTML = 'Placeholders found: ' + placeholders.map(p => '<code class="bg-gray-100 px-1 rounded">{{' + p + '}}</code>').join(', ');
        } else {
            preview.innerHTML = '';
        }
    }

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Close modal on escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') hideModal();
    });

    // Close modal on backdrop click
    document.getElementById('draft-modal').addEventListener('click', (e) => {
        if (e.target.id === 'draft-modal') hideModal();
    });

    loadDrafts();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// HandleDraftEditPage handles /drafts/{id} for viewing/editing a specific draft
func (h *WebHandler) HandleDraftEditPage(w http.ResponseWriter, r *http.Request) {
	// This could redirect to the drafts page with the edit modal open
	// For simplicity, we'll redirect to /drafts
	http.Redirect(w, r, "/drafts", http.StatusTemporaryRedirect)
}

// HandleContactDetailPage renders the contact detail page with attributes
func (h *WebHandler) HandleContactDetailPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract JID from path: /contact/{jid}
	jid := strings.TrimPrefix(r.URL.Path, "/contact/")
	if jid == "" {
		http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Contact Details - Friday</title>
    ` + sharedHead + `
</head>
<body class="bg-gray-50 min-h-screen">
    ` + navComponent + `
    <script>` + toastScript + `</script>

    <main class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <!-- Contact Info Card -->
        <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6 mb-6">
            <div class="flex items-start gap-4">
                <div class="w-16 h-16 bg-gradient-to-br from-whatsapp-500 to-whatsapp-700 rounded-full flex items-center justify-center text-white text-2xl font-semibold" id="contact-avatar">
                    ?
                </div>
                <div class="flex-1">
                    <h1 class="text-xl font-bold text-gray-900" id="contact-name">Loading...</h1>
                    <p class="text-gray-500" id="contact-phone"></p>
                    <p class="text-sm text-gray-400 font-mono mt-1" id="contact-jid"></p>
                </div>
                <a id="send-link" href="/send" class="inline-flex items-center gap-2 px-4 py-2 bg-whatsapp-500 text-white rounded-lg hover:bg-whatsapp-600 transition-colors">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/>
                    </svg>
                    Send Message
                </a>
            </div>
        </div>

        <!-- Attributes Section -->
        <div class="bg-white rounded-xl shadow-sm border border-gray-100">
            <div class="p-6 border-b border-gray-100">
                <div class="flex justify-between items-center">
                    <div>
                        <h2 class="text-lg font-semibold text-gray-900">Custom Attributes</h2>
                        <p class="text-sm text-gray-500">Add custom values to use in message placeholders</p>
                    </div>
                    <button onclick="showAddAttribute()" class="inline-flex items-center gap-1 px-3 py-1.5 text-sm bg-whatsapp-50 text-whatsapp-600 rounded-lg hover:bg-whatsapp-100 transition-colors">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
                        </svg>
                        Add Attribute
                    </button>
                </div>
            </div>

            <!-- Add Attribute Form (hidden by default) -->
            <div id="add-form" class="hidden p-6 bg-gray-50 border-b border-gray-100">
                <form onsubmit="saveAttribute(event)" class="flex gap-3">
                    <input type="text" id="attr-key" placeholder="Key (e.g., company)" required
                        pattern="[a-zA-Z0-9_]+"
                        class="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-whatsapp-500 focus:border-whatsapp-500">
                    <input type="text" id="attr-value" placeholder="Value" required
                        class="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-whatsapp-500 focus:border-whatsapp-500">
                    <button type="submit" class="px-4 py-2 bg-whatsapp-500 text-white rounded-lg hover:bg-whatsapp-600 transition-colors">
                        Save
                    </button>
                    <button type="button" onclick="hideAddAttribute()" class="px-4 py-2 text-gray-600 hover:bg-gray-100 rounded-lg transition-colors">
                        Cancel
                    </button>
                </form>
            </div>

            <!-- Attributes List -->
            <div id="attributes-list" class="divide-y divide-gray-100">
                <div class="p-6 text-center text-gray-500 animate-pulse">Loading attributes...</div>
            </div>

            <!-- Empty state -->
            <div id="empty-attrs" class="hidden p-8 text-center">
                <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"/>
                </svg>
                <p class="text-gray-500">No custom attributes yet</p>
                <p class="text-sm text-gray-400 mt-1">Add attributes to personalize messages with {{placeholders}}</p>
            </div>
        </div>
    </main>

    <script>
    const contactJid = decodeURIComponent('` + jid + `');
    let contact = null;
    let attributes = [];
    let editingKey = null; // Track which attribute is being edited (null = adding new)

    async function loadContact() {
        try {
            const response = await fetch('/api/contacts');
            const data = await response.json();
            if (data.success) {
                contact = data.contacts.find(c => c.jid === contactJid);
                if (contact) {
                    renderContact();
                } else {
                    document.getElementById('contact-name').textContent = 'Contact not found';
                }
            }
        } catch (error) {
            console.error('Failed to load contact:', error);
        }
    }

    function renderContact() {
        const initial = (contact.name || contact.phone || '?')[0].toUpperCase();
        document.getElementById('contact-avatar').textContent = initial;
        document.getElementById('contact-name').textContent = contact.name || contact.phone;
        document.getElementById('contact-phone').textContent = contact.phone;
        document.getElementById('contact-jid').textContent = contact.jid;
        document.getElementById('send-link').href = '/send?jid=' + encodeURIComponent(contact.jid);
    }

    async function loadAttributes() {
        try {
            const response = await fetch('/api/contacts/' + encodeURIComponent(contactJid) + '/attributes');
            const data = await response.json();
            if (data.success) {
                attributes = data.attributes || [];
                renderAttributes();
            }
        } catch (error) {
            console.error('Failed to load attributes:', error);
        }
    }

    function renderAttributes() {
        const list = document.getElementById('attributes-list');
        const empty = document.getElementById('empty-attrs');

        if (attributes.length === 0) {
            list.classList.add('hidden');
            empty.classList.remove('hidden');
            return;
        }

        empty.classList.add('hidden');
        list.classList.remove('hidden');

        list.innerHTML = attributes.map(attr => ` + "`" + `
            <div class="p-4 flex items-center justify-between hover:bg-gray-50">
                <div>
                    <code class="text-sm bg-gray-100 px-2 py-0.5 rounded text-gray-700">{{${attr.key}}}</code>
                    <span class="mx-2 text-gray-400">=</span>
                    <span class="text-gray-900">${escapeHtml(attr.value)}</span>
                </div>
                <div class="flex items-center gap-1">
                    <button onclick="editAttribute('${attr.key}', '${escapeHtml(attr.value).replace(/'/g, "\\'")}')" class="p-1.5 text-gray-400 hover:text-whatsapp-600 hover:bg-whatsapp-50 rounded transition-colors" title="Edit">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                        </svg>
                    </button>
                    <button onclick="deleteAttribute('${attr.key}')" class="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors" title="Delete">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                        </svg>
                    </button>
                </div>
            </div>
        ` + "`" + `).join('');
    }

    function showAddAttribute() {
        editingKey = null;
        const keyInput = document.getElementById('attr-key');
        keyInput.value = '';
        keyInput.readOnly = false;
        keyInput.classList.remove('bg-gray-100', 'cursor-not-allowed');
        document.getElementById('attr-value').value = '';
        document.getElementById('add-form').classList.remove('hidden');
        keyInput.focus();
    }

    function editAttribute(key, value) {
        editingKey = key;
        const keyInput = document.getElementById('attr-key');
        keyInput.value = key;
        keyInput.readOnly = true;
        keyInput.classList.add('bg-gray-100', 'cursor-not-allowed');
        document.getElementById('attr-value').value = value;
        document.getElementById('add-form').classList.remove('hidden');
        document.getElementById('attr-value').focus();
    }

    function hideAddAttribute() {
        document.getElementById('add-form').classList.add('hidden');
        editingKey = null;
        const keyInput = document.getElementById('attr-key');
        keyInput.value = '';
        keyInput.readOnly = false;
        keyInput.classList.remove('bg-gray-100', 'cursor-not-allowed');
        document.getElementById('attr-value').value = '';
    }

    async function saveAttribute(e) {
        e.preventDefault();
        const key = document.getElementById('attr-key').value.trim();
        const value = document.getElementById('attr-value').value.trim();

        if (!key || !value) {
            Toast.error('Key and value are required');
            return;
        }

        try {
            const response = await fetch('/api/contacts/' + encodeURIComponent(contactJid) + '/attributes', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ key, value })
            });
            const data = await response.json();
            if (data.success) {
                Toast.success(editingKey ? 'Attribute updated' : 'Attribute saved');
                hideAddAttribute();
                loadAttributes();
            } else {
                Toast.error('Failed to save: ' + data.message);
            }
        } catch (error) {
            Toast.error(editingKey ? 'Failed to update attribute' : 'Failed to save attribute');
        }
    }

    async function deleteAttribute(key) {
        if (!confirm('Delete attribute "' + key + '"?')) return;

        try {
            const response = await fetch('/api/contacts/' + encodeURIComponent(contactJid) + '/attributes/' + encodeURIComponent(key), {
                method: 'DELETE'
            });
            const data = await response.json();
            if (data.success) {
                Toast.success('Attribute deleted');
                loadAttributes();
            } else {
                Toast.error('Failed to delete: ' + data.message);
            }
        } catch (error) {
            Toast.error('Failed to delete attribute');
        }
    }

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    loadContact();
    loadAttributes();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// HandleSendPage renders the send-with-draft page with live preview
func (h *WebHandler) HandleSendPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Send Message - Friday</title>
    ` + sharedHead + `
</head>
<body class="bg-gray-50 min-h-screen">
    ` + navComponent + `
    <script>` + toastScript + `</script>

    <main class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div class="mb-8">
            <h1 class="text-2xl font-bold text-gray-900">Send Message</h1>
            <p class="text-gray-500 mt-1">Send a personalized message using a draft template</p>
        </div>

        <div class="grid md:grid-cols-2 gap-6">
            <!-- Left: Form -->
            <div class="space-y-6">
                <!-- Draft Selection -->
                <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
                    <label class="block text-sm font-medium text-gray-700 mb-2">Select Draft</label>
                    <select id="draft-select" onchange="onDraftChange()" class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-whatsapp-500 focus:border-whatsapp-500">
                        <option value="">-- Choose a draft --</option>
                    </select>
                    <div id="draft-content-preview" class="mt-3 p-3 bg-gray-50 rounded-lg text-sm text-gray-600 whitespace-pre-wrap hidden"></div>
                </div>

                <!-- Recipient Selection -->
                <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
                    <label class="block text-sm font-medium text-gray-700 mb-3">Select Recipient</label>

                    <!-- Mode Toggle -->
                    <div class="flex rounded-lg bg-gray-100 p-1 mb-4">
                        <button type="button" id="mode-contact" onclick="setMode('contact')"
                            class="flex-1 py-2 px-3 text-sm font-medium rounded-md transition-colors bg-white text-gray-900 shadow-sm">
                            <span class="flex items-center justify-center gap-2">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                                </svg>
                                Single Contact
                            </span>
                        </button>
                        <button type="button" id="mode-group" onclick="setMode('group')"
                            class="flex-1 py-2 px-3 text-sm font-medium rounded-md transition-colors text-gray-500 hover:text-gray-700">
                            <span class="flex items-center justify-center gap-2">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                                </svg>
                                Contact Group
                            </span>
                        </button>
                    </div>

                    <!-- Contact Selection (Single Mode) -->
                    <div id="contact-selection">
                        <div class="relative">
                            <input type="text" id="contact-search" placeholder="Search contacts..." oninput="searchContacts()" onfocus="showContactDropdown()"
                                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-whatsapp-500 focus:border-whatsapp-500">
                            <div id="contact-dropdown" class="absolute z-10 mt-1 w-full bg-white border border-gray-200 rounded-lg shadow-lg max-h-48 overflow-y-auto hidden"></div>
                        </div>
                        <div id="selected-contact" class="mt-3 hidden">
                            <div class="flex items-center gap-3 p-3 bg-whatsapp-50 rounded-lg">
                                <div class="w-10 h-10 bg-whatsapp-500 rounded-full flex items-center justify-center text-white font-semibold" id="selected-avatar">?</div>
                                <div class="flex-1">
                                    <div class="font-medium text-gray-900" id="selected-name"></div>
                                    <div class="text-sm text-gray-500" id="selected-phone"></div>
                                </div>
                                <button onclick="clearContact()" class="p-1 text-gray-400 hover:text-gray-600">
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                                    </svg>
                                </button>
                            </div>
                            <a id="edit-attrs-link" href="#" class="inline-flex items-center gap-1 text-sm text-whatsapp-600 hover:text-whatsapp-700 mt-2">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                                </svg>
                                Edit contact attributes
                            </a>
                        </div>
                    </div>

                    <!-- Group Selection (Batch Mode) -->
                    <div id="group-selection" class="hidden">
                        <div class="relative">
                            <input type="text" id="group-search" placeholder="Search contact groups..." oninput="searchGroups()" onfocus="showGroupDropdown()"
                                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-whatsapp-500 focus:border-whatsapp-500">
                            <div id="group-dropdown" class="absolute z-10 mt-1 w-full bg-white border border-gray-200 rounded-lg shadow-lg max-h-48 overflow-y-auto hidden"></div>
                        </div>
                        <div id="selected-group" class="mt-3 hidden">
                            <div class="flex items-center gap-3 p-3 bg-blue-50 rounded-lg">
                                <div class="w-10 h-10 bg-blue-500 rounded-full flex items-center justify-center text-white">
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                                    </svg>
                                </div>
                                <div class="flex-1">
                                    <div class="font-medium text-gray-900" id="selected-group-name"></div>
                                    <div class="text-sm text-gray-500" id="selected-group-count"></div>
                                </div>
                                <button onclick="clearGroup()" class="p-1 text-gray-400 hover:text-gray-600">
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                                    </svg>
                                </button>
                            </div>
                            <a id="edit-group-link" href="#" class="inline-flex items-center gap-1 text-sm text-blue-600 hover:text-blue-700 mt-2">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                                </svg>
                                Edit group members
                            </a>
                        </div>
                        <p id="no-groups-msg" class="mt-3 text-sm text-gray-500 hidden">
                            No contact groups found. <a href="/groups" class="text-whatsapp-600 hover:text-whatsapp-700">Create one</a>
                        </p>
                    </div>
                </div>

                <!-- Send Button -->
                <button onclick="sendMessage()" id="send-btn" disabled
                    class="w-full py-3 bg-whatsapp-500 text-white rounded-lg font-medium hover:bg-whatsapp-600 transition-colors disabled:bg-gray-300 disabled:cursor-not-allowed flex items-center justify-center gap-2">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/>
                    </svg>
                    Send Message
                </button>
            </div>

            <!-- Right: Preview -->
            <div class="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
                <div class="p-4 bg-gray-50 border-b border-gray-100">
                    <h3 class="font-medium text-gray-900">Message Preview</h3>
                    <p class="text-sm text-gray-500">How the message will look with placeholders filled</p>
                </div>
                <div id="preview-container" class="p-6">
                    <div class="text-center text-gray-400 py-8">
                        <svg class="w-12 h-12 mx-auto mb-3 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"/>
                        </svg>
                        <p>Select a draft and contact to preview</p>
                    </div>
                </div>
                <div id="preview-placeholders" class="hidden px-6 pb-6">
                    <div class="text-sm">
                        <div id="filled-placeholders" class="text-green-600"></div>
                        <div id="missing-placeholders" class="text-amber-600 mt-1"></div>
                    </div>
                </div>
            </div>
        </div>
    </main>

    <!-- Batch Success Modal -->
    <div id="batch-success-modal" class="fixed inset-0 z-50 hidden">
        <div class="fixed inset-0 bg-black/50 backdrop-blur-sm" onclick="closeBatchModal()"></div>
        <div class="fixed inset-0 flex items-center justify-center p-4 pointer-events-none">
            <div class="bg-white rounded-2xl shadow-xl max-w-md w-full p-6 pointer-events-auto transform transition-all">
                <div class="flex items-center justify-center w-12 h-12 mx-auto bg-green-100 rounded-full mb-4">
                    <svg class="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                    </svg>
                </div>
                <h3 class="text-lg font-semibold text-gray-900 text-center mb-2">Batch Created Successfully!</h3>
                <p id="batch-modal-message" class="text-sm text-gray-600 text-center mb-6">Your messages have been queued and will be sent shortly.</p>
                <div class="flex flex-col sm:flex-row gap-3">
                    <button onclick="closeBatchModal()" class="flex-1 px-4 py-2.5 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 font-medium transition-colors">
                        Stay Here
                    </button>
                    <a id="batch-view-link" href="/batch-runs" class="flex-1 px-4 py-2.5 bg-whatsapp-500 text-white rounded-lg hover:bg-whatsapp-600 font-medium transition-colors text-center flex items-center justify-center gap-2">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
                        </svg>
                        View Progress
                    </a>
                </div>
            </div>
        </div>
    </div>

    <script>
    let drafts = [];
    let contacts = [];
    let groups = [];
    let selectedDraft = null;
    let selectedContact = null;
    let selectedGroup = null;
    let currentMode = 'contact'; // 'contact' or 'group'

    function setMode(mode) {
        currentMode = mode;

        // Update toggle button styles
        const contactBtn = document.getElementById('mode-contact');
        const groupBtn = document.getElementById('mode-group');

        if (mode === 'contact') {
            contactBtn.className = 'flex-1 py-2 px-3 text-sm font-medium rounded-md transition-colors bg-white text-gray-900 shadow-sm';
            groupBtn.className = 'flex-1 py-2 px-3 text-sm font-medium rounded-md transition-colors text-gray-500 hover:text-gray-700';
            document.getElementById('contact-selection').classList.remove('hidden');
            document.getElementById('group-selection').classList.add('hidden');
        } else {
            groupBtn.className = 'flex-1 py-2 px-3 text-sm font-medium rounded-md transition-colors bg-white text-gray-900 shadow-sm';
            contactBtn.className = 'flex-1 py-2 px-3 text-sm font-medium rounded-md transition-colors text-gray-500 hover:text-gray-700';
            document.getElementById('group-selection').classList.remove('hidden');
            document.getElementById('contact-selection').classList.add('hidden');
        }

        updatePreview();
        updateSendButton();
    }

    async function loadGroups() {
        try {
            const response = await fetch('/api/groups');
            const data = await response.json();
            if (data.success) {
                groups = data.groups || [];
                // Show "no groups" message if empty
                const noGroupsMsg = document.getElementById('no-groups-msg');
                if (groups.length === 0) {
                    noGroupsMsg.classList.remove('hidden');
                } else {
                    noGroupsMsg.classList.add('hidden');
                }
            }
        } catch (error) {
            Toast.error('Failed to load contact groups');
        }
    }

    function showGroupDropdown() {
        const query = document.getElementById('group-search').value.toLowerCase();
        renderGroupDropdown(query);
    }

    function searchGroups() {
        const query = document.getElementById('group-search').value.toLowerCase();
        renderGroupDropdown(query);
    }

    function renderGroupDropdown(query) {
        const dropdown = document.getElementById('group-dropdown');

        const matches = query
            ? groups.filter(g => g.name && g.name.toLowerCase().includes(query)).slice(0, 10)
            : groups.slice(0, 10);

        if (matches.length === 0) {
            dropdown.innerHTML = query
                ? '<div class="p-3 text-sm text-gray-500">No groups found matching "' + escapeHtml(query) + '"</div>'
                : '<div class="p-3 text-sm text-gray-500">No contact groups yet. <a href="/groups" class="text-whatsapp-600 hover:text-whatsapp-700">Create one</a></div>';
        } else {
            dropdown.innerHTML = matches.map(g => ` + "`" + `
                <div onclick="selectGroup(${JSON.stringify(g).replace(/"/g, '&quot;')})"
                     class="p-3 hover:bg-gray-50 cursor-pointer flex items-center gap-3">
                    <div class="w-8 h-8 bg-blue-100 text-blue-700 rounded-full flex items-center justify-center">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                        </svg>
                    </div>
                    <div>
                        <div class="font-medium text-gray-900">${escapeHtml(g.name)}</div>
                        <div class="text-sm text-gray-500">${g.member_count || 0} members</div>
                    </div>
                </div>
            ` + "`" + `).join('');
        }
        dropdown.classList.remove('hidden');
    }

    function selectGroup(group) {
        selectedGroup = group;
        document.getElementById('group-search').value = '';
        document.getElementById('group-dropdown').classList.add('hidden');

        const selected = document.getElementById('selected-group');
        document.getElementById('selected-group-name').textContent = group.name;
        document.getElementById('selected-group-count').textContent = (group.member_count || 0) + ' members';
        document.getElementById('edit-group-link').href = '/groups/' + group.id;
        selected.classList.remove('hidden');

        updatePreview();
        updateSendButton();
    }

    function clearGroup() {
        selectedGroup = null;
        document.getElementById('selected-group').classList.add('hidden');
        updatePreview();
        updateSendButton();
    }

    async function loadDrafts() {
        try {
            const response = await fetch('/api/drafts');
            const data = await response.json();
            if (data.success) {
                drafts = data.drafts;
                renderDraftOptions();
                checkUrlParams();
            }
        } catch (error) {
            Toast.error('Failed to load drafts');
        }
    }

    async function loadContacts() {
        try {
            const response = await fetch('/api/contacts');
            const data = await response.json();
            if (data.success) {
                contacts = data.contacts;
                checkUrlParams();
            }
        } catch (error) {
            Toast.error('Failed to load contacts');
        }
    }

    function checkUrlParams() {
        const params = new URLSearchParams(window.location.search);
        const draftId = params.get('draft');
        const jid = params.get('jid');

        if (draftId && drafts.length > 0) {
            document.getElementById('draft-select').value = draftId;
            onDraftChange();
        }

        if (jid && contacts.length > 0) {
            const contact = contacts.find(c => c.jid === jid);
            if (contact) {
                selectContact(contact);
            }
        }
    }

    function renderDraftOptions() {
        const select = document.getElementById('draft-select');
        select.innerHTML = '<option value="">-- Choose a draft --</option>' +
            drafts.map(d => '<option value="' + d.id + '">' + escapeHtml(d.title) + '</option>').join('');
    }

    function onDraftChange() {
        const id = parseInt(document.getElementById('draft-select').value);
        selectedDraft = drafts.find(d => d.id === id) || null;

        const preview = document.getElementById('draft-content-preview');
        if (selectedDraft) {
            preview.textContent = selectedDraft.content;
            preview.classList.remove('hidden');
        } else {
            preview.classList.add('hidden');
        }

        updatePreview();
        updateSendButton();
    }

    function showContactDropdown() {
        const query = document.getElementById('contact-search').value.toLowerCase();
        renderContactDropdown(query);
    }

    function searchContacts() {
        const query = document.getElementById('contact-search').value.toLowerCase();
        renderContactDropdown(query);
    }

    function renderContactDropdown(query) {
        const dropdown = document.getElementById('contact-dropdown');

        const matches = query
            ? contacts.filter(c =>
                (c.name && c.name.toLowerCase().includes(query)) ||
                (c.phone && c.phone.includes(query))
              ).slice(0, 10)
            : contacts.slice(0, 10);

        if (matches.length === 0) {
            dropdown.innerHTML = query
                ? '<div class="p-3 text-sm text-gray-500">No contacts found matching "' + escapeHtml(query) + '"</div>'
                : '<div class="p-3 text-sm text-gray-500">No contacts available</div>';
        } else {
            dropdown.innerHTML = matches.map(c => ` + "`" + `
                <div onclick="selectContact(${JSON.stringify(c).replace(/"/g, '&quot;')})"
                     class="p-3 hover:bg-gray-50 cursor-pointer flex items-center gap-3">
                    <div class="w-8 h-8 bg-whatsapp-100 text-whatsapp-700 rounded-full flex items-center justify-center text-sm font-medium">
                        ${(c.name || c.phone || '?')[0].toUpperCase()}
                    </div>
                    <div>
                        <div class="font-medium text-gray-900">${escapeHtml(c.name || c.phone)}</div>
                        <div class="text-sm text-gray-500">${c.phone}</div>
                    </div>
                </div>
            ` + "`" + `).join('');
        }
        dropdown.classList.remove('hidden');
    }

    function selectContact(contact) {
        selectedContact = contact;
        document.getElementById('contact-search').value = '';
        document.getElementById('contact-dropdown').classList.add('hidden');

        const selected = document.getElementById('selected-contact');
        document.getElementById('selected-avatar').textContent = (contact.name || contact.phone || '?')[0].toUpperCase();
        document.getElementById('selected-name').textContent = contact.name || contact.phone;
        document.getElementById('selected-phone').textContent = contact.phone;
        document.getElementById('edit-attrs-link').href = '/contact/' + encodeURIComponent(contact.jid);
        selected.classList.remove('hidden');

        updatePreview();
        updateSendButton();
    }

    function clearContact() {
        selectedContact = null;
        document.getElementById('selected-contact').classList.add('hidden');
        updatePreview();
        updateSendButton();
    }

    async function updatePreview() {
        const container = document.getElementById('preview-container');
        const placeholdersDiv = document.getElementById('preview-placeholders');

        if (currentMode === 'contact') {
            await updateContactPreview(container, placeholdersDiv);
        } else {
            updateGroupPreview(container, placeholdersDiv);
        }
    }

    async function updateContactPreview(container, placeholdersDiv) {
        if (!selectedDraft || !selectedContact) {
            container.innerHTML = ` + "`" + `
                <div class="text-center text-gray-400 py-8">
                    <svg class="w-12 h-12 mx-auto mb-3 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"/>
                    </svg>
                    <p>Select a draft and contact to preview</p>
                </div>
            ` + "`" + `;
            placeholdersDiv.classList.add('hidden');
            return;
        }

        try {
            const response = await fetch('/api/drafts/' + selectedDraft.id + '/preview', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ jid: selectedContact.jid })
            });
            const data = await response.json();

            if (data.success && data.preview) {
                container.innerHTML = ` + "`" + `
                    <div class="bg-whatsapp-50 rounded-lg p-4">
                        <div class="text-sm text-gray-600 whitespace-pre-wrap">${escapeHtml(data.preview.preview)}</div>
                    </div>
                ` + "`" + `;

                const filled = data.preview.placeholders_filled || [];
                const missing = data.preview.placeholders_missing || [];

                placeholdersDiv.classList.remove('hidden');
                document.getElementById('filled-placeholders').innerHTML = filled.length > 0
                    ? 'Filled: ' + filled.map(p => '<code class="bg-green-100 px-1 rounded">{{' + p + '}}</code>').join(', ')
                    : '';
                document.getElementById('missing-placeholders').innerHTML = missing.length > 0
                    ? 'Missing: ' + missing.map(p => '<code class="bg-amber-100 px-1 rounded">{{' + p + '}}</code>').join(', ')
                    : '';
            }
        } catch (error) {
            container.innerHTML = '<div class="text-red-500 p-4">Failed to generate preview</div>';
        }
    }

    function updateGroupPreview(container, placeholdersDiv) {
        if (!selectedDraft || !selectedGroup) {
            container.innerHTML = ` + "`" + `
                <div class="text-center text-gray-400 py-8">
                    <svg class="w-12 h-12 mx-auto mb-3 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                    </svg>
                    <p>Select a draft and contact group to preview</p>
                </div>
            ` + "`" + `;
            placeholdersDiv.classList.add('hidden');
            return;
        }

        const memberCount = selectedGroup.member_count || 0;

        // For group mode, show the template and batch summary
        container.innerHTML = ` + "`" + `
            <div class="space-y-4">
                <div class="bg-blue-50 rounded-lg p-4">
                    <div class="flex items-center gap-2 mb-2">
                        <svg class="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                        </svg>
                        <span class="font-medium text-blue-900">Batch Send Summary</span>
                    </div>
                    <ul class="text-sm text-blue-800 space-y-1">
                        <li><strong>Group:</strong> ${escapeHtml(selectedGroup.name)}</li>
                        <li><strong>Recipients:</strong> ${memberCount} contacts</li>
                        <li><strong>Draft:</strong> ${escapeHtml(selectedDraft.title)}</li>
                    </ul>
                </div>
                <div class="bg-gray-50 rounded-lg p-4">
                    <div class="text-xs text-gray-500 uppercase tracking-wide mb-2">Message Template</div>
                    <div class="text-sm text-gray-600 whitespace-pre-wrap">${escapeHtml(selectedDraft.content)}</div>
                </div>
                <div class="text-xs text-gray-500">
                    <svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
                    </svg>
                    Messages will be personalized using each contact's attributes
                </div>
            </div>
        ` + "`" + `;

        // Extract placeholders from draft content
        const placeholderRegex = /\{\{(\w+)\}\}/g;
        const placeholders = [];
        let match;
        while ((match = placeholderRegex.exec(selectedDraft.content)) !== null) {
            if (!placeholders.includes(match[1])) {
                placeholders.push(match[1]);
            }
        }

        if (placeholders.length > 0) {
            placeholdersDiv.classList.remove('hidden');
            document.getElementById('filled-placeholders').innerHTML = '';
            document.getElementById('missing-placeholders').innerHTML =
                'Placeholders: ' + placeholders.map(p => '<code class="bg-blue-100 px-1 rounded">{{' + p + '}}</code>').join(', ');
        } else {
            placeholdersDiv.classList.add('hidden');
        }
    }

    function updateSendButton() {
        const btn = document.getElementById('send-btn');
        if (currentMode === 'contact') {
            btn.disabled = !selectedDraft || !selectedContact;
            btn.innerHTML = '<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/></svg> Send Message';
        } else {
            btn.disabled = !selectedDraft || !selectedGroup;
            const count = selectedGroup ? (selectedGroup.member_count || 0) : 0;
            btn.innerHTML = '<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/></svg> Send to ' + count + ' Contacts';
        }
    }

    async function sendMessage() {
        if (currentMode === 'contact') {
            await sendSingleMessage();
        } else {
            await sendBatchMessage();
        }
    }

    async function sendSingleMessage() {
        if (!selectedDraft || !selectedContact) return;

        const btn = document.getElementById('send-btn');
        btn.disabled = true;
        btn.innerHTML = '<svg class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg> Sending...';

        try {
            const response = await fetch('/api/drafts/' + selectedDraft.id + '/send', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ jid: selectedContact.jid })
            });
            const data = await response.json();

            if (data.success) {
                Toast.success('Message sent successfully!');
                // Reset form
                clearContact();
                document.getElementById('draft-select').value = '';
                selectedDraft = null;
                document.getElementById('draft-content-preview').classList.add('hidden');
                updatePreview();
            } else {
                Toast.error('Failed to send: ' + data.message);
            }
        } catch (error) {
            Toast.error('Failed to send message');
        }

        btn.disabled = false;
        updateSendButton();
    }

    async function sendBatchMessage() {
        if (!selectedDraft || !selectedGroup) return;

        // Check if group has members
        if ((selectedGroup.member_count || 0) === 0) {
            Toast.error('Cannot send to an empty group');
            return;
        }

        const btn = document.getElementById('send-btn');
        btn.disabled = true;
        btn.innerHTML = '<svg class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg> Creating batch...';

        try {
            const response = await fetch('/api/batch-runs', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    draft_id: selectedDraft.id,
                    group_id: selectedGroup.id
                })
            });
            const data = await response.json();

            if (data.success) {
                const memberCount = selectedGroup.member_count || 0;
                const batchId = data.batch ? data.batch.id : null;

                Toast.success('Batch created! ' + memberCount + ' messages queued.');

                // Reset form
                clearGroup();
                document.getElementById('draft-select').value = '';
                selectedDraft = null;
                document.getElementById('draft-content-preview').classList.add('hidden');
                updatePreview();

                showBatchSuccessModal(memberCount, batchId);
            } else {
                Toast.error('Failed to create batch: ' + data.message);
            }
        } catch (error) {
            Toast.error('Failed to create batch');
        }

        btn.disabled = false;
        updateSendButton();
    }

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    function showBatchSuccessModal(messageCount, batchId) {
        const modal = document.getElementById('batch-success-modal');
        const message = document.getElementById('batch-modal-message');
        const viewLink = document.getElementById('batch-view-link');

        message.textContent = messageCount + ' message' + (messageCount !== 1 ? 's have' : ' has') + ' been queued and will be sent shortly.';

        // Link to specific batch run if ID is available, otherwise to batch runs list
        if (batchId) {
            viewLink.href = '/batch-runs/' + batchId;
        } else {
            viewLink.href = '/batch-runs';
        }

        modal.classList.remove('hidden');
        document.body.style.overflow = 'hidden';
    }

    function closeBatchModal() {
        const modal = document.getElementById('batch-success-modal');
        modal.classList.add('hidden');
        document.body.style.overflow = '';
    }

    // Close modal on Escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            closeBatchModal();
        }
    });

    // Close dropdowns when clicking outside
    document.addEventListener('click', (e) => {
        if (!e.target.closest('#contact-search') && !e.target.closest('#contact-dropdown')) {
            document.getElementById('contact-dropdown').classList.add('hidden');
        }
        if (!e.target.closest('#group-search') && !e.target.closest('#group-dropdown')) {
            document.getElementById('group-dropdown').classList.add('hidden');
        }
    });

    // Initialize: load all required data in parallel
    loadDrafts();
    loadContacts();
    loadGroups();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// HandleContactsPage renders the contacts list page with attribute management links
func (h *WebHandler) HandleContactsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Contacts - Friday</title>
    ` + sharedHead + `
</head>
<body class="bg-gray-50 min-h-screen">
    ` + navComponent + `
    <script>` + toastScript + `</script>

    <main class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div class="mb-8">
            <h1 class="text-2xl font-bold text-gray-900">Contacts</h1>
            <p class="text-gray-500 mt-1">Manage contacts and their custom attributes for message personalization</p>
        </div>

        <!-- Search -->
        <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-6">
            <div class="relative">
                <input type="text" id="contact-search" placeholder="Search contacts by name or phone..." oninput="searchContacts()"
                    class="w-full pl-10 pr-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-whatsapp-500 focus:border-whatsapp-500">
                <svg class="w-5 h-5 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                </svg>
            </div>
        </div>

        <!-- Attribute Keys Summary -->
        <div id="attr-keys-section" class="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-6 hidden">
            <div class="flex items-center gap-2 mb-3">
                <svg class="w-5 h-5 text-whatsapp-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"/>
                </svg>
                <span class="text-sm font-medium text-gray-700">Attribute keys in use:</span>
            </div>
            <div id="attr-keys-list" class="flex flex-wrap gap-2"></div>
        </div>

        <!-- Contacts List -->
        <div class="bg-white rounded-xl shadow-sm border border-gray-100">
            <div id="contacts-list" class="divide-y divide-gray-100">
                <div class="p-8 text-center text-gray-500 animate-pulse">Loading contacts...</div>
            </div>

            <!-- Empty state -->
            <div id="empty-contacts" class="hidden p-8 text-center">
                <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                </svg>
                <p class="text-gray-500">No contacts found</p>
                <p class="text-sm text-gray-400 mt-1">Connect WhatsApp to see your contacts</p>
            </div>
        </div>
    </main>

    <script>
    let allContacts = [];
    let contactAttributes = {}; // Map of jid -> attribute count

    async function loadContacts() {
        try {
            const response = await fetch('/api/contacts');
            const data = await response.json();
            if (data.success) {
                allContacts = data.contacts || [];
                await loadAttributeCounts();
                displayContacts(allContacts);
            } else {
                document.getElementById('contacts-list').innerHTML = '<p class="p-8 text-center text-red-500">Error loading contacts</p>';
                Toast.error('Failed to load contacts');
            }
        } catch (error) {
            document.getElementById('contacts-list').innerHTML = '<p class="p-8 text-center text-red-500">Error: ' + error.message + '</p>';
            Toast.error('Failed to load contacts');
        }
    }

    async function loadAttributeCounts() {
        try {
            const response = await fetch('/api/attributes/keys');
            const data = await response.json();
            if (data.success && data.keys && data.keys.length > 0) {
                // Show attribute keys section
                const section = document.getElementById('attr-keys-section');
                section.classList.remove('hidden');

                const keysList = document.getElementById('attr-keys-list');
                keysList.innerHTML = data.keys.map(key => {
                    const count = data.counts[key] || 0;
                    return '<span class="inline-flex items-center gap-1 px-2.5 py-1 bg-whatsapp-50 text-whatsapp-700 text-sm rounded-full">' +
                        '<code>{{' + escapeHtml(key) + '}}</code>' +
                        '<span class="text-whatsapp-500">(' + count + ')</span></span>';
                }).join('');
            }
        } catch (error) {
            console.error('Failed to load attribute keys:', error);
        }
    }

    function displayContacts(contacts) {
        const container = document.getElementById('contacts-list');
        const emptyState = document.getElementById('empty-contacts');

        if (contacts.length === 0) {
            container.innerHTML = '';
            emptyState.classList.remove('hidden');
            return;
        }

        emptyState.classList.add('hidden');
        container.innerHTML = contacts.map(contact => {
            const name = contact.name || contact.phone;
            const initials = name.charAt(0).toUpperCase();
            const jid = contact.jid || '';

            return '<a href="/contact/' + encodeURIComponent(jid) + '" ' +
                'class="flex items-center gap-4 p-4 hover:bg-gray-50 transition-colors">' +
                '<div class="w-12 h-12 rounded-full bg-gradient-to-br from-whatsapp-500 to-whatsapp-700 flex items-center justify-center text-white text-lg font-semibold">' + initials + '</div>' +
                '<div class="flex-1 min-w-0">' +
                '<p class="text-sm font-medium text-gray-900 truncate">' + escapeHtml(name) + '</p>' +
                '<p class="text-sm text-gray-500">' + escapeHtml(contact.phone) + '</p>' +
                '</div>' +
                '<div class="flex items-center gap-2 text-gray-400">' +
                '<span class="text-sm">View attributes</span>' +
                '<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">' +
                '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>' +
                '</svg></div></a>';
        }).join('');
    }

    function searchContacts() {
        const query = document.getElementById('contact-search').value.toLowerCase().trim();

        if (!query) {
            displayContacts(allContacts);
            return;
        }

        const filtered = allContacts.filter(c =>
            (c.name && c.name.toLowerCase().includes(query)) ||
            c.phone.includes(query)
        );
        displayContacts(filtered);
    }

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    loadContacts();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// HandleGroupsPage renders the groups management page
func (h *WebHandler) HandleGroupsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Contact Groups - Friday</title>
    ` + sharedHead + `
</head>
<body class="min-h-screen bg-gray-50">
    ` + navComponent + `
    <script>` + toastScript + `</script>

    <!-- Main Content -->
    <main class="max-w-6xl mx-auto px-4 py-8">
        <div class="flex items-center justify-between mb-6">
            <div>
                <h1 class="text-2xl font-semibold text-gray-900">Contact Groups</h1>
                <p class="text-gray-500 mt-1">Organize contacts into groups for batch messaging</p>
            </div>
            <button onclick="showCreateModal()" class="inline-flex items-center gap-2 px-4 py-2.5 bg-whatsapp-500 text-white font-medium rounded-lg hover:bg-whatsapp-600 transition-colors">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/>
                </svg>
                New Group
            </button>
        </div>

        <!-- Groups Grid -->
        <div id="groups-grid" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        </div>

        <!-- Empty State -->
        <div id="empty-state" class="hidden text-center py-16">
            <div class="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg class="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                </svg>
            </div>
            <h3 class="text-lg font-medium text-gray-900 mb-2">No groups yet</h3>
            <p class="text-gray-500 mb-6">Create your first contact group to start batch messaging</p>
            <button onclick="showCreateModal()" class="inline-flex items-center gap-2 px-4 py-2 bg-whatsapp-500 text-white rounded-lg hover:bg-whatsapp-600 transition-colors">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/>
                </svg>
                Create Group
            </button>
        </div>

        <!-- Loading State -->
        <div id="loading-state" class="text-center py-16">
            <div class="w-8 h-8 border-4 border-whatsapp-500 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
            <p class="text-gray-500">Loading groups...</p>
        </div>
    </main>

    <!-- Create/Edit Modal -->
    <div id="group-modal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 hidden">
        <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4">
            <div class="p-6 border-b border-gray-100">
                <h2 id="modal-title" class="text-lg font-semibold text-gray-900">Create Group</h2>
            </div>
            <form id="group-form" class="p-6">
                <input type="hidden" id="group-id">
                <div class="mb-4">
                    <label for="group-name" class="block text-sm font-medium text-gray-700 mb-1">Group Name</label>
                    <input type="text" id="group-name" required
                        class="w-full px-4 py-2.5 border border-gray-200 rounded-lg focus:ring-2 focus:ring-whatsapp-500 focus:border-transparent transition-all"
                        placeholder="e.g., VIP Customers">
                </div>
                <div class="flex justify-end gap-3">
                    <button type="button" onclick="hideModal()" class="px-4 py-2 text-gray-600 hover:bg-gray-100 rounded-lg transition-colors">
                        Cancel
                    </button>
                    <button type="submit" class="px-4 py-2 bg-whatsapp-500 text-white rounded-lg hover:bg-whatsapp-600 transition-colors">
                        Save
                    </button>
                </div>
            </form>
        </div>
    </div>

    <script>
    let groups = [];

    async function loadGroups() {
        try {
            const response = await fetch('/api/groups');
            const data = await response.json();
            if (data.success) {
                groups = data.groups;
                renderGroups();
            }
        } catch (e) {
            Toast.error('Failed to load groups');
        } finally {
            document.getElementById('loading-state').classList.add('hidden');
        }
    }

    function renderGroups() {
        const grid = document.getElementById('groups-grid');
        const empty = document.getElementById('empty-state');

        if (groups.length === 0) {
            grid.classList.add('hidden');
            empty.classList.remove('hidden');
            return;
        }

        empty.classList.add('hidden');
        grid.classList.remove('hidden');

        grid.innerHTML = groups.map(group => ` + "`" + `
            <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-5 hover:shadow-md transition-shadow">
                <div class="flex items-start justify-between mb-3">
                    <div class="flex items-center gap-3">
                        <div class="w-10 h-10 bg-whatsapp-50 rounded-full flex items-center justify-center">
                            <svg class="w-5 h-5 text-whatsapp-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                            </svg>
                        </div>
                        <div>
                            <h3 class="font-medium text-gray-900">${escapeHtml(group.name)}</h3>
                            <p class="text-sm text-gray-500">${group.member_count} members</p>
                        </div>
                    </div>
                    <div class="flex items-center gap-1">
                        <button onclick="editGroup(${group.id})" class="p-1.5 text-gray-400 hover:text-whatsapp-600 hover:bg-whatsapp-50 rounded-lg transition-colors" title="Edit">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                            </svg>
                        </button>
                        <button onclick="deleteGroup(${group.id})" class="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors" title="Delete">
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                            </svg>
                        </button>
                    </div>
                </div>
                <a href="/groups/${group.id}" class="block w-full py-2 text-center text-sm font-medium text-whatsapp-600 bg-whatsapp-50 rounded-lg hover:bg-whatsapp-100 transition-colors">
                    Manage Members
                </a>
            </div>
        ` + "`" + `).join('');
    }

    function showCreateModal() {
        document.getElementById('modal-title').textContent = 'Create Group';
        document.getElementById('group-id').value = '';
        document.getElementById('group-name').value = '';
        document.getElementById('group-modal').classList.remove('hidden');
        document.getElementById('group-name').focus();
    }

    function editGroup(id) {
        const group = groups.find(g => g.id === id);
        if (!group) return;
        document.getElementById('modal-title').textContent = 'Edit Group';
        document.getElementById('group-id').value = id;
        document.getElementById('group-name').value = group.name;
        document.getElementById('group-modal').classList.remove('hidden');
        document.getElementById('group-name').focus();
    }

    function hideModal() {
        document.getElementById('group-modal').classList.add('hidden');
    }

    async function deleteGroup(id) {
        const group = groups.find(g => g.id === id);
        if (!confirm('Delete "' + group.name + '"? This will also remove all member associations.')) return;
        try {
            const response = await fetch('/api/groups/' + id, { method: 'DELETE' });
            const data = await response.json();
            if (data.success) {
                Toast.success('Group deleted');
                loadGroups();
            } else {
                Toast.error(data.message || 'Failed to delete group');
            }
        } catch (e) {
            Toast.error('Failed to delete group');
        }
    }

    document.getElementById('group-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const id = document.getElementById('group-id').value;
        const name = document.getElementById('group-name').value.trim();
        if (!name) {
            Toast.error('Group name is required');
            return;
        }
        try {
            const url = id ? '/api/groups/' + id : '/api/groups';
            const method = id ? 'PUT' : 'POST';
            const response = await fetch(url, {
                method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name })
            });
            const data = await response.json();
            if (data.success) {
                Toast.success(id ? 'Group updated' : 'Group created');
                hideModal();
                loadGroups();
            } else {
                Toast.error(data.message || 'Failed to save group');
            }
        } catch (e) {
            Toast.error('Failed to save group');
        }
    });

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') hideModal();
    });
    document.getElementById('group-modal').addEventListener('click', (e) => {
        if (e.target.id === 'group-modal') hideModal();
    });

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    loadGroups();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// HandleGroupDetailPage renders the group detail page with member management
func (h *WebHandler) HandleGroupDetailPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/groups/")
	if path == "" {
		http.Redirect(w, r, "/groups", http.StatusFound)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Group Details - Friday</title>
    ` + sharedHead + `
</head>
<body class="min-h-screen bg-gray-50">
    ` + navComponent + `
    <script>` + toastScript + `</script>

    <main class="max-w-4xl mx-auto px-4 py-8">
        <div class="mb-6">
            <a href="/groups" class="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 mb-4">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
                </svg>
                Back to Groups
            </a>
            <div class="flex items-center justify-between">
                <div>
                    <h1 id="group-name" class="text-2xl font-semibold text-gray-900">Loading...</h1>
                    <p id="member-count" class="text-gray-500 mt-1">0 members</p>
                </div>
                <button onclick="showSendModal()" class="inline-flex items-center gap-2 px-4 py-2.5 bg-whatsapp-500 text-white font-medium rounded-lg hover:bg-whatsapp-600 transition-colors">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/>
                    </svg>
                    Send to Group
                </button>
            </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-5 mb-6">
            <h2 class="font-medium text-gray-900 mb-4">Add Members</h2>
            <div class="flex gap-3">
                <div class="flex-1 relative">
                    <input type="text" id="contact-search" placeholder="Search contacts..."
                        class="w-full px-4 py-2.5 border border-gray-200 rounded-lg focus:ring-2 focus:ring-whatsapp-500"
                        oninput="searchContacts()">
                    <div id="contact-results" class="absolute z-10 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg hidden max-h-64 overflow-y-auto"></div>
                </div>
                <button onclick="addSelectedContacts()" id="add-btn" disabled
                    class="px-4 py-2.5 bg-whatsapp-500 text-white rounded-lg hover:bg-whatsapp-600 disabled:opacity-50">
                    Add Selected
                </button>
            </div>
            <div id="selected-contacts" class="flex flex-wrap gap-2 mt-3"></div>
        </div>

        <div class="bg-white rounded-xl shadow-sm border border-gray-100">
            <div class="p-5 border-b border-gray-100">
                <h2 class="font-medium text-gray-900">Members</h2>
            </div>
            <div id="members-list" class="divide-y divide-gray-100"></div>
            <div id="no-members" class="hidden p-8 text-center">
                <p class="text-gray-500">No members in this group yet</p>
            </div>
        </div>
    </main>

    <div id="send-modal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 hidden">
        <div class="bg-white rounded-xl shadow-xl w-full max-w-lg mx-4">
            <div class="p-6 border-b border-gray-100">
                <h2 class="text-lg font-semibold text-gray-900">Send to Group</h2>
            </div>
            <div class="p-6">
                <div class="mb-4">
                    <label class="block text-sm font-medium text-gray-700 mb-2">Select Draft</label>
                    <select id="draft-select" class="w-full px-4 py-2.5 border border-gray-200 rounded-lg">
                        <option value="">Select a draft...</option>
                    </select>
                </div>
                <div id="draft-preview" class="mb-4 p-4 bg-gray-50 rounded-lg hidden">
                    <p class="text-sm text-gray-600" id="draft-content"></p>
                </div>
                <div class="bg-amber-50 border border-amber-200 rounded-lg p-4 mb-4">
                    <p class="text-sm text-amber-700">Messages will be sent with 10-15 second random delays to avoid spam detection.</p>
                </div>
                <div class="flex justify-end gap-3">
                    <button onclick="hideSendModal()" class="px-4 py-2 text-gray-600 hover:bg-gray-100 rounded-lg">Cancel</button>
                    <button onclick="startBatch()" id="start-batch-btn" disabled class="px-4 py-2 bg-whatsapp-500 text-white rounded-lg disabled:opacity-50">Start Batch</button>
                </div>
            </div>
        </div>
    </div>

    <script>
    const groupId = ` + "`" + path + "`" + `;
    let group = null;
    let members = [];
    let allContacts = [];
    let selectedToAdd = [];
    let drafts = [];

    async function loadGroup() {
        try {
            const response = await fetch('/api/groups/' + groupId);
            const data = await response.json();
            if (data.success) {
                group = data.group;
                members = data.members || [];
                document.getElementById('group-name').textContent = group.name;
                document.getElementById('member-count').textContent = members.length + ' members';
                renderMembers();
            } else {
                Toast.error('Failed to load group');
                window.location.href = '/groups';
            }
        } catch (e) {
            Toast.error('Failed to load group');
        }
    }

    async function loadContacts() {
        try {
            const response = await fetch('/api/contacts');
            const data = await response.json();
            if (data.success) allContacts = data.contacts || [];
        } catch (e) {}
    }

    async function loadDrafts() {
        try {
            const response = await fetch('/api/drafts');
            const data = await response.json();
            if (data.success) {
                drafts = data.drafts || [];
                document.getElementById('draft-select').innerHTML = '<option value="">Select a draft...</option>' +
                    drafts.map(d => '<option value="' + d.id + '">' + escapeHtml(d.title) + '</option>').join('');
            }
        } catch (e) {}
    }

    function renderMembers() {
        const list = document.getElementById('members-list');
        const noMembers = document.getElementById('no-members');
        if (members.length === 0) {
            list.innerHTML = '';
            noMembers.classList.remove('hidden');
            return;
        }
        noMembers.classList.add('hidden');
        list.innerHTML = members.map(m => ` + "`" + `
            <div class="flex items-center justify-between p-4 hover:bg-gray-50">
                <div class="flex items-center gap-3">
                    <div class="w-10 h-10 bg-gray-100 rounded-full flex items-center justify-center">
                        <span class="text-gray-600 font-medium">${escapeHtml((m.name || '?').charAt(0).toUpperCase())}</span>
                    </div>
                    <div>
                        <p class="font-medium text-gray-900">${escapeHtml(m.name)}</p>
                        <p class="text-sm text-gray-500">${escapeHtml(m.phone)}</p>
                    </div>
                </div>
                <button onclick="removeMember('${m.jid}')" class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                    </svg>
                </button>
            </div>
        ` + "`" + `).join('');
    }

    function searchContacts() {
        const query = document.getElementById('contact-search').value.toLowerCase();
        const results = document.getElementById('contact-results');
        if (query.length < 2) { results.classList.add('hidden'); return; }
        const memberJids = members.map(m => m.jid);
        const filtered = allContacts.filter(c => {
            const jid = c.jid.String || c.jid;
            return !memberJids.includes(jid) && !selectedToAdd.some(s => (s.jid.String || s.jid) === jid) &&
                ((c.name && c.name.toLowerCase().includes(query)) || c.phone.includes(query));
        }).slice(0, 10);
        if (filtered.length === 0) { results.classList.add('hidden'); return; }
        results.classList.remove('hidden');
        results.innerHTML = filtered.map(c => ` + "`" + `
            <div class="p-3 hover:bg-gray-50 cursor-pointer" onclick="selectContact('${c.jid.String || c.jid}')">
                <p class="font-medium text-gray-900">${escapeHtml(c.name)}</p>
                <p class="text-sm text-gray-500">${escapeHtml(c.phone)}</p>
            </div>
        ` + "`" + `).join('');
    }

    function selectContact(jid) {
        const contact = allContacts.find(c => (c.jid.String || c.jid) === jid);
        if (!contact) return;
        selectedToAdd.push(contact);
        renderSelectedContacts();
        document.getElementById('contact-search').value = '';
        document.getElementById('contact-results').classList.add('hidden');
    }

    function removeSelected(jid) {
        selectedToAdd = selectedToAdd.filter(c => (c.jid.String || c.jid) !== jid);
        renderSelectedContacts();
    }

    function renderSelectedContacts() {
        document.getElementById('add-btn').disabled = selectedToAdd.length === 0;
        document.getElementById('selected-contacts').innerHTML = selectedToAdd.map(c => ` + "`" + `
            <span class="inline-flex items-center gap-1 px-3 py-1.5 bg-whatsapp-50 text-whatsapp-700 rounded-full text-sm">
                ${escapeHtml(c.name)}
                <button onclick="removeSelected('${c.jid.String || c.jid}')" class="hover:text-whatsapp-900">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
                </button>
            </span>
        ` + "`" + `).join('');
    }

    async function addSelectedContacts() {
        if (selectedToAdd.length === 0) return;
        try {
            const jids = selectedToAdd.map(c => c.jid.String || c.jid);
            const response = await fetch('/api/groups/' + groupId + '/members', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ jids })
            });
            const data = await response.json();
            if (data.success) {
                Toast.success('Members added');
                selectedToAdd = [];
                renderSelectedContacts();
                loadGroup();
            } else {
                Toast.error(data.message);
            }
        } catch (e) {
            Toast.error('Failed to add members');
        }
    }

    async function removeMember(jid) {
        if (!confirm('Remove this member?')) return;
        try {
            const response = await fetch('/api/groups/' + groupId + '/members/' + encodeURIComponent(jid), { method: 'DELETE' });
            const data = await response.json();
            if (data.success) {
                Toast.success('Member removed');
                loadGroup();
            } else {
                Toast.error(data.message);
            }
        } catch (e) {
            Toast.error('Failed to remove member');
        }
    }

    function showSendModal() {
        if (members.length === 0) { Toast.warning('Add members first'); return; }
        document.getElementById('send-modal').classList.remove('hidden');
        loadDrafts();
    }

    function hideSendModal() {
        document.getElementById('send-modal').classList.add('hidden');
    }

    document.getElementById('draft-select').addEventListener('change', (e) => {
        const draftId = e.target.value;
        const preview = document.getElementById('draft-preview');
        const btn = document.getElementById('start-batch-btn');
        if (draftId) {
            const draft = drafts.find(d => d.id == draftId);
            if (draft) {
                document.getElementById('draft-content').textContent = draft.content;
                preview.classList.remove('hidden');
                btn.disabled = false;
            }
        } else {
            preview.classList.add('hidden');
            btn.disabled = true;
        }
    });

    async function startBatch() {
        const draftId = document.getElementById('draft-select').value;
        if (!draftId) return;
        try {
            const response = await fetch('/api/batch-runs', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ draft_id: parseInt(draftId), group_id: parseInt(groupId) })
            });
            const data = await response.json();
            if (data.success) {
                Toast.success(data.message);
                window.location.href = '/batch-runs/' + data.batch.id;
            } else {
                Toast.error(data.message);
            }
        } catch (e) {
            Toast.error('Failed to start batch');
        }
    }

    document.addEventListener('click', (e) => {
        const results = document.getElementById('contact-results');
        if (!results.contains(e.target) && e.target.id !== 'contact-search') results.classList.add('hidden');
    });
    document.addEventListener('keydown', (e) => { if (e.key === 'Escape') hideSendModal(); });
    document.getElementById('send-modal').addEventListener('click', (e) => { if (e.target.id === 'send-modal') hideSendModal(); });

    function escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    loadGroup();
    loadContacts();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// HandleBatchRunsPage renders the batch runs list page
func (h *WebHandler) HandleBatchRunsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Batch Runs - Friday</title>
    ` + sharedHead + `
</head>
<body class="min-h-screen bg-gray-50">
    ` + navComponent + `
    <script>` + toastScript + `</script>

    <main class="max-w-6xl mx-auto px-4 py-8">
        <div class="flex items-center justify-between mb-6">
            <div>
                <h1 class="text-2xl font-semibold text-gray-900">Batch Runs</h1>
                <p class="text-gray-500 mt-1">Track and manage batch message sending</p>
            </div>
        </div>

        <div id="active-banner" class="hidden bg-gradient-to-r from-whatsapp-500 to-whatsapp-600 text-white rounded-xl p-5 mb-6">
            <div class="flex items-center justify-between">
                <div class="flex items-center gap-4">
                    <div class="w-12 h-12 bg-white/20 rounded-full flex items-center justify-center">
                        <div class="w-6 h-6 border-3 border-white border-t-transparent rounded-full animate-spin"></div>
                    </div>
                    <div>
                        <p class="font-medium">Batch in Progress</p>
                        <p id="active-info" class="text-white/80 text-sm">Sending...</p>
                    </div>
                </div>
                <a id="active-link" href="#" class="px-4 py-2 bg-white text-whatsapp-600 rounded-lg font-medium hover:bg-white/90">View Progress</a>
            </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
            <div class="overflow-x-auto">
                <table class="w-full">
                    <thead class="bg-gray-50 border-b border-gray-100">
                        <tr>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Batch</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Progress</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
                            <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                        </tr>
                    </thead>
                    <tbody id="batches-body" class="divide-y divide-gray-100"></tbody>
                </table>
            </div>
            <div id="no-batches" class="hidden p-12 text-center">
                <h3 class="text-lg font-medium text-gray-900 mb-2">No batch runs yet</h3>
                <p class="text-gray-500 mb-4">Start by creating a group and sending a draft to it</p>
                <a href="/groups" class="inline-flex items-center gap-2 px-4 py-2 bg-whatsapp-500 text-white rounded-lg hover:bg-whatsapp-600">Go to Groups</a>
            </div>
            <div id="loading-state" class="p-12 text-center">
                <div class="w-8 h-8 border-4 border-whatsapp-500 border-t-transparent rounded-full animate-spin mx-auto"></div>
            </div>
        </div>
    </main>

    <script>
    let batches = [];

    async function loadBatches() {
        try {
            const response = await fetch('/api/batch-runs');
            const data = await response.json();
            if (data.success) {
                batches = data.batches || [];
                renderBatches();
                checkActiveBatch();
            }
        } catch (e) {
            Toast.error('Failed to load batches');
        } finally {
            document.getElementById('loading-state').classList.add('hidden');
        }
    }

    async function checkActiveBatch() {
        try {
            const response = await fetch('/api/batch-runs/active');
            const data = await response.json();
            const banner = document.getElementById('active-banner');
            if (data.has_active && data.batch) {
                banner.classList.remove('hidden');
                document.getElementById('active-info').textContent = 'Sending "' + data.batch.draft_title + '" to "' + data.batch.group_name + '" - ' + data.batch.sent_count + '/' + data.batch.total_count + ' sent';
                document.getElementById('active-link').href = '/batch-runs/' + data.batch.id;
            } else {
                banner.classList.add('hidden');
            }
        } catch (e) {}
    }

    function renderBatches() {
        const tbody = document.getElementById('batches-body');
        const noBatches = document.getElementById('no-batches');
        if (batches.length === 0) {
            tbody.innerHTML = '';
            noBatches.classList.remove('hidden');
            return;
        }
        noBatches.classList.add('hidden');
        tbody.innerHTML = batches.map(b => {
            const statusColors = { 'queued': 'bg-gray-100 text-gray-700', 'running': 'bg-blue-100 text-blue-700', 'completed': 'bg-green-100 text-green-700', 'cancelled': 'bg-gray-100 text-gray-500', 'failed': 'bg-red-100 text-red-700' };
            const statusColor = statusColors[b.status] || 'bg-gray-100 text-gray-700';
            const progress = b.total_count > 0 ? Math.round((b.sent_count + b.failed_count) / b.total_count * 100) : 0;
            const created = new Date(b.created_at).toLocaleString();
            return ` + "`" + `
                <tr class="hover:bg-gray-50">
                    <td class="px-6 py-4">
                        <p class="font-medium text-gray-900">${escapeHtml(b.draft_title)}</p>
                        <p class="text-sm text-gray-500">to ${escapeHtml(b.group_name)}</p>
                    </td>
                    <td class="px-6 py-4">
                        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor}">
                            ${b.status === 'running' ? '<span class="w-1.5 h-1.5 bg-blue-500 rounded-full mr-1.5 animate-pulse"></span>' : ''}
                            ${b.status.charAt(0).toUpperCase() + b.status.slice(1)}
                        </span>
                    </td>
                    <td class="px-6 py-4">
                        <div class="flex items-center gap-2">
                            <div class="w-24 h-2 bg-gray-200 rounded-full overflow-hidden">
                                <div class="h-full bg-whatsapp-500 rounded-full" style="width: ${progress}%"></div>
                            </div>
                            <span class="text-sm text-gray-600">${b.sent_count}/${b.total_count}</span>
                        </div>
                    </td>
                    <td class="px-6 py-4 text-sm text-gray-500">${created}</td>
                    <td class="px-6 py-4 text-right">
                        <div class="flex items-center justify-end gap-2">
                            <a href="/batch-runs/${b.id}" class="px-3 py-1.5 text-sm text-whatsapp-600 hover:bg-whatsapp-50 rounded-lg">View</a>
                            ${b.status === 'running' || b.status === 'queued' ?
                                '<button onclick="cancelBatch(' + b.id + ')" class="px-3 py-1.5 text-sm text-red-600 hover:bg-red-50 rounded-lg">Cancel</button>' :
                                '<button onclick="deleteBatch(' + b.id + ')" class="px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100 rounded-lg">Delete</button>'}
                        </div>
                    </td>
                </tr>
            ` + "`" + `;
        }).join('');
    }

    async function cancelBatch(id) {
        if (!confirm('Cancel this batch?')) return;
        try {
            const response = await fetch('/api/batch-runs/' + id + '/cancel', { method: 'POST' });
            const data = await response.json();
            if (data.success) { Toast.success('Batch cancelled'); loadBatches(); }
            else { Toast.error(data.message); }
        } catch (e) { Toast.error('Failed to cancel batch'); }
    }

    async function deleteBatch(id) {
        if (!confirm('Delete this batch run?')) return;
        try {
            const response = await fetch('/api/batch-runs/' + id, { method: 'DELETE' });
            const data = await response.json();
            if (data.success) { Toast.success('Batch deleted'); loadBatches(); }
            else { Toast.error(data.message); }
        } catch (e) { Toast.error('Failed to delete batch'); }
    }

    function escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    loadBatches();
    setInterval(loadBatches, 5000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// HandleBatchRunDetailPage renders the batch run detail page with real-time SSE progress
func (h *WebHandler) HandleBatchRunDetailPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/batch-runs/")
	if path == "" {
		http.Redirect(w, r, "/batch-runs", http.StatusFound)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Batch Progress - Friday</title>
    ` + sharedHead + `
</head>
<body class="min-h-screen bg-gray-50">
    ` + navComponent + `
    <script>` + toastScript + `</script>

    <main class="max-w-4xl mx-auto px-4 py-8">
        <div class="mb-6">
            <a href="/batch-runs" class="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 mb-4">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
                </svg>
                Back to Batches
            </a>
        </div>

        <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6 mb-6">
            <div class="flex items-start justify-between mb-6">
                <div>
                    <h1 id="batch-title" class="text-xl font-semibold text-gray-900">Loading...</h1>
                    <p id="batch-subtitle" class="text-gray-500 mt-1"></p>
                </div>
                <span id="status-badge" class="px-3 py-1 rounded-full text-sm font-medium bg-gray-100 text-gray-700">Loading</span>
            </div>

            <div class="mb-6">
                <div class="flex items-center justify-between mb-2">
                    <span class="text-sm text-gray-600">Progress</span>
                    <span id="progress-text" class="text-sm font-medium text-gray-900">0/0</span>
                </div>
                <div class="h-3 bg-gray-200 rounded-full overflow-hidden">
                    <div id="progress-bar" class="h-full bg-whatsapp-500 rounded-full transition-all duration-300" style="width: 0%"></div>
                </div>
                <div class="flex justify-between mt-2">
                    <span id="sent-count" class="text-sm text-green-600">0 sent</span>
                    <span id="failed-count" class="text-sm text-red-600">0 failed</span>
                </div>
            </div>

            <div id="current-status" class="bg-gray-50 rounded-lg p-4 hidden">
                <div class="flex items-center gap-3">
                    <div class="w-10 h-10 bg-whatsapp-100 rounded-full flex items-center justify-center">
                        <div class="w-5 h-5 border-2 border-whatsapp-600 border-t-transparent rounded-full animate-spin"></div>
                    </div>
                    <div>
                        <p class="font-medium text-gray-900">Sending to <span id="current-contact">...</span></p>
                        <p class="text-sm text-gray-500">Next message in <span id="countdown" class="font-medium text-whatsapp-600">--</span> seconds</p>
                    </div>
                </div>
            </div>

            <div id="actions" class="mt-6 hidden">
                <button onclick="cancelBatch()" class="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600">Cancel Batch</button>
            </div>
        </div>

        <div class="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
            <div class="p-5 border-b border-gray-100">
                <h2 class="font-medium text-gray-900">Message History</h2>
            </div>
            <div id="messages-list" class="divide-y divide-gray-100 max-h-96 overflow-y-auto"></div>
            <div id="no-messages" class="p-8 text-center text-gray-500 hidden">No messages sent yet</div>
        </div>
    </main>

    <div id="message-modal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 hidden">
        <div class="bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 max-h-[80vh] flex flex-col">
            <div class="p-6 border-b border-gray-100 flex items-center justify-between">
                <h2 class="text-lg font-semibold text-gray-900">Message Details</h2>
                <button onclick="hideMessageModal()" class="p-2 text-gray-400 hover:text-gray-600 rounded-lg">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
                </button>
            </div>
            <div class="p-6 overflow-y-auto">
                <div class="mb-4"><label class="text-sm font-medium text-gray-500">Recipient</label><p id="modal-recipient" class="text-gray-900"></p></div>
                <div class="mb-4"><label class="text-sm font-medium text-gray-500">Status</label><p id="modal-status" class="text-gray-900"></p></div>
                <div id="modal-content-section"><label class="text-sm font-medium text-gray-500">Sent Message</label><div id="modal-content" class="mt-1 p-4 bg-gray-50 rounded-lg text-gray-900 whitespace-pre-wrap"></div></div>
                <div id="modal-error-section" class="hidden"><label class="text-sm font-medium text-gray-500">Error</label><p id="modal-error" class="mt-1 p-4 bg-red-50 rounded-lg text-red-700"></p></div>
            </div>
        </div>
    </div>

    <script>
    const batchId = ` + "`" + path + "`" + `;
    let batch = null;
    let messages = [];
    let eventSource = null;

    async function loadBatch() {
        try {
            const response = await fetch('/api/batch-runs/' + batchId);
            const data = await response.json();
            if (data.success) {
                batch = data.batch;
                messages = data.messages || [];
                updateUI();
                if (batch.status === 'running' || batch.status === 'queued') startSSE();
            } else {
                Toast.error('Batch not found');
                window.location.href = '/batch-runs';
            }
        } catch (e) {
            Toast.error('Failed to load batch');
        }
    }

    function updateUI() {
        document.getElementById('batch-title').textContent = 'Sending "' + batch.draft_title + '"';
        document.getElementById('batch-subtitle').textContent = 'to ' + batch.group_name + ' (' + batch.total_count + ' contacts)';
        const badge = document.getElementById('status-badge');
        const statusColors = { 'queued': 'bg-gray-100 text-gray-700', 'running': 'bg-blue-100 text-blue-700', 'completed': 'bg-green-100 text-green-700', 'cancelled': 'bg-gray-100 text-gray-500', 'failed': 'bg-red-100 text-red-700' };
        badge.className = 'px-3 py-1 rounded-full text-sm font-medium ' + (statusColors[batch.status] || 'bg-gray-100 text-gray-700');
        badge.innerHTML = (batch.status === 'running' ? '<span class="inline-block w-2 h-2 bg-blue-500 rounded-full mr-2 animate-pulse"></span>' : '') + batch.status.charAt(0).toUpperCase() + batch.status.slice(1);
        const total = batch.total_count;
        const done = batch.sent_count + batch.failed_count;
        const progress = total > 0 ? (done / total * 100) : 0;
        document.getElementById('progress-bar').style.width = progress + '%';
        document.getElementById('progress-text').textContent = done + '/' + total;
        document.getElementById('sent-count').textContent = batch.sent_count + ' sent';
        document.getElementById('failed-count').textContent = batch.failed_count + ' failed';
        const currentStatus = document.getElementById('current-status');
        const actions = document.getElementById('actions');
        if (batch.status === 'running' || batch.status === 'queued') {
            currentStatus.classList.remove('hidden');
            actions.classList.remove('hidden');
        } else {
            currentStatus.classList.add('hidden');
            actions.classList.add('hidden');
        }
        renderMessages();
    }

    function renderMessages() {
        const list = document.getElementById('messages-list');
        const noMessages = document.getElementById('no-messages');
        if (messages.length === 0) { list.innerHTML = ''; noMessages.classList.remove('hidden'); return; }
        noMessages.classList.add('hidden');
        const sorted = [...messages].sort((a, b) => { if (a.sent_at && b.sent_at) return new Date(b.sent_at) - new Date(a.sent_at); return b.id - a.id; });
        const statusIcons = {
            'pending': '<svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" stroke-width="2"/></svg>',
            'sending': '<div class="w-5 h-5 border-2 border-whatsapp-600 border-t-transparent rounded-full animate-spin"></div>',
            'sent': '<svg class="w-5 h-5 text-green-500" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/></svg>',
            'failed': '<svg class="w-5 h-5 text-red-500" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/></svg>'
        };
        list.innerHTML = sorted.map(m => {
            const time = m.sent_at ? new Date(m.sent_at).toLocaleTimeString() : '--';
            const name = m.contact_name || (m.jid ? m.jid.split('@')[0] : '?');
            return ` + "`" + `
                <div class="flex items-center justify-between p-4 hover:bg-gray-50 cursor-pointer" onclick="showMessageDetail(${m.id})">
                    <div class="flex items-center gap-3">
                        ${statusIcons[m.status] || statusIcons.pending}
                        <div><p class="font-medium text-gray-900">${escapeHtml(name)}</p><p class="text-sm text-gray-500">${time}</p></div>
                    </div>
                    <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>
                </div>
            ` + "`" + `;
        }).join('');
    }

    function startSSE() {
        if (eventSource) eventSource.close();
        eventSource = new EventSource('/api/batch-runs/' + batchId + '/stream');
        eventSource.onmessage = function(event) {
            try { handleSSEEvent(JSON.parse(event.data)); } catch (e) {}
        };
    }

    function handleSSEEvent(data) {
        if (data.total_count !== undefined) {
            batch.sent_count = data.sent_count;
            batch.failed_count = data.failed_count;
            batch.total_count = data.total_count;
            batch.status = data.status;
        }
        if (data.current_contact) document.getElementById('current-contact').textContent = data.current_contact;
        if (data.next_send_in_seconds !== undefined) document.getElementById('countdown').textContent = Math.max(0, data.next_send_in_seconds);
        if (data.last_message) {
            const existing = messages.find(m => m.jid === data.last_message.jid);
            if (existing) {
                existing.status = data.last_message.status;
                existing.sent_content = data.last_message.sent_content;
                existing.sent_at = data.last_message.sent_at;
                existing.error_message = data.last_message.error;
            }
        }
        if (data.type === 'completed' || data.type === 'cancelled') {
            if (eventSource) { eventSource.close(); eventSource = null; }
            Toast.success(data.type === 'completed' ? 'Batch completed!' : 'Batch cancelled');
            // Refresh messages to get final state from database
            refreshMessages();
        }
        updateUI();
    }

    async function refreshMessages() {
        try {
            const response = await fetch('/api/batch-runs/' + batchId + '/messages');
            const data = await response.json();
            if (data.success) {
                messages = data.messages || [];
                renderMessages();
            }
        } catch (e) { /* ignore refresh errors */ }
    }

    async function cancelBatch() {
        if (!confirm('Cancel this batch?')) return;
        try {
            const response = await fetch('/api/batch-runs/' + batchId + '/cancel', { method: 'POST' });
            const data = await response.json();
            if (data.success) {
                Toast.success('Batch cancelled');
                batch.status = 'cancelled';
                updateUI();
                if (eventSource) { eventSource.close(); eventSource = null; }
            } else { Toast.error(data.message); }
        } catch (e) { Toast.error('Failed to cancel batch'); }
    }

    function showMessageDetail(id) {
        const msg = messages.find(m => m.id === id);
        if (!msg) return;
        const name = msg.contact_name || (msg.jid ? msg.jid.split('@')[0] : '?');
        document.getElementById('modal-recipient').textContent = name + ' (' + (msg.jid ? msg.jid.split('@')[0] : '') + ')';
        document.getElementById('modal-status').textContent = msg.status.charAt(0).toUpperCase() + msg.status.slice(1);
        if (msg.sent_content) { document.getElementById('modal-content').textContent = msg.sent_content; document.getElementById('modal-content-section').classList.remove('hidden'); }
        else { document.getElementById('modal-content-section').classList.add('hidden'); }
        if (msg.error_message) { document.getElementById('modal-error').textContent = msg.error_message; document.getElementById('modal-error-section').classList.remove('hidden'); }
        else { document.getElementById('modal-error-section').classList.add('hidden'); }
        document.getElementById('message-modal').classList.remove('hidden');
    }

    function hideMessageModal() { document.getElementById('message-modal').classList.add('hidden'); }
    document.addEventListener('keydown', (e) => { if (e.key === 'Escape') hideMessageModal(); });
    document.getElementById('message-modal').addEventListener('click', (e) => { if (e.target.id === 'message-modal') hideMessageModal(); });
    window.addEventListener('beforeunload', () => { if (eventSource) eventSource.close(); });

    function escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    loadBatch();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}
