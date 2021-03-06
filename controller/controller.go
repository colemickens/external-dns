/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/kubernetes-incubator/external-dns/plan"
	"github.com/kubernetes-incubator/external-dns/provider"
	"github.com/kubernetes-incubator/external-dns/source"
)

// Controller is responsible for orchestrating the different components.
// It works in the following way:
// * Ask the DNS provider for current list of endpoints.
// * Ask the Source for the desired list of endpoints.
// * Take both lists and calculate a Plan to move current towards desired state.
// * Tell the DNS provider to apply the changes calucated by the Plan.
type Controller struct {
	Zone string

	Source   source.Source
	Provider provider.Provider
}

// RunOnce runs a single iteration of a reconciliation loop.
func (c *Controller) RunOnce() error {
	records, err := c.Provider.Records(c.Zone)
	if err != nil {
		return err
	}

	endpoints, err := c.Source.Endpoints()
	if err != nil {
		return err
	}

	plan := &plan.Plan{
		Current: records,
		Desired: endpoints,
	}

	plan = plan.Calculate()

	return c.Provider.ApplyChanges(c.Zone, &plan.Changes)
}

// Run runs RunOnce in a loop with a delay until stopChan receives a value.
func (c *Controller) Run(stopChan <-chan struct{}) {
	for {
		err := c.RunOnce()
		if err != nil {
			log.Fatal(err)
		}

		select {
		case <-time.After(time.Minute):
		case <-stopChan:
			log.Infoln("terminating main controller loop")
			return
		}
	}
}
