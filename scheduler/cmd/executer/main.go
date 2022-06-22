package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/golang/protobuf/proto"

	"github.com/liamzebedee/goliath-blockchain/sequencer/mvp/sequencer/messages"
)

func main() {
	endpoint := "http://0.0.0.0:49000/"
	
	fmt.Printf("goliath executer\n")
	fmt.Printf("connecting to sequencer chain on %s\n", endpoint)
	
	// Read all blocks up until a point, and execute their txs.
	client, err := rpc.Dial(endpoint)
	if err != nil {
		panic("Couldn't connect to RPC endpoint")
	}

	fmt.Printf("connected to sequencer\n")

	var res messages.GetSequencerInfo
	var buf []byte
	err = client.Call(&buf, "sequencer_info")
	if err != nil {
		panic(err)
	}

	err = proto.Unmarshal(buf, &res)
	if err != nil {
		panic(err)
	}

	height := res.Count
	fmt.Printf("latest_height=%d\n", height)
	fmt.Printf("syncing %d blocks...\n", height)
	
	var i uint64
	
	for i = 0; i < height; {
		minHeight := i
		maxHeight := i + 1000

		buf := []byte{}
		var res messages.GetTransactions

		err = client.Call(&buf, "sequencer_get", minHeight, maxHeight)
		if err != nil {
			panic(err.Error())
		}

		err = proto.Unmarshal(buf, &res)
		if err != nil {
			panic(err)
		}

		fmt.Printf("%d %d num=%d\n", res.From, res.To, len(res.Txs))

		i += 1000

		if ((i - 1) % 100) == 0 {
			fmt.Printf("synced %d/%d blocks...\n", maxHeight, height)
		}
	}
}