// Package rpcxauth defines the versioned wire authentication contract.
package rpcxauth

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/smallnest/rpcx/protocol"
	"strconv"
	"time"
)

const Version = "X-Auth-Version"
const Domain = "X-Auth-Domain"
const AppID = "X-Auth-AppId"
const Timestamp = "X-Auth-Timestamp"
const Nonce = "X-Auth-Nonce"
const Sign = "X-Auth-Sign"

type ReplayStore interface {
	Claim(context.Context, string, time.Duration) (bool, error)
}

func canonical(r *protocol.Message) []byte {
	var b bytes.Buffer
	hash := sha256.Sum256(r.Payload)
	for _, v := range []string{"v2", r.Metadata[Domain], r.Metadata[AppID], r.ServicePath, r.ServiceMethod, r.Metadata[Timestamp], r.Metadata[Nonce], hex.EncodeToString(hash[:])} {
		binary.Write(&b, binary.BigEndian, uint32(len(v)))
		b.WriteString(v)
	}
	return b.Bytes()
}
func Digest(r *protocol.Message, key string) string {
	m := hmac.New(sha256.New, []byte(key))
	m.Write(canonical(r))
	return hex.EncodeToString(m.Sum(nil))
}
func SignRequest(r *protocol.Message, app, key, domain string) error {
	if r.Metadata == nil {
		r.Metadata = map[string]string{}
	}
	delete(r.Metadata, Sign)
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	if app == "" || key == "" || domain == "" {
		return errors.New("rpcxauth: credentials/domain required")
	}
	r.Metadata[Version] = "v2"
	r.Metadata[Domain] = domain
	r.Metadata[AppID] = app
	r.Metadata[Timestamp] = strconv.FormatInt(time.Now().Unix(), 10)
	r.Metadata[Nonce] = hex.EncodeToString(nonce[:])
	r.Metadata[Sign] = Digest(r, key)
	return nil
}
func Verify(ctx context.Context, r *protocol.Message, key, domain string, skew time.Duration, store ReplayStore) error {
	if store == nil || skew <= 0 || skew > 10*time.Minute || r.Metadata[Version] != "v2" || r.Metadata[Domain] != domain {
		return errors.New("rpcxauth: invalid policy/version/domain")
	}
	if len(r.Metadata[AppID]) == 0 || len(r.Metadata[AppID]) > 128 || len(r.Metadata[Nonce]) != 48 || len(r.Metadata[Timestamp]) > 20 || len(r.Metadata[Sign]) != 64 {
		return errors.New("rpcxauth: invalid credentials")
	}
	nonce, err := hex.DecodeString(r.Metadata[Nonce])
	if err != nil || len(nonce) != 24 {
		return errors.New("rpcxauth: invalid nonce")
	}
	ts, err := strconv.ParseInt(r.Metadata[Timestamp], 10, 64)
	now := time.Now().Unix()
	limit := int64(skew / time.Second)
	if err != nil || ts < now-limit || ts > now+limit {
		return errors.New("rpcxauth: expired request")
	}
	expected := Digest(r, key)
	if !hmac.Equal([]byte(expected), []byte(r.Metadata[Sign])) {
		return errors.New("rpcxauth: invalid signature")
	}
	digest := sha256.Sum256([]byte(domain + "\x00" + r.Metadata[AppID] + "\x00" + r.Metadata[Nonce]))
	ok, err := store.Claim(ctx, hex.EncodeToString(digest[:]), 2*skew+time.Second)
	if err != nil {
		return errors.New("rpcxauth: replay store unavailable")
	}
	if !ok {
		return errors.New("rpcxauth: replay rejected")
	}
	return nil
}
