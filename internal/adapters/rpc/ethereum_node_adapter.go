// Package rpc implements an Ethereum client using JSON-RPC communication with an Ethereum node.
package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"trust_wallet_homework/internal/core/domain"
	"trust_wallet_homework/internal/core/domain/client"
)

// EthereumNodeAdapter implements the client.EthereumClient interface by making JSON-RPC calls to an Ethereum node.
type EthereumNodeAdapter struct {
	rpcURL     string
	httpClient *http.Client
	requestID  int
}

// Compile-time check to ensure EthereumNodeAdapter implements client.EthereumClient
var _ client.EthereumClient = (*EthereumNodeAdapter)(nil)

// NewEthereumNodeAdapter creates a new RPC adapter.
func NewEthereumNodeAdapter(rpcURL string, httpClient *http.Client) *EthereumNodeAdapter {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &EthereumNodeAdapter{
		rpcURL:     rpcURL,
		httpClient: httpClient,
		requestID:  0,
	}
}

// GetLatestBlockNumber fetches the number of the most recent block.
func (a *EthereumNodeAdapter) GetLatestBlockNumber(ctx context.Context) (domain.BlockNumber, error) {
	respBody, err := a.doRPC(ctx, "eth_blockNumber", []interface{}{})
	if err != nil {
		return domain.BlockNumber{}, fmt.Errorf("RPC call failed: %w", err)
	}

	if respBody.Result == nil {
		return domain.BlockNumber{}, fmt.Errorf("RPC result is null for eth_blockNumber")
	}

	var resultStr string
	if err := json.Unmarshal(respBody.Result, &resultStr); err != nil {
		return domain.BlockNumber{}, fmt.Errorf("failed to unmarshal block number result: %w", err)
	}

	blockNumberInt, err := hexToInt64(resultStr)
	if err != nil {
		return domain.BlockNumber{}, fmt.Errorf("failed to parse block number hex '%s': %w", resultStr, err)
	}

	return domain.NewBlockNumber(blockNumberInt)
}

// GetBlockWithTransactions fetches a block by its number and includes its transactions.
func (a *EthereumNodeAdapter) GetBlockWithTransactions(
	ctx context.Context,
	blockNumber domain.BlockNumber,
) (*domain.Block, error) {
	blockNumberHex := fmt.Sprintf("0x%x", blockNumber.Value())
	params := []interface{}{blockNumberHex, true}

	respBody, err := a.doRPC(ctx, "eth_getBlockByNumber", params)
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	if respBody.Result == nil {
		log.Printf("Received null result for block %d (0x%x)", blockNumber.Value(), blockNumber.Value())
		return nil, nil
	}

	var rpcBlock *Block
	if err := json.Unmarshal(respBody.Result, &rpcBlock); err != nil {
		log.Printf("Error unmarshaling block %d (0x%x): %v. JSON: %s",
			blockNumber.Value(),
			blockNumber.Value(),
			err,
			string(respBody.Result),
		)
		return nil, fmt.Errorf("failed to unmarshal block result for block %s: %w. JSON: %s",
			blockNumberHex,
			err,
			string(respBody.Result),
		)
	}

	if rpcBlock == nil {
		log.Printf("Block %d unmarshalled to nil unexpectedly (after non-null raw result for 0x%x)\n",
			blockNumber.Value(),
			blockNumber.Value(),
		)
		return nil, nil
	}

	return mapRPCBlockToDomain(rpcBlock)
}

// doRPC performs the actual JSON-RPC call.
func (a *EthereumNodeAdapter) doRPC(
	ctx context.Context,
	method string,
	params []interface{},
) (*JSONRPCResponse, error) {
	a.requestID++
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      a.requestID,
	}

	jsonReqBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.rpcURL, bytes.NewBuffer(jsonReqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := a.httpClient.Do(httpReq.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	if httpResp.Body != nil {
		defer func() {
			if errClose := httpResp.Body.Close(); errClose != nil {
				log.Printf("[WARN] Failed to close response body in doRPC for method %s: %v", method, errClose)
			}
		}()
	}

	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP request failed with status %s: %s", httpResp.Status, string(bodyBytes))
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RPC response: %w, body: %s", err, string(bodyBytes))
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: code=%d, message='%s'", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return &rpcResp, nil
}

// mapRPCBlockToDomain converts the RPC DTO for a block to the domain model.
func mapRPCBlockToDomain(rpcBlock *Block) (*domain.Block, error) {
	num, err := hexToInt64(rpcBlock.Number)
	if err != nil {
		return nil, fmt.Errorf("invalid block number hex '%s': %w", rpcBlock.Number, err)
	}
	domainBlockNum, err := domain.NewBlockNumber(num)
	if err != nil {
		return nil, fmt.Errorf("failed creating domain block number: %w", err)
	}

	domainBlockHash, err := domain.NewBlockHash(rpcBlock.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed creating domain block hash: %w", err)
	}

	timestamp, err := hexToUint64(rpcBlock.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("invalid block timestamp hex '%s': %w", rpcBlock.Timestamp, err)
	}

	domainTxs := make([]domain.Transaction, 0, len(rpcBlock.Transactions))
	for i, rpcTx := range rpcBlock.Transactions {
		domainTx, err := mapRPCTransactionToDomain(&rpcTx, domainBlockNum, timestamp)
		if err != nil {
			fmt.Printf("Error mapping transaction index %d (hash: %s) in block %d: %v\n", i, rpcTx.Hash, num, err)
			continue
		}
		domainTxs = append(domainTxs, *domainTx)
	}

	domainBlock := domain.NewBlock(domainBlockNum, domainBlockHash, timestamp, domainTxs)
	return &domainBlock, nil
}

// mapRPCTransactionToDomain converts the RPC DTO for a transaction to the domain model.
func mapRPCTransactionToDomain(
	rpcTx *Transaction,
	blockNum domain.BlockNumber,
	blockTimestamp uint64,
) (*domain.Transaction, error) {
	hash, err := domain.NewTransactionHash(rpcTx.Hash)
	if err != nil {
		return nil, fmt.Errorf("invalid tx hash '%s': %w", rpcTx.Hash, err)
	}

	from, err := domain.NewAddress(rpcTx.From)
	if err != nil {
		return nil, fmt.Errorf("invalid tx from address '%s': %w", rpcTx.From, err)
	}

	var to domain.Address
	if rpcTx.To != nil && *rpcTx.To != "" {
		to, err = domain.NewAddress(*rpcTx.To)
		if err != nil {
			return nil, fmt.Errorf("invalid tx to address '%s': %w", *rpcTx.To, err)
		}
	}

	value, err := domain.NewWeiValue(rpcTx.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid tx value '%s': %w", rpcTx.Value, err)
	}

	domainTx := domain.NewTransaction(hash, from, to, value, blockNum, blockTimestamp)
	return &domainTx, nil
}

// hexToInt64 converts a hex string (e.g., "0x1a") to int64.
func hexToInt64(hexStr string) (int64, error) {
	cleaned := strings.TrimPrefix(strings.ToLower(hexStr), "0x")
	if cleaned == "" {
		return 0, fmt.Errorf("empty hex string")
	}
	return strconv.ParseInt(cleaned, 16, 64)
}

// hexToUint64 converts a hex string (e.g., "0x1a") to uint64.
func hexToUint64(hexStr string) (uint64, error) {
	cleaned := strings.TrimPrefix(strings.ToLower(hexStr), "0x")
	if cleaned == "" {
		return 0, fmt.Errorf("empty hex string")
	}
	if cleaned == "0" {
		return 0, nil
	}
	return strconv.ParseUint(cleaned, 16, 64)
}
