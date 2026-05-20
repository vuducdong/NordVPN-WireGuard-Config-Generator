# NordGen Backend

A minimalist, high-performance backend service for generating NordVPN WireGuard configurations. Built on the **Go** programming language and the **Fiber** framework, this service handles server data caching, credential exchange, and configuration generation with extreme efficiency.

## Overview

This application serves as the API layer for the NordGen project. It interfaces directly with NordVPN's infrastructure to retrieve server lists and exchange authentication tokens for WireGuard private keys. It provides endpoints to generate configuration files in text, file, or QR code formats.

## Prerequisites

- Go 1.26+ (Recommended)

## Installation

Clone the repository and download the dependencies:

```bash
git clone https://github.com/mustafachyi/NordVPN-WireGuard-Config-Generator
cd NordVPN-WireGuard-Config-Generator/Web/web-Backend
go mod download
```

## Development

Start the server directly using the Go toolchain:

```bash
go run main.go
```

The server listens on port `3000` by default.

## Production

To build and run the optimized production binary:

```bash
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o server main.go
./server
```

## Static Assets

The server looks for a `./public` directory to serve static frontend files. If an `index.html` is present, it is served for the root path and any unknown routes (SPA fallback), with the server data injected directly into the HTML to prevent an initial round-trip fetch.

## API Documentation

For detailed endpoint specifications, request/response formats, and validation rules, please refer to the [API.md](./API.md).