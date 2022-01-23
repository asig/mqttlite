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
package server

import (
	"net"
	"sync"
	"time"
)

type Server struct {
	hostPort string

	shouldStop chan bool
	sessionsLock     sync.Mutex
	sessions []*Session

}

func New(hostPort string) *Server {
	return &Server{
		hostPort: hostPort,
		shouldStop: make(chan bool),
	}
}

func (s *Server) listenAndServe(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Warningf("Can't accept connection: %s", err)
		}
		go func() {
			sess := s.NewSession(conn)
			logger.Infof("Starting session %+v", sess)
			sess.Run()
			s.Remove(sess)
		}()
	}
}

func (s *Server) Start() error {
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			s.RemoveDead()
		}
	}()

	logger.Infof("Listening on %s", s.hostPort)
	listener, err := net.Listen("tcp", s.hostPort)
	if err != nil {
		return err
	}
	go s.listenAndServe(listener)
	<- s.shouldStop
	listener.Close()
	return nil
}

func (s* Server) Stop() {
	s.shouldStop <- true
}

func (s *Server) NewSession(conn net.Conn) *Session {
	id := nextSessionId
	nextSessionId++
	sess := &Session{
		id:                      id,
		conn:                    conn,
		createdAt:               time.Now(),
		connected:               false,
		nextPacketId:            1,
		subscriptions:           make(map[TopicFilter]*Subscription),
		unacknowledgedPublishes: make(map[uint16]*outstandingPublishMessage),
		unacknowledgedPubRels:   make(map[uint16]*outstandingPubRelMessage),
		unacknowledgedPubRecs:   make(map[uint16]*outstandingPubRecMessage),
		server:                s,
	}
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()
	s.sessions = append(s.sessions, sess)
	return sess
}

func (s *Server) RemoveDead() {
	toRemove := make(map[*Session]bool)
	for _, sess := range s.sessions {
		if sess.deadlineExceeded() {
			logger.Infof("Session %d: No message within %s, closing.", sess.id, sess.keepAliveDuration)
			sess.Close()
			toRemove[sess] = true
		}
	}
	s.sessionsLock.Lock()
	var newSessions []*Session
	for _, sess := range s.sessions {
		if _, ok := toRemove[sess]; ok {
			continue
		}
		newSessions = append(newSessions, sess)
	}
	s.sessions = newSessions
	s.sessionsLock.Unlock()
}

func (s *Server) Remove(c *Session) {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()
	for i := 0; i < len(s.sessions); i++ {
		if s.sessions[i] == c {
			// See https://github.com/golang/go/wiki/SliceTricks
			s.sessions[i] = s.sessions[len(s.sessions)-1]
			s.sessions[len(s.sessions)-1] = nil
			s.sessions = s.sessions[:len(s.sessions)-1]
			return
		}
	}
}

