package tailsplit

import (
	"encoding/json"
	"testing"

	"github.com/sagernet/sing/common/domain"
)

func TestControlURLReachesEveryEmbeddedDevice(t *testing.T) {
	config, err := loadTestConfig(t, `
control_url: HTTPS://HS.Example.com:8443/
tailscale:
  exit_node: 100.64.0.10
exits:
  - id: office
    exit_node: 100.64.0.11
rules:
  source: ""
`)
	if err != nil {
		t.Fatal(err)
	}
	if config.ControlURL != "https://hs.example.com:8443" {
		t.Fatalf("control_url = %q", config.ControlURL)
	}
	compiled, err := Compile(config)
	if err != nil {
		t.Fatal(err)
	}
	var native struct {
		Endpoints []map[string]any `json:"endpoints"`
	}
	if err := json.Unmarshal(compiled.JSON, &native); err != nil {
		t.Fatal(err)
	}
	if len(native.Endpoints) != 2 {
		t.Fatalf("endpoints = %d", len(native.Endpoints))
	}
	for _, endpoint := range native.Endpoints {
		if endpoint["control_url"] != "https://hs.example.com:8443" {
			t.Fatalf("endpoint %v control_url = %v", endpoint["tag"], endpoint["control_url"])
		}
	}
}

func TestDefaultControlPlaneIsLeftToTheCore(t *testing.T) {
	config, err := loadTestConfig(t, "tailscale:\n  exit_node: 100.64.0.10\nrules:\n  source: \"\"\n")
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := Compile(config)
	if err != nil {
		t.Fatal(err)
	}
	var native struct {
		Endpoints []map[string]any `json:"endpoints"`
	}
	if err := json.Unmarshal(compiled.JSON, &native); err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range native.Endpoints {
		if _, ok := endpoint["control_url"]; ok {
			t.Fatalf("endpoint %v has an unexpected control_url", endpoint["tag"])
		}
	}
}

// A whole top-level domain is a valid routing rule. The core matches suffixes
// on label boundaries, so "ru" covers example.ru without capturing guru.com.
func TestRoutingRulesAcceptTopLevelDomains(t *testing.T) {
	config, err := loadTestConfig(t, `
rules:
  source: ""
  direct_domains: [ru, .su, "*.xn--p1ai", "рф"]
  proxy_domains: [io]
`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ru", "su", "xn--p1ai", "xn--p1ai"}
	if len(config.Rules.DirectDomains) != len(want) {
		t.Fatalf("direct_domains = %v", config.Rules.DirectDomains)
	}
	for index, value := range want {
		if config.Rules.DirectDomains[index] != value {
			t.Fatalf("direct_domains = %v", config.Rules.DirectDomains)
		}
	}
	matcher := domain.NewMatcher(nil, config.Rules.DirectDomains, false)
	for name, direct := range map[string]bool{
		"yandex.ru": true, "mail.yandex.ru": true, "xn--e1afmkfd.xn--p1ai": true,
		"guru.com": false, "guru": false, "ru.example.com": false,
	} {
		if matcher.Match(name) != direct {
			t.Fatalf("%s direct = %t", name, !direct)
		}
	}
}
