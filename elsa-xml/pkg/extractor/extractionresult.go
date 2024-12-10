package extractor

// ExtractionResult holds the result of an extraction process.
// It contains a map where keys and values are strings.
type ExtractionResult struct {
	res map[string]string
}

// NewExtractionResult creates a new instance of ExtractionResult with an initialized map.
func NewExtractionResult() *ExtractionResult {
	return &ExtractionResult{res: make(map[string]string)}
}

// Add inserts a key-value pair into the ExtractionResult's map.
func (e *ExtractionResult) Add(key, value string) {
	e.res[key] = value
}

// Value returns the value associated with the given key in the ExtractionResult's map.
func (e *ExtractionResult) Value(key string) string {
	return e.res[key]
}
