package doctor

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnalyze_ClassifiesRepresentativeMethods(t *testing.T) {
	report, err := Analyze()
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if report.UpstreamMethods == 0 {
		t.Fatal("expected upstream methods in report")
	}
	if report.Implemented == 0 {
		t.Fatal("expected implemented methods in report")
	}
	if report.Stubbed == 0 {
		t.Fatal("expected stubbed methods in report")
	}
	if report.Missing == 0 {
		t.Fatal("expected missing methods in report")
	}
	if report.Extensions == 0 {
		t.Fatal("expected extension methods in report")
	}

	assertDomainContains(t, report, "Target", "Target.createTarget", "implemented")
	assertDomainContains(t, report, "Debugger", "Debugger.disable", "stubbed")
	assertDomainContains(t, report, "Accessibility", "Accessibility.getPartialAXTree", "missing")
	assertDomainContains(t, report, "Page", "Page.getOptimizedDOM", "extension")
}

func TestAnalyze_CoversAllCurrentBridgeMethods(t *testing.T) {
	report, err := Analyze()
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	seen := map[string]bool{}
	for _, domain := range report.Domains {
		for _, method := range domain.ImplementedMethods {
			seen[method] = true
		}
		for _, method := range domain.StubbedMethods {
			seen[method] = true
		}
		for _, method := range domain.ExtensionMethods {
			seen[method] = true
		}
	}

	var current currentBridgeSnapshot
	if err := json.Unmarshal(currentBridgeMethodsJSON, &current); err != nil {
		t.Fatalf("decode current bridge snapshot: %v", err)
	}
	for method := range current.Methods {
		if !seen[method] {
			t.Fatalf("current bridge method %s was not classified", method)
		}
	}
}

func TestFormatText_IncludesSummary(t *testing.T) {
	report, err := Analyze()
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	text := FormatText(report)
	for _, want := range []string{
		"foxbridge doctor",
		"Protocol snapshot:",
		"Implemented:",
		"Stubbed:",
		"Missing:",
		"Extensions:",
		"Per-domain coverage:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in doctor output", want)
		}
	}
}

func assertDomainContains(t *testing.T, report Report, domainName, method, status string) {
	t.Helper()

	for _, domain := range report.Domains {
		if domain.Domain != domainName {
			continue
		}
		var methods []string
		switch status {
		case "implemented":
			methods = domain.ImplementedMethods
		case "stubbed":
			methods = domain.StubbedMethods
		case "missing":
			methods = domain.MissingMethods
		case "extension":
			methods = domain.ExtensionMethods
		default:
			t.Fatalf("unknown status %s", status)
		}
		for _, candidate := range methods {
			if candidate == method {
				return
			}
		}
		t.Fatalf("method %s not found in %s methods for domain %s", method, status, domainName)
	}

	t.Fatalf("domain %s not found", domainName)
}
