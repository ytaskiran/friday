package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/skip2/go-qrcode"
)

type QRHandler struct {
	currentQR string
}

func NewQRHandler() *QRHandler {
	return &QRHandler{}
}

type QRResponse struct {
	QRCode    string `json:"qr_code"`
	Available bool   `json:"available"`
	Message   string `json:"message"`
}

func (h *QRHandler) HandleGetQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if h.currentQR == "" {
		json.NewEncoder(w).Encode(QRResponse{
			Available: false,
			Message:   "No QR code available. Try connecting to WhatsApp first.",
		})
		return
	}

	json.NewEncoder(w).Encode(QRResponse{
		QRCode:    h.currentQR,
		Available: true,
		Message:   "QR code ready for scanning",
	})
}

func (h *QRHandler) SetQR(qrCode string) {
	h.currentQR = qrCode
}

func (h *QRHandler) ClearQR() {
	h.currentQR = ""
}

func (h *QRHandler) HandleQRImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.currentQR == "" {
		http.Error(w, "No QR code available. Try connecting to WhatsApp first.", http.StatusNotFound)
		return
	}

	qrBytes, err := qrcode.Encode(h.currentQR, qrcode.Medium, 512)
	if err != nil {
		http.Error(w, "Failed to generate QR code image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(qrBytes)))
	w.Write(qrBytes)
}
