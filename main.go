/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"os"
	"runtime"

	l "github.com/ernestio/builder-library"
)

var s l.Scheduler

func main() {
	s.Setup(os.Getenv("NATS_URI"))

	// Process requests
	s.ProcessRequest("nats.create", "nat.create")
	s.ProcessRequest("nats.delete", "nat.delete")
	s.ProcessRequest("nats.update", "nat.update")

	// Process resulting success
	s.ProcessSuccessResponse("nat.create.done", "nat.create", "nats.create.done")
	s.ProcessSuccessResponse("nat.delete.done", "nat.delete", "nats.delete.done")
	s.ProcessSuccessResponse("nat.update.done", "nat.update", "nats.update.done")

	// Process resulting errors
	s.ProcessFailedResponse("nat.create.error", "nats.create.error")
	s.ProcessFailedResponse("nat.delete.error", "nats.delete.error")
	s.ProcessFailedResponse("nat.update.error", "nats.update.error")

	runtime.Goexit()
}
