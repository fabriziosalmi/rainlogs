package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fabriziosalmi/rainlogs/internal/cloudflare"
	"github.com/fabriziosalmi/rainlogs/internal/config"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/fabriziosalmi/rainlogs/internal/models"
	"github.com/fabriziosalmi/rainlogs/internal/notifications"
	"github.com/fabriziosalmi/rainlogs/internal/storage"
	"go.uber.org/zap"
)

type InstantLogsManager struct {
	db       *db.DB
	kms      *kms.Encryptor
	storage  *storage.MultiStore
	cfCfg    config.CloudflareConfig
	log      *zap.Logger
	notifier notifications.NotificationService
	wg       sync.WaitGroup
	mu       sync.Mutex
	// streams tracks active stream cancellations by ZoneID
	streams map[string]context.CancelFunc
}

func NewInstantLogsManager(db *db.DB, kms *kms.Encryptor, storage *storage.MultiStore, cfCfg config.CloudflareConfig, log *zap.Logger, notifier notifications.NotificationService) *InstantLogsManager {
	return &InstantLogsManager{
		db:       db,
		kms:      kms,
		storage:  storage,
		cfCfg:    cfCfg,
		log:      log,
		notifier: notifier,
		streams:  make(map[string]context.CancelFunc),
	}
}

// Start watches for Business zones and manages their log streams.
// It handles dynamic addition/removal of zones and ensures robust reconnection.
func (m *InstantLogsManager) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial sync
	m.syncStreams(ctx)

	for {
		select {
		case <-ctx.Done():
			m.stopAll()
			return
		case <-ticker.C:
			m.syncStreams(ctx)
		}
	}
}

func (m *InstantLogsManager) stopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, cancel := range m.streams {
		cancel()
		delete(m.streams, id)
	}
	m.wg.Wait()
}

func (m *InstantLogsManager) syncStreams(ctx context.Context) {
	zones, err := m.db.Zones.ListActive(ctx)
	if err != nil {
		m.log.Error("instant logs: list zones failed", zap.Error(err))
		return
	}

	// Identify active Business zones
	businessZones := make(map[string]models.Zone)
	for _, z := range zones {
		if z.Plan == models.PlanBusiness {
			businessZones[z.ID.String()] = *z
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Remove stopped/downgraded streams
	for id, cancel := range m.streams {
		if _, ok := businessZones[id]; !ok {
			m.log.Info("stopping instant logs stream (zone removed or downgraded)", zap.String("zone_id", id))
			cancel()
			delete(m.streams, id)
		}
	}

	// 2. Start new streams
	for id, zone := range businessZones {
		if _, exists := m.streams[id]; !exists {
			m.log.Info("starting instant logs stream", zap.String("zone", zone.Name))

			// Create a child context for this stream
			ctxZone, cancel := context.WithCancel(ctx)
			m.streams[id] = cancel
			m.wg.Add(1)

			go func(z models.Zone) {
				defer m.wg.Done()
				// Run the stream manager for this zone until ctxZone is cancelled
				m.runZoneStream(ctxZone, z)

				// Cleanup on exit (if it wasn't cancelled by syncStreams)
				m.mu.Lock()
				if _, ok := m.streams[z.ID.String()]; ok {
					// Only delete if we are still in the map (avoid race with syncStreams removal)
					// Actually, we shouldn't delete here if we want to restart on error?
					// runZoneStream loops forever until context cancel, so if we are here, context is done.
					// We can just let the map be cleaned up by syncStreams or stopAll.
				}
				m.mu.Unlock()
			}(zone)
		}
	}
}

// runZoneStream manages the persistent connection life-cycle for a single zone.
// It handles retries, backoffs, and uploading.
func (m *InstantLogsManager) runZoneStream(ctx context.Context, zone models.Zone) {
	// Exponential backoff for connection retries
	minBackoff := 5 * time.Second
	maxBackoff := 5 * time.Minute
	backoff := minBackoff

	for {
		if ctx.Err() != nil {
			return
		}

		if err := m.streamSession(ctx, zone); err == nil {
			// Clean exit (context done)
			return
		} else {
			// Stream errored
			m.log.Error("instant logs stream disconnected, retrying...",
				zap.String("zone", zone.Name),
				zap.Error(err),
				zap.Duration("backoff", backoff),
			)
			// Only alert if we've been backing off for a while (e.g. > 1 minute), indicating persistent failure
			if backoff > 1*time.Minute {
				m.notifier.SendAlert(ctx, zone.ID.String(), "error", fmt.Sprintf("Instant logs stream persistent failure for zone %s: %v", zone.Name, err))
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			// Increase backoff
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (m *InstantLogsManager) streamSession(ctx context.Context, zone models.Zone) error {
	// Get Customer & Key (Refresh from DB to get latest key if rotated)
	customer, err := m.db.Customers.GetByID(ctx, zone.CustomerID)
	if err != nil {
		return fmt.Errorf("get customer: %w", err)
	}
	cfKey, err := m.kms.Decrypt(customer.CFAPIKeyEnc)
	if err != nil {
		return fmt.Errorf("decrypt key: %w", err)
	}

	client := cloudflare.NewInstantLogsClient(cfKey, zone.ZoneID)

	// Create Session
	wsURL, err := client.StartSession(ctx)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}

	// Stream
	ch, err := client.Stream(ctx, wsURL)
	if err != nil {
		return fmt.Errorf("connect stream: %w", err)
	}

	buffer := make([][]byte, 0, 5000)
	lastUpload := time.Now()

	// Helper to flush buffer
	upload := func() {
		if len(buffer) == 0 {
			return
		}

		var raw []byte
		for _, line := range buffer {
			raw = append(raw, line...)
			raw = append(raw, '\n')
		}

		start := lastUpload
		end := time.Now()

		// Use a detached context for upload to ensure data isn't lost if stream cancels mid-upload
		uploadCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		_, _, _, _, _, err := m.storage.PutLogs(uploadCtx, customer.ID, zone.ID, start, end, raw, "instant")
		if err != nil {
			m.log.Error("upload failed", zap.Error(err))
		} else {
			m.log.Info("uploaded instant logs batch",
				zap.String("zone", zone.Name),
				zap.Int("lines", len(buffer)),
			)
		}

		buffer = buffer[:0]
		lastUpload = end
	}

	uploadTicker := time.NewTicker(30 * time.Second) // Upload more frequently for instant feel
	defer uploadTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			upload()
			return nil
		case <-uploadTicker.C:
			upload()
		case msg, ok := <-ch:
			if !ok {
				upload()
				return fmt.Errorf("stream closed by remote")
			}
			buffer = append(buffer, msg)
			if len(buffer) >= 2000 { // Batch size
				upload()
			}
		}
	}
}
