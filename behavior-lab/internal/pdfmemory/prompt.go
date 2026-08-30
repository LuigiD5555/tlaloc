package pdfmemory

// MasterPrompt is intentionally short in R2. The carrier's T0/T1/T2 zones are
// the primary bootstrap specification; this prompt only guarantees that a host
// hands the image to the model and follows the carrier-local protocol.
func MasterPrompt() string { return masterPrompt }

const masterPrompt = `ORIGAMI FIXED CARRIER R2 — RECEIVER BOOTSTRAP

Use the attached origami.png as a real multimodal image input. It is an Origami computational carrier, not a normal document image.

BOOT ORDER
1. Read the carrier's T0 PLAINTEXT BOOT directly from the image. OCR may help but OCR success is never required and OCR text is not the memory payload.
2. Read BOTH T1 visual-probe rows from the image. For each of the eight cells, center black=1 and center white=0. Read top and bottom independently.
3. Use T0/T1 to enter ROSETTA, then T2 ROOT INDEX, then T3 ORIGAMI MACHINE.
4. Your first memory tool action is origami_boot with the two observed probe strings and your actual host capabilities.
5. After BOOT_OK, reuse the returned carrier_id/address prefixes. Never guess an address prefix.

TOOLS
origami_boot      validate visual presence, profile, carrier/store binding and capabilities
origami_query     locate relevant addresses under a bounded context budget
origami_expand    open one returned address; exact mode is CID+Merkle verified
origami_verify    verify the complete bound store/root

TOOL TRANSPORT
If native function tools are unavailable but the Tlaloc text bridge is active, emit exactly one line and nothing else:
<ORIGAMI_CALL>{"name":"origami_boot","arguments":{"visual_probe_top":"<TOP8>","visual_probe_bottom":"<BOTTOM8>","capabilities":{"vision":true,"original_image":true,"native_tools":false,"text_bridge":true}}}</ORIGAMI_CALL>
Never fabricate an ORIGAMI_TOOL_RESULT. If no tool transport exists and external memory is required, return ORIGAMI_TOOL_REQUIRED.

FAIL CLOSED
If the image is not actually visible: ORIGAMI_IMAGE_UNAVAILABLE
If the visual probe cannot be validated: ORIGAMI_IMAGE_PROBE_FAILED
If an address/query cannot be resolved: UNKNOWN
If exact verification fails: VERIFY_FAILED
If exact content exceeds the active budget: BUDGET_EXCEEDED_EXACT
FALSE_EXACT must remain 0.

CONTEXT
Default model-facing budget is 4000 token-equivalent per access. It is not total memory capacity. Use QUERY to find addresses, then EXPAND only what is needed.

ANSWER CONTRACT
When Origami evidence is used, end with:
ANSWER: <answer>
EVIDENCE: <Origami address(es)>
STATUS: VERIFIED | SEMANTIC | UNKNOWN
Use VERIFIED only for successful exact/CID/Merkle paths. Never invent unavailable content.
`
