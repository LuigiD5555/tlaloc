package closedloop

// ValidateReady validates the config and every currently required local file
// without performing any model inference. Candidate PNGs may be absent only
// when an explicit external build_command is declared.
func ValidateReady(cfg Config) error {
	_, err := prepare(cfg, true)
	return err
}
