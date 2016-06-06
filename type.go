/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"bytes"
	"encoding/json"
	"log"

	"gopkg.in/redis.v3"
)

// NatsCreate : Represents a nats.create message
type NatsCreate struct {
	Service              string `json:"service"`
	Status               string `json:"status"`
	ErrorCode            string `json:"error_code"`
	ErrorMessage         string `json:"error_message"`
	Nats                 []nat  `json:"nats"`
	SequentialProcessing bool   `json:"sequential_processing"`
}

func (e *NatsCreate) toJSON() string {
	message, _ := json.Marshal(e)
	return string(message)
}

func (e *NatsCreate) cacheKey() string {
	return composeCacheKey(e.Service)
}

func composeCacheKey(service string) string {
	var key bytes.Buffer
	key.WriteString("GPBNats_")
	key.WriteString(service)

	return key.String()
}

type rule struct {
	Type            string `json:"type"`
	OriginIP        string `json:"origin_ip"`
	OriginPort      string `json:"origin_port"`
	TranslationIP   string `json:"translation_ip"`
	TranslationPort string `json:"translation_port"`
	Protocol        string `json:"protocol"`
	Network         string `json:"network"`
}

type nat struct {
	Name               string `json:"name"`
	Rules              []rule `json:"rules,omitempty"`
	RouterName         string `json:"router_name"`
	RouterType         string `json:"router_type"`
	RouterIP           string `json:"router_ip"`
	ClientName         string `json:"client_name,omitempty"`
	DatacenterType     string `json:"datacenter_type,omitempty"`
	DatacenterName     string `json:"datacenter_name,omitempty"`
	DatacenterUsername string `json:"datacenter_username,omitempty"`
	DatacenterPassword string `json:"datacenter_password,omitempty"`
	DatacenterRegion   string `json:"datacenter_region,omitempty"`
	ExternalNetwork    string `json:"external_network"`
	VCloudURL          string `json:"vcloud_url"`
	Status             string `json:"status"`
	ErrorCode          string `json:"error_code"`
	ErrorMessage       string `json:"error_message"`
}

func (n *nat) fail() {
	n.Status = "errored"
}

func (n *nat) complete() {
	n.Status = "completed"
}

func (n *nat) processing() {
	n.Status = "processed"
}

func (n *nat) errored() bool {
	return n.Status == "errored"
}

func (n *nat) completed() bool {
	println(n.Status)
	return n.Status == "completed"
}

func (n *nat) isProcessed() bool {
	return n.Status == "processed"
}

func (n *nat) isErrored() bool {
	return n.Status == "errored"
}

func (n *nat) toBeProcessed() bool {
	return n.Status != "processed" && n.Status != "completed" && n.Status != "errored"
}

func (n *nat) Valid() (bool, string) {
	if n.Name == "" {
		return false, "Nat name is empty"
	}
	if n.RouterName == "" {
		return false, "Nat router name is empty"
	}

	for _, rule := range n.Rules {
		if rule.Network == "" {
			return false, "Empty network on rule detected"
		}
		if rule.OriginPort == "" {
			return false, "Empty original port on rule detected"
		}
		if rule.Protocol == "" {
			return false, "Empty protocol on rule detected"
		}
		if rule.TranslationPort == "" {
			return false, "Empty translation port on rule detected"
		}
		if rule.Type == "" {
			return false, "Empty type on rule detected"
		}
	}
	return true, ""
}

type natEvent struct {
	Service            string `json:"service_id"`
	Type               string `json:"type"`
	NatName            string `json:"nat_name"`
	NatRules           []rule `json:"nat_rules"`
	RouterName         string `json:"router_name"`
	RouterType         string `json:"router_type"`
	RouterIP           string `json:"router_ip"`
	ClientName         string `json:"client_name"`
	DatacenterName     string `json:"datacenter_name"`
	DatacenterUsername string `json:"datacenter_username"`
	DatacenterPassword string `json:"datacenter_password"`
	DatacenterRegion   string `json:"datacenter_region"`
	DatacenterType     string `json:"datacenter_type"`
	ExternalNetwork    string `json:"external_network"`
	VCloudURL          string `json:"vcloud_url"`
	Status             string `json:"status"`
}

func (e *natEvent) load(n nat, t string, s string) {
	e.Service = s
	e.Type = t
	e.NatName = n.Name
	e.NatRules = n.Rules
	e.RouterType = n.RouterType
	e.RouterName = n.RouterName
	e.RouterIP = n.RouterIP
	e.ClientName = n.ClientName
	e.DatacenterName = n.DatacenterName
	e.DatacenterUsername = n.DatacenterUsername
	e.DatacenterPassword = n.DatacenterPassword
	e.DatacenterRegion = n.DatacenterRegion
	e.DatacenterType = n.DatacenterType
	e.ExternalNetwork = n.ExternalNetwork
	e.VCloudURL = n.VCloudURL
	e.Status = n.Status
}

func (e *natEvent) toJSON() string {
	message, _ := json.Marshal(e)
	return string(message)
}

type Error struct {
	Code    json.Number `json:"code,Number"`
	Message string      `json:"message"`
}

type natCreatedEvent struct {
	Type    string `json:"type"`
	Service string `json:"service_id"`
	NatID   string `json:"nat_id"`
	NatName string `json:"nat_name"`
	Error   Error  `json:"error"`
}

func (e *natCreatedEvent) cacheKey() string {
	return composeCacheKey(e.Service)
}

func persistEvent(redisClient *redis.Client, event *NatsCreate) {
	if event.Service == "" {
		panic("Service is null!")
	}
	if err := redisClient.Set(event.cacheKey(), event.toJSON(), 0).Err(); err != nil {
		log.Println(err)
	}
}
