# Coord

[![GitHub](https://img.shields.io/github/license/lemon-mint/coord?style=for-the-badge)](https://github.com/lemon-mint/coord/blob/main/LICENSE)
[![Go Reference](https://img.shields.io/badge/go-reference-%23007d9c?style=for-the-badge&logo=go)](https://pkg.go.dev/github.com/lemon-mint/coord)

![Coord](https://github.com/lemon-mint/coord/assets/55233766/59c5db0a-f496-4c34-9838-a321bac7a8b4)

## What is Coord?

Coord is a Go library designed to simplify interactions with various AI services, providing a unified interface for Large Language Models (LLMs), Text-to-Speech (TTS) systems, and Embedding models.

This allows developers to seamlessly integrate and utilize different AI services without grappling with the complexities of each provider's specific APIs and requirements.

## Key Features

- **Unified Interface:**  Interact with LLMs, TTS, and Embedding models using a consistent API, reducing code complexity and learning curves.
- **Abstraction:**  Coord handles the intricacies of model communication, data formatting, and result processing, letting you focus on your application logic.
- **Flexibility:**  Easily switch between different LLM, TTS, or Embedding providers without significant code changes.

## Use Cases

Coord is ideal for a wide range of AI-powered applications, including:

- **Chatbots and Conversational AI:** Build interactive chatbots that leverage the power of LLMs for natural language understanding and generation.
- **Content Generation:** Generate high-quality text, articles, summaries, and more using various LLM providers.
- **Speech Synthesis:**  Integrate natural-sounding speech into your applications with support for different TTS engines.
- **Semantic Search and Recommendation:**  Utilize embedding models to power features like semantic search, similarity comparisons, and personalized recommendations.

## Modules

### LLM

- Provides a standardized way to interact with various LLMs.
- Supports streaming responses, chat history management, and function calling for enhanced interaction design.

### TTS

- Offers a unified interface for text-to-speech synthesis.
- Supports different audio formats (MP3, WAV, OGG, etc.) for flexible output.

### Embedding

- Simplifies working with embedding models for text representation.
- Supports various embedding tasks, including semantic similarity, classification, and clustering.

## Getting Started

- **Installation:** `go get -u github.com/lemon-mint/coord`
- **Documentation:** [https://pkg.go.dev/github.com/lemon-mint/coord](https://pkg.go.dev/github.com/lemon-mint/coord)
- **Examples:** Explore the examples directory for practical implementations.

## Contributions

Contributions to Coord are welcome!  Please submit issues or pull requests to help improve and expand the library.
