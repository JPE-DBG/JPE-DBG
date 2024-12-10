package extractor

import (
	"github.com/antchfx/xmlquery"
)

// findOne searches for a single node in the XML document that matches the given XPath expression.
// It returns the inner text of the found node, or an empty string if no node is found or if the input node is nil.
func findOne(node *xmlquery.Node, path string) string {
	if node == nil {
		return ""
	}
	res := xmlquery.FindOne(node, path)
	if res == nil {
		return ""
	}
	return res.InnerText()
}
