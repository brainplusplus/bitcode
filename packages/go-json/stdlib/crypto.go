package stdlib

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/google/uuid"
)

// CryptoNamespace returns a map of crypto functions for injection as "crypto" variable.
// Enables dot-notation calls: crypto.sha256("hello"), crypto.uuid()
func CryptoNamespace() map[string]any {
	return map[string]any{
		"sha256": func(args ...any) (any, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("crypto.sha256: requires a string argument")
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.sha256: argument must be a string")
			}
			h := sha256.Sum256([]byte(s))
			return hex.EncodeToString(h[:]), nil
		},
		"md5": func(args ...any) (any, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("crypto.md5: requires a string argument")
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.md5: argument must be a string")
			}
			h := md5.Sum([]byte(s))
			return hex.EncodeToString(h[:]), nil
		},
		"uuid": func(args ...any) (any, error) {
			return uuid.New().String(), nil
		},
		"hmac": func(args ...any) (any, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("crypto.hmac: requires (data, key) arguments")
			}
			s, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.hmac: first argument must be a string")
			}
			key, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("crypto.hmac: second argument must be a key string")
			}
			algo := "sha256"
			if len(args) > 2 {
				if a, ok := args[2].(string); ok {
					algo = a
				}
			}
			var hashFunc func() hash.Hash
			switch algo {
			case "sha256":
				hashFunc = sha256.New
			case "sha512":
				hashFunc = sha512.New
			default:
				return nil, fmt.Errorf("crypto.hmac: unsupported algorithm '%s' (use sha256 or sha512)", algo)
			}
			mac := hmac.New(hashFunc, []byte(key))
			mac.Write([]byte(s))
			return hex.EncodeToString(mac.Sum(nil)), nil
		},
	}
}
