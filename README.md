# dump-swarm-nodes

```
./dump-swarm-nodes -h
Usage of ./dump-swarm-nodes:
  -end int
    	end block number (default "latest")
  -out-file string
    	out file (default "out.txt")
  -rpc-provider string
    	RPC provider (default "https://rpc.gnosischain.com")
  -start int
    	start block number (default 16515647)
  -worker-count int
    	worker count (default 10)
```

The program can be run with the options described above.

A quick run can be done with the following commands:

```
go build
./dump-swarm-nodes -start 32923118 -end 32923122
```

https://github.com/szkl/dump-swarm-nodes/assets/1136739/7b74e42a-16e2-48f9-9b35-219f7bcddb87

## Speed

Tests with 10 cores:

- from block 30500962 to 32926204 (2.425.242 blocks) took about 15
  minutes.
  - The out file had 11872 lines. ~620 transactions were not found on
    RPC provider.
  
- from block 27419649 to 30500962 (3.081.313 blocks) took about 3
  minutes 41 seconds.
  - Most of the transactions were not found on RPC provider.
  - Rerunning this block range resulted in 5 times more transactions.
  
It could take about 2~3 hours to find all transactions assuming
they're known to the RPC provider.

## Issues

- `ethClient.Client.FilterLogs` do not return any logs for some old
blocks. Example: 18687544

  - The block hash do not match with the hash read from `ethclient`.

- RPC provider returns not found for some transactions.
