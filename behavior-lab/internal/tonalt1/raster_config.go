package tonalt1

const (
	// Frozen source document
	SourcePDFPath = "/mnt/resources/Libros/Aprendizaje/Ingeniería Civil/Cálculo Estructural/Estructuras de concreto/ACI_318-05_Espanhol (1).pdf"
	SourcePDFSHA  = "79db14b30d58a528c858b48d859ef24ab61378cb5879782ea550043472edd8b4"

	// Frozen store
	FreshCorpusStoreDir = "experiments/tonal-t1/fresh-corpus/stores/cand03"
	StorePageCount      = 495

	// Frozen rasterization parameters
	RasterDPI           = 150
	CanvasPx            = 512
	TargetLineHeightPx  = 32
	DrawCue             = true

	// Frozen renderer colors
	CueColorR           = uint8(230)
	CueColorG           = uint8(0)
	CueColorB           = uint8(130)
	CueColorA           = uint8(255)
	NeutralBGR          = uint8(200)
	NeutralBGG          = uint8(200)
	NeutralBGB          = uint8(200)
	NeutralBGA          = uint8(255)

	// Frozen workflow specification
	WorkflowCount       = 60
	OperandCount        = 144
	ExpectedUniquePages = 56 // Approximate, will derive from actual data

	// Call budget (from independent derivation)
	ArmAExpectedCalls     = 60
	ArmBExpectedCalls     = 492
	ArmCExpectedCalls     = 144
	PrimaryTotalCalls     = 696
	CounterfactualCalls   = 0 // Zero additional calls for counterfactuals

	// Hard invariants
	BridgePageOverlapRequired         = 0
	BridgeInstanceOverlapRequired     = 0
	PriorUseOverlapRequired           = 0
	SingleDigitCountRequired          = 0
	DuplicatePhysicalIdentityRequired = 0
	OperandReuseRequired              = 0
)
