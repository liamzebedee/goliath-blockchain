package sequencer

import (
	"bytes"
	"fmt"

	"database/sql"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/messages"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/utils"
	_ "github.com/mattn/go-sqlite3"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// Event listeners.

type onBlockFn func (block *messages.Block)
type OnBlockEventListener struct {
	handler onBlockFn
}

// Sequencer Core.
type SequencerCore struct {
	signer utils.Signer
	db *sql.DB

	sequenceTxs chan *sequenceWork
	blockIngestion chan *messages.Block
	blockListeners []*OnBlockEventListener
	
	LastBlock *messages.Block

	outOfOrderBlocks map[int64]*messages.Block

	Total int64
	// milliseconds.
	LastSequenceTime int64
}

type operatorChange struct {
	BlockHash []byte
	Pubkey []byte
}
var operatorChangeHistory []operatorChange


func NewSequencerCore(db *sql.DB, operatorPrivateKey string) (*SequencerCore) {
	fmt.Println("migrating database")

	operatorChangeHistory = []operatorChange{
		{
			BlockHash: []byte{0},
			Pubkey: hexutil.MustDecode("0x043e0b751273070a517b4c54393deb672e75a6d9dd731bd0b90f11bb178343dc2084ac3c86e289d0902fe40fbb7bb24efd2a342a95220347ed7cedd0dd19d629f5"),
		},
	}
	
	// TODO: handle migation errors
	// panic: table sequence already exists
	db.Exec(`
	CREATE TABLE sequence (
		num INTEGER PRIMARY KEY AUTOINCREMENT, 
		msg BLOB,
		hash BLOB
	);
	CREATE TABLE blocks (
		num INTEGER PRIMARY KEY AUTOINCREMENT, 
		block BLOB,
		hash BLOB
	);
	`)
	fmt.Println("migration complete")

	// if err != nil {
	// 	panic(err)
	// }

	s := &SequencerCore{
		blockIngestion: make(chan *messages.Block, 5),
		sequenceTxs: make(chan *sequenceWork, 5),
		blockListeners: make([]*OnBlockEventListener, 0),
		db: db,
		// outOfOrderBlocks: make([]*messages.Block, 100),
		outOfOrderBlocks: make(map[int64]*messages.Block),
	}

	// Insert genesis block.
	s.LastBlock = &messages.Block{
		Height: 0,
		PrevBlockHash: []byte{0},
		Body: nil,
	}

	if operatorPrivateKey != "" {
		s.signer = utils.NewEthereumECDSASigner(operatorPrivateKey)
		fmt.Printf("operator pubkey: %s\n", s.signer.String())
	}

	go s.sequenceRoutine()
	go s.ingestBlockRoutine()

	// Every 10ms, check blocks we've received out-of-order for future block heights.
	// If we've processed the parent block, we process the child.
	go func(){
		for {
			block := s.outOfOrderBlocks[s.LastBlock.Height]
			if block != nil {
				s.outOfOrderBlocks[s.LastBlock.Height] = nil
				s.blockIngestion <- block
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	return s
}

type sequenceWork struct {
	msg *messages.SequenceTx
}

func (s *SequencerCore) sequenceRoutine() (error) {
	for {
		// Process sequence txs serially.
		work := <-s.sequenceTxs
		sequenceTx := work.msg

		sequenceBuf, err := proto.Marshal(sequenceTx)
		if err != nil {
			return err
		}

		// Insert sequence into storage, generating a sequence number.
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("error writing tx to db: %s", err)
		}

		_, err = tx.Exec(
			"INSERT INTO sequence values (?, ?, ?)",
			nil,
			sequenceBuf,
			nil,
		)
		if err != nil {
			return fmt.Errorf("error writing tx to db: %s", err)
		}

		// Create a block, chain and sign it.
		block := messages.ConstructBlock(sequenceTx)
		block.Height = s.LastBlock.Height + 1
		block.PrevBlockHash = s.LastBlock.SigHash()
		block = block.Signed(s.signer)
		
		// Insert into database.
		blockBuf, err := proto.Marshal(block)
		if err != nil {
			return err
		}

		_, err = tx.Exec(
			"INSERT INTO blocks values (?, ?, ?)",
			nil,
			blockBuf,
			block.SigHash(),
		)
		if err != nil {
			return fmt.Errorf("error writing tx to db: %s", err)
		}

		// Commit the new state.
		err = tx.Commit()
		if err != nil {
			return err
		}
		
		s.LastBlock = block

		fmt.Println("chained a block:", block.PrettyString())

		// Notify the block listeners.
		for _, list := range s.blockListeners {
			go list.handler(block)
		}
	}
}

func (s *SequencerCore) ingestBlockRoutine() (error) {
	for {
		// Process blocks serially.
		block := <-s.blockIngestion

		sequenceBuf, err := proto.Marshal(block.Body)
		if err != nil {
			return err
		}

		// Insert sequence into storage, generating a sequence number.
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("error writing tx to db: %s", err)
		}

		_, err = tx.Exec(
			"INSERT INTO sequence values (?, ?, ?)",
			nil,
			sequenceBuf,
			nil,
		)
		if err != nil {
			return fmt.Errorf("error writing tx to db: %s", err)
		}

		// Insert into database.
		blockBuf, err := proto.Marshal(block)
		if err != nil {
			return err
		}

		_, err = tx.Exec(
			"INSERT INTO blocks values (?, ?, ?)",
			nil,
			blockBuf,
			block.SigHash(),
		)
		if err != nil {
			return fmt.Errorf("error writing tx to db: %s", err)
		}

		// Commit the new state.
		err = tx.Commit()
		if err != nil {
			return err
		}

		s.LastBlock = block

		fmt.Println("ingested a block:", block.PrettyString())
	}
}

// Processes a block from the sequencer primary. Used by replicas.
// NOTE: This method is NOT threadsafe with the `Sequence` method.
func (s *SequencerCore) ProcessBlock(block *messages.Block) (error) {
	fmt.Printf("processing block: %s\n", block.PrettyString())

	// 
	// 1. Verify block.
	// 
	if block.Sig == nil {
		return fmt.Errorf("missing signature")
	}

	// Compute the digest which was signed, aka the "sighash".
	digestHash := block.SigHash()

	// Recover pubkey.
	pubkey, err := crypto.Ecrecover(digestHash, block.Sig)
	if err != nil {
		// TODO
		fmt.Println("error while recovering pubkey:", err.Error())
		return fmt.Errorf("invalid signature")
	}

	// Verify block was signed by the sequencer operator.
	expectedPubkey := s.GetOperatorPubkey()
	if !bytes.Equal(pubkey, expectedPubkey) {
		return fmt.Errorf("invalid signer for block\n     got: %s\nexpected: %s\n", hexutil.Encode(pubkey), hexutil.Encode(expectedPubkey))
	}

	// Verify signature is valid.
	// remove recovery id (last byte) from signature.
	signatureValid := crypto.VerifySignature(pubkey, digestHash, block.Sig[:len(block.Sig)-1])
	if !signatureValid {
		return fmt.Errorf("invalid signature")
	}
	
	// 
	// 2. Verify block body.
	// 
	if block.Height <= s.LastBlock.Height {
		// We have already processed up to this block height.
		return nil
	}
	// TODO memoize sighash here.
	if !bytes.Equal(block.PrevBlockHash, s.LastBlock.SigHash()) {
		// Block is out-of-order. 
		// Store it for later.
		fmt.Println("got block out-of-order:", block.PrettyString())
		s.outOfOrderBlocks[block.Height - 1] = block
		// s.outOfOrderBlocks = append(s.outOfOrderBlocks, block)
		return nil
	}

	body := block.GetBody()
	if body == nil {
		return fmt.Errorf("block body is empty")
	}

	err = s.verifySequenceMessage(body, false)
	if err != nil {
		return err
	}

	// Block was valid.
	
	// 
	// 3. Update sequencer state.
	// 
	s.blockIngestion <- block

	return nil
}

func (s *SequencerCore) verifySequenceMessage(msg *messages.SequenceTx, checkUnixExpiry bool) (error) {
	if len(msg.Data) == 0 || msg.Sig == nil {
		return fmt.Errorf("message is malformed")
	}

	if len(msg.From) == 0 || msg.From == nil {
		return fmt.Errorf("message is malformed")
	}
	
	// Verify signature.
	digestHash := msg.SigHash()

	pubkey, err := crypto.Ecrecover(digestHash, msg.Sig)
	if err != nil {
		// TODO
		fmt.Println("error while recovering pubkey:", err.Error())
		return fmt.Errorf("invalid signature")
	}

	fromPubkey, err := crypto.DecompressPubkey(msg.From)
	if err != nil {
		return fmt.Errorf("message is malformed")
	}

	if !bytes.Equal(pubkey, crypto.FromECDSAPub(fromPubkey)) {
		fmt.Println("message signature is for different pubkey:")
		return fmt.Errorf("invalid signature")
	}

	// remove recovery id (last byte) from signature.
	signatureValid := crypto.VerifySignature(pubkey, digestHash, msg.Sig[:len(msg.Sig)-1])
	if !signatureValid {
		return fmt.Errorf("invalid signature")
	}

	// Check expiry conditions.
	for _, expiryCondition := range msg.Expires {
		if cond := expiryCondition.GetUnix(); cond != nil {
			if (!checkUnixExpiry) {
				continue
			}

			if err != nil {
				// TODO
				fmt.Println("error while parsing expiry check time", err.Error())
				return fmt.Errorf("message is malformed")
			}
			
			if cond.Time < uint64(time.Now().UnixMilli()) {
				return fmt.Errorf("message expired")
			}
		} else {
			// TODO this won't actually print the Condition id. Do we need this? 
			return fmt.Errorf("unknown expiry condition '%s'", expiryCondition.GetCondition())
		}
	}

	if err != nil {
		return err
	}

	return nil
}

// Assigns a sequence number for the transaction.
func (s *SequencerCore) Sequence(msgData string) (int64, error) {
	if (s.signer == nil) {
		return 0, fmt.Errorf("sequencer is in replica mode, it will not produce blocks")
	}
	
	// Decode message.
	msg := &messages.SequenceTx{}
	
	msgBuf, err := hexutil.Decode(msgData)
	if err != nil {
		return 0, err
	}

	err = proto.Unmarshal(msgBuf, msg)
	if err != nil {
		return 0, err
	}

	// Verify message.
	err = s.verifySequenceMessage(msg, true)
	if err != nil {
		return 0, err
	}
	
	fmt.Printf("sequence hash=%s\n", hexutil.Encode(msg.SigHash()))

	s.sequenceTxs <- &sequenceWork{msg: msg}

	return 0, nil
}

// Returns the transactions between index `from` and `to`.
func (s *SequencerCore) Get(from, to uint64) (int, error) {
	return 1, nil
}

type SequencerInfo struct {
	Total int64            `json:"total"`
	// milliseconds.
	LastSequenceTime int64 `json:"lastSequenceTime"`
}

// Get the sequencer info.
// - total number of sequenced txs.
// - latest received tx time.
func (s *SequencerCore) Info() (SequencerInfo, error) {
	// info := make(map[string]interface{})
	// info["total"] = 2
	// info["latest"] = 21

	info := SequencerInfo{
		Total: 0,
		LastSequenceTime: 0,
	}
	
	return info, nil
}

func (s *SequencerCore) OnNewBlock(onBlock onBlockFn) () {
	list := &OnBlockEventListener{
		handler: onBlock,
	}
	s.blockListeners = append(s.blockListeners, list)
}

func (s *SequencerCore) GetOperatorPubkey() ([]byte) {
	// TODO load from Ethereum.
	return operatorChangeHistory[0].Pubkey
}

func (s *SequencerCore) Close() {
	s.db.Close()
}