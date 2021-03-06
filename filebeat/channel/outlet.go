// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package channel

import (
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common/atomic"
)

type outlet struct {
	wg     eventCounter
	client beat.Client
	isOpen atomic.Bool
}

func newOutlet(client beat.Client, wg eventCounter) *outlet {
	o := &outlet{
		wg:     wg,
		client: client,
		isOpen: atomic.MakeBool(true),
	}
	return o
}

func (o *outlet) Close() error {
	isOpen := o.isOpen.Swap(false)
	if isOpen {
		return o.client.Close()
	}
	return nil
}

func (o *outlet) OnEvent(d *util.Data) bool {
	if !o.isOpen.Load() {
		return false
	}

	event := d.GetEvent()
	if d.HasState() {
		event.Private = d.GetState()
	}

	if o.wg != nil {
		o.wg.Add(1)
	}

	o.client.Publish(event)

	// Note: race condition on shutdown:
	//  The underlying beat.Client is asynchronous. Without proper ACK
	//  handler we can not tell if the event made it 'through' or the client
	//  close has been completed before sending. In either case,
	//  we report 'false' here, indicating the event eventually being dropped.
	//  Returning false here, prevents the harvester from updating the state
	//  to the most recently published events. Therefore, on shutdown the harvester
	//  might report an old/outdated state update to the registry, overwriting the
	//  most recently
	//  published offset in the registry on shutdown.
	return o.isOpen.Load()
}
