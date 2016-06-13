/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats"
	"gopkg.in/redis.v3"
)

func provisionNat(n *nats.Conn, r nat, s string, t string) {
	event := natEvent{}
	event.load(r, t, s)

	n.Publish(t, []byte(event.toJSON()))
}

func processRequest(n *nats.Conn, r *redis.Client, subject string, resSubject string) {
	n.Subscribe(subject, func(m *nats.Msg) {
		event := NatsCreate{}
		json.Unmarshal(m.Data, &event)
		persistEvent(r, &event)

		if len(event.Nats) == 0 || event.Status == "completed" {
			event.Status = "completed"
			event.ErrorCode = ""
			event.ErrorMessage = ""
			n.Publish(subject+".done", []byte(event.toJSON()))
			return
		}
		for _, nat := range event.Nats {
			if ok, msg := nat.Valid(); ok == false {
				event.Status = "error"
				event.ErrorCode = "0001"
				event.ErrorMessage = msg
				n.Publish(subject+".error", []byte(event.toJSON()))
				return
			}
		}
		sw := false
		for i, nat := range event.Nats {
			log.Println(event.Nats[i].Status)
			if event.Nats[i].completed() == false {
				sw = true
				event.Nats[i].processing()
				provisionNat(n, nat, event.Service, resSubject)
				if true == event.SequentialProcessing {
					break
				}
			}
		}
		if sw == false {
			event.Status = "completed"
			event.ErrorCode = ""
			event.ErrorMessage = ""
			n.Publish(subject+".done", []byte(event.toJSON()))
			return
		}
		persistEvent(r, &event)
	})
}

func processResponse(n *nats.Conn, r *redis.Client, s string, res string, p string, t string) {
	n.Subscribe(s, func(m *nats.Msg) {
		stored, completed := processNext(n, r, s, p, m.Data, t)

		if completed {
			complete(n, stored, res)
		}
	})
}

func complete(n *nats.Conn, stored *NatsCreate, subject string) {
	if isErrored(stored) == true {
		stored.Status = "error"
		stored.ErrorCode = "0002"
		stored.ErrorMessage = "Some nats could not be successfully processed"
		n.Publish(subject+"error", []byte(stored.toJSON()))
	} else {
		stored.Status = "completed"
		n.Publish(subject+"done", []byte(stored.toJSON()))
	}
}

func isErrored(stored *NatsCreate) bool {
	for _, v := range stored.Nats {
		if v.isErrored() {
			return true
		}
	}
	return false
}

func processNext(n *nats.Conn, r *redis.Client, subject string, procSubject string, body []byte, status string) (*NatsCreate, bool) {
	event := &natCreatedEvent{}
	json.Unmarshal(body, event)

	message, err := r.Get(event.cacheKey()).Result()
	if err != nil {
		log.Println(err)
	}
	stored := &NatsCreate{}
	json.Unmarshal([]byte(message), stored)
	completed := true
	scheduled := false
	for i := range stored.Nats {
		if stored.Nats[i].Name == event.NatName {
			stored.Nats[i].Status = status
			stored.Nats[i].ErrorCode = string(event.Error.Code)
			stored.Nats[i].ErrorMessage = event.Error.Message
		}
		if stored.Nats[i].completed() == false && stored.Nats[i].errored() == false {
			completed = false
		}
		if stored.Nats[i].toBeProcessed() && scheduled == false {
			scheduled = true
			completed = false
			stored.Nats[i].processing()
			provisionNat(n, stored.Nats[i], event.Service, procSubject)
		}
	}
	persistEvent(r, stored)

	return stored, completed
}
