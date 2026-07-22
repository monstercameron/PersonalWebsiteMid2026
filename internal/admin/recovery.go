package admin

import (
	_ "embed"
	"crypto/rand"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

//go:embed wordlist.txt
var wordlistRaw string

// words is the recovery-phrase vocabulary (whitespace-separated in wordlist.txt). At 286 words each
// word carries ~8.16 bits, so the default 6-word phrase is ~49 bits of entropy.
var words = strings.Fields(wordlistRaw)

// phraseWords is how many words a generated recovery phrase has.
const phraseWords = 6

// normalizePhrase lower-cases a phrase and collapses runs of whitespace to single spaces, so a phrase
// compares equal regardless of casing or spacing when the owner types it back.
func normalizePhrase(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// generatePhrase returns a fresh space-separated recovery phrase drawn uniformly (crypto/rand, no
// modulo bias) from the wordlist.
func generatePhrase() string {
	out := make([]string, phraseWords)
	n := big.NewInt(int64(len(words)))
	for i := range out {
		idx, err := rand.Int(rand.Reader, n)
		if err != nil {
			// crypto/rand failing is catastrophic and not something the caller can recover from.
			panic("admin: cannot read randomness for recovery phrase: " + err.Error())
		}
		out[i] = words[idx.Int64()]
	}
	return strings.Join(out, " ")
}

// hashSecret returns the bcrypt hash of a secret (password or recovery phrase). Empty input yields an
// empty hash (used when no recovery phrase is set).
func hashSecret(secret string) (string, error) {
	if secret == "" {
		return "", nil
	}
	h, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// matchSecret reports whether secret matches the bcrypt hash, in the constant-time way bcrypt
// provides. An empty hash never matches (no secret configured).
func matchSecret(hash, secret string) bool {
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) == nil
}
