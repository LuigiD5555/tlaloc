package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tlaloc.local/behaviorlab/internal/automata"
)

func main() {
	in := flag.String("in", "", "Tlaloque action trace JSON")
	out := flag.String("out", "-", "output JSON path or -")
	flag.Parse()
	if *in == "" {
		fatal("-in is required")
	}
	b, err := os.ReadFile(*in)
	if err != nil {
		fatal(err.Error())
	}
	var trace automata.ActionTrace
	if err := json.Unmarshal(b, &trace); err != nil {
		fatal(err.Error())
	}
	result, err := automata.Distill(trace)
	if err != nil {
		fatal(err.Error())
	}
	var f *os.File
	if *out == "" || *out == "-" {
		f = os.Stdout
	} else {
		f, err = os.Create(*out)
		if err != nil {
			fatal(err.Error())
		}
		defer f.Close()
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fatal(err.Error())
	}
}

func fatal(msg string) { fmt.Fprintln(os.Stderr, msg); os.Exit(2) }
