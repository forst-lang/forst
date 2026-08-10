// invoke_proof implements HMAC-SHA256 challenge-response proofs for invoke RPC auth.
package invokeserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strconv"
)

// invokeProofVersion is the MAC domain separator baked into every proof message.
const invokeProofVersion = "forst-invoke-v1"

// invokeProofMessage formats the MAC input: version|generation|nonce.
func invokeProofMessage(generation uint64, nonce string) string {
	return invokeProofVersion + "|" + strconv.FormatUint(generation, 10) + "|" + nonce
}

// computeInvokeProof returns the raw HMAC-SHA256 bytes for token, generation, and nonce.
func computeInvokeProof(token []byte, generation uint64, nonce string) []byte {
	mac := hmac.New(sha256.New, token)
	_, _ = mac.Write([]byte(invokeProofMessage(generation, nonce)))
	return mac.Sum(nil)
}

// encodeInvokeProof base64url-encodes raw proof bytes without padding.
func encodeInvokeProof(proof []byte) string {
	return base64.RawURLEncoding.EncodeToString(proof)
}

// decodeInvokeProof decodes a base64url proof string.
func decodeInvokeProof(encoded string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(encoded)
}

// verifyInvokeProof compares candidateBase64 to the expected MAC using constant-time compare.
// Decode failures take the same comparison path as a wrong proof.
func verifyInvokeProof(token []byte, generation uint64, nonce, candidateBase64 string) bool {
	expected := computeInvokeProof(token, generation, nonce)
	candidate, err := decodeInvokeProof(candidateBase64)
	if err != nil {
		candidate = make([]byte, len(expected))
	}
	if len(candidate) != len(expected) {
		padded := make([]byte, len(expected))
		copy(padded, candidate)
		candidate = padded
	}
	return subtle.ConstantTimeCompare(expected, candidate) == 1
}

// InvokeProofMessageForTest exposes the MAC input for cross-language test vectors.
func InvokeProofMessageForTest(generation uint64, nonce string) string {
	return invokeProofMessage(generation, nonce)
}

// ComputeInvokeProofForTest exposes proof computation for tests.
func ComputeInvokeProofForTest(token []byte, generation uint64, nonce string) string {
	return encodeInvokeProof(computeInvokeProof(token, generation, nonce))
}
