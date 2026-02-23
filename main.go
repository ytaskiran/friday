package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"friday/internal/batch"
	"friday/internal/database"
	"friday/internal/handlers"
	"friday/internal/models"
	"friday/internal/whatsapp"
)

func main() {
	appDB, err := database.New("friday.db")
	if err != nil {
		log.Fatalf("Failed to create app database: %v", err)
	}
	defer appDB.Close()
	log.Println("Application database initialized: friday.db")

	whatsappClient, err := whatsapp.NewClient()
	if err != nil {
		log.Fatalf("Failed to create WhatsApp client: %v", err)
	}
	defer whatsappClient.Disconnect()

	draftRepo := models.NewDraftRepository(appDB)
	attrRepo := models.NewAttributeRepository(appDB)
	groupRepo := models.NewGroupRepository(appDB)
	memberRepo := models.NewGroupMemberRepository(appDB)
	batchRepo := models.NewBatchRunRepository(appDB)
	batchMsgRepo := models.NewBatchMessageRepository(appDB)

	batchWorker := batch.NewWorker(batchRepo, batchMsgRepo, memberRepo, draftRepo, attrRepo, whatsappClient)
	go batchWorker.Run()

	// Initialize handlers
	whatsappHandler := handlers.NewWhatsAppHandler(whatsappClient)
	contactHandler := handlers.NewContactHandler(whatsappClient)
	qrHandler := handlers.NewQRHandler()
	webHandler := handlers.NewWebHandler(draftRepo, attrRepo, whatsappClient)

	// New handlers for drafts and attributes
	draftHandler := handlers.NewDraftHandler(draftRepo, attrRepo, whatsappClient)
	attrHandler := handlers.NewAttributeHandler(attrRepo)

	// Contact groups and batch messaging handlers
	groupHandler := handlers.NewGroupHandler(groupRepo, memberRepo, whatsappClient)
	batchHandler := handlers.NewBatchHandler(batchRepo, batchMsgRepo, groupRepo, memberRepo, draftRepo, batchWorker, whatsappClient)

	// Wire up QR code callbacks
	whatsappClient.SetQRHandler(qrHandler.SetQR)
	whatsappClient.SetQRClearHandler(qrHandler.ClearQR)

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok", "service": "friday-whatsapp-api"}`)
	})

	// Web interface
	// "/" serves the landing/connection page - users must connect before accessing dashboard
	mux.HandleFunc("/", webHandler.HandleLandingPage)
	mux.HandleFunc("/dashboard", webHandler.HandleDashboard)
	mux.HandleFunc("/qr-scan", webHandler.HandleQRScanPage)

	// WhatsApp API
	mux.HandleFunc("/api/whatsapp/status", whatsappHandler.HandleStatus)
	mux.HandleFunc("/api/whatsapp/connect", whatsappHandler.HandleConnect)
	mux.HandleFunc("/api/whatsapp/disconnect", whatsappHandler.HandleDisconnect)
	mux.HandleFunc("/api/whatsapp/send", whatsappHandler.HandleSendMessage)
	mux.HandleFunc("/api/whatsapp/qr", qrHandler.HandleGetQR)
	mux.HandleFunc("/api/whatsapp/qr.png", qrHandler.HandleQRImage)

	// Contact API
	mux.HandleFunc("/api/contacts", contactHandler.HandleGetContacts)
	mux.HandleFunc("/api/contacts/search", contactHandler.HandleSearchContacts)
	mux.HandleFunc("/api/contacts/validate", contactHandler.HandleValidatePhones)

	// Draft API
	mux.HandleFunc("/api/drafts", draftHandler.HandleDrafts)     // GET (list), POST (create)
	mux.HandleFunc("/api/drafts/", draftHandler.HandleDraft)     // GET/{id}, PUT/{id}, DELETE/{id}, POST/{id}/preview, POST/{id}/send

	// Contact Attributes API
	mux.HandleFunc("/api/contacts/", attrHandler.HandleContactAttributes) // /api/contacts/{jid}/attributes
	mux.HandleFunc("/api/attributes/keys", attrHandler.HandleAttributeKeys)

	// Contact Groups API
	mux.HandleFunc("/api/groups", groupHandler.HandleGroups)      // GET (list), POST (create)
	mux.HandleFunc("/api/groups/", groupHandler.HandleGroup)      // GET/{id}, PUT/{id}, DELETE/{id}, POST/{id}/members

	// Batch Runs API
	mux.HandleFunc("/api/batch-runs", batchHandler.HandleBatches) // GET (list), POST (create)
	mux.HandleFunc("/api/batch-runs/", batchHandler.HandleBatch)  // GET/{id}, DELETE/{id}, POST/{id}/cancel, GET/{id}/stream

	// New web pages
	mux.HandleFunc("/drafts", webHandler.HandleDraftsPage)
	mux.HandleFunc("/drafts/", webHandler.HandleDraftEditPage)
	mux.HandleFunc("/contacts", webHandler.HandleContactsPage)
	mux.HandleFunc("/contact/", webHandler.HandleContactDetailPage)
	mux.HandleFunc("/send", webHandler.HandleSendPage)
	mux.HandleFunc("/groups", webHandler.HandleGroupsPage)
	mux.HandleFunc("/groups/", webHandler.HandleGroupDetailPage)
	mux.HandleFunc("/batch-runs", webHandler.HandleBatchRunsPage)
	mux.HandleFunc("/batch-runs/", webHandler.HandleBatchRunDetailPage)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting Friday WhatsApp API server on %s", server.Addr)
		log.Printf("Web: / (dashboard) | /drafts | /qr-scan | /groups | /batch-runs | /health")
		log.Printf("API: /api/whatsapp/{status,connect,send,qr,qr.png}")
		log.Printf("API: /api/contacts | /api/drafts | /api/groups | /api/batch-runs")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Shutdown batch worker first
	batchWorker.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}
