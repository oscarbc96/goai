//go:build ignore

// Example: OpenAI File Search tool.
//
// Searches uploaded files in a vector store using semantic/keyword search.
// Requires a pre-created vector store with uploaded files.
//
// Setup (using curl with OpenAI API):
//
//  1. Create a vector store:
//     curl -s https://api.openai.com/v1/vector_stores \
//       -H "Authorization: Bearer $OPENAI_API_KEY" \
//       -H "Content-Type: application/json" \
//       -d '{"name": "my-docs"}' | jq .id
//
//  2. Upload a file (purpose=assistants):
//     curl -s https://api.openai.com/v1/files \
//       -H "Authorization: Bearer $OPENAI_API_KEY" \
//       -F purpose=assistants \
//       -F file=@your-document.txt | jq .id
//
//  3. Add the file to the vector store:
//     curl -s https://api.openai.com/v1/vector_stores/$STORE_ID/files \
//       -H "Authorization: Bearer $OPENAI_API_KEY" \
//       -H "Content-Type: application/json" \
//       -d '{"file_id": "'$FILE_ID'"}'
//
//  4. Wait for indexing to complete (status → "completed"):
//     curl -s https://api.openai.com/v1/vector_stores/$STORE_ID \
//       -H "Authorization: Bearer $OPENAI_API_KEY" | jq .status
//
// Via OpenAI direct:
//
//	export OPENAI_API_KEY=...
//	go run examples/file-search/main.go openai <vector-store-id>
//
// Via Azure OpenAI:
//
//	export AZURE_OPENAI_API_KEY=... AZURE_RESOURCE_NAME=...
//	go run examples/file-search/main.go azure <vector-store-id>
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/azure"
	"github.com/zendev-sh/goai/provider/openai"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: go run main.go <openai|azure> <vector-store-id>")
	}
	auth := os.Args[1]
	storeID := os.Args[2]

	var model provider.LanguageModel
	switch auth {
	case "openai":
		model = openai.Chat("gpt-4.1-mini",
			openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")))
	case "azure":
		model = azure.Chat("gpt-4.1-mini",
			azure.WithAPIKey(os.Getenv("AZURE_OPENAI_API_KEY")))
	default:
		log.Fatalf("Unknown: %s (use openai or azure)", auth)
	}

	def := openai.Tools.FileSearch(
		openai.WithVectorStoreIDs(storeID),
		openai.WithMaxNumResults(5),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := goai.GenerateText(ctx, model,
		goai.WithPrompt("What information is in the uploaded documents?"),
		goai.WithMaxOutputTokens(500),
		goai.WithTools(goai.Tool{
			Name:                   def.Name,
			ProviderDefinedType:    def.ProviderDefinedType,
			ProviderDefinedOptions: def.ProviderDefinedOptions,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Text)
}
