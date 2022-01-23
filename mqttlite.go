/*
 * Copyright (c) 2018 Andreas Signer <asigner@gmail.com>
 *
 * This file is part of mqttlite.
 *
 * mqttlite is free software: you can redistribute it and/or
 * modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * mqttlite is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with mqttlite.  If not, see <http://www.gnu.org/licenses/>.
 */
package main

import (
	"flag"
	"sync"

	"github.com/asig/go-logging/logging"

	"github.com/asig/mqttlite/management"
	"github.com/asig/mqttlite/server"
)

var (
	logger *logging.Logger

	flagAddress           = flag.String("address", ":1883", "Address to listen on, e.g. localhost:1883. If only a port is specified, the server will listen on all addresses.")
	flagManagementAddress = flag.String("management_address", "localhost:8883", "Address the management server will listen on.")
)

func init() {
	flag.Parse()
	logging.Initialize()
	logger = logging.Get("main")
	server.Init()
	management.Init()
}

func main() {
	wg := &sync.WaitGroup{}

	// Start MQTT Server
	srv := server.New(*flagAddress)
	wg.Add(1)
	go func() {
		defer wg.Done()
		srv.Start()
	}()

	if *flagManagementAddress != "" {
		// Start Management Server
		mgmr := management.New(*flagManagementAddress, srv)
		wg.Add(1)
		go func() {
			defer wg.Done()
			mgmr.Start()
		}()
	}

	wg.Wait()
}
