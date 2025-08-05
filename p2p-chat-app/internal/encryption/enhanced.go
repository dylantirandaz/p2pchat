package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"sync"
)

type KeyManager struct {
	peerKeys map[string][]byte 
	myPrivKey *ecdh.PrivateKey
	myPubKey  *ecdh.PublicKey
	mu        sync.RWMutex
}

type EncryptedMessage struct {
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
	KeyID      string `json:"key_id"`
}

func NewKeyManager() (*KeyManager, error) {
	curve := ecdh.P256()
	privKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &KeyManager{
		peerKeys:  make(map[string][]byte),
		myPrivKey: privKey,
		myPubKey:  privKey.PublicKey(),
	}, nil
}

func (km *KeyManager) GetPublicKey() []byte {
	return km.myPubKey.Bytes()
}

func (km *KeyManager) EstablishSharedKey(userID string, peerPubKeyBytes []byte) error {
	curve := ecdh.P256()
	peerPubKey, err := curve.NewPublicKey(peerPubKeyBytes)
	if err != nil {
		return err
	}

	sharedSecret, err := km.myPrivKey.ECDH(peerPubKey)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(sharedSecret)
	
	km.mu.Lock()
	km.peerKeys[userID] = hash[:]
	km.mu.Unlock()

	return nil
}

func (km *KeyManager) EncryptForPeer(userID string, plaintext []byte) (*EncryptedMessage, error) {
	km.mu.RLock()
	key, exists := km.peerKeys[userID]
	km.mu.RUnlock()

	if !exists {
		return nil, errors.New("no shared key for peer")
	}

	return km.encryptWithKey(key, plaintext, userID)
}

func (km *KeyManager) DecryptFromPeer(userID string, encMsg *EncryptedMessage) ([]byte, error) {
	km.mu.RLock()
	key, exists := km.peerKeys[userID]
	km.mu.RUnlock()

	if !exists {
		return nil, errors.New("no shared key for peer")
	}

	return km.decryptWithKey(key, encMsg)
}

func (km *KeyManager) encryptWithKey(key []byte, plaintext []byte, keyID string) (*EncryptedMessage, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedMessage{
		Nonce:      hex.EncodeToString(nonce),
		Ciphertext: hex.EncodeToString(ciphertext),
		KeyID:      keyID,
	}, nil
}

func (km *KeyManager) decryptWithKey(key []byte, encMsg *EncryptedMessage) ([]byte, error) {
	nonce, err := hex.DecodeString(encMsg.Nonce)
	if err != nil {
		return nil, err
	}

	ciphertext, err := hex.DecodeString(encMsg.Ciphertext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func EncryptWithECDH(plaintext []byte, sharedKey []byte) (string, error) {
	km := &KeyManager{}
	encMsg, err := km.encryptWithKey(sharedKey, plaintext, "legacy")
	if err != nil {
		return "", err
	}
	
	return encMsg.Ciphertext, nil
}

func DecryptWithECDH(ciphertextHex string, sharedKey []byte) ([]byte, error) {
	km := &KeyManager{}
	
	encMsg := &EncryptedMessage{
		Ciphertext: ciphertextHex,
		Nonce:      "000000000000000000000000", //dummy nonce for legacy
	}
	
	return km.decryptWithKey(sharedKey, encMsg)
}

type ForwardSecureEncryption struct {
	km          *KeyManager
	rootKeys    map[string][]byte    //userID -> root key
	chainKeys   map[string][]byte    //userID -> current chain key
	messageKeys map[string][][]byte  //userID -> message keys history
	mu          sync.RWMutex
}

func NewForwardSecureEncryption(km *KeyManager) *ForwardSecureEncryption {
	return &ForwardSecureEncryption{
		km:          km,
		rootKeys:    make(map[string][]byte),
		chainKeys:   make(map[string][]byte),
		messageKeys: make(map[string][][]byte),
	}
}

func (fse *ForwardSecureEncryption) InitializeWithPeer(userID string) error {
	fse.km.mu.RLock()
	sharedKey, exists := fse.km.peerKeys[userID]
	fse.km.mu.RUnlock()

	if !exists {
		return errors.New("no shared key established")
	}

	fse.mu.Lock()
	defer fse.mu.Unlock()

	hash := sha256.Sum256(append(sharedKey, []byte("root")...))
	fse.rootKeys[userID] = hash[:]

	hash = sha256.Sum256(append(sharedKey, []byte("chain")...))
	fse.chainKeys[userID] = hash[:]

	fse.messageKeys[userID] = make([][]byte, 0)

	return nil
}

func (fse *ForwardSecureEncryption) EncryptMessage(userID string, plaintext []byte) (*EncryptedMessage, error) {
	fse.mu.Lock()
	defer fse.mu.Unlock()

	chainKey, exists := fse.chainKeys[userID]
	if !exists {
		return nil, errors.New("no chain key for peer")
	}

	hash := sha256.Sum256(append(chainKey, []byte("message")...))
	messageKey := hash[:]

	hash = sha256.Sum256(append(chainKey, []byte("advance")...))
	fse.chainKeys[userID] = hash[:]

	fse.messageKeys[userID] = append(fse.messageKeys[userID], messageKey)

	return fse.km.encryptWithKey(messageKey, plaintext, userID)
}

func (fse *ForwardSecureEncryption) DecryptMessage(userID string, encMsg *EncryptedMessage, messageIndex int) ([]byte, error) {
	fse.mu.RLock()
	defer fse.mu.RUnlock()

	messageKeys, exists := fse.messageKeys[userID]
	if !exists || messageIndex >= len(messageKeys) {
		return nil, errors.New("message key not found")
	}

	messageKey := messageKeys[messageIndex]
	return fse.km.decryptWithKey(messageKey, encMsg)
}
