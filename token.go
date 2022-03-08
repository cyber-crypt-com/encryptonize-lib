// Copyright 2020-2022 CYBERCRYPT
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package encryptonize

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"time"

	"encryptonize/crypto"
)

const TokenValidity = time.Hour

type token struct {
	Plaintext  []byte
	ExpiryTime time.Time
}

type SealedToken struct {
	Ciphertext []byte
	WrappedKey []byte
	ExpiryTime time.Time
}

func newToken(plaintext []byte, validityPeriod time.Duration) token {
	expiryTime := time.Now().Add(validityPeriod)
	return token{plaintext, expiryTime}
}

func (t *token) seal(cryptor crypto.CryptorInterface) (SealedToken, error) {
	associatedData, err := t.ExpiryTime.GobEncode()
	if err != nil {
		return SealedToken{}, err
	}

	wrappedKey, ciphertext, err := cryptor.Encrypt(t.Plaintext, associatedData)
	if err != nil {
		return SealedToken{}, err
	}

	return SealedToken{ciphertext, wrappedKey, t.ExpiryTime}, nil
}

func (t *SealedToken) unseal(cryptor crypto.CryptorInterface) (token, error) {
	associatedData, err := t.ExpiryTime.GobEncode()
	if err != nil {
		return token{}, err
	}

	plaintext := []byte{}
	err = cryptor.Decrypt(&plaintext, associatedData, t.WrappedKey, t.Ciphertext)
	if err != nil {
		return token{}, err
	}

	if t.ExpiryTime.Before(time.Now()) {
		return token{}, errors.New("Token expired")
	}

	return token{plaintext, t.ExpiryTime}, nil
}

func (t *SealedToken) verify(cryptor crypto.CryptorInterface) bool {
	_, err := t.unseal(cryptor)
	if err != nil {
		return false
	}
	return t.ExpiryTime.After(time.Now())
}

func (t *SealedToken) String() (string, error) {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	if err := enc.Encode(t); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buffer.Bytes()), nil
}

func TokenFromString(tokenString string) (SealedToken, error) {
	tokenBytes, err := base64.RawURLEncoding.DecodeString(tokenString)
	if err != nil {
		return SealedToken{}, err
	}
	var token SealedToken
	dec := gob.NewDecoder(bytes.NewReader(tokenBytes))
	if err := dec.Decode(&token); err != nil {
		return SealedToken{}, err
	}
	return token, nil
}
