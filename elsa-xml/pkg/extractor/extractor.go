// Package extractor provides functions and types for extracting data from ISO20022/ISO20022+ XML documents.
package extractor

import (
	"bytes"
	"errors"
	"github.com/antchfx/xmlquery"
)

// Extract parses the given XML and extracts data based on the message type.
// It returns an ExtractionResult containing the extracted data or an error if the XML is invalid or empty.
func Extract(xml []byte, msgType string) (*ExtractionResult, error) {
	if len(xml) == 0 {
		return nil, errors.New("empty xml")
	}

	doc, err := xmlquery.Parse(bytes.NewReader(xml))
	if err != nil {
		return nil, err
	}

	res := NewExtractionResult()
	exParams := getExParams(msgType)
	if exParams == nil {
		return nil, errors.New("extraction - unsupported message type")
	}
	for _, p := range exParams {
		res.Add(p.mapKey, p.exFunc(doc))
	}
	return res, nil
}

// extractionParam defines a parameter for the extraction process,
// including the key to map the extracted value and the function to extract the value.
type extractionParam struct {
	mapKey string
	exFunc extractorFunc
}

// extractorFunc defines a function type that takes an *xmlquery.Node and returns a string.
type extractorFunc func(node *xmlquery.Node) string

// createExtractorFunc creates an extractor function that searches for a specific XML path
// and returns the inner text of the found node. If the node is not found, it returns an empty string.
func createExtractorFunc(path string) extractorFunc {
	return func(node *xmlquery.Node) string {
		res := xmlquery.FindOne(node, path)
		if res == nil {
			return ""
		}
		return res.InnerText()
	}
}

// getExParams returns a slice of extractionParam based on the provided message type.
// It uses the message type to determine which specific extraction parameters to return.
func getExParams(msgType string) []extractionParam {
	switch msgType {
	case MsgTypeSese023Plus:
		return sese023tsExtractors()
	default:
		return nil
	}
}
