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
	processBlock chan *messages.Block

	outOfOrderBlockChan chan *messages.Block
	unprocessedBlockAtHeight map[int64]*messages.Block
	LastBlock *messages.Block
	TotalSeen int
	blockListeners []*OnBlockEventListener
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
		// blockIngestion: make(chan *messages.Block),
		processBlock: make(chan *messages.Block),
		sequenceTxs: make(chan *sequenceWork),

		blockListeners: make([]*OnBlockEventListener, 0),
		db: db,
		// outOfOrderBlocks: make([]*messages.Block, 100),
		unprocessedBlockAtHeight: make(map[int64]*messages.Block),
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

	go s.loop()
	return s
}


func (s *SequencerCore) loop() {
	for {
		// Process blocks serially.
		select {
		case block := <-s.processBlock:
			// Check since last time if we have satisfied dependencies for other blocks.
			s.checkOutOfOrderBlocks()

			// Process the block.
			err := s.doProcessBlock(block)
			if err != nil {
				fmt.Println("error while processing block", fmt.Sprint(block.Height), ":", err)
				continue
			}
		case work := <-s.sequenceTxs:
			s.doSequenceWork(work)
		}
	}
}

type sequenceWork struct {
	msg *messages.SequenceTx
}

func (s *SequencerCore) WaitedBlocks(max int64) (int64) {
	for height := s.LastBlock.Height; height < max; height++ {
		b := s.unprocessedBlockAtHeight[height]
		if b != nil {
			return height
		}
	}
	return -1
}

func (s *SequencerCore) checkOutOfOrderBlocks() (error) {
	// If we've processed the parent block, we process the child.
	for {
		block_satisfied := s.unprocessedBlockAtHeight[s.LastBlock.Height + 1]

		if block_satisfied == nil {
			break
		}

		err := s.doProcessBlock(block_satisfied)
		if err != nil {
			return fmt.Errorf("error processing an out of order block: %s", err)
		}

		delete(s.unprocessedBlockAtHeight, s.LastBlock.Height)
	}
	
	return nil
}

func (s *SequencerCore) doSequenceWork(work *sequenceWork) (error) {
	// Process sequence txs serially.
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

	return nil
}

func (s *SequencerCore) ingestBlock(block *messages.Block) (error) {
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
	fmt.Printf("new chain height %d\n", block.Height)

	return nil
}

// Processes a block from the sequencer primary. Used by replicas.
// NOTE: This method is NOT threadsafe with the `Sequence` method.
func (s *SequencerCore) ProcessBlock(block *messages.Block) (error) {
	s.processBlock <- block
	return nil
}

func (s *SequencerCore) doProcessBlock(block *messages.Block) (error) {
	fmt.Println("process block", fmt.Sprint(block.Height))
	// fmt.Printf("processing block: %s\n", block.PrettyString())

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
	s.TotalSeen += 1

	// Block was received out-of-order. We can process it later.
	if s.LastBlock.Height + 1 < block.Height {
		// Store it for later.
		fmt.Println("got block out-of-order:", block.PrettyString())

		// Maps the out-of-order block to the block height which satisfies it.
		s.unprocessedBlockAtHeight[block.Height] = block
		return nil
	}

	// Verify hash chain.
	if !bytes.Equal(block.PrevBlockHash, s.LastBlock.SigHash()) {
		return fmt.Errorf("block prevhash is not lastblock prevhash")
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

	err = s.ingestBlock(block)
	if err != nil {
		return err
	}

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

// Returns the blocks between index `from` and `to`.
func (s *SequencerCore) GetBlocks(from, to uint64) (int, error) {
	return 1, nil
}

type SequencerInfo struct {
	Total int64            `json:"total"`
	LastSequenceTime int64 `json:"lastSequenceTime"` // milliseconds.
}

// Get the sequencer info.
// - total number of sequenced txs.
// - latest received tx time.
func (s *SequencerCore) Info() (SequencerInfo, error) {
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