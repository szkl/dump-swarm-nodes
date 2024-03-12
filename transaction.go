package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type TransactionEntry = struct {
	blockHash common.Hash
	txHash    common.Hash
}

type TransactionJob = struct {
	client *ethclient.Client
	in     <-chan *TransactionEntry
	out    chan<- string
}

func NewTransactionJob(
	client *ethclient.Client,
	in <-chan *TransactionEntry,
	out chan<- string,
) *TransactionJob {
	return &TransactionJob{client, in, out}
}

func processTransaction(job *TransactionJob) {
	for entry := range job.in {
		tx, err := getTransaction(entry.txHash, job.client)
		if err != nil {
			// RPC provider does not know the transaction
			if err == ethereum.NotFound {
				continue
			}
			log.Printf("getTransaction: hash=%s err=%s\n", entry.txHash, err)
			continue
		}

		sender, err := getTransactionSender(tx)
		if err != nil {
			log.Printf("getTransactionSender: hash=%s err=%s\n", entry.txHash, err)
			continue
		}

		// Block timestamps can be cached
		block, err := getBlock(entry.blockHash, job.client)
		if err != nil {
			log.Printf("getBlock: block=%s tx=%s err=%s", entry.blockHash, entry.txHash, err)
			continue
		}

		job.out <- fmt.Sprintf("%s %d", sender.Hex(), block.Time())
	}
}

func getTransaction(hash common.Hash, client *ethclient.Client) (*types.Transaction, error) {
	ctx := context.Background()
	for {
		tx, _, err := client.TransactionByHash(ctx, hash)
		if err != nil {
			if ok := checkErrorRetry(err); ok {
				continue
			}
			return nil, err
		}

		return tx, nil
	}
}

func getTransactionSender(tx *types.Transaction) (common.Address, error) {
	var signer types.Signer
	switch tx.Type() {
	case types.AccessListTxType:
		signer = types.NewEIP2930Signer(tx.ChainId())
	case types.DynamicFeeTxType:
		signer = types.NewLondonSigner(tx.ChainId())
	default:
		signer = types.NewEIP155Signer(tx.ChainId())
	}

	return types.Sender(signer, tx)
}
