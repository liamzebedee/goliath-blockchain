package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/golang/protobuf/proto"

	"github.com/liamzebedee/goliath-blockchain/sequencer/mvp/sequencer/messages"
)

type ExecutionNode struct {
	tx *messages.SequenceTx
	deps []StateLeaf
}
type ExecutionGraph struct {
	root *ExecutionNode
	nodes []*ExecutionNode
	edges [][]*ExecutionNode
	lastToModifyLeaf map[*StateLeaf]*ExecutionNode
}

type StateLeaf struct {
	val []byte
}



// File looks something like:
// address 0x58311aaf5ebf42095ee0d620b97697a0b4c2f11c
// address 0x96fc1d3b4b39982a82776bcc88e32dc934d7b6fa
// storage 0x58311aaf5ebf42095ee0d620b97697a0b4c2f11c 0x0000000000000000000000000000000000000000000000000000000000000000
func parseStateLeaves(data string) []StateLeaf {
	leaves := []StateLeaf{}

	for _, line := range(strings.Split(data, "\n")) {
		buf := []byte{}

		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")
		if parts[0] == "address" {
			account := hexutil.MustDecode(parts[1])
			buf = append(buf, account...)
		} else if parts[0] == "storage" {
			contract := hexutil.MustDecode(parts[1])
			index := hexutil.MustDecode(parts[2])
			buf = append(buf, contract...)
			buf = append(buf, index...)
		}

		if len(buf) == 0 {
			panic("no data for state leaf")
		}

		leaves = append(leaves, StateLeaf{buf})
	}
	
	return leaves
}

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

		// Firstly, we construct an dependency tree of txs.
		g := &ExecutionGraph{
			nodes: []*ExecutionNode{},
			edges: [][]*ExecutionNode{},
			lastToModifyLeaf: make(map[*StateLeaf]*ExecutionNode),
		}
		g.root = &ExecutionNode{}
		
		// For every tx, we insert it into the graph, pointing it at the 
		// txs it depends on.
		for _, tx := range res.Txs {
			reads := parseStateLeaves(string(tx.StateReads))

			node := &ExecutionNode{
				tx: tx,
				deps: reads,
			}

			g.nodes = append(g.nodes, node)

			// Special case: if the tx doesn't read any previous state,
			// then we construct an artificial dependency on the root execution node.
			if len(reads) == 0 {
				// Point the current tx at the execution root (no state leaf deps).
				edge := []*ExecutionNode{node, g.root}
				g.edges = append(g.edges, edge)
				continue
			}

			for _, leaf := range(reads) {
				dep := g.lastToModifyLeaf[&leaf]
				var edge []*ExecutionNode
				
				if dep == nil {
					// Point the current tx at the execution root.
					edge = []*ExecutionNode{node, g.root}
				} else {
					// Point the current tx at the tx it depends on.
					// (aka the tx that wrote this state leaf we're reading).
					edge = []*ExecutionNode{node, dep}
				}
				
				g.lastToModifyLeaf[&leaf] = node
				g.edges = append(g.edges, edge)
			}
		}

		// We've constructed the full execution graph.
		// Now we perform a breadth-first search from the root, in order to concurrently execute txs.
		toVisit := []*ExecutionNode{g.root}
		execBatchSize := 100

		
		for len(toVisit) > 0 {
			// pop.
			node := toVisit[0]
			toVisit = toVisit[1:]
			
			// visit.
			batch := []*ExecutionNode{}
			var wg sync.WaitGroup

			for _, edge := range(g.edges) {
				from, to := edge[0], edge[1]
				if to == node {
					batch = append(batch, from)
				}
			}

			if len(batch) == 0 {
				goto child
			}

			fmt.Printf("executing batch of txs size=%d\n", len(batch))

			for i := 0; i < (len(batch) / execBatchSize) + 1; i++ {
				start := execBatchSize * i
				end := start + execBatchSize
				if end > len(batch) {
					end = len(batch)
				}
				fmt.Printf("executing sub-batch of txs start=%d,end=%d\n", start, end)

				perfStart := time.Now()
				
				subbatch := batch[start:end]

				for _, node := range(subbatch) {
					tx := node.tx
					log.Debug("executing tx seq=%d hash=%s\n", i, hexutil.Encode(tx.SigHash()))
					go func(){
						wg.Add(1)
						execute(*tx)
						wg.Done()
					}()
				}
				wg.Wait()

				perfElapsed := time.Since(perfStart)
				fmt.Printf("subbatch %s %s\n", perfElapsed, perfElapsed / time.Duration(execBatchSize))
			}

			child:

			// visit children.
			for _, edge := range(g.edges) {
				from, to := edge[0], edge[1]
				if to == node {
					toVisit = append(toVisit, from)
				}
			}
		}

		// fmt.Printf("%d %d num=%d\n", res.From, res.To, len(res.Txs))

		i += 1000

		if (i % 100) == 0 {
			fmt.Printf("synced %d/%d txs...\n", maxHeight, height)
		}
	}
}

type EthTx struct {
	Data string     `json:"data"`
	From string     `json:"from"`
	To string       `json:"to"`
	Gas string      `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Value string    `json:"value"`
}


func execute(tx messages.SequenceTx) {
	binary := "/Users/liamz/Documents/Projects/shard/goliath/sputnikvm/target/debug/quarkevm"

	ethTx := &EthTx{
		Data: hexutil.Encode(tx.Data),
		From: "0x0x033e0b751273070a517b4c54393deb672e75a622",
		//  + hexutil.Encode(tx.From[0:20]),
		// From: hexutil.Encode(tx.From),
		// To: hexutil.Encode(tx.To),
		// To: hexutil.Encode([]byte{}),
		To: "0x",
		Gas: "0xffffffff",
		GasPrice: "0x00",
		Value: "0x00",
	}

	// const txRaw = {
    //     data: bufferToHex(tx.data),
    //     from: bufferToHex(tx.getSenderAddress().buf),
    //     to: bufferToHex(tx.to),
    //     gas: bufferToHex(tx.gasLimit),
    //     gasPrice: bufferToHex(tx.gasPrice),
    //     value: bufferToHex(tx.value),
    // }

	// simply json encode the tx onw
	txString, err := json.Marshal(ethTx)
	if err != nil {
		panic(err)
	}

	// outputFile, err := os.CreateTemp("/tmp", "vm-output")
	// if err != nil {
	// 	panic(err)
	// }

	// stateLeavesFile, err := os.CreateTemp("/tmp", "state-leaves")
	// if err != nil {
	// 	panic(err)
	// }
	
	args := []string{
		binary,
		"--db-path",
		// "file:/Users/liamz/Documents/Projects/shard/goliath/scheduler/chain.sqlite?cache=shared",
		"file:memory::?cache=shared",
		
		"--data", 
		// We don't escape the tx string, as Go does it for us.
		string(txString),
		// fmt.Sprintf("'%s'", txString),
		
		"--output-file", "/dev/null",
		"--state-leaves-file", "/dev/null",
		// "--write",
	}

	
	// runCmd := fmt.Sprintf("%s %s", binary, strings.Join(args, " "))
	// shellRunCmd := fmt.Sprintf("`%s`", runCmd)

	// Shell args.
	// shArgs := []string{
	// 	"-c",
	// 	shellRunCmd,
	// }

	// cmd := exec.Command("sh", shArgs...)
	// cmd := exec.Command(binary, args...)
	cmd := &exec.Cmd{
        Path:   binary,
        Args:   args,
        // Stdout: os.Stdout,
        // Stderr: os.Stderr,
    }

	// cmd.Env = append(cmd.Env, "DB_GENESIS=0")
	cmd.Env = append(os.Environ(), "DB_GENESIS=1")
	
	// fmt.Println(runCmd)
	log.Debug(cmd.String())
	// fmt.Println(cmd.String())
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}

	start := time.Now()
		
	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	err = cmd.Wait()
	if err != nil {
		// Write the stderr/stdout if the VM failed during execution.
		stdoutBuf, _ := io.ReadAll(stdout)
		stderrBuf, _ := io.ReadAll(stderr)
		fmt.Printf("%s\n", stdoutBuf)
		fmt.Printf("%s\n", stderrBuf)

		panic(err)
	}

	elapsed := time.Since(start)
	fmt.Printf("exec %s\n", elapsed)
}