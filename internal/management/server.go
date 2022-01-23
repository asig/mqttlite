/*
 * Copyright (c) 2022 Andreas Signer <asigner@gmail.com>
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
package management

import (
	"github.com/asig/go-logging/logging"
	"github.com/asig/mqttlite/server"
	"net/http"
)

var (
	logger *logging.Logger
)

func Init() {
	logger = logging.Get("management")
}

type Server struct {
	hostPort string
	mqttServer *server.Server

	shouldStop chan bool
}

func New(hostPort string, mqttServer *server.Server) *Server {
	return &Server{
		hostPort : hostPort,
		mqttServer: mqttServer,
		shouldStop: make(chan bool),
	}
}

func (s *Server) quit(w http.ResponseWriter, req *http.Request) {
	logger.Info("Shutdown requested")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	s.mqttServer.Stop()
	s.Stop()
}

func (s *Server) overview(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte("<h1>mqttlite</h1>"))
}

func (s *Server) Start() error {
	sm := http.NewServeMux()
	sm.HandleFunc("/quit", s.quit)
	sm.HandleFunc("/", s.overview)


	logger.Infof("Listening on %s", s.hostPort)

	httpServer :=  &http.Server{Addr: s.hostPort, Handler: sm}
	go httpServer.ListenAndServe()

	<- s.shouldStop
	return nil
}

func (s* Server) Stop() {
	s.shouldStop <- true
}


