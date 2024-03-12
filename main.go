package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

const DEFAULT_RPC_PROVIDER = "https://rpc.gnosischain.com"
const CONTRACT_ADDRESS_HEX = "0xc2d5a532cf69aa9a1378737d8ccdef884b6e7420"
const DEFAULT_START_BLOCK_NUMBER = 16515646 + 1 // Next one after the contract is created

func main() {
	outFilename := flag.String("out-file", "out.txt", "out file")
	startBlockNumber := flag.Int64("start", DEFAULT_START_BLOCK_NUMBER, "start block number")
	endBlockNumber := flag.Int64("end", 0, "end block number (default \"latest\")")
	rpcProvider := flag.String("rpc-provider", DEFAULT_RPC_PROVIDER, "RPC provider")
	workerCount := flag.Int64("worker-count", int64(runtime.GOMAXPROCS(0)), "worker count")

	flag.Parse()

	client, err := ethclient.Dial(*rpcProvider)
	if err != nil {
		log.Fatal(err)
	}

	if *endBlockNumber == 0 {
		latestBlockNumber, err := getLatestBlockNumber(client)
		if err != nil {
			log.Fatalln(err)
		}
		*endBlockNumber = int64(latestBlockNumber)
	}

	outFile, closeOutFile, err := getOutFile(*outFilename)
	defer closeOutFile()

	done := make(chan bool)
	results := make(chan string)

	go runResultWorker(results, outFile, done)

	startJobs(
		*startBlockNumber,
		*endBlockNumber,
		client,
		results,
		*workerCount,
	)

	<-done
}

func startJobs(
	start int64,
	end int64,
	client *ethclient.Client,
	results chan<- string,
	workerCount int64,
) {
	blockWorkerCount := int64(0)
	if end == start {
		blockWorkerCount = 1
	} else if (end - start) < workerCount {
		blockWorkerCount = end - start
	} else {
		blockWorkerCount = workerCount
	}

	workerBlockRange := int64(0)
	if (end - start) <= blockWorkerCount {
		workerBlockRange = 1
	} else {
		workerBlockRange = (end - start) / workerCount
	}

	toAddress := common.HexToAddress(CONTRACT_ADDRESS_HEX)
	txEntries := make(chan *TransactionEntry, 100*workerCount)

	wgBlock := &sync.WaitGroup{}
	for i := int64(0); i < blockWorkerCount; i++ {
		workerStart := start + (i * workerBlockRange)
		workerEnd := workerStart + workerBlockRange - 1
		if i == blockWorkerCount-1 {
			workerEnd = end
		}

		wgBlock.Add(1)
		go func() {
			defer wgBlock.Done()
			job := NewBlockRangeJob(client, workerStart, workerEnd, toAddress, txEntries)
			scanBlockRange(job)
		}()
	}

	wgTx := &sync.WaitGroup{}
	for i := int64(0); i < workerCount; i++ {
		wgTx.Add(1)
		go func() {
			defer wgTx.Done()
			job := NewTransactionJob(client, txEntries, results)
			processTransaction(job)
		}()
	}

	wgBlock.Wait()
	log.Println("Finished scanning block ranges")
	close(txEntries)

	wgTx.Wait()
	log.Println("Finished processing transactions")
	close(results)
}

func getLatestBlockNumber(client *ethclient.Client) (uint64, error) {
	blockNumber, err := client.BlockNumber(context.Background())
	if err != nil {
		return 0, err
	}
	return blockNumber, nil
}

// checkErrorRetry checks if it is a retriable error then sleeps for
// 10 seconds. The status codes 401 and 429 are usually used when the
// request can be retried in this program. The return value signals
// the caller whether it can be retried or not.
func checkErrorRetry(err error) bool {
	if rpcErr, ok := err.(rpc.HTTPError); ok {
		switch rpcErr.StatusCode {
		case http.StatusUnauthorized:
			fallthrough
		case http.StatusTooManyRequests:
			fallthrough
		case http.StatusBadGateway:
			fallthrough
		case http.StatusServiceUnavailable:
			fallthrough
		case http.StatusGatewayTimeout:
			time.Sleep(5 * time.Second)
			return true
		}
	}

	if _, ok := err.(*url.Error); ok {
		time.Sleep(5 * time.Second)
		return true
	}

	if _, ok := err.(*net.OpError); ok {
		time.Sleep(5 * time.Second)
		return true
	}

	return false
}
