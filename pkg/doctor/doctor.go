package doctor

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Status string

const (
	StatusImplemented Status = "implemented"
	StatusStubbed     Status = "stubbed"
	StatusMissing     Status = "missing"
	StatusExtension   Status = "extension"
)

type DomainReport struct {
	Domain             string   `json:"domain"`
	ImplementedMethods []string `json:"implementedMethods,omitempty"`
	StubbedMethods     []string `json:"stubbedMethods,omitempty"`
	MissingMethods     []string `json:"missingMethods,omitempty"`
	ExtensionMethods   []string `json:"extensionMethods,omitempty"`
}

type Report struct {
	ProtocolSnapshot string         `json:"protocolSnapshot"`
	UpstreamMethods  int            `json:"upstreamMethods"`
	Implemented      int            `json:"implemented"`
	Stubbed          int            `json:"stubbed"`
	Missing          int            `json:"missing"`
	Extensions       int            `json:"extensions"`
	Domains          []DomainReport `json:"domains"`
}

type upstreamSnapshot struct {
	Source  string              `json:"source"`
	Domains map[string][]string `json:"domains"`
}

type currentBridgeSnapshot struct {
	Methods map[string]string `json:"methods"`
}

//go:embed data/upstream_methods.json
var upstreamMethodsJSON []byte

//go:embed data/current_bridge_methods.json
var currentBridgeMethodsJSON []byte

var stubbedDomains = map[string]bool{
	"Audits":            true,
	"BackgroundService": true,
	"CacheStorage":      true,
	"Cast":              true,
	"Database":          true,
	"Debugger":          true,
	"DeviceAccess":      true,
	"HeapProfiler":      true,
	"IndexedDB":         true,
	"Inspector":         true,
	"Log":               true,
	"Media":             true,
	"Memory":            true,
	"Overlay":           true,
	"Profiler":          true,
	"Security":          true,
	"ServiceWorker":     true,
	"WebAuthn":          true,
}

var stubbedMethods = map[string]bool{
	"Accessibility.disable":                       true,
	"Accessibility.enable":                        true,
	"Browser.getWindowForTarget":                  true,
	"Browser.grantPermissions":                    true,
	"Browser.setDownloadBehavior":                 true,
	"Browser.setWindowBounds":                     true,
	"Console.disable":                             true,
	"Console.enable":                              true,
	"CSS.disable":                                 true,
	"CSS.enable":                                  true,
	"DOMStorage.disable":                          true,
	"DOMStorage.enable":                           true,
	"Emulation.clearDeviceMetricsOverride":        true,
	"Emulation.setDefaultBackgroundColorOverride": true,
	"Emulation.setScrollbarsHidden":               true,
	"Network.deleteCookies":                       true,
	"Network.disable":                             true,
	"Network.emulateNetworkConditions":            true,
	"Network.enable":                              true,
	"Network.setCacheDisabled":                    true,
	"Page.bringToFront":                           true,
	"Page.enable":                                 true,
	"Page.navigateToHistoryEntry":                 true,
	"Page.resetNavigationHistory":                 true,
	"Page.setBypassCSP":                           true,
	"Page.setDownloadBehavior":                    true,
	"Page.setLifecycleEventsEnabled":              true,
	"Page.stopLoading":                            true,
	"Performance.disable":                         true,
	"Performance.enable":                          true,
	"Runtime.addBinding":                          true,
	"Runtime.discardConsoleEntries":               true,
	"Runtime.releaseObjectGroup":                  true,
	"Runtime.runIfWaitingForDebugger":             true,
	"Target.activateTarget":                       true,
	"Target.detachFromTarget":                     true,
}

func Analyze() (Report, error) {
	var upstream upstreamSnapshot
	if err := json.Unmarshal(upstreamMethodsJSON, &upstream); err != nil {
		return Report{}, fmt.Errorf("decode upstream snapshot: %w", err)
	}

	var current currentBridgeSnapshot
	if err := json.Unmarshal(currentBridgeMethodsJSON, &current); err != nil {
		return Report{}, fmt.Errorf("decode current bridge snapshot: %w", err)
	}

	statuses := make(map[string]Status, len(current.Methods))
	for method := range current.Methods {
		statuses[method] = StatusImplemented
	}
	for method := range stubbedMethods {
		statuses[method] = StatusStubbed
	}

	domains := make(map[string]*DomainReport, len(upstream.Domains))
	getDomain := func(name string) *DomainReport {
		if report, ok := domains[name]; ok {
			return report
		}
		report := &DomainReport{Domain: name}
		domains[name] = report
		return report
	}

	var result Report
	result.ProtocolSnapshot = upstream.Source

	for domain, methods := range upstream.Domains {
		report := getDomain(domain)
		for _, name := range methods {
			method := domain + "." + name
			switch {
			case stubbedDomains[domain]:
				report.StubbedMethods = append(report.StubbedMethods, method)
				result.Stubbed++
			case statuses[method] == StatusStubbed:
				report.StubbedMethods = append(report.StubbedMethods, method)
				result.Stubbed++
			case statuses[method] == StatusImplemented:
				report.ImplementedMethods = append(report.ImplementedMethods, method)
				result.Implemented++
			default:
				report.MissingMethods = append(report.MissingMethods, method)
				result.Missing++
			}
			result.UpstreamMethods++
		}
	}

	for method := range statuses {
		if methodInUpstream(upstream.Domains, method) {
			continue
		}
		domain := methodDomain(method)
		report := getDomain(domain)
		report.ExtensionMethods = append(report.ExtensionMethods, method)
		result.Extensions++
	}

	result.Domains = make([]DomainReport, 0, len(domains))
	for _, report := range domains {
		sort.Strings(report.ImplementedMethods)
		sort.Strings(report.StubbedMethods)
		sort.Strings(report.MissingMethods)
		sort.Strings(report.ExtensionMethods)
		result.Domains = append(result.Domains, *report)
	}
	sort.Slice(result.Domains, func(i, j int) bool {
		return result.Domains[i].Domain < result.Domains[j].Domain
	})

	return result, nil
}

func FormatText(report Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "foxbridge doctor\n")
	fmt.Fprintf(&b, "Protocol snapshot: %s\n", report.ProtocolSnapshot)
	fmt.Fprintf(&b, "Upstream methods: %d\n", report.UpstreamMethods)
	fmt.Fprintf(&b, "Implemented: %d\n", report.Implemented)
	fmt.Fprintf(&b, "Stubbed: %d\n", report.Stubbed)
	fmt.Fprintf(&b, "Missing: %d\n", report.Missing)
	fmt.Fprintf(&b, "Extensions: %d\n", report.Extensions)
	fmt.Fprintf(&b, "\nPer-domain coverage:\n")

	for _, domain := range report.Domains {
		fmt.Fprintf(
			&b,
			"- %s: %d implemented, %d stubbed, %d missing, %d extensions\n",
			domain.Domain,
			len(domain.ImplementedMethods),
			len(domain.StubbedMethods),
			len(domain.MissingMethods),
			len(domain.ExtensionMethods),
		)
	}

	if report.Extensions > 0 {
		fmt.Fprintf(&b, "\nExtensions:\n")
		for _, domain := range report.Domains {
			for _, method := range domain.ExtensionMethods {
				fmt.Fprintf(&b, "- %s\n", method)
			}
		}
	}

	return strings.TrimSpace(b.String()) + "\n"
}

func methodInUpstream(domains map[string][]string, method string) bool {
	domain, name, ok := strings.Cut(method, ".")
	if !ok {
		return false
	}
	for _, candidate := range domains[domain] {
		if candidate == name {
			return true
		}
	}
	return false
}

func methodDomain(method string) string {
	if domain, _, ok := strings.Cut(method, "."); ok {
		return domain
	}
	return "Unknown"
}
