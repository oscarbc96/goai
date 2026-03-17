package xai

import "testing"

// ---------------------------------------------------------------------------
// WebSearch
// ---------------------------------------------------------------------------

func TestTools_WebSearch_Default(t *testing.T) {
	def := Tools.WebSearch()
	if def.Name != "web_search" {
		t.Errorf("Name = %q, want web_search", def.Name)
	}
	if def.ProviderDefinedType != "web_search" {
		t.Errorf("ProviderDefinedType = %q, want web_search", def.ProviderDefinedType)
	}
	if len(def.ProviderDefinedOptions) != 0 {
		t.Errorf("expected empty options, got %v", def.ProviderDefinedOptions)
	}
}

func TestTools_WebSearch_AllOptions(t *testing.T) {
	def := Tools.WebSearch(
		WithAllowedDomains("example.com", "test.org"),
		WithExcludedDomains("spam.com"),
		WithWebSearchImageUnderstanding(true),
	)

	opts := def.ProviderDefinedOptions
	allowed, ok := opts["allowed_domains"].([]string)
	if !ok || len(allowed) != 2 || allowed[0] != "example.com" {
		t.Errorf("allowed_domains = %v", opts["allowed_domains"])
	}
	excluded, ok := opts["excluded_domains"].([]string)
	if !ok || len(excluded) != 1 || excluded[0] != "spam.com" {
		t.Errorf("excluded_domains = %v", opts["excluded_domains"])
	}
	if opts["enable_image_understanding"] != true {
		t.Errorf("enable_image_understanding = %v", opts["enable_image_understanding"])
	}
}

func TestTools_WebSearch_ImageUnderstandingFalse(t *testing.T) {
	// When false, enable_image_understanding should NOT be set.
	def := Tools.WebSearch(WithWebSearchImageUnderstanding(false))
	if _, ok := def.ProviderDefinedOptions["enable_image_understanding"]; ok {
		t.Error("enable_image_understanding should not be set when false")
	}
}

// ---------------------------------------------------------------------------
// XSearch
// ---------------------------------------------------------------------------

func TestTools_XSearch_Default(t *testing.T) {
	def := Tools.XSearch()
	if def.Name != "x_search" {
		t.Errorf("Name = %q, want x_search", def.Name)
	}
	if def.ProviderDefinedType != "x_search" {
		t.Errorf("ProviderDefinedType = %q, want x_search", def.ProviderDefinedType)
	}
	if len(def.ProviderDefinedOptions) != 0 {
		t.Errorf("expected empty options, got %v", def.ProviderDefinedOptions)
	}
}

func TestTools_XSearch_AllOptions(t *testing.T) {
	def := Tools.XSearch(
		WithAllowedXHandles("@alice", "@bob"),
		WithExcludedXHandles("@spam"),
		WithXSearchDateRange("2025-01-01", "2025-12-31"),
		WithXSearchImageUnderstanding(true),
		WithXSearchVideoUnderstanding(true),
	)

	opts := def.ProviderDefinedOptions
	handles, ok := opts["allowed_x_handles"].([]string)
	if !ok || len(handles) != 2 {
		t.Errorf("allowed_x_handles = %v", opts["allowed_x_handles"])
	}
	excluded, ok := opts["excluded_x_handles"].([]string)
	if !ok || len(excluded) != 1 {
		t.Errorf("excluded_x_handles = %v", opts["excluded_x_handles"])
	}
	if opts["from_date"] != "2025-01-01" {
		t.Errorf("from_date = %v", opts["from_date"])
	}
	if opts["to_date"] != "2025-12-31" {
		t.Errorf("to_date = %v", opts["to_date"])
	}
	if opts["enable_image_understanding"] != true {
		t.Errorf("enable_image_understanding = %v", opts["enable_image_understanding"])
	}
	if opts["enable_video_understanding"] != true {
		t.Errorf("enable_video_understanding = %v", opts["enable_video_understanding"])
	}
}

func TestTools_XSearch_PartialDateRange(t *testing.T) {
	// Only from_date set (via empty to_date in range).
	def := Tools.XSearch(WithXSearchDateRange("2025-01-01", ""))
	opts := def.ProviderDefinedOptions
	if opts["from_date"] != "2025-01-01" {
		t.Errorf("from_date = %v", opts["from_date"])
	}
	if _, ok := opts["to_date"]; ok {
		t.Error("to_date should not be set when empty")
	}
}

func TestTools_XSearch_UnderstandingFalse(t *testing.T) {
	def := Tools.XSearch(
		WithXSearchImageUnderstanding(false),
		WithXSearchVideoUnderstanding(false),
	)
	if _, ok := def.ProviderDefinedOptions["enable_image_understanding"]; ok {
		t.Error("enable_image_understanding should not be set when false")
	}
	if _, ok := def.ProviderDefinedOptions["enable_video_understanding"]; ok {
		t.Error("enable_video_understanding should not be set when false")
	}
}
