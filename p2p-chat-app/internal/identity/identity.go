package identity

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
)

// Identity represents a user's cryptographic identity
type Identity struct {
	Username   string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	ID         string // Hash of public key
}

// NewIdentity creates a new cryptographic identity
func NewIdentity(username string) (*Identity, error) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	publicKey := &privateKey.PublicKey

	// Generate user ID from public key hash
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	
	hash := sha256.Sum256(pubKeyBytes)
	userID := hex.EncodeToString(hash[:])[:16] // Use first 16 chars

	return &Identity{
		Username:   username,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		ID:         userID,
	}, nil
}

// Sign signs a message with the private key
func (i *Identity) Sign(message []byte) ([]byte, error) {
	hash := sha256.Sum256(message)
	return rsa.SignPKCS1v15(rand.Reader, i.PrivateKey, crypto.SHA256, hash[:])
}

// Verify verifies a signature with the public key
func (i *Identity) Verify(message, signature []byte, publicKey *rsa.PublicKey) error {
	hash := sha256.Sum256(message)
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature)
}

// EncryptMessage encrypts a message for a specific recipient
func (i *Identity) EncryptMessage(message []byte, recipientPublicKey *rsa.PublicKey) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, recipientPublicKey, message, nil)
}

// DecryptMessage decrypts a message with the private key
func (i *Identity) DecryptMessage(ciphertext []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, i.PrivateKey, ciphertext, nil)
}

// ExportPublicKey exports the public key as PEM string
func (i *Identity) ExportPublicKey() (string, error) {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(i.PublicKey)
	if err != nil {
		return "", err
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return string(pubKeyPEM), nil
}

// ImportPublicKey imports a public key from PEM string
func ImportPublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaPubKey, nil
}

// ExportPrivateKey exports the private key as PEM string (for persistence)
func (i *Identity) ExportPrivateKey() (string, error) {
	privKeyBytes := x509.MarshalPKCS1PrivateKey(i.PrivateKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	return string(privKeyPEM), nil
}

// ImportPrivateKey imports a private key from PEM string
func ImportPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
