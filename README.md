# PicoLM Server

An OpenAI-compatible HTTP API server for PicoLM - an ultra-lightweight LLM inference engine.

## Overview

PicoLM Server exposes [PicoLM](https://github.com/RightNow-AI/picolm/) functionality over HTTP via an OpenAI-compatible REST API. This enables applications using OpenAI client libraries to seamlessly switch to local LLM inference with minimal code changes.

## Features

- **OpenAI-compatible API** - Works with existing OpenAI client libraries
- **Streaming support** - Real-time token streaming via SSE
- **Tool calling** - Function calling support for AI agents
- **Lightweight** - Minimal resource overhead (~8MB binary)
- **Portable** - Runs on Linux, macOS, Windows, ARM devices

## Quick Start

### Prerequisites

- Go 1.21+
- [PicoLM binary](https://github.com/RightNow-AI/picolm/)
- A GGUF model file (e.g., TinyLlama)

### Installation

#### From Source

```bash
git clone https://github.com/picolm/picolm-server.git
cd picolm-server
go build -o picolm-server ./cmd/server/
```

#### From Release

Download the latest binary from [Releases](https://github.com/picolm/picolm-server/releases)

### Configuration

```bash
cp config.example.yaml config.yaml
```

Edit `config.yaml` with your settings:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  api_key: ""  # Optional: set an API key for authentication

picolm:
  binary: "/path/to/picolm"           # Path to picolm binary
  model_path: "/path/to/model.gguf"    # Path to GGUF model
  max_tokens: 256
  threads: 4
  temperature: 0.7
  top_p: 0.9
  context_length: 2048
```

### Run

```bash
./picolm-server -config config.yaml
```

Or with Docker:

```bash
docker-compose up -d
```

## API Reference

### Chat Completions

**Endpoint:** `POST /v1/chat/completions`

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "picolm-local",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "temperature": 0.7
  }'
```

### List Models

**Endpoint:** `GET /v1/models`

```bash
curl http://localhost:8080/v1/models
```

### Streaming

Set `stream: true` in your request for streaming responses:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "picolm-local",
    "messages": [{"role": "user", "content": "Count to 5"}],
    "stream": true
  }'
```

### Tool Calling

Define tools in your request for function calling:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "picolm-local",
    "messages": [{"role": "user", "content": "What is the weather?"}],
    "tools": [{
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get current weather",
        "parameters": {
          "type": "object",
          "properties": {"location": {"type": "string"}},
          "required": ["location"]
        }
      }
    }]
  }'
```

## Using with OpenAI Clients

### Python

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="dummy"  # or your configured API key
)

response = client.chat.completions.create(
    model="picolm-local",
    messages=[{"role": "user", "content": "Hello!"}]
)

print(response.choices[0].message.content)
```

### Node.js

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'dummy'
});

const response = await client.chat.completions.create({
  model: 'picolm-local',
  messages: [{role: 'user', content: 'Hello!'}]
});

console.log(response.choices[0].message.content);
```

### Go

```go
package main

import (
	"context"
	"fmt"
	openai "github.com/sashabaranov/go-openai"
)

func main() {
	client := openai.NewClientWithConfig(openai.DefaultConfig("dummy"))
	client.BaseURL = "http://localhost:8080/v1"

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "picolm-local",
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: "Hello!"},
			},
		},
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(resp.Choices[0].Message.Content)
}
```

## Development

### Build

```bash
make build
```

### Test

```bash
make test
```

### Lint

```bash
make lint
```

### Run

```bash
make run
```

### Docker

```bash
make docker-build
make docker-compose
```

## Docker

### Quick Start

```bash
docker-compose up --build
```

### Configuration for Docker

Create `config.yaml` with Docker-specific paths:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

picolm:
  binary: "/usr/local/bin/picolm"
  model_path: "/models/model.gguf"
  timeout_seconds: 300
  max_tokens: 256
  threads: 4
  temperature: 0.7
  top_p: 0.9

logging:
  level: "info"
  format: "text"
  output: "stdout"
  log_requests: true
```

### Providing a Model

The Docker image includes picolm but you need to provide a GGUF model file:

**Option 1: Download during build**
```bash
MODEL_URL=https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf \
docker-compose build
```

**Option 2: Mount your own model**
```bash
mkdir -p models
cp your-model.gguf models/model.gguf
docker-compose up
```

### Build Arguments

| Argument | Default | Description |
|----------|---------|-------------|
| `MODEL_URL` | (none) | URL to download model from during build |
| `MODEL_NAME` | `model.gguf` | Filename for downloaded model |

### Architecture

The Dockerfile automatically detects your architecture and builds picolm accordingly:
- `arm64` → `make pi`
- `arm` → `make pi-arm32`
- `x86_64` → `make native`

## Architecture

```
picolm-server/
├── cmd/server/main.go      # Entry point
├── pkg/
│   ├── config/            # Configuration loading
│   ├── handlers/          # HTTP handlers
│   ├── picolm/            # PicoLM client (subprocess)
│   └── types/             # OpenAI API types
├── Dockerfile
├── docker-compose.yaml
├── Makefile
└── CONTRIBUTING.md
```

## Performance

PicoLM is designed for ultra-lightweight inference:

| Metric | Value |
|--------|-------|
| Binary size | ~8MB |
| RAM usage | ~45MB |
| Startup time | <1s |

First token latency depends on your hardware and model size.

## Compatibility

Tested with models that support ChatML format:
- TinyLlama
- Phi-2
- Qwen
- Other ChatML-fine-tuned models

## Related Projects

- [PicoLM](https://github.com/RightNow-AI/picolm) - Ultra-lightweight LLM inference engine
- [PicoClaw](https://github.com/sipeed/picoclaw) - AI Assistant using PicoLM

## License

MIT License - see [LICENSE](LICENSE)

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.
