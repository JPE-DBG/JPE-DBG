package extractor

import "github.com/antchfx/xmlquery"

// sese023tsExtractors extracts parameters from the sese023 message.
// It returns a slice of extractionParam which contains the mapping keys and their corresponding extractor functions.
func sese023tsExtractors() []extractionParam {
	res := make([]extractionParam, 0)

	// simple extractions (no logic, just value retrieval)
	extractions := []struct {
		mapKey string
		xPath  string
	}{
		{TxIDKey, sese023TxID},
		{MovementTypeKey, sese023MovementType},
		{PaymentTypeKey, sese023PaymentType},
		{MessageTypeKey, sese023AppHdrMsgDefIdfr},
		{ReceivedFromKey, sese023ReceivedFrom},
	}

	for _, v := range extractions {
		res = append(res, extractionParam{v.mapKey, createExtractorFunc(v.xPath)})
	}

	// special extraction (logic involved)
	res = append(res, extractionParam{InstructingPartyKey, sese023t2sInstructingPartyKey()})
	return res
}

// sese023t2sInstructingPartyKey returns an extractor function that retrieves the instructing party key
// from the XML node. If the instructing party is identified as T2S (by BIC "TRGTXE2SXXX", also stored in constant
// ), it retrieves the
// related party key instead.
func sese023t2sInstructingPartyKey() extractorFunc {
	return func(node *xmlquery.Node) string {
		instParty := findOne(node, sese023AppHdrBICFI)
		if instParty == T2SBic {
			instParty = findOne(node, sese023AppHdrRltd)
		}
		return instParty
	}
}
