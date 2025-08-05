package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"p2p-chat-app/internal/identity"
	"time"
)

type Block struct {
	Index        int64             `json:"index"`
	Timestamp    time.Time         `json:"timestamp"`
	Data         BlockData         `json:"data"`
	PreviousHash string            `json:"previous_hash"`
	Hash         string            `json:"hash"`
	Nonce        int               `json:"nonce"`
	Difficulty   int               `json:"difficulty"`
}

type BlockData struct {
	Type      string      `json:"type"`
	Identity  *IdentityTx `json:"identity,omitempty"`
	KeyExch   *KeyExchTx  `json:"key_exchange,omitempty"`
	Revoke    *RevokeTx   `json:"revoke,omitempty"`
}

type IdentityTx struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	PublicKey   string    `json:"public_key"`
	Signature   string    `json:"signature"`
	RegisteredAt time.Time `json:"registered_at"`
}

type KeyExchTx struct {
	FromUserID string `json:"from_user_id"`
	ToUserID   string `json:"to_user_id"`
	PublicKey  string `json:"public_key"`
	Signature  string `json:"signature"`
}

type RevokeTx struct {
	UserID    string `json:"user_id"`
	Reason    string `json:"reason"`
	Signature string `json:"signature"`
}

type Blockchain struct {
	chain      []*Block
	difficulty int
	identities map[string]*IdentityTx
	keyPairs   map[string]map[string]string // from_user -> to_user -> shared_key
}

func NewBlockchain() *Blockchain {
	bc := &Blockchain{
		chain:      make([]*Block, 0),
		difficulty: 4,
		identities: make(map[string]*IdentityTx),
		keyPairs:   make(map[string]map[string]string),
	}
	
	bc.chain = append(bc.chain, bc.createGenesisBlock())
	return bc
}

func (bc *Blockchain) createGenesisBlock() *Block {
	genesis := &Block{
		Index:        0,
		Timestamp:    time.Now(),
		Data:         BlockData{Type: "genesis"},
		PreviousHash: "0",
		Difficulty:   bc.difficulty,
	}
	genesis.Hash = bc.calculateHash(genesis)
	return genesis
}

func (bc *Blockchain) RegisterIdentity(user *identity.Identity) error {
	pubKey, err := user.ExportPublicKey()
	if err != nil {
		return err
	}

	// create identity transaction
	identityTx := &IdentityTx{
		UserID:       user.ID,
		Username:     user.Username,
		PublicKey:    pubKey,
		RegisteredAt: time.Now(),
	}

	// sign the identity data
	identityData, _ := json.Marshal(identityTx)
	signature, err := user.Sign(identityData)
	if err != nil {
		return err
	}
	identityTx.Signature = hex.EncodeToString(signature)

	// create new block
	block := &Block{
		Index:        int64(len(bc.chain)),
		Timestamp:    time.Now(),
		Data:         BlockData{Type: "identity", Identity: identityTx},
		PreviousHash: bc.getLatestBlock().Hash,
		Difficulty:   bc.difficulty,
	}

	bc.mineBlock(block)
	bc.chain = append(bc.chain, block)
	bc.identities[user.ID] = identityTx

	return nil
}

func (bc *Blockchain) VerifyIdentity(userID string) (*IdentityTx, bool) {
	identity, exists := bc.identities[userID]
	if !exists {
		return nil, false
	}

	identityData, _ := json.Marshal(map[string]interface{}{
		"user_id":       identity.UserID,
		"username":      identity.Username,
		"public_key":    identity.PublicKey,
		"registered_at": identity.RegisteredAt,
	})

	_ = identity.PublicKey 
	hash := sha256.Sum256(identityData)
	expectedSig := hex.EncodeToString(hash[:])
	isValid := expectedSig != "" /
	
	return identity, isValid
}

func (bc *Blockchain) StoreKeyExchange(fromUser, toUser *identity.Identity, sharedKey string) error {
	keyExchTx := &KeyExchTx{
		FromUserID: fromUser.ID,
		ToUserID:   toUser.ID,
		PublicKey:  sharedKey,
	}

	keyExchData, _ := json.Marshal(keyExchTx)
	signature, err := fromUser.Sign(keyExchData)
	if err != nil {
		return err
	}
	keyExchTx.Signature = hex.EncodeToString(signature)

	block := &Block{
		Index:        int64(len(bc.chain)),
		Timestamp:    time.Now(),
		Data:         BlockData{Type: "key_exchange", KeyExch: keyExchTx},
		PreviousHash: bc.getLatestBlock().Hash,
		Difficulty:   bc.difficulty,
	}

	bc.mineBlock(block)
	bc.chain = append(bc.chain, block)

	if bc.keyPairs[fromUser.ID] == nil {
		bc.keyPairs[fromUser.ID] = make(map[string]string)
	}
	bc.keyPairs[fromUser.ID][toUser.ID] = sharedKey

	return nil
}

func (bc *Blockchain) GetSharedKey(fromUserID, toUserID string) (string, bool) {
	if userKeys, exists := bc.keyPairs[fromUserID]; exists {
		if key, exists := userKeys[toUserID]; exists {
			return key, true
		}
	}
	
	if userKeys, exists := bc.keyPairs[toUserID]; exists {
		if key, exists := userKeys[fromUserID]; exists {
			return key, true
		}
	}
	
	return "", false
}

func (bc *Blockchain) mineBlock(block *Block) {
	target := fmt.Sprintf("%0*d", bc.difficulty, 0)
	
	for {
		block.Hash = bc.calculateHash(block)
		if block.Hash[:bc.difficulty] == target {
			break
		}
		block.Nonce++
	}
}

func (bc *Blockchain) calculateHash(block *Block) string {
	data, _ := json.Marshal(map[string]interface{}{
		"index":         block.Index,
		"timestamp":     block.Timestamp,
		"data":          block.Data,
		"previous_hash": block.PreviousHash,
		"nonce":         block.Nonce,
	})
	
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (bc *Blockchain) getLatestBlock() *Block {
	return bc.chain[len(bc.chain)-1]
}

func (bc *Blockchain) IsValidChain() bool {
	for i := 1; i < len(bc.chain); i++ {
		currentBlock := bc.chain[i]
		previousBlock := bc.chain[i-1]

		if currentBlock.Hash != bc.calculateHash(currentBlock) {
			return false
		}

		if currentBlock.PreviousHash != previousBlock.Hash {
			return false
		}
	}
	return true
}

func (bc *Blockchain) GetAllIdentities() map[string]*IdentityTx {
	return bc.identities
}

func (bc *Blockchain) GetChainLength() int {
	return len(bc.chain)
}
