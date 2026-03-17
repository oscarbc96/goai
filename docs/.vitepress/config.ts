import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'GoAI',
  description: 'AI SDK, the Go way.',
  base: process.env.VITEPRESS_BASE || '/',

  markdown: {
    theme: {
      light: 'github-light',
      dark: 'github-dark',
    },
  },
  head: [
    ['link', { rel: 'icon', type: 'image/png', href: '/goai-icon.png' }],
    ['meta', { property: 'og:title', content: 'GoAI' }],
    ['meta', { property: 'og:description', content: 'AI SDK, the Go way.' }],
    ['meta', { property: 'og:image', content: '/goai.png' }],
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
  ],

  themeConfig: {
    logo: '/goai-icon.png',
    siteTitle: 'GoAI',

    nav: [
      { text: 'Guide', link: '/getting-started/installation' },
      { text: 'Providers', link: '/providers/' },
      { text: 'API', link: '/api/core-functions' },
      { text: 'Examples', link: '/examples' },
      {
        text: 'Links',
        items: [
          { text: 'GitHub', link: 'https://github.com/zendev-sh/goai' },
          { text: 'GoDoc', link: 'https://pkg.go.dev/github.com/zendev-sh/goai' },
        ],
      },
    ],

    sidebar: {
      '/': [
        {
          text: 'Getting Started',
          items: [
            { text: 'Installation', link: '/getting-started/installation' },
            { text: 'Quick Start', link: '/getting-started/quick-start' },
            { text: 'Structured Output', link: '/getting-started/structured-output' },
          ],
        },
        {
          text: 'Concepts',
          items: [
            { text: 'Providers & Models', link: '/concepts/providers-and-models' },
            { text: 'Streaming', link: '/concepts/streaming' },
            { text: 'Tools', link: '/concepts/tools' },
            { text: 'Provider-Defined Tools', link: '/concepts/provider-tools' },
            { text: 'TokenSource', link: '/concepts/token-source' },
            { text: 'Error Handling', link: '/concepts/error-handling' },
            { text: 'Prompt Caching', link: '/concepts/prompt-caching' },
          ],
        },
        {
          text: 'Providers',
          collapsed: false,
          items: [
            { text: 'Overview', link: '/providers/' },
            {
              text: 'Tier 1',
              items: [
                { text: 'OpenAI', link: '/providers/openai' },
                { text: 'Anthropic', link: '/providers/anthropic' },
                { text: 'Google', link: '/providers/google' },
                { text: 'AWS Bedrock', link: '/providers/bedrock' },
                { text: 'Azure', link: '/providers/azure' },
                { text: 'Vertex AI', link: '/providers/vertex' },
              ],
            },
            {
              text: 'Tier 2',
              items: [
                { text: 'Cohere', link: '/providers/cohere' },
                { text: 'Mistral', link: '/providers/mistral' },
                { text: 'xAI (Grok)', link: '/providers/xai' },
                { text: 'Groq', link: '/providers/groq' },
                { text: 'DeepSeek', link: '/providers/deepseek' },
              ],
            },
            {
              text: 'Tier 3',
              items: [
                { text: 'Fireworks', link: '/providers/fireworks' },
                { text: 'Together', link: '/providers/together' },
                { text: 'DeepInfra', link: '/providers/deepinfra' },
                { text: 'OpenRouter', link: '/providers/openrouter' },
                { text: 'Perplexity', link: '/providers/perplexity' },
                { text: 'Cerebras', link: '/providers/cerebras' },
              ],
            },
            {
              text: 'Local / Custom',
              items: [
                { text: 'Ollama', link: '/providers/ollama' },
                { text: 'vLLM', link: '/providers/vllm' },
                { text: 'Compatible', link: '/providers/compat' },
              ],
            },
          ],
        },
        {
          text: 'API Reference',
          items: [
            { text: 'Core Functions', link: '/api/core-functions' },
            { text: 'Types', link: '/api/types' },
            { text: 'Options', link: '/api/options' },
            { text: 'Errors', link: '/api/errors' },
          ],
        },
        {
          text: 'Examples',
          link: '/examples',
        },
      ],
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/zendev-sh/goai' },
    ],

    search: {
      provider: 'local',
    },

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2026 GoAI',
    },

    editLink: {
      pattern: 'https://github.com/zendev-sh/goai/edit/main/docs/:path',
      text: 'Edit this page on GitHub',
    },
  },
})
