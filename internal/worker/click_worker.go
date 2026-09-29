package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/asceded/url-shortener/internal/model"
	"github.com/asceded/url-shortener/internal/repository"
)

const flushTimeout = 5 * time.Second

type ClickWorker struct {
	repo      *repository.ClickRepository
	ch        chan model.Click
	batchSize int
	interval  time.Duration
	log       *slog.Logger
}

func NewClickWorker(
	repo *repository.ClickRepository,
	bufferSize int,
	batchSize int,
	interval time.Duration,
	log *slog.Logger,
) *ClickWorker {
	return &ClickWorker{
		repo:      repo,
		ch:        make(chan model.Click, bufferSize),
		batchSize: batchSize,
		interval:  interval,
		log:       log,
	}
}

func (w *ClickWorker) Start(ctx context.Context) {
	go w.run(ctx)
}

func (w *ClickWorker) Enqueue(click model.Click) {
	select {
	case w.ch <- click:
	default:
		w.log.Warn("click buffer full, dropping", "link_id", click.LinkID)
	}
}

func (w *ClickWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	batch := make([]model.Click, 0, w.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		flushCtx, cancel := context.WithTimeout(context.Background(), flushTimeout)
		defer cancel()
		if err := w.repo.SaveBatch(flushCtx, batch); err != nil {
			w.log.Error("save clicks batch failed", "error", err, "count", len(batch))
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case click := <-w.ch:
			batch = append(batch, click)
			if len(batch) >= w.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}
