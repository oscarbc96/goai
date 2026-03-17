# Providers

GoAI ships 20 providers organized by tier.

## Tier 1

Dedicated implementations with extended API support.

| Provider | API | Features |
|----------|-----|----------|
| [OpenAI](openai.md) | Chat Completions + Responses API | Embedding, Image, 4 provider tools |
| [Anthropic](anthropic.md) | Messages API | 10 provider tools, thinking, cache control |
| [Google](google.md) | Gemini REST API | Embedding, Image, 3 provider tools, thinking |
| [Bedrock](bedrock.md) | AWS Converse API | SigV4, EventStream, multi-vendor |
| [Azure](azure.md) | Multi-endpoint routing | OpenAI + Claude + AI Services, Image |
| [Vertex AI](vertex.md) | Vertex AI + Gemini fallback | Embedding, Image, ADC auth |

## Tier 2

| Provider | API | Features |
|----------|-----|----------|
| [Cohere](cohere.md) | Native Chat v2 + Embed API | Embedding, citations, reasoning |
| [Mistral](mistral.md) | OpenAI-compatible | |
| [xAI (Grok)](xai.md) | OpenAI-compatible | 2 provider tools (pending Responses API) |
| [Groq](groq.md) | OpenAI-compatible | BrowserSearch tool |
| [DeepSeek](deepseek.md) | OpenAI-compatible | Reasoning (R1) |

## Tier 3

All use the shared `internal/openaicompat` codec.

| Provider | Endpoint | Special Features |
|----------|----------|-----------------|
| [Fireworks](fireworks.md) | `api.fireworks.ai` | |
| [Together](together.md) | `api.together.xyz` | |
| [DeepInfra](deepinfra.md) | `api.deepinfra.com` | |
| [OpenRouter](openrouter.md) | `openrouter.ai` | Multi-provider routing |
| [Perplexity](perplexity.md) | `api.perplexity.ai` | Search-augmented, citations |
| [Cerebras](cerebras.md) | `api.cerebras.ai` | |

## Local / Custom

| Provider | Default Endpoint | Features |
|----------|-----------------|----------|
| [Ollama](ollama.md) | `localhost:11434` | Embedding, no auth required |
| [vLLM](vllm.md) | `localhost:8000` | Embedding, optional auth |
| [Generic Compatible](compat.md) | (required) | Any OpenAI-compatible endpoint |

## Common Options

All providers support these options (except where noted):

```go
openai.WithAPIKey(key)        // Static API key
openai.WithTokenSource(ts)    // Dynamic auth (OAuth, service accounts)
openai.WithBaseURL(url)       // Override endpoint
openai.WithHeaders(h)         // Custom HTTP headers
openai.WithHTTPClient(c)      // Custom HTTP transport
```

Each provider package exports its own `With*` functions with the same signatures.
