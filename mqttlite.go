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

	"github.com/asig/go-logging/logging"

	"github.com/asig/mqttlite/server"
)

var (
	logger *logging.Logger

	flagAddress = flag.String("address", ":1883", "Address to listen on, e.g. localhost:1883. If only a port is specified, the server will listen on all addresses.")
)

func init() {
	flag.Parse()
	logging.Initialize()
	logger = logging.Get("main")
	server.Init()
}

func main() {
	srv := server.New(*flagAddress)
	srv.Start()
}
