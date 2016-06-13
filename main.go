/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import "runtime"

func main() {
	n := natsClient()
	r := redisClient()

	// Process requests
	processRequest(n, r, "nats.create", "nat.create")
	processRequest(n, r, "nats.update", "nat.update")
	processRequest(n, r, "nats.delete", "nat.delete")

	// Process resulting success
	processResponse(n, r, "nat.create.done", "nats.create.", "nat.create", "completed")
	processResponse(n, r, "nat.update.done", "nats.update.", "nat.update", "completed")
	processResponse(n, r, "nat.delete.done", "nats.delete.", "nat.delete", "completed")

	// Process resulting errors
	processResponse(n, r, "nat.create.error", "nats.create.", "nat.create", "errored")
	processResponse(n, r, "nat.update.error", "nats.create.", "nat.update", "errored")
	processResponse(n, r, "nat.delete.error", "nats.delete.", "nat.delete", "errored")

	runtime.Goexit()
}
