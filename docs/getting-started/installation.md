# Installation

## Requirements

- Go 1.25 or later

## Install

```bash
go get github.com/zendev-sh/goai@latest
```

This installs the core SDK. Provider packages are included - no separate installs needed.

## Import

```go
import (
    "github.com/zendev-sh/goai"
    "github.com/zendev-sh/goai/provider/openai"
)
```

Each provider has its own sub-package under `provider/`. Import only the providers you use:

```go
import "github.com/zendev-sh/goai/provider/anthropic"
import "github.com/zendev-sh/goai/provider/google"
import "github.com/zendev-sh/goai/provider/bedrock"
```

## Verify

Create a file `main.go`:

```go
package main

import (
    "context"
    "fmt"

    "github.com/zendev-sh/goai"
    "github.com/zendev-sh/goai/provider/openai"
)

func main() {
    model := openai.Chat("gpt-4o")

    result, err := goai.GenerateText(context.Background(), model,
        goai.WithPrompt("Say hello in one sentence."),
    )
    if err != nil {
        panic(err)
    }
    fmt.Println(result.Text)
}
```

Set your API key and run:

```bash
export OPENAI_API_KEY="sk-..."
go run main.go
```

If you see a response from the model, the installation is working.

## Dependencies

The only external dependency is `golang.org/x/oauth2`, used by the Vertex AI provider for Application Default Credentials. All other providers use the standard library.
