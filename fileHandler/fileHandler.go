package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	envInputDir  = "INPUT_DIR"
	envOutputDir = "OUTPUT_DIR"
	envdoneDir   = "DONE_DIR"
	envErrorDir  = "ERROR_DIR"
)

func NewFileHandler(logger *zap.SugaredLogger) *FileHandler {
	return &FileHandler{logger: logger,
		inputDir:   os.Getenv(envInputDir),
		outputDirs: strings.Split(os.Getenv(envOutputDir), ","),
		doneDir:    os.Getenv(envdoneDir),
		errorDir:   os.Getenv(envErrorDir),
	}
}

type FileHandler struct {
	inputDir   string
	outputDirs []string
	doneDir    string
	errorDir   string
	logger     *zap.SugaredLogger

	// Track the current file being processed
	currentFile string
}

// Read reads a file from the input directory
func (fh *FileHandler) Read() (*ReadResult, error) {
	if fh.inputDir == "" {
		return nil, errors.New("input directory not configured")
	}

	// List files in input directory
	files, err := os.ReadDir(fh.inputDir)
	if err != nil {
		fh.logger.Errorw("Failed to read input directory", "error", err)
		return nil, err
	}

	// No files to process
	if len(files) == 0 {
		return &ReadResult{
			Msg:      nil,
			EndBatch: true,
		}, nil
	}

	// Process first non-directory file
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(fh.inputDir, file.Name())
		fh.currentFile = filePath

		data, err := os.ReadFile(filePath)
		if err != nil {
			fh.logger.Errorw("Failed to read file", "file", filePath, "error", err)
			return nil, err
		}

		fh.logger.Infow("Read file", "file", filePath, "size", len(data))

		return &ReadResult{
			Msg:      data,
			EndBatch: len(files) == 1,
		}, nil
	}

	// Only found directories, no actual files
	return &ReadResult{
		Msg:      nil,
		EndBatch: true,
	}, nil
}

// Write writes data to a file in the target output directory
func (fh *FileHandler) Write(ctx context.Context, target string, data []byte) error {
	if len(fh.outputDirs) == 0 {
		return errors.New("output directories not configured")
	}

	// Determine output directory
	var outputDir string
	if target == "" {
		// Use first output directory if no target specified
		outputDir = fh.outputDirs[0]
	} else {
		// Find matching output directory
		for _, dir := range fh.outputDirs {
			if filepath.Base(dir) == target {
				outputDir = dir
				break
			}
		}

		if outputDir == "" {
			return errors.New("target output directory not found: " + target)
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fh.logger.Errorw("Failed to create output directory", "dir", outputDir, "error", err)
		return err
	}

	// Create output filename
	fileName := "event_" + time.Now().Format("20060102_150405") + ".txt"

	outPath := filepath.Join(outputDir, fileName)

	// Write file
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		fh.logger.Errorw("Failed to write file", "file", outPath, "error", err)
		return err
	}

	fh.logger.Infow("Wrote file", "file", outPath, "size", len(data))
	return nil
}

// Commit moves the processed file to the done directory
func (fh *FileHandler) Commit() error {
	if fh.currentFile == "" {
		return nil // Nothing to commit
	}

	if fh.doneDir == "" {
		return errors.New("done directory not configured")
	}

	// Ensure done directory exists
	if err := os.MkdirAll(fh.doneDir, 0755); err != nil {
		fh.logger.Errorw("Failed to create done directory", "error", err)
		return err
	}

	// Move file to done directory
	fileName := filepath.Base(fh.currentFile)
	destPath := filepath.Join(fh.doneDir, fileName)

	if err := os.Rename(fh.currentFile, destPath); err != nil {
		fh.logger.Errorw("Failed to move file to done directory", "error", err)
		return err
	}

	fh.logger.Infow("Committed file", "source", fh.currentFile, "dest", destPath)
	fh.currentFile = "" // Clear current file
	return nil
}

// Rollback moves the file to the error directory if processing failed
func (fh *FileHandler) Rollback() error {
	if fh.currentFile == "" {
		return nil // Nothing to rollback
	}

	if fh.errorDir == "" {
		fh.logger.Warn("Error directory not configured, cannot rollback file")
		return nil
	}

	// Ensure error directory exists
	if err := os.MkdirAll(fh.errorDir, 0755); err != nil {
		fh.logger.Errorw("Failed to create error directory", "error", err)
		return err
	}

	// Move file to error directory
	fileName := filepath.Base(fh.currentFile)
	destPath := filepath.Join(fh.errorDir, fileName)

	if err := os.Rename(fh.currentFile, destPath); err != nil {
		fh.logger.Errorw("Failed to move file to error directory", "error", err)
		return err
	}

	fh.logger.Infow("Rolled back file", "source", fh.currentFile, "dest", destPath)
	fh.currentFile = "" // Clear current file
	return nil
}

// Close performs cleanup
func (fh *FileHandler) Close() {
	// If there's still a file being processed, roll it back
	if fh.currentFile != "" {
		if err := fh.Rollback(); err != nil {
			fh.logger.Errorw("Failed to rollback file during Close", "error", err)
		}
	}

	fh.logger.Debug("FileHandler closed")
}
