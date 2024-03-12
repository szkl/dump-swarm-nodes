package main

import (
	"context"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type BlockRangeJob = struct {
	client    *ethclient.Client
	start     int64
	end       int64
	toAddress common.Address
	txEntries chan<- *TransactionEntry
}

func NewBlockRangeJob(
	client *ethclient.Client,
	start int64,
	end int64,
	toAddress common.Address,
	txEntries chan<- *TransactionEntry,
) *BlockRangeJob {
	return &BlockRangeJob{
		client,
		start,
		end,
		toAddress,
		txEntries,
	}
}

func scanBlockRange(job *BlockRangeJob) error {
	q := ethereum.FilterQuery{
		Addresses: []common.Address{job.toAddress},
		FromBlock: big.NewInt(job.start),
		ToBlock:   big.NewInt(job.end),
	}

	logs, err := getFilterLogs(q, job.client)
	if err != nil {
		log.Fatalln(err)
	}

	count := 0
	for _, l := range logs {
		if l.Address != job.toAddress {
			continue
		}

		job.txEntries <- &TransactionEntry{
			txHash:    l.TxHash,
			blockHash: l.BlockHash,
		}
		count += 1
	}

	log.Printf("start=%d end=%d count=%d\n", job.start, job.end, count)
	return nil
}

func getFilterLogs(q ethereum.FilterQuery, client *ethclient.Client) ([]types.Log, error) {
	ctx := context.Background()
	for {
		logs, err := client.FilterLogs(ctx, q)
		if err != nil {
			if ok := checkErrorRetry(err); ok {
				continue
			}
			return nil, err
		}
		return logs, nil
	}
}

func getBlock(hash common.Hash, client *ethclient.Client) (*types.Block, error) {
	ctx := context.Background()
	for {
		block, err := client.BlockByHash(ctx, hash)
		if err != nil {
			if ok := checkErrorRetry(err); ok {
				continue
			}
			return nil, err
		}
		return block, nil
	}
}
