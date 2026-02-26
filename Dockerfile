FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git gcc make curl

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .

ARG TARGETARCH
ARG MODEL_URL
ARG MODEL_NAME=model.gguf

RUN CGO_ENABLED=0 GOOS=linux go build -o picolm-server ./cmd/server/

RUN git clone --depth 1 https://github.com/RightNow-AI/picolm.git /tmp/picolm && \
    cd /tmp/picolm && \
    if [ "$TARGETARCH" = "arm64" ]; then \
        make pi; \
    elif [ "$TARGETARCH" = "arm" ]; then \
        make pi-arm32; \
    else \
        make native; \
    fi

RUN mkdir -p /models && \
    if [ -n "$MODEL_URL" ]; then \
        curl -L -o /models/$MODEL_NAME "$MODEL_URL"; \
    fi

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/picolm-server .
COPY --from=builder /tmp/picolm/picolm /usr/local/bin/picolm
RUN chmod +x /usr/local/bin/picolm

COPY --from=builder /models /models

COPY config.example.yaml ./config.yaml

EXPOSE 8080

CMD ["./picolm-server", "-config", "config.yaml"]
