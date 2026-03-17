package openai

import (
	"testing"
)

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
	// No options set → map should be empty.
	if len(def.ProviderDefinedOptions) != 0 {
		t.Errorf("expected empty options, got %v", def.ProviderDefinedOptions)
	}
}

func TestTools_WebSearch_AllOptions(t *testing.T) {
	def := Tools.WebSearch(
		WithSearchContextSize("high"),
		WithUserLocation(WebSearchLocation{
			Country:  "US",
			City:     "Minneapolis",
			Region:   "Minnesota",
			Timezone: "America/Chicago",
		}),
		WithSearchFilters(WebSearchFilters{
			AllowedDomains: []string{"example.com", "test.org"},
		}),
		WithExternalWebAccess(false),
	)

	opts := def.ProviderDefinedOptions
	if opts["search_context_size"] != "high" {
		t.Errorf("search_context_size = %v", opts["search_context_size"])
	}
	loc, ok := opts["user_location"].(map[string]any)
	if !ok {
		t.Fatal("user_location not set")
	}
	if loc["type"] != "approximate" {
		t.Errorf("type = %v, want approximate", loc["type"])
	}
	if loc["country"] != "US" {
		t.Errorf("country = %v", loc["country"])
	}
	if loc["city"] != "Minneapolis" {
		t.Errorf("city = %v", loc["city"])
	}
	if loc["region"] != "Minnesota" {
		t.Errorf("region = %v", loc["region"])
	}
	if loc["timezone"] != "America/Chicago" {
		t.Errorf("timezone = %v", loc["timezone"])
	}

	filters, ok := opts["filters"].(map[string]any)
	if !ok {
		t.Fatal("filters not set")
	}
	domains, ok := filters["allowed_domains"].([]string)
	if !ok || len(domains) != 2 {
		t.Errorf("allowed_domains = %v", filters["allowed_domains"])
	}

	if opts["external_web_access"] != false {
		t.Errorf("external_web_access = %v", opts["external_web_access"])
	}
}

func TestTools_WebSearch_PartialLocation(t *testing.T) {
	// Only country set -- other fields should be absent.
	def := Tools.WebSearch(WithUserLocation(WebSearchLocation{
		Country: "GB",
	}))
	loc, ok := def.ProviderDefinedOptions["user_location"].(map[string]any)
	if !ok {
		t.Fatal("user_location not set")
	}
	if _, ok := loc["city"]; ok {
		t.Error("city should not be set")
	}
	if _, ok := loc["region"]; ok {
		t.Error("region should not be set")
	}
	if _, ok := loc["timezone"]; ok {
		t.Error("timezone should not be set")
	}
}

func TestTools_WebSearch_EmptyFilters(t *testing.T) {
	// Filters with empty AllowedDomains should not be set.
	def := Tools.WebSearch(WithSearchFilters(WebSearchFilters{}))
	if _, ok := def.ProviderDefinedOptions["filters"]; ok {
		t.Error("filters should not be set when AllowedDomains is empty")
	}
}

// ---------------------------------------------------------------------------
// CodeInterpreter
// ---------------------------------------------------------------------------

func TestTools_CodeInterpreter_Default(t *testing.T) {
	def := Tools.CodeInterpreter()
	if def.Name != "code_interpreter" {
		t.Errorf("Name = %q", def.Name)
	}
	if def.ProviderDefinedType != "code_interpreter" {
		t.Errorf("ProviderDefinedType = %q", def.ProviderDefinedType)
	}
	// Default should have container type "auto" with no file_ids.
	container, ok := def.ProviderDefinedOptions["container"].(map[string]any)
	if !ok {
		t.Fatal("container not set as map")
	}
	if container["type"] != "auto" {
		t.Errorf("container type = %v, want auto", container["type"])
	}
}

func TestTools_CodeInterpreter_StringContainer(t *testing.T) {
	def := Tools.CodeInterpreter(WithContainerID("container_123"))
	if def.ProviderDefinedOptions["container"] != "container_123" {
		t.Errorf("container = %v", def.ProviderDefinedOptions["container"])
	}
}

func TestTools_CodeInterpreter_StructContainer(t *testing.T) {
	def := Tools.CodeInterpreter(WithContainerFiles(&CodeInterpreterContainer{
		FileIDs: []string{"file_1", "file_2"},
	}))
	container, ok := def.ProviderDefinedOptions["container"].(map[string]any)
	if !ok {
		t.Fatal("container not set as map")
	}
	if container["type"] != "auto" {
		t.Errorf("type = %v", container["type"])
	}
	fileIDs, ok := container["file_ids"].([]string)
	if !ok || len(fileIDs) != 2 {
		t.Errorf("file_ids = %v", container["file_ids"])
	}
}

// ---------------------------------------------------------------------------
// FileSearch
// ---------------------------------------------------------------------------

func TestTools_FileSearch_Default(t *testing.T) {
	def := Tools.FileSearch()
	if def.Name != "file_search" {
		t.Errorf("Name = %q", def.Name)
	}
	if def.ProviderDefinedType != "file_search" {
		t.Errorf("ProviderDefinedType = %q", def.ProviderDefinedType)
	}
	if len(def.ProviderDefinedOptions) != 0 {
		t.Errorf("expected empty options, got %v", def.ProviderDefinedOptions)
	}
}

func TestTools_FileSearch_AllOptions(t *testing.T) {
	def := Tools.FileSearch(
		WithVectorStoreIDs("vs_1", "vs_2"),
		WithMaxNumResults(10),
		WithRanking(FileSearchRanking{
			Ranker:         "default_2024_08_21",
			ScoreThreshold: 0.8,
		}),
		WithFileSearchFilters(&FileSearchComparisonFilter{
			Key:   "author",
			Type:  "eq",
			Value: "Alice",
		}),
	)

	opts := def.ProviderDefinedOptions
	vsIDs, ok := opts["vector_store_ids"].([]string)
	if !ok || len(vsIDs) != 2 {
		t.Errorf("vector_store_ids = %v", opts["vector_store_ids"])
	}
	if opts["max_num_results"] != 10 {
		t.Errorf("max_num_results = %v", opts["max_num_results"])
	}
	ranking, ok := opts["ranking_options"].(map[string]any)
	if !ok {
		t.Fatal("ranking_options not set")
	}
	if ranking["ranker"] != "default_2024_08_21" {
		t.Errorf("ranker = %v", ranking["ranker"])
	}
	if ranking["score_threshold"] != 0.8 {
		t.Errorf("score_threshold = %v", ranking["score_threshold"])
	}

	// Filter should be serialized.
	filter, ok := opts["filters"].(map[string]any)
	if !ok {
		t.Fatal("filters not set")
	}
	if filter["key"] != "author" {
		t.Errorf("filter key = %v", filter["key"])
	}
}

func TestTools_FileSearch_RankingPartial(t *testing.T) {
	// Ranker only (no threshold) -- score_threshold should not be set.
	def := Tools.FileSearch(WithRanking(FileSearchRanking{Ranker: "auto"}))
	ranking, ok := def.ProviderDefinedOptions["ranking_options"].(map[string]any)
	if !ok {
		t.Fatal("ranking_options not set")
	}
	if _, ok := ranking["score_threshold"]; ok {
		t.Error("score_threshold should not be set when 0")
	}

	// Threshold only (no ranker) -- ranker should not be set.
	def = Tools.FileSearch(WithRanking(FileSearchRanking{ScoreThreshold: 0.5}))
	ranking, ok = def.ProviderDefinedOptions["ranking_options"].(map[string]any)
	if !ok {
		t.Fatal("ranking_options not set")
	}
	if _, ok := ranking["ranker"]; ok {
		t.Error("ranker should not be set when empty")
	}
}

func TestSerializeFilter_CompoundFilter(t *testing.T) {
	def := Tools.FileSearch(WithFileSearchFilters(&FileSearchCompoundFilter{
		Type: "and",
		Filters: []FileSearchFilter{
			&FileSearchComparisonFilter{Key: "status", Type: "eq", Value: "active"},
			&FileSearchComparisonFilter{Key: "priority", Type: "gt", Value: 5},
		},
	}))

	filter, ok := def.ProviderDefinedOptions["filters"].(map[string]any)
	if !ok {
		t.Fatal("filters not set")
	}
	if filter["type"] != "and" {
		t.Errorf("type = %v", filter["type"])
	}
	subFilters, ok := filter["filters"].([]any)
	if !ok || len(subFilters) != 2 {
		t.Errorf("sub-filters = %v", filter["filters"])
	}
}

// ---------------------------------------------------------------------------
// ImageGeneration
// ---------------------------------------------------------------------------

func TestTools_ImageGeneration_Default(t *testing.T) {
	def := Tools.ImageGeneration()
	if def.Name != "image_generation" {
		t.Errorf("Name = %q", def.Name)
	}
	if def.ProviderDefinedType != "image_generation" {
		t.Errorf("ProviderDefinedType = %q", def.ProviderDefinedType)
	}
	if len(def.ProviderDefinedOptions) != 0 {
		t.Errorf("expected empty options, got %v", def.ProviderDefinedOptions)
	}
}

func TestTools_ImageGeneration_AllOptions(t *testing.T) {
	def := Tools.ImageGeneration(
		WithBackground("transparent"),
		WithInputFidelity("high"),
		WithInputImageMask(ImageGenerationMask{
			FileID:   "file_mask_1",
			ImageURL: "https://example.com/mask.png",
		}),
		WithImageModel("gpt-image-1"),
		WithModeration("auto"),
		WithOutputCompression(85),
		WithOutputFormat("webp"),
		WithPartialImages(2),
		WithImageQuality("high"),
		WithImageSize("1024x1536"),
	)

	opts := def.ProviderDefinedOptions
	if opts["background"] != "transparent" {
		t.Errorf("background = %v", opts["background"])
	}
	if opts["input_fidelity"] != "high" {
		t.Errorf("input_fidelity = %v", opts["input_fidelity"])
	}
	mask, ok := opts["input_image_mask"].(map[string]any)
	if !ok {
		t.Fatal("input_image_mask not set")
	}
	if mask["file_id"] != "file_mask_1" {
		t.Errorf("file_id = %v", mask["file_id"])
	}
	if mask["image_url"] != "https://example.com/mask.png" {
		t.Errorf("image_url = %v", mask["image_url"])
	}
	if opts["model"] != "gpt-image-1" {
		t.Errorf("model = %v", opts["model"])
	}
	if opts["moderation"] != "auto" {
		t.Errorf("moderation = %v", opts["moderation"])
	}
	if opts["output_compression"] != 85 {
		t.Errorf("output_compression = %v", opts["output_compression"])
	}
	if opts["output_format"] != "webp" {
		t.Errorf("output_format = %v", opts["output_format"])
	}
	if opts["partial_images"] != 2 {
		t.Errorf("partial_images = %v", opts["partial_images"])
	}
	if opts["quality"] != "high" {
		t.Errorf("quality = %v", opts["quality"])
	}
	if opts["size"] != "1024x1536" {
		t.Errorf("size = %v", opts["size"])
	}
}

// ---------------------------------------------------------------------------
// fileSearchFilter marker methods + serializeFilter default branch
// ---------------------------------------------------------------------------

func TestFileSearchFilter_MarkerMethods(t *testing.T) {
	// Exercise the marker methods to achieve coverage.
	// These are no-op methods that exist only to seal the interface.
	var comp FileSearchFilter = &FileSearchComparisonFilter{}
	comp.fileSearchFilter()

	var compound FileSearchFilter = &FileSearchCompoundFilter{}
	compound.fileSearchFilter()
}

func TestSerializeFilter_NilReturnsNil(t *testing.T) {
	// A nil FileSearchFilter should hit the default branch and return nil.
	result := serializeFilter(nil)
	if result != nil {
		t.Errorf("serializeFilter(nil) = %v, want nil", result)
	}
}

func TestTools_ImageGeneration_PartialMask(t *testing.T) {
	// Only FileID set.
	def := Tools.ImageGeneration(WithInputImageMask(ImageGenerationMask{FileID: "f1"}))
	mask := def.ProviderDefinedOptions["input_image_mask"].(map[string]any)
	if mask["file_id"] != "f1" {
		t.Errorf("file_id = %v", mask["file_id"])
	}
	if _, ok := mask["image_url"]; ok {
		t.Error("image_url should not be set")
	}

	// Only ImageURL set.
	def = Tools.ImageGeneration(WithInputImageMask(ImageGenerationMask{ImageURL: "https://x"}))
	mask = def.ProviderDefinedOptions["input_image_mask"].(map[string]any)
	if _, ok := mask["file_id"]; ok {
		t.Error("file_id should not be set")
	}
	if mask["image_url"] != "https://x" {
		t.Errorf("image_url = %v", mask["image_url"])
	}
}
