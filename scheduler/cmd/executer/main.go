package main

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/rpc"
	// "github.com/tendermint/tendermint/rpc/coretypes"
	// coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

// Single block (with meta)
type RequestBlockInfo struct {
	Height int64 `json:"height,omitempty"`
}

type ResultBlock struct {
	Block *Block `json:"block"`
}

type Block struct {
	Header *BlockHeader `json:"header"`
}

type BlockHeader struct {
	Height string `json:"height"`
	NumTxs string `json:"num_txs"`
}

func intToStr(i int) string {
	return strconv.FormatInt(int64(i), 10)
}

type ResultBlockchain struct {
	BlockMetas []BlockMeta `json:"block_metas"`
}

type BlockMeta struct {
	Header *BlockHeader `json:"header"`
}

func main() {
	endpoint := "http://0.0.0.0:54046/"

	fmt.Printf("goliath executer\n")
	fmt.Printf("connecting to sequencer chain on %s\n", endpoint)

	// Read all blocks up until a point, and execute their txs.

	client, err := rpc.Dial(endpoint)
	if err != nil {
		panic("Couldn't connect to RPC endpoint")
	}

	fmt.Printf("connected to sequencer\n")


	var res ResultBlock
	err = client.Call(&res, "block", nil)
	if err != nil {
		panic(err)
	}

	lastHeight, err := strconv.Atoi(res.Block.Header.Height)
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("latest_height=%d\n", lastHeight)
	fmt.Printf("syncing %d blocks...\n", lastHeight - 1)
	
	for i := 1; i < lastHeight; {
		var res ResultBlockchain

		minHeight := i
		maxHeight := i + 1000

		err = client.Call(&res, "blockchain", intToStr(minHeight), intToStr(maxHeight))
		if err != nil {
			panic(err.Error())
		}

		for _, meta := range res.BlockMetas {
			num, err := strconv.Atoi(meta.Header.NumTxs)
			if err != nil { panic(err) }
			if num > 0 {
				fmt.Printf("block=%s n(txs)=%d\n", meta.Header.Height, num)
			}
		}

		i += 1000

		if ((i - 1) % 100) == 0 {
			fmt.Printf("synced %d/%d blocks...\n", i, lastHeight - 1)
		}
	}

	// For each block, we check if there were txs.
	// If there were, we load them from state.



	fmt.Println("Hey", lastHeight)
}