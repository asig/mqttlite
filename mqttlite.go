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
	"fmt"
	"net"
	"time"

	"github.com/asig/go-logging/logging"

	"github.com/asig/mqttlite/server"
)

var (
	logger *logging.Logger

	flagPort    = flag.Int("port", 1883, "Port to listen on")
	flagAddress = flag.String("address", "", "Address to listen on")

	sessions = server.SessionList{}
)

func init() {
	flag.Parse()
	logging.Initialize()
	logger = logging.Get("main")
	server.Init()
}

func main() {

	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			sessions.RemoveDead()
		}
	}()

	addr := fmt.Sprintf("%s:%d", *flagAddress, *flagPort)
	logger.Infof("Listening on %s", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("Can't listen: %s", err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Warningf("Can't accept connection: %s", err)
		}
		go func() {
			s := sessions.NewSession(conn)
			logger.Infof("Starting session %+v", s)
			s.Run()
			sessions.Remove(s)
		}()
	}
}
