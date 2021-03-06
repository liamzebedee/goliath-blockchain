package sequencer

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/liamzebedee/goliath-blockchain/sequencer/mvp/sequencer"
	"github.com/liamzebedee/goliath-blockchain/sequencer/mvp/sequencer/messages"
	"github.com/liamzebedee/goliath-blockchain/sequencer/mvp/sequencer/utils"
	"github.com/stretchr/testify/assert"
)

func getMockSequencer() (*sequencer.SequencerCore, error) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	// db, err := sql.Open("sqlite3", "data.sqlite")
	if err != nil {
		return nil, err
	}

	seq := sequencer.NewSequencerCore(db)
	return seq, nil
}


func TestSequence(t *testing.T) {
    seq, err := getMockSequencer()
	if err != nil {
		t.Error(err)
	}

	// 
	// Failure conditions for sequencing:
	// 
	
	// 1. Message is malformed.
	seqno, err := seq.Sequence("0x")
	assert.EqualError(t, err, "message is malformed")

	// 2. Invalid signature.
	// pubkey 0x0466724a07b5fc7937b0a5ef42d9d25b496958426e2d36c69e44e7e33c0b1f835e29127894ac9183a8f9353e78bd2a0b2667c23ae1ec88b4e6f9ba18b2854465aa
	signer := utils.NewEthereumECDSASigner("3977045d27df7e401ecf1596fd3ae86b59f666944f81ba8dbf547c2269902f6b")
	txData := "0xc4a6abb1cc341e7b796bdc0fb11c50a12d4e998cc4e8e3cb44badf185a8e00f7"
	
	// // 2a. Empty signature data.
	msg := messages.ConstructSequenceMessage(txData, 5 * time.Second)
	msg.Sig = hexutil.MustDecode("0x1234")
	msg.From = hexutil.MustDecode("0x0266724a07b5fc7937b0a5ef42d9d25b496958426e2d36c69e44e7e33c0b1f835e")
	seqno, err = seq.Sequence(msg.ToHex())
	assert.EqualError(t, err, "invalid signature")

	// // 2b. Signature for a different message.
	msg = messages.ConstructSequenceMessage(txData, 5 * time.Second)
	badData, _ := hexutil.Decode("0xaaaa")
	badSig, err := signer.Sign(crypto.Keccak256Hash(badData).Bytes())
	if err != nil {
		panic(err)
	}
	msg.Sig = badSig
	msg.SetFrom(signer.GetPubkey())
	seqno, err = seq.Sequence(msg.ToHex())
	assert.EqualError(t, err, "invalid signature")

	// // 3. Message is expired.
	msg = messages.ConstructSequenceMessage(txData, 1 * time.Second)
	ts := uint64(time.Now().Add(time.Duration(-1) * time.Minute).UnixMilli())
	msg.Expires[0].Condition = &messages.ExpiryCondition_Unix{
		Unix: &messages.UNIXExpiryCondition{
			Time: ts,
		},
	}
	fmt.Println(msg.ToHex())
	msg.SetFrom(signer.GetPubkey())
	msg = msg.Signed(signer)
	seqno, err = seq.Sequence(msg.ToHex())
	assert.EqualError(t, err, "message expired")

	// Happy path!
	msg = messages.ConstructSequenceMessage(txData, 1 * time.Second)
	msg.SetFrom(signer.GetPubkey())
	msg = msg.Signed(signer)
	seqno, err = seq.Sequence(msg.ToHex())
	
	if err != nil {
		t.Error(err)
	}

	assert.True(t, seqno == 1, "First tx sequenced should have sequence number of 1, got %d", seqno)
}

// func TestGet(t *testing.T) {
//     seq, err := getMockSequencer()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	sequences, err := seq.Get(0, 3)
// }

func TestInfo(t *testing.T) {
    seq, err := getMockSequencer()
	if err != nil {
		t.Error(err)
	}

	info, err := seq.Info()
	if err != nil {
		t.Error(err)
	}
	
	assert.Equal(t, info.Total, 0, "Total should be 0")
	assert.Equal(t, info.LastSequenceTime, 0, "LastSequenceTime should be 0")


	start := time.Now().UnixMilli()
	seq.Sequence("")
	end := time.Now().UnixMilli()
	
	assert.Equal(t, info.Total, 1, "Total should be 1")
	assert.True(
		t,
		start <= info.LastSequenceTime && info.LastSequenceTime < end,
		"LastSequenceTime should be a recent timestamp.\nstart=%d\nLastSequenceTime=%d\nend=%d\n",
		start, info.LastSequenceTime, end,
	)
}