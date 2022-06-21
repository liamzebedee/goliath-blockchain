package sequencer

import (
	"testing"
	// "github.com/stretchr/testify/assert"
)

func TestReplicaRequestsHistory(t *testing.T) {
	// The P2P pubsub network is used for disseminating new blocks only.
	// Replicas sync blocks they've missed by requesting history from peers.
	// In the V1, this is simple and not efficiently load balanced. We request blocks from one peer in our table.

	
}