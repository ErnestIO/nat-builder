/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats"
)

type DummyEvent struct {
	Type string `json:"type"`
	Name string `json:"nat_name"`
}

func wait(ch chan bool) error {
	return waitTime(ch, 500*time.Millisecond)
}

func waitTime(ch chan bool, timeout time.Duration) error {
	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
	}
	return errors.New("timeout")
}

func TestProvisionAllNatsBasic(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processRequest(n, r, "nats.create", "nat.provision")

	ch := make(chan bool)

	n.Subscribe("nat.provision", func(ev *nats.Msg) {
		event := natEvent{}
		json.Unmarshal(ev.Data, &event)

		if event.Type == "nat.provision" &&
			event.NatName == "test" &&
			event.RouterName == "test" &&
			event.NatRules[0].Type == "SNAT" {
			eventKey := "GPBNats_" + event.Service
			message, _ := r.Get(eventKey).Result()
			stored := &NatsCreate{}
			json.Unmarshal([]byte(message), stored)
			if stored.Service != event.Service {
				t.Fatal("Event is not persisted correctly")
			}
			ch <- true
		} else {
			t.Fatal("Message received from nats does not match")
		}
	})

	message := `{"service":"service", "nats":[{"name": "test", "router": "test", "rules": [{"type": "SNAT", "origin_ip": "10.64.0.0/16", "origin_port": "ANY", "translation_ip": "8.8.8.1", "translation_port": "ANY", "protocol": "any", "network": "test"}], "router_name": "test",	"router_type": "vcloud", "router_ip": "8.8.8.24", "datacenter": "test", "datacenter_name": "test",	"datacenter_type": "vcloud", "datacenter_region": "LON-001", "datacenter_username": "test@test", "datacenter_password": "test", "client_id": "test", "client_name": "test"}]}`
	n.Publish("nats.create", []byte(message))
	time.Sleep(100 * time.Millisecond)

	if e := wait(ch); e != nil {
		t.Fatal("Message not received from nats for subscription")
	}
}

func TestProvisionAllNatsWithInvalidMessage(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processRequest(n, r, "nats.create", "nat.provision")

	ch := make(chan bool)
	ch2 := make(chan bool)

	n.Subscribe("nat.provision", func(msg *nats.Msg) {
		ch <- true
	})

	n.Subscribe("nats.create.error", func(msg *nats.Msg) {
		ch2 <- true
	})

	message := `{"service":"service", "nats":[{"name": "test", "router": "test", "rules": [{"type": "SNAT"}] }]}`
	n.Publish("nats.create", []byte(message))

	if e := wait(ch); e == nil {
		t.Fatal("Produced a nat.provision message when I shouldn't")
	}
	if e := wait(ch2); e != nil {
		t.Fatal("Should produce a provision-all-nats-error message on nats")
	}
}

func TestProvisionAllnatsWithDifferentMessageType(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processRequest(n, r, "nats.create", "nat.provision")

	ch := make(chan bool)

	n.Subscribe("nat.provision", func(msg *nats.Msg) {
		ch <- true
	})

	message := `{"service":"service", "routers":[{"name":"test"}]}`
	n.Publish("nats.create", []byte(message))

	if e := wait(ch); e == nil {
		t.Fatal("Produced a nat.provision message when I shouldn't")
	}
}

func TestHandleProvisionNatErrorEvent(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processResponse(n, r, "nat.create.error", "nats.create.", "nat.provision", "errored")

	// Add record to Redis
	message := `{"service":"service", "nats":[{"name": "test", "router": "test", "rules": [{"type": "SNAT", "origin_ip": "10.64.0.0/16", "origin_port": "ANY", "translation_ip": "8.8.8.1", "translation_port": "ANY", "protocol": "any", "network": "test"}], "router_name": "test",	"router_type": "vcloud", "router_ip": "8.8.8.24", "datacenter": "test", "datacenter_name": "test",	"datacenter_type": "vcloud", "datacenter_region": "LON-001", "datacenter_username": "test@test", "datacenter_password": "test", "client_id": "test", "client_name": "test"}]}`
	r.Set("GPBNats_service", message, 0)

	ch := make(chan bool)
	ch2 := make(chan bool)

	n.Subscribe("nats.create.error", func(msg *nats.Msg) {
		ch <- true
	})

	n.Subscribe("nat.provision", func(msg *nats.Msg) {
		ev := DummyEvent{}
		json.Unmarshal(msg.Data, &ev)
		if ev.Type == "nat.provision" {
			ch2 <- true
		}
	})

	ev := `{"type": "nat.create.error", "service_id": "service", "nat_id": "1", "nat_name": "test", "router_id": "test", "nat_rules": [{"type": "SNAT", "origin_ip": "10.64.0.0/16", "origin_port": "ANY", "translation_ip": "8.8.8.1", "translation_port": "ANY", "protocol": "any", "network": "test"}],"error":{"code":"001","message":"lol"}}`

	n.Publish("nat.create.error", []byte(ev))

	if e := wait(ch); e != nil {
		t.Fatal("Didnt Produce an error event when I should have")
	}

	if e := wait(ch2); e == nil {
		t.Fatal("Produced a new nat.provision event when i shouldn't")
	}
}

func TestHandleNatCompletedEvent(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processResponse(n, r, "nat.create.done", "nats.create.", "nat.provision", "completed")

	// Add record to Redis
	message := `{"service":"service", "nats":[{"name": "test", "router": "test", "rules": [{"type": "SNAT", "origin_ip": "10.64.0.0/16", "origin_port": "ANY", "translation_ip": "8.8.8.1", "translation_port": "ANY", "protocol": "any", "network": "test"}]}, {"name": "test2", "router": "test", "rules": [{"type": "SNAT", "origin_ip": "192.168.0.0/16", "origin_port": "ANY", "translation_ip": "8.8.8.1", "translation_port": "ANY", "protocol": "any", "network": "test"}] , "router_name": "test",	"router_type": "vcloud", "router_ip": "8.8.8.24", "datacenter": "test", "datacenter_name": "test",	"datacenter_type": "vcloud", "datacenter_region": "LON-001", "datacenter_username": "test@test", "datacenter_password": "test", "client_id": "test", "client_name": "test"}]}`
	r.Set("GPBNats_service", message, 0)

	ch := make(chan bool)
	ch2 := make(chan bool)

	n.Subscribe("nats.create.error", func(msg *nats.Msg) {
		ch <- true
	})

	n.Subscribe("nat.provision", func(msg *nats.Msg) {
		ev := DummyEvent{}
		json.Unmarshal(msg.Data, &ev)
		if ev.Type == "nat.provision" {
			ch2 <- true
		}
	})

	ev := `{"type": "nat.create.done", "service_id": "service", "nat_id": "1", "nat_name": "test", "router_id": "test", "nat_type":"vcloud", "nat_rules": [{"type": "SNAT", "origin_ip": "10.64.0.0/16", "origin_port": "ANY", "translation_ip": "8.8.8.1", "translation_port": "ANY", "protocol": "any", "network": "test"}]}`
	n.Publish("nat.create.done", []byte(ev))

	// Should receive a provision event
	if e := wait(ch); e == nil {
		t.Fatal("Produced an error when i shouldn't have")
	}

	if e := wait(ch2); e != nil {
		t.Fatal("Didn't produce a nat.provision event when i should have")
	}
}

func TestHandleNextInSequenceEvent(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processResponse(n, r, "nat.create.done", "nats.create.", "nat.provision", "completed")

	// Add record to Redis
	message := `{"service":"service", "nats":[{"name": "test", "router": "test", "rules": [{"type": "SNAT", "origin_ip": "10.64.0.0/16", "origin_port": "ANY", "translation_ip": "8.8.8.1", "translation_port": "ANY", "protocol": "any", "network": "test"}]}, {"name": "test2", "router": "test", "rules": [{"type": "SNAT", "origin_ip": "192.168.0.0/16", "origin_port": "ANY", "translation_ip": "8.8.8.1", "translation_port": "ANY", "protocol": "any", "network": "test"}], "router_name": "test",	"router_type": "vcloud", "router_ip": "8.8.8.24", "datacenter": "test", "datacenter_name": "test",	"datacenter_type": "vcloud", "datacenter_region": "LON-001", "datacenter_username": "test@test", "datacenter_password": "test", "client_id": "test", "client_name": "test" }]}`
	r.Set("GPBNats_service", message, 0)

	ch := make(chan bool)
	ch2 := make(chan []byte)

	n.Subscribe("nats.create.error", func(msg *nats.Msg) {
		ch <- true
	})

	n.Subscribe("nat.provision", func(msg *nats.Msg) {
		ev := DummyEvent{}
		json.Unmarshal(msg.Data, &ev)
		if ev.Type == "nat.provision" {
			ch2 <- msg.Data
		}
	})

	ev := `{"type": "nat.create.done", "service_id": "service", "nat_id": "1", "nat_name": "test", "router_id": "test", "nat_type":"vcloud", "nat_rules": [{"type": "SNAT", "origin_ip": "10.64.0.0/16", "origin_port": "ANY", "translation_ip": "8.8.8.1", "translation_port": "ANY", "protocol": "any", "network": "test"}]}`
	n.Publish("nat.create.done", []byte(ev))

	if e := wait(ch); e == nil {
		t.Fatal("Produced an error when i shouldn't have")
	}

	// Should receive next provision event in sequence
	nev := <-ch2
	nextEvent := natEvent{}
	json.Unmarshal(nev, &nextEvent)

	if nextEvent.Service == "service" && nextEvent.NatName == "test2" && nextEvent.Type == "nat.provision" {
		log.Println("Correct event received")
	} else {
		t.Fatal("Did not produce the correct next event")
	}
}

func TestHandleFinalEvent(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processResponse(n, r, "nat.create.done", "nats.create.", "nat.provision", "completed")

	// Add record to Redis
	message := `{"service":"service", "nats":[{"name": "test", "router": "test", "rules": [{"type": "SNAT", "origin_ip": "10.64.0.0/16", "origin_port": "ANY", "translation_ip": "8.8.8.1", "translation_port": "ANY", "protocol": "any", "network": "test"}], "router_name": "test",	"router_type": "vcloud", "router_ip": "8.8.8.24", "datacenter": "test", "datacenter_name": "test",	"datacenter_type": "vcloud", "datacenter_region": "LON-001", "datacenter_username": "test@test", "datacenter_password": "test", "client_id": "test", "client_name": "test" }]}`
	r.Set("GPBNats_service", message, 0)

	ch := make(chan bool)

	n.Subscribe("nats.create.done", func(msg *nats.Msg) {
		ch <- true
	})

	ev := `{"type": "nat.create.done", "service_id": "service", "nat_id": "1", "nat_name": "test", "router_id": "test", "nat_type":"vcloud", "nat_rules": [{"type": "SNAT", "origin_ip": "10.64.0.0/16", "origin_port": "ANY", "translation_ip": "8.8.8.1", "translation_port": "ANY", "protocol": "any", "network": "test"}]}`
	n.Publish("nat.create.done", []byte(ev))

	// Should receive next provision event in sequence
	if e := wait(ch); e != nil {
		t.Fatal("Did not produce a completed event")
	}
}
