package worker

import (
	"context"
	"sync"
	"time"

	"github.com/fabriziosalmi/rainlogs/internal/cloudflare"
	"github.com/fabriziosalmi/rainlogs/internal/config"
	"github.com/fabriziosalmi/rainlogs/internal/db"
	"github.com/fabriziosalmi/rainlogs/internal/kms"
	"github.com/fabriziosalmi/rainlogs/internal/models"
	"github.com/fabriziosalmi/rainlogs/internal/storage"
	"go.uber.org/zap"
)

type InstantLogsManager struct {
	db      *db.DB
	kms     *kms.Encryptor
	storage *storage.MultiStore
	cfCfg   config.CloudflareConfig
	log     *zap.Logger
	wg      sync.WaitGroup
	mu      sync.Mutex
}

func NewInstantLogsManager(db *db.DB, kms *kms.Encryptor, storage *storage.MultiStore, cfCfg config.CloudflareConfig, log *zap.Logger) *InstantLogsManager {
	return &InstantLogsManager{
		db:      db,
		kms:     kms,
		storage: storage,
		cfCfg:   cfCfg,
		log:     log,
	}
}

// Start watching for Business zones and streaming logs.
// This is a naive implementation: it just loops through all business zones and starts streaming.
// A production version would need dynamic add/remove handling.
func (m *InstantLogsManager) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	activeStreams := make(map[string]context.CancelFunc)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			zones, err := m.db.Zones.ListActive(ctx)
			if err != nil {
				m.log.Error("instant logs: list zones", zap.Error(err))
				continue
			}

			m.mu.Lock()
			for _, z := range zones {
				if z.Plan != models.PlanBusiness {
					continue
				}
				if _, exists := activeStreams[z.ID.String()]; exists {
					continue
				}

				// Start streaming for new zone
				ctxZone, cancel := context.WithCancel(ctx)
				activeStreams[z.ID.String()] = cancel
				m.wg.Add(1)
				go func(zone models.Zone) {
					defer m.wg.Done()
					defer func() {
						m.mu.Lock()
						delete(activeStreams, zone.ID.String()) // Safe delete
						m.mu.Unlock()
						cancel()
					}()
					m.streamZone(ctxZone, zone)
				}(*z)
			}
			m.mu.Unlock()
		}
	}
}

func (m *InstantLogsManager) streamZone(ctx context.Context, zone models.Zone) {
	m.log.Info("starting instant logs stream", zap.String("zone", zone.Name))

	// Get Customer & Key
	customer, err := m.db.Customers.GetByID(ctx, zone.CustomerID)
	if err != nil {
		m.log.Error("get customer", zap.Error(err))
		return
	}
	cfKey, err := m.kms.Decrypt(customer.CFAPIKeyEnc)
	if err != nil {
		m.log.Error("decrypt key", zap.Error(err))
		return
	}

	client := cloudflare.NewInstantLogsClient(cfKey, zone.ZoneID)

	// Create Session
	wsURL, err := client.StartSession(ctx)
	if err != nil {
		m.log.Error("start session failed", zap.String("zone", zone.Name), zap.Error(err))
		time.Sleep(10 * time.Second) // Backoff
		return
	}

	// Stream
	ch, err := client.Stream(ctx, wsURL)
	if err != nil {
		m.log.Error("stream failed", zap.String("zone", zone.Name), zap.Error(err))
		return
	}

	buffer := make([][]byte, 0, 1000)
	lastUpload := time.Now()

	upload := func() {
		if len(buffer) == 0 {
			return
		}
		// Flatten
		var raw []byte
		for _, line := range buffer {
			raw = append(raw, line...)
			raw = append(raw, '\n')
		}

		// Upload
		start := lastUpload
		end := time.Now()
		_, _, _, _, _, err := m.storage.PutLogs(ctx, customer.ID, zone.ID, start, end, raw, "instant-logs")
		if err != nil {
			m.log.Error("upload failed", zap.Error(err))
			// Data loss here if we continue? Yes. But it's logs.
		} else {
			m.log.Info("uploaded instant logs batch", zap.Int("lines", len(buffer)))
		}

		buffer = buffer[:0]
		lastUpload = end
	}

	uploadTicker := time.NewTicker(5 * time.Minute) // Upload batch every 5m
	defer uploadTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			upload()
			return
		case <-uploadTicker.C:
			upload()
		case msg, ok := <-ch:
			if !ok {
				upload()
				return
			}
			buffer = append(buffer, msg)
			if len(buffer) >= 5000 { // Or max batch size
				upload()
			}
		}
	}
}
