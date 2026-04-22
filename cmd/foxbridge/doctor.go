package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/VulpineOS/foxbridge/pkg/doctor"
)

func runDoctorCommand(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	format := fs.String("format", "text", "Output format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	report, err := doctor.Analyze()
	if err != nil {
		fmt.Fprintf(os.Stderr, "doctor failed: %v\n", err)
		return 1
	}

	switch *format {
	case "text":
		fmt.Print(doctor.FormatText(report))
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "doctor failed: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(os.Stderr, "unsupported doctor format: %s\n", *format)
		return 2
	}

	return 0
}
