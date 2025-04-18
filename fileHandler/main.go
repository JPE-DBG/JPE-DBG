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
	handler          messageHandler
	logger           *zap.SugaredLogger
	readWaitTime     time.Duration
	maxEmptyReads    int
	processedInBatch int
	batchSize        int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(handler messageHandler, logger *zap.SugaredLogger) *BatchProcessor {
	return &BatchProcessor{
		handler:          handler,
		logger:           logger,
		readWaitTime:     500 * time.Millisecond,
		maxEmptyReads:    5,
		processedInBatch: 0,
		batchSize:        10, // Default batch size
	}
}

// ProcessFile adds a file to the current batch
// Returns true if the batch is full and ready for commit
func (bp *BatchProcessor) ProcessFile(msg []byte) (bool, error) {
	if msg == nil {
		return false, nil
	}

	ctx := context.Background()

	// Generate a target filename based on timestamp to ensure uniqueness
	target := "output_" + time.Now().Format("20060102_150405.000") + ".dat"

	bp.logger.Infow("Processing file", "target", target)

	if err := bp.handler.Write(ctx, target, msg); err != nil {
		bp.logger.Errorw("Error processing file", "target", target, "error", err)
		bp.handler.Rollback()
		return false, err
	}

	bp.processedInBatch++

	// Return true if batch is full
	return bp.processedInBatch >= bp.batchSize, nil
}

// CommitBatch commits the current batch of processed files
func (bp *BatchProcessor) CommitBatch() error {
	if bp.processedInBatch == 0 {
		return nil
	}

	bp.logger.Infof("Committing batch of %d files", bp.processedInBatch)

	if err := bp.handler.Commit(); err != nil {
		bp.logger.Errorw("Error committing batch", "error", err)
		bp.handler.Rollback()
		return err
	}

	// Reset the counter
	bp.processedInBatch = 0
	return nil
}

// ProcessFiles reads and processes files in batches
func (bp *BatchProcessor) ProcessFiles() error {
	done := make(chan struct{})
	sigChan := make(chan os.Signal, 1) // Channel for OS signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	defer func() {
		close(done)
		signal.Stop(sigChan) // Stop receiving signals
	}()

	// Keep track of consecutive empty reads
	emptyReads := 0

	for {
		// Check for OS signals
		select {
		case sig := <-sigChan:
			bp.logger.Infow("Received termination signal", "signal", sig)
			// Commit any pending files before shutdown
			if bp.processedInBatch > 0 {
				bp.logger.Info("Committing remaining items before shutdown")
				if err := bp.CommitBatch(); err != nil {
					bp.logger.Errorw("Error committing batch on shutdown", "error", err)
				}
			}
			return nil // Graceful exit
		default:
			// Continue normal processing
		}

		// Read a file
		result, err := bp.handler.Read()
		if err != nil {
			bp.logger.Errorw("Error reading file", "error", err)
			if result == nil {
				return err
			}
		}

		// Process the file if we have one
		if result.Msg != nil {
			emptyReads = 0 // Reset the empty reads counter

			// Add file to batch, check if batch is full
			batchFull, err := bp.ProcessFile(result.Msg)
			if err != nil {
				return err
			}

			// Commit if batch is full
			if batchFull {
				bp.logger.Info("Batch is full, committing")
				if err := bp.CommitBatch(); err != nil {
					return err
				}
			}
		} else {
			// No files found, increment empty reads counter
			emptyReads++
			bp.logger.Debugw("No files found", "emptyReads", emptyReads)

			// If we've had too many consecutive empty reads and we have files in the batch, commit the batch
			if emptyReads >= bp.maxEmptyReads && bp.processedInBatch > 0 {
				bp.logger.Info("Maximum empty reads reached with pending files, committing batch")
				if err := bp.CommitBatch(); err != nil {
					return err
				}
			}

			// If we've had too many consecutive empty reads and no files in the batch, end processing
			if emptyReads >= bp.maxEmptyReads && bp.processedInBatch == 0 {
				bp.logger.Info("Maximum empty reads reached with no pending files, ending batch processing")
				break
			}
		}

		// Wait between reads to prevent CPU spinning
		time.Sleep(bp.readWaitTime)
	}

	return nil
}

func main() {
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Create file handler implementation of messageHandler interface
	handler := NewFileHandler(log)
	defer handler.Close()

	// Create a batch processor
	batchProcessor := NewBatchProcessor(handler, log)

	// Process files in batches
	log.Info("Starting file processing")
	err := batchProcessor.ProcessFiles()
	if err != nil {
		log.Errorw("Error processing files", "error", err)
	}

	log.Info("File processing complete")
}
