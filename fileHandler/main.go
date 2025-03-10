package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

type messageHandler interface {
	Read() (*ReadResult, error)
	Write(ctx context.Context, target string, data []byte) error
	Commit() error
	Rollback() error
	Close()
}

// ReadResult represents the result of a read operation.
type ReadResult struct {
	// Msg contains the message read from the handler.
	Msg []byte
	// EndBatch indicates whether this message is the last one in the current batch.
	EndBatch bool
}

// BatchProcessor reads files one by one but processes them in batches
type BatchProcessor struct {
	handler      messageHandler
	batchSize    int
	logger       *zap.SugaredLogger
	currentBatch [][]byte
	targets      []string
	lastReadTime time.Time
	readTimeout  time.Duration
}

// NewBatchProcessor creates a new batch processor with the given batch size
func NewBatchProcessor(handler messageHandler, batchSize int, logger *zap.SugaredLogger) *BatchProcessor {
	return &BatchProcessor{
		handler:      handler,
		batchSize:    batchSize,
		logger:       logger,
		currentBatch: make([][]byte, 0, batchSize),
		targets:      make([]string, 0, batchSize),
		lastReadTime: time.Now(),
		readTimeout:  5 * time.Second,
	}
}

// AddToBatch adds a message to the current batch
// Returns true if the batch is full and ready to be processed
func (bp *BatchProcessor) AddToBatch(target string, msg []byte) bool {
	bp.currentBatch = append(bp.currentBatch, msg)
	bp.targets = append(bp.targets, target)
	return len(bp.currentBatch) >= bp.batchSize
}

// ProcessBatch processes all messages in the current batch
func (bp *BatchProcessor) ProcessBatch() error {
	if len(bp.currentBatch) == 0 {
		return nil
	}

	bp.logger.Infof("Processing batch of %d files", len(bp.currentBatch))
	ctx := context.Background()

	for i, msg := range bp.currentBatch {
		target := bp.targets[i]
		if err := bp.handler.Write(ctx, target, msg); err != nil {
			bp.logger.Errorw("Error processing file in batch", "target", target, "error", err)
			bp.handler.Rollback()
			return err
		}
	}

	// Commit the batch
	if err := bp.handler.Commit(); err != nil {
		bp.logger.Errorw("Error committing batch", "error", err)
		bp.handler.Rollback()
		return err
	}

	// Clear the batch after successful processing
	bp.currentBatch = make([][]byte, 0, bp.batchSize)
	bp.targets = make([]string, 0, bp.batchSize)
	return nil
}

// ProcessFiles reads and processes files one by one in batches
func (bp *BatchProcessor) ProcessFiles() error {
	done := make(chan struct{})
	timeoutSignal := make(chan struct{}, 1) // Signal channel for timeout notifications
	sigChan := make(chan os.Signal, 1)      // Channel for OS signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	defer func() {
		close(done)
		signal.Stop(sigChan) // Stop receiving signals
	}()

	// Start a goroutine to check for read timeout
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// If it's been more than readTimeout since last read and we have pending items
				if time.Since(bp.lastReadTime) > bp.readTimeout && len(bp.currentBatch) > 0 {
					bp.logger.Infow("Timeout reached with pending items",
						"timeout", bp.readTimeout,
						"itemsInBatch", len(bp.currentBatch))

					// Send timeout signal (non-blocking)
					select {
					case timeoutSignal <- struct{}{}:
					default: // Don't block if channel is full
					}
				}
			case <-done:
				return
			}
		}
	}()

	for {
		// Check for timeout signal or OS signal
		select {
		case <-timeoutSignal:
			bp.logger.Info("Processing batch due to timeout signal")
			if err := bp.ProcessBatch(); err != nil {
				bp.logger.Errorw("Error processing batch on timeout", "error", err)
				return err
			}
		case sig := <-sigChan:
			bp.logger.Infow("Received termination signal", "signal", sig)
			// Process any remaining items in the batch
			if len(bp.currentBatch) > 0 {
				bp.logger.Info("Processing remaining items before shutdown")
				if err := bp.ProcessBatch(); err != nil {
					bp.logger.Errorw("Error processing batch on shutdown", "error", err)
				}
			}
			return nil // Graceful exit
		default:
			// Continue normal processing
		}

		// Check if we need to end due to no new files for a while
		result, err := bp.handler.Read()
		if err != nil {
			bp.logger.Errorw("Error reading file", "error", err)
			if result == nil {
				return err
			}
		}

		if result.Msg != nil {
			// Reset counter and update last read time on successful read
			bp.lastReadTime = time.Now()

			// Add to current batch with a placeholder target
			if bp.AddToBatch("output", result.Msg) {
			} else {
				// No files found, wait a bit
				time.Sleep(500 * time.Millisecond)
			}

			// Check if we should end due to timeout
			if time.Since(bp.lastReadTime) > bp.readTimeout && len(bp.currentBatch) > 0 {
				bp.logger.Info("No new files for 5 seconds, ending batch")
				if err := bp.ProcessBatch(); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func main() {
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	fh := NewFileHandler(log)

	// Create a batch processor with batch size of 10
	batchProcessor := NewBatchProcessor(fh, 10, log)

	// Process files one by one but in batches
	err := batchProcessor.ProcessFiles()
	if err != nil {
		log.Errorw("Error processing files in batches", "error", err)
	}

	// Cleanup
	fh.Close()
}
