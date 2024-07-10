// Copyright 2016 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// These event types are shared between the Events API and used as Webhook payloads.

package sdkadapter

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

const (
	WebhookUserAgentKey   = "User-Agent"
	WebhookUserAgentValue = "AtomGit-Hookshot"
)

type GenericEvent struct {
	EventType int
	EventName string
	EventUUID string
	Action    string
	Org       string
	Repo      string
	HtmlURL   string
	Ref       string // PushEvent
	Head      string // PushEvent
	Review    string // PullRequestReviewEvent || PullRequestCommentEvent
	Reviewer  string // PullRequestReviewEvent || PullRequestCommentEvent
	Payload   []byte
}

func (ge *GenericEvent) ConvertToBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(ge); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (ge *GenericEvent) ConvertToMap() map[string]interface{} {

	m := make(map[string]interface{})

	if ge.Repo == "" {
		if ge.Payload == nil {
			return nil
		}

		var b map[string]any
		_ = json.Unmarshal(ge.Payload, &b)
		ge.Action, _ = b["action"].(string)
		repository, _ := b["repository"].(map[string]any)
		orgRepo, _ := repository["full_name"].(string)
		if strings.Index(orgRepo, "/") > 0 {
			orgRepoSlice := strings.Split(orgRepo, "/")
			ge.Org = orgRepoSlice[0]
			ge.Repo = orgRepoSlice[1]
		}
	}

	if ge.EventName != "" {
		m["event-type"] = ge.EventName
	}
	if ge.EventUUID != "" {
		m["event-uuid"] = ge.EventUUID
	}
	if ge.Action != "" {
		m["event-action"] = ge.Action
	}
	if ge.Org != "" {
		m["org"] = ge.Org
	}
	if ge.Repo != "" {
		m["repo"] = ge.Repo
	}
	if ge.HtmlURL != "" {
		m["url"] = ge.HtmlURL
	}

	return m
}

func (ge *GenericEvent) ConvertFromBytes(b []byte) error {
	if b == nil {
		return errors.New("no data to convert")
	}

	buf := new(bytes.Buffer)
	buf.Write(b)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(ge); err != nil {
		return err
	}

	return nil
}

func (ge *GenericEvent) GetOrg() *string {

	return &ge.Org
}

func (ge *GenericEvent) GetRepo() *string {

	return &ge.Repo
}

// Event-Type Value
const (
	accessEvent = iota
	issueEvent
	pullRequestEvent
	pushEvent
	issueCommentEvent
	pullRequestReviewEvent
	pullRequestCommentEvent
)

func GetEventType(header *http.Header) (int, string, error) {
	ua := header.Get(WebhookUserAgentKey)
	if ua == WebhookUserAgentValue {
		// later
	}
	return accessEvent, header.Get("X-AtomGit-Event"), nil
}

func CheckUserAgent(header *http.Header, value string) error {
	ua := header.Get(WebhookUserAgentKey)
	if ua != value {
		return errors.New("unknown User-Agent header")
	}
	return nil
}

func GetEventUUID(header *http.Header) (string, error) {
	uuid := header.Get("X-AtomGit-Delivery")
	if uuid == "" {
		return "", errors.New("missing X-AtomGit-Delivery header")
	}
	return uuid, nil
}

func AuthSign(header *http.Header, body *[]byte, hmacKey []byte) error {
	sign := header.Get("X-Hub-Signature-256")
	if sign == "" || !strings.HasPrefix(sign, "sha256=") {
		return errors.New("missing X-Hub-Signature-256 header")
	}

	//if !hmac.Equal([]byte(sign[7:]), []byte(payloadSignature(body, hmacKey))) {
	//	return errors.New("invalid X-Hub-Signature-256")
	//}

	return nil
}

func payloadSignature(payload *[]byte, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(*payload)
	return hex.EncodeToString(mac.Sum(nil))
}
