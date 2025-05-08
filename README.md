[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue.svg)](https://golang.org/)
[![GitHub last commit](https://img.shields.io/github/last-commit/Dorafanboy/trust_wallet_homework.svg)](https://github.com/Dorafanboy/trust_wallet_homework/commits/main)
[![GitHub repo size](https://img.shields.io/github/repo-size/Dorafanboy/trust_wallet_homework.svg)](https://github.com/Dorafanboy/trust_wallet_homework)
[![GitHub issues](https://img.shields.io/github/issues/Dorafanboy/trust_wallet_homework.svg)](https://github.com/Dorafanboy/trust_wallet_homework/issues)
[![GitHub contributors](https://img.shields.io/github/contributors/Dorafanboy/trust_wallet_homework.svg)](https://github.com/Dorafanboy/trust_wallet_homework/graphs/contributors)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](#) <!-- Replace with CI build status badge if available, e.g., GitHub Actions -->
[![Go Report Card](https://goreportcard.com/badge/github.com/dorafanboy/trust_wallet_homework)](https://goreportcard.com/report/github.com/dorafanboy/trust_wallet_homework)
[![Coverage Status](https://coveralls.io/repos/github/Dorafanboy/trust_wallet_homework/badge.svg?branch=main)](https://coveralls.io/github/Dorafanboy/trust_wallet_homework?branch=main)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# Ethereum Blockchain Parser

## Overview

This project is an Ethereum blockchain parser designed to monitor new blocks, identify transactions for subscribed addresses (both incoming and outgoing), and provide a REST API to access this information. All parsed data is currently stored in memory.

## Tech Stack

- Go (v1.24+)
- Docker & Docker Compose
- Makefile for build/task automation
- net/http (for REST API, using standard library)
- slog (for structured logging)
- testify/assert & testify/require (for testing)
- mockery (for mock generation)
- golangci-lint (for linting)

## Prerequisites

- Go (version 1.24 or later recommended)
- Docker & Docker Compose (for containerized execution)
- Make

## Installation and Building

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/Dorafanboy/trust_wallet_homework.git
    cd trust_wallet_homework
    ```

2.  **Build the application:**
    ```bash
    make build
    ```
    Alternatively, build directly with Go:
    ```bash
    go build -o parserapi ./cmd/parserapi/main.go
    ```

## Running the Application

### Configuration

The application uses a configuration file located at `config/config.yml`. Ensure this file is correctly set up before running the application. Key parameters include:

-   `server.port`: Port for the API server (e.g., `":8080"`).
-   `ethereum.rpc_url`: URL of your Ethereum JSON-RPC provider.
-   `parser.polling_interval_seconds`: Interval in seconds for polling new blocks.
-   `parser.initial_scan_block_number`: Block number to start scanning from. Use `-1` to start from the latest block, or `0` or a positive number for a specific block.

**Example `config/config.yml`:**
```yaml
server:
  port: ":8080"

ethereum:
  rpc_url: "https://cloudflare-eth.com" # Replace with your preferred RPC URL

parser:
  polling_interval_seconds: 15
  initial_scan_block_number: -1 # Start from the latest block
```

### Local Execution

To build and run the application locally:
```bash
make run
```
Or, if you have already built the binary:
```bash
./parserapi
```

### Docker Execution

1.  **Build the Docker image:**
    ```bash
    make docker-build
    ```
    *Note: If you encounter `error getting credentials` during the Docker build, ensure your Docker daemon is correctly configured, can access the internet, and can pull base images from Docker Hub (e.g., `golang:1.24-alpine`). This might involve checking Docker login status or network configuration.*

2.  **Run the container using `docker run`:**
    ```bash
    make docker-run
    ```

3.  **Run using Docker Compose (recommended for managing services):**
    ```bash
    make infra-up
    ```
    This will build the image if necessary and start the service in detached mode.

4.  **Stop services run with Docker Compose:**
    ```bash
    make infra-down
    ```

## Makefile Commands

A `Makefile` is provided for common tasks:

-   `make build`: Builds the application binary (`parserapi`).
-   `make run`: Builds and then runs the application locally.
-   `make test`: Runs all Go tests in the project.
-   `make lint`: Runs `golangci-lint` to check for code style and errors.
-   `make clean`: Removes the built binary and cleans test cache.
-   `make docker-build`: Builds the Docker image for the application.
-   `make docker-run`: Runs the application inside a Docker container (after building the image).
-   `make infra-up`: Starts the application using `docker-compose` (builds image if needed).
-   `make infra-down`: Stops the services started by `docker-compose`.
-   `make help`: Displays a list of all available `make` targets.

## API Endpoints

The following REST API endpoints are available:

-   **`GET /current_block`**
    -   Description: Returns the number of the last successfully processed block.
    -   Response: `{"block_number": 1234567}`

-   **`POST /subscribe`**
    -   Description: Subscribes a new Ethereum address for transaction monitoring.
    -   Request Body: `{"address":"0xYOUR_ETHEREUM_ADDRESS_HERE"}`
    -   Example: `curl -X POST -H "Content-Type: application/json" -d '{"address":"0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B"}' http://localhost:8080/subscribe`
    -   Success Response: `200 OK` (or `201 Created`)
    -   Error Responses: `400 Bad Request` (invalid address format), `500 Internal Server Error`.

-   **`GET /transactions/{address}`**
    -   Description: Retrieves a list of transactions associated with a given monitored Ethereum address.
    -   Example: `curl http://localhost:8080/transactions/0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B`
    -   Response: 
        ```json
        [
          {
            "hash": "0x...",
            "from": "0x...",
            "to": "0x...",
            "value": "1000000000000000000",
            "block_number": 1234560,
            "timestamp": 1600000000
          }
        ]
        ```