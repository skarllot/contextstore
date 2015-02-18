/*
 * Copyright (C) 2015 Fabrício Godoy <skarllot@gmail.com>
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License
 * as published by the Free Software Foundation; either version 2
 * of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 59 Temple Place - Suite 330, Boston, MA  02111-1307, USA.
 */

package appcontext

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

// A TokenStore provides a temporary token to uniquely identify an user session.
type TokenStore struct {
	tstore       *TimedStore
	salt         []byte
	authDuration time.Duration
}

// NewTokenStore creates a new instance of TokenStore and defines a lifetime for
// unauthenticated and authenticated sessions and a salt for random input.
func NewTokenStore(noAuth, auth time.Duration, salt string) *TokenStore {
	hash := sha256.New()
	hash.Write([]byte(salt))

	ts := NewTimedStore(noAuth)
	return &TokenStore{
		tstore:       ts,
		salt:         hash.Sum(nil),
		authDuration: auth,
	}
}

// Count gets the number of tokens stored by current instance.
func (s *TokenStore) Count() int {
	return s.tstore.Count()
}

// getInvalidTokenError gets the default error when an invalid or expired token
// is requested.
func (s *TokenStore) getInvalidTokenError(token string) error {
	return errors.New(fmt.Sprintf(
		"The requested token '%s' is invalid or is expired", token))
}

// GetValue gets the value stored by specified token.
func (s *TokenStore) GetValue(token string) (interface{}, error) {
	v, err := s.tstore.GetValue(token)
	if err != nil {
		return nil, s.getInvalidTokenError(token)
	}
	return v, err
}

// NewToken creates a new unique token and stores it into current TokenStore
// instance.
func (s *TokenStore) NewToken() string {
	mac := hmac.New(sha256.New, s.salt)
	now := time.Now().Format(time.RFC3339Nano)

	// Tries to create unpredictable token
	// Most strength comes from 'rand.Read'
	// Another bits are used to avoid the chance of system random genarator
	//   is compromissed by internal issue
	mac.Write(getRandomBytes(128))
	mac.Write(getRandomBytes(time.Now().Second() / 2))
	mac.Write([]byte(now))
	macSum := mac.Sum(nil)
	s.salt = macSum
	strSum := base64.URLEncoding.EncodeToString(macSum)

	_, err := s.tstore.AddValue(strSum, nil)
	if err != nil {
		panic("Something is seriously wrong, a duplicated token was generated")
	}

	return strSum
}

// RemoveToken removes specified token from current TokenStore instance.
func (s *TokenStore) RemoveToken(token string) error {
	err := s.tstore.RemoveValue(token)
	if err != nil {
		return s.getInvalidTokenError(token)
	}
	return nil
}

// SetTokenAsAuthenticated updates the lifetime of specified token to specified
// lifetime for authenticated sessions.
func (s *TokenStore) SetTokenAsAuthenticated(token string) error {
	err := s.tstore.SetValueDuration(token, s.authDuration)
	if err != nil {
		return s.getInvalidTokenError(token)
	}
	return nil
}

// SetValue store a value to specified token.
func (s *TokenStore) SetValue(token string, value interface{}) error {
	err := s.tstore.SetValue(token, value)
	if err != nil {
		return s.getInvalidTokenError(token)
	}
	return nil
}

// getRandomBytes gets secure random bytes.
func getRandomBytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic("Could not access secure random generator")
	}

	return b
}
