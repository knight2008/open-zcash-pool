package proxy

import (
	"log"
	"math/big"
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/jkkgbe/open-zcash-pool/util"
)

const maxBacklog = 3

type heightDiffPair struct {
	diff   *big.Int
	height uint64
}

type transaction struct {
	fee  int    `json:"fee"`
	hash string `json:"hash"`
}

type coinbaseTxn struct {
	data           string `json:"data"`
	hash           string `json:"hash"`
	foundersReward int    `json:"foundersreward"`
}

type BlockTemplate struct {
	sync.RWMutex
	prevBlockHash string        `json:"prevblockhash"`
	transactions  []transaction `json:"transactions"`
	coinbaseTxn   coinbaseTxn   `json:"coinbasetxn"`
	longpollId    string        `json:""`
	minTime       int           `json:""`
	nonceRange    string        `json:""`
	curtime       int           `json:""`
	bits          string        `json:""`
	height        int           `json:""`
}

type Block struct {
	difficulty         int
	version            string
	prevHashReversed   string
	merkleRootReversed string
	reservedField      string
	nTime              string
	bits               string
	nonce              string
	header             string
}

// func (b Block) Difficulty() *big.Int     { return b.difficulty }
// func (b Block) HashNoNonce() common.Hash { return b.hashNoNonce }
// func (b Block) Nonce() uint64            { return b.nonce }
// func (b Block) MixDigest() common.Hash   { return b.mixDigest }
// func (b Block) NumberU64() uint64        { return b.number }

func (s *ProxyServer) fetchBlockTemplate() {
	rpc := s.rpc()
	t := s.currentBlockTemplate()
	var reply BlockTemplate
	err := rpc.GetBlockTemplate(&reply)
	if err != nil {
		log.Printf("Error while refreshing block template on %s: %s", rpc.Name, err)
		return
	}
	// No need to update, we have fresh job
	if t != nil && t.prevBlockHash == reply.prevBlockHash {
		return
	}

	// TODO calc merkle root etc
	// if (blockHeight.toString(16).length % 2 === 0) {
	//     var blockHeightSerial = blockHeight.toString(16);
	// } else {
	//     var blockHeightSerial = '0' + blockHeight.toString(16);
	// }
	// var height = Math.ceil((blockHeight << 1).toString(2).length / 8);
	// var lengthDiff = blockHeightSerial.length/2 - height;
	// for (var i = 0; i < lengthDiff; i++) {
	//     blockHeightSerial = blockHeightSerial + '00';
	// }
	// length = '0' + height;
	// var serializedBlockHeight = new Buffer.concat([
	//     new Buffer(length, 'hex'),
	//     util.reverseBuffer(new Buffer(blockHeightSerial, 'hex')),
	//     new Buffer('00', 'hex') // OP_0
	// ]);

	// tx.addInput(new Buffer('0000000000000000000000000000000000000000000000000000000000000000', 'hex'),
	//     4294967295,
	//     4294967295,
	//     new Buffer.concat([serializedBlockHeight,
	//         Buffer('5a2d4e4f4d50212068747470733a2f2f6769746875622e636f6d2f6a6f7368756179616275742f7a2d6e6f6d70', 'hex')]) //Z-NOMP! https://github.com/joshuayabut/z-nomp
	// );

	generatedTxHash := CreateRawTransaction(inputs, outputs).TxHash()
	txHashes := make([]chainhash.Hash, len(reply.transactions)+1)
	txHashes[0] = util.ReverseHash(generatedTxHash)
	for i, transaction := range reply.transactions {
		txHashes[i+1] = transaction.hash
	}
	merkleRootReversed := util.ReverseHash(getRoot(txHashes))

	// TODO
	newBlock := Block{
		difficulty:         1,
		version:            "",
		prevHashReversed:   "",
		merkleRootReversed: merkleRootReversed,
		reservedField:      "",
		nTime:              "",
		bits:               "",
		nonce:              "",
		header:             "",
	}

	// Copy job backlog and add current one
	newBlock.headers[reply[0]] = heightDiffPair{
		diff:   util.TargetHexToDiff(reply[2]),
		height: height,
	}
	if t != nil {
		for k, v := range t.headers {
			if v.height > height-maxBacklog {
				newBlock.headers[k] = v
			}
		}
	}
	s.blockTemplate.Store(&newBlock)
	log.Printf("New block to mine on %s at height %d / %s", rpc.Name, height, reply[0][0:10])

	// Stratum
	if s.config.Proxy.Stratum.Enabled {
		go s.broadcastNewJobs()
	}
}
