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

// type EventListener interface {
// 	Unsubscribe()
// }

type onBlockFn func (block *messages.Block)

type OnBlockEventListener struct {
	handler onBlockFn
}

type SequencerCore struct {
	signer utils.Signer
	db *sql.DB

	blockChannel chan *messages.Block
	blockListeners []*OnBlockEventListener
	
	lastBlock *messages.Block

	Total int64
	// milliseconds.
	LastSequenceTime int64
}

func NewSequencerCore(db *sql.DB, operatorPrivateKey string) (*SequencerCore) {
	fmt.Println("migrating database")
	
	// TODO: handle migation errors
	// panic: table sequence already exists
	db.Exec(`
	CREATE TABLE sequence (
		num INTEGER PRIMARY KEY AUTOINCREMENT, 
		msg BLOB,
		txid BLOB
	);
	`)
	fmt.Println("migration complete")

	// if err != nil {
	// 	panic(err)
	// }

	s := &SequencerCore{
		blockChannel: make(chan *messages.Block, 5),
		blockListeners: make([]*OnBlockEventListener, 0),
		db: db,
	}

	if operatorPrivateKey != "" {
		s.signer = utils.NewEthereumECDSASigner(operatorPrivateKey)
		fmt.Printf("operator pubkey: %s\n", hexutil.Encode(crypto.FromECDSAPub(s.signer.GetPubkey())))
	}

	go s.produceBlocks()

	return s
}

func (s *SequencerCore) produceBlocks() {
	for {
		// Process blocks serially.
		block := <-s.blockChannel

		// Sign each block.
		block.PrevBlockHash = s.getLastBlockHash()
		block = block.Signed(s.signer)
		s.lastBlock = block
		
		fmt.Println("chained a block:", block.PrettyString())
		for _, list := range s.blockListeners {
			go list.handler(block)
		}
	}
}

func (s *SequencerCore) getLastBlockHash() ([]byte) {
	// Genesis.
	if (s.lastBlock == nil) {
		return []byte{0}
	}
	
	return s.lastBlock.SigHash()
}

func (s *SequencerCore) OnNewBlock(onBlock onBlockFn) () {
	list := &OnBlockEventListener{
		handler: onBlock,
	}
	s.blockListeners = append(s.blockListeners, list)
}

func (s *SequencerCore) GetOperatorPubkey() ([]byte) {
	return hexutil.MustDecode("0x043e0b751273070a517b4c54393deb672e75a6d9dd731bd0b90f11bb178343dc2084ac3c86e289d0902fe40fbb7bb24efd2a342a95220347ed7cedd0dd19d629f5")
}

func (s *SequencerCore) Close() {
	s.db.Close()
}

func (s *SequencerCore) ProcessBlock(block *messages.Block) (error) {
	// current block = 5
	// new block = ?
	// if currBlock.num < newBlock.num { core.ProcessBlock }
	fmt.Printf("processing block: %s\n", block.PrettyString())

	// 
	// Verify block.
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

	// Verify signer.
	// expectedPubkeyECDSA, err := crypto.DecompressPubkey(hexutil.MustDecode("0x0266724a07b5fc7937b0a5ef42d9d25b496958426e2d36c69e44e7e33c0b1f835e"))
	// if err != nil {
	// 	return fmt.Errorf("message is malformed")
	// }

	// expectedPubkey := crypto.FromECDSAPub(expectedPubkeyECDSA)
	expectedPubkey := s.GetOperatorPubkey()
	if !bytes.Equal(pubkey, expectedPubkey) {
		return fmt.Errorf("invalid signer for block\n     got: %s\nexpected: %s\n", hexutil.Encode(pubkey), hexutil.Encode(expectedPubkey))
	}

	// Verify signature.
	// remove recovery id (last byte) from signature.
	signatureValid := crypto.VerifySignature(pubkey, digestHash, block.Sig[:len(block.Sig)-1])
	if !signatureValid {
		return fmt.Errorf("invalid signature")
	}
	
	// Now verify the sequence message inside.
	body := block.GetBody()
	if body == nil {
		return fmt.Errorf("block body is empty")
	}

	err = s.verifySequenceMessage(body, false)
	if err != nil {
		return err
	}
	
	// Then append to the log.
	_, err = s.db.Exec(
		"INSERT INTO sequence values (?, ?, ?)",
		nil,
		body.ToHex(),
		nil,
	)
	if err != nil {
		return fmt.Errorf("error writing tx to db: %s", err)
	}

	return nil
}

func (s *SequencerCore) verifySequenceMessage(msg *messages.SequenceTx, checkUnixExpiry bool) (error) {
	// if msg.Type != SEQUENCE_MESSAGE_TYPE {
	// 	return fmt.Errorf("unhandled message type: %s", msg.Type)
	// }

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
		return 0, fmt.Errorf("sequencer is in replica mode, does not sequence itself")
	}
	
	msg := &messages.SequenceTx{}

	msgBuf, err := hexutil.Decode(msgData)
	if err != nil {
		return 0, err
	}

	err = proto.Unmarshal(msgBuf, msg)
	if err != nil {
		return 0, err
	}

	err = s.verifySequenceMessage(msg, true)
	if err != nil {
		return 0, err
	}
	
	fmt.Printf("sequence hash=%s\n", hexutil.Encode(msg.SigHash()))

	// Then append to the log.
	// Save to DB.
	res, err := s.db.Exec(
		"INSERT INTO sequence values (?, ?, ?)",
		nil,
		msgData,
		nil,
	)
	if err != nil {
		return 0, fmt.Errorf("error writing tx to db: %s", err)
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		// TODO
		panic(err)
	}

	// Now chain a block.
	block := messages.ConstructBlock([]byte{}, msg)
	s.blockChannel <- block

	return lastId, nil
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