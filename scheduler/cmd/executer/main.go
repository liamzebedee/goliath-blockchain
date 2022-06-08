package main

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/rpc"
	// "github.com/tendermint/tendermint/rpc/coretypes"
	// coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"encoding/json"
)

// Single block (with meta)
type RequestBlockInfo struct {
	Height int64 `json:"height,omitempty"`
}

func (args RequestBlockInfo) String() string {
	s, err := json.Marshal(args)
	if err == nil {
		return string(s)
	}
	return err.Error()
}



type ResultBlock struct {
	Block *Block `json:"block"`
}

type Block struct {
	Header *BlockHeader `json:"header"`
}

type BlockHeader struct {
	Height string `json:"height"`
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

	// Using latest Tendermint package.
	// client, err := httpclient.New(endpoint)
	// if err != nil {
	// 	panic(err)
	// }
	// var height int64 = 2
	// ctx := context.Background()
	// res, err := client.Block(ctx, &height)
	// if err != nil {
	// 	panic(err)
	// }
	
	// req := RequestBlockInfo{
	// 	height: 2,
	// }
	// params := make(map[string]interface{})
	// params["height"] = 2

	// var res coretypes.ResultBlock

	// Get latest block.
	// req := make(map[string]interface{})
	// var res map[string]interface{}
	// var res coretypes.ResultBlock
	var res ResultBlock
	
	// fmt.Println(json.Marshal(&req))
	
	err = client.Call(&res, "block", nil)
	if err != nil {
		panic(err)
	}

	// latestBlock := res["block"]["header"]["height"]
	lastHeight, err := strconv.Atoi(res.Block.Header.Height)
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("latest_height=%d\n", lastHeight)
	fmt.Printf("syncing %d blocks...\n", lastHeight - 1)
	
	for i := 1; i < lastHeight; i++ {
		req := make(map[string]interface{})
		// req := RequestBlockInfo{
		// 	Height: int64(i),
		// }
		
		req["height"] = i
		
		err = client.Call(&res, "block", strconv.FormatInt(int64(i), 10))
		if err != nil {
			panic(err.Error())
		}
	}



	fmt.Println("Hey", lastHeight)
}