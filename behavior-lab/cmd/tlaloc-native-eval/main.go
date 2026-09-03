package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/nativeeval"
)

func main() {
	in := flag.String("in", "-", "native trial JSON or - for stdin")
	out := flag.String("out", "-", "evaluation JSON or - for stdout")
	flag.Parse()
	body, err := read(*in)
	die(err)
	var trial nativeeval.Trial
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	die(dec.Decode(&trial))
	if trial.Schema != "" && trial.Schema != nativeeval.SchemaR0+".trial" {
		die(fmt.Errorf("unexpected schema %q", trial.Schema))
	}
	result := nativeeval.Evaluate(trial)
	encoded, err := json.MarshalIndent(result, "", "  ")
	die(err)
	encoded = append(encoded, '\n')
	if *out == "-" {
		_, err = os.Stdout.Write(encoded)
		die(err)
		return
	}
	die(os.WriteFile(*out, encoded, 0o644))
}

func read(path string) ([]byte, error) {
	if path == "-" {
		return os.ReadFile("/dev/stdin")
	}
	return os.ReadFile(path)
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
