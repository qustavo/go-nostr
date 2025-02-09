package nostr

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/fiatjaf/bip340"
	"github.com/valyala/fastjson"
)

type Event struct {
	ID        string
	PubKey    string
	CreatedAt time.Time
	Kind      int
	Tags      Tags
	Content   string
	Sig       string
}

const (
	KindSetMetadata            int = 0
	KindTextNote               int = 1
	KindRecommendServer        int = 2
	KindContactList            int = 3
	KindEncryptedDirectMessage int = 4
	KindDeletion               int = 5
)

// GetID serializes and returns the event ID as a string
func (evt *Event) GetID() string {
	h := sha256.Sum256(evt.Serialize())
	return hex.EncodeToString(h[:])
}

// Serialize outputs a byte array that can be hashed/signed to identify/authenticate
func (evt *Event) Serialize() []byte {
	// the serialization process is just putting everything into a JSON array
	// so the order is kept
	var arena fastjson.Arena

	arr := arena.NewArray()

	// version: 0
	arr.SetArrayItem(0, arena.NewNumberInt(0))

	// pubkey
	arr.SetArrayItem(1, arena.NewString(evt.PubKey))

	// created_at
	arr.SetArrayItem(2, arena.NewNumberInt(int(evt.CreatedAt.Unix())))

	// kind
	arr.SetArrayItem(3, arena.NewNumberInt(evt.Kind))

	// tags
	arr.SetArrayItem(4, tagsToFastjsonArray(&arena, evt.Tags))

	// content
	arr.SetArrayItem(5, arena.NewString(evt.Content))

	return arr.MarshalTo(nil)
}

// CheckSignature checks if the signature is valid for the id
// (which is a hash of the serialized event content).
// returns an error if the signature itself is invalid.
func (evt Event) CheckSignature() (bool, error) {
	// read and check pubkey
	pubkey, err := bip340.ParsePublicKey(evt.PubKey)
	if err != nil {
		return false, fmt.Errorf("Event has invalid pubkey '%s': %w", evt.PubKey, err)
	}

	s, err := hex.DecodeString(evt.Sig)
	if err != nil {
		return false, fmt.Errorf("signature is invalid hex: %w", err)
	}
	if len(s) != 64 {
		return false, fmt.Errorf("signature must be 64 bytes, not %d", len(s))
	}

	var sig [64]byte
	copy(sig[:], s)

	hash := sha256.Sum256(evt.Serialize())
	return bip340.Verify(pubkey, hash, sig)
}

// Sign signs an event with a given privateKey
func (evt *Event) Sign(privateKey string) error {
	h := sha256.Sum256(evt.Serialize())

	s, err := bip340.ParsePrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("Sign called with invalid private key '%s': %w", privateKey, err)
	}

	aux := make([]byte, 32)
	rand.Read(aux)
	sig, err := bip340.Sign(s, h, aux)
	if err != nil {
		return err
	}

	evt.ID = hex.EncodeToString(h[:])
	evt.Sig = hex.EncodeToString(sig[:])
	return nil
}
