package groq

import "testing"

func TestTools_BrowserSearch(t *testing.T) {
	def := Tools.BrowserSearch()
	if def.Name != "browser_search" {
		t.Errorf("Name = %q, want browser_search", def.Name)
	}
	if def.ProviderDefinedType != "browser_search" {
		t.Errorf("ProviderDefinedType = %q, want browser_search", def.ProviderDefinedType)
	}
	if def.ProviderDefinedOptions != nil {
		t.Errorf("expected nil options, got %v", def.ProviderDefinedOptions)
	}
}
