package extractor

const (
	T2SBic = "TRGTXE2SXXX"
)

const (
	// msg types
	MsgTypeSese023Plus = "sese023plus"
)
const (
	// Result keys
	TxIDKey             = "TxID"
	MovementTypeKey     = "MovementType"
	PaymentTypeKey      = "PaymentType"
	MessageTypeKey      = "MessageType"
	ReceivedFromKey     = "ReceivedFrom"
	InstructingPartyKey = "InstructingParty"
)

const (
	// Xpath expressions

	// sese 023 ex T2S
	sese023TxID             = "/CST2SMsg/T2SPayload/Document/SctiesSttlmTxInstr/TxId"
	sese023MovementType     = "/CST2SMsg/T2SPayload/Document/SctiesSttlmTxInstr/SttlmTpAndAddtlParams/SctiesMvmntTp"
	sese023PaymentType      = "/CST2SMsg/T2SPayload/Document/SctiesSttlmTxInstr/SttlmTpAndAddtlParams/Pmt"
	sese023AppHdrBICFI      = "/CST2SMsg/T2SPayload/cst2s:AppHdr/Fr/FIId/FinInstnId/BICFI"
	sese023AppHdrRltd       = "/CST2SMsg/T2SPayload/cst2s:AppHdr/Rltd/Fr/FIId/FinInstnId/BICFI"
	sese023AppHdrMsgDefIdfr = "/CST2SMsg/T2SPayload/cst2s:AppHdr/MsgDefIdr"
	sese023ReceivedFrom     = "/CST2SMsg/CSPayload/IntApplHead/ApplFrom/Id"
)
