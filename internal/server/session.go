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
package server

import (
	"net"
	"sync"
	"time"

	"github.com/asig/mqttlite/internal/messages"
)

var (
	nextSessionId uint32
)

type Subscription struct {
	qos    uint8
	filter TopicFilter
}

type outstandingMessage struct {
	nextSendTime time.Time // Time when the next retry should happen
	sendCount    int       // how often the message was sent
	packetId     uint16
}

func (msg *outstandingMessage) computeNextSendTime() {
	msg.sendCount++
	delay := time.Duration(10*msg.sendCount) * time.Second
	if delay > 60*time.Second {
		delay = 60 * time.Second
	}
	msg.nextSendTime = msg.nextSendTime.Add(delay)
}

type outstandingPublishMessage struct {
	outstandingMessage

	topic   TopicName
	payload []byte
	dup     bool
	qos     uint8
	retain  bool
}

type outstandingPubRelMessage struct {
	outstandingMessage
}

type outstandingPubRecMessage struct {
	outstandingMessage
}

func (dm *outstandingPublishMessage) toMessage() *messages.Message {
	var dupFlag uint8 = 0
	if dm.dup {
		dupFlag = 1
	}
	var retainFlag uint8 = 0
	if dm.retain {
		retainFlag = 1
	}
	msg := &messages.Message{messages.Publish, dupFlag<<3 | dm.qos<<1 | retainFlag, []byte{}}
	pw := msg.PayloadWriter()
	pw.WriteString(string(dm.topic))
	if dm.qos > 0 {
		pw.WriteUint16(dm.packetId)
	}
	pw.WriteBytes(dm.payload)
	return msg
}

func (dm *outstandingPubRelMessage) toMessage() *messages.Message {
	msg := &messages.Message{messages.PubRel, 2, []byte{}}
	pw := msg.PayloadWriter()
	pw.WriteUint16(dm.packetId)
	return msg
}

func (dm *outstandingPubRecMessage) toMessage() *messages.Message {
	msg := &messages.Message{messages.PubRec, 0, []byte{}}
	pw := msg.PayloadWriter()
	pw.WriteUint16(dm.packetId)
	return msg
}

type will struct {
	retain bool
	qos    uint8
	topic  TopicName
	data   []byte
}

type Session struct {
	id                uint32
	conn              net.Conn
	createdAt         time.Time
	connected         bool
	keepAliveDuration time.Duration
	nextPacketId      uint16
	lock              sync.Mutex
	subscriptions     map[TopicFilter]*Subscription

	will *will

	lastMessageReceived time.Time

	unacknowledgedPublishes map[uint16]*outstandingPublishMessage
	unacknowledgedPubRels   map[uint16]*outstandingPubRelMessage
	unacknowledgedPubRecs   map[uint16]*outstandingPubRecMessage

	server *Server
}

func (s *Session) deadlineExceeded() bool {
	deadline := s.lastMessageReceived.Add(s.keepAliveDuration)
	return time.Now().After(deadline)
}

func (s *Session) newOutstandingPublishMessage(topicName TopicName, data []byte, retain bool, qos uint8) *outstandingPublishMessage {
	om := &outstandingPublishMessage{
		outstandingMessage: outstandingMessage{
			packetId: s.GetNextPacketId(),
		},
		topic:   topicName,
		payload: data,
		dup:     false,
		qos:     qos,
		retain:  retain,
	}
	return om
}

func (s *Session) sendPublish(msg *outstandingPublishMessage) {
	logger.Infof("Session %d: --> PUBLISH %#v", s.id, msg)
	if msg.qos > 0 {
		msg.nextSendTime = time.Now().Add(10 * time.Second)
		msg.sendCount = 1
		s.lock.Lock()
		s.unacknowledgedPublishes[msg.packetId] = msg
		s.lock.Unlock()
	}
	msg.toMessage().Send(s.conn)
	logger.Infof("Session %d: --> PUBLISH DONE %#v", s.id, msg)
}

func (s *Session) sendSubAck(packetId uint16, maxQoS int) {
	logger.Infof("Session %d: --> SUBACK(%d)", s.id, packetId)
	msg := &messages.Message{messages.SubAck, 0, []byte{}}
	pw := msg.PayloadWriter()
	pw.WriteUint16(packetId)
	pw.WriteUint8(uint8(maxQoS))
	msg.Send(s.conn)
}

func (s *Session) sendPubRec(packetId uint16) {
	logger.Infof("Session %d: --> PUBREC(%d)", s.id, packetId)
	m := &outstandingPubRecMessage{
		outstandingMessage: outstandingMessage{
			packetId:     packetId,
			nextSendTime: time.Now().Add(10 * time.Second),
			sendCount:    1,
		},
	}
	s.lock.Lock()
	s.unacknowledgedPubRecs[packetId] = m
	s.lock.Unlock()
	m.toMessage().Send(s.conn)
}

func (s *Session) sendPubRel(packetId uint16) {
	logger.Infof("Session %d: --> PUBREL(%d)", s.id, packetId)
	m := &outstandingPubRelMessage{
		outstandingMessage: outstandingMessage{
			packetId:     packetId,
			nextSendTime: time.Now().Add(10 * time.Second),
			sendCount:    1,
		},
	}
	s.lock.Lock()
	s.unacknowledgedPubRels[packetId] = m
	s.lock.Unlock()
	m.toMessage().Send(s.conn)
}

func (s *Session) sendPubComp(packetId uint16) {
	logger.Infof("Session %d: --> PUBCOMP(%d)", s.id, packetId)
	m := &messages.Message{messages.PubComp, 0, []byte{}}
	pw := m.PayloadWriter()
	pw.WriteUint16(packetId)
	m.Send(s.conn)
}

func (s *Session) AddSubscription(filter TopicFilter, qos uint8) {
	sub := &Subscription{qos, filter}
	logger.Infof("Session %d: New Subscription %+v", s.id, sub)

	// [MQTT-3.3.1-6].
	logger.Infof("Session %d: Matching topics:", s.id)
	matchingTopics := topics.filter(filter)
	for _, topic := range matchingTopics {
		logger.Infof("  %s", topic.name)
		if topic.retainedMessage != nil {
			copy := *topic.retainedMessage
			if copy.qos < qos {
				copy.qos = qos
			}
			copy.retain = true // [MQTT-3.3.1-8]
			s.sendPublish(&copy)
		}
	}
	s.subscriptions[filter] = sub
}

func (s *Session) RemoveSubscription(filter TopicFilter) {
	delete(s.subscriptions, filter)
}

func (s *Session) findSubscriptions(name TopicName) []*Subscription {
	var res []*Subscription
	for _, sub := range s.subscriptions {
		if sub.filter.matches(name) {
			res = append(res, sub)
		}
	}
	return res
}

func (s *Session) Close() {
	logger.Infof("Session %d: Closing session", s.id)

	if s.will != nil {
		om := s.newOutstandingPublishMessage(s.will.topic, s.will.data, s.will.retain, s.will.qos)
		s.sendToSubscribers(om)
	}

	s.conn.Close()
}

func (s *Session) GetNextPacketId() uint16 {
	s.lock.Lock()
	res := s.nextPacketId
	s.nextPacketId++
	s.lock.Unlock()
	return res
}

func (s *Session) sendConnAck(res byte, sessionPresent bool) {
	logger.Infof("Session %d: --> CONACK", s.id)
	b0 := byte(0)
	if sessionPresent {
		b0 = 1
	}
	msg := messages.Message{messages.ConnAck, 0, []byte{b0, res}}
	msg.Send(s.conn)
}

func (s *Session) sendPingResp() {
	logger.Infof("Session %d: --> PINGRESP", s.id)
	msg := &messages.Message{messages.PingResp, 0, []byte{}}
	msg.Send(s.conn)
}

func (s *Session) sendToSubscribers(om *outstandingPublishMessage) {
	// Publish to subscribed sessions
	for _, sess := range s.server.sessions {
		logger.Infof("Session %d: checking Session %d", s.id, sess.id)
		if sess == s {
			continue
		}
		if subs := sess.findSubscriptions(om.topic); len(subs) > 0 {
			logger.Infof("Session %d: publish message to Session %d", s.id, sess.id)
			omCopy := *om
			omCopy.retain = false // [MQTT-3.3.1-9]
			if subs[0].qos < omCopy.qos {
				omCopy.qos = subs[0].qos
			}
			sess.sendPublish(&omCopy)
		} else {
			logger.Infof("Session %d: Session %d not subscribed to %s", s.id, sess.id, om.topic)
		}
	}
}

func (s *Session) handlePublish(msg *messages.Message) {
	dupFlag := (msg.Flags & 4) > 0
	retainFlag := (msg.Flags & 1) > 0
	qos := (msg.Flags >> 1) & 3

	logger.Infof("Session %d: <-- PUBLISH", s.id)
	logger.Infof("  DUP: %t", dupFlag)
	logger.Infof("  QoS: %d", qos)
	logger.Infof("  RETAIN: %t", retainFlag)

	pr := msg.PayloadReader(0)
	topicName := TopicName(pr.GetString())
	logger.Infof("  topicName: %s", topicName)
	var packetId uint16
	if qos > 0 {
		packetId = pr.GetUint16()
		logger.Infof("  packetId: %d", packetId)
	}
	data := msg.Data[pr.GetCurPos():]
	logger.Infof("  data: %+v", data)
	logger.Infof("  data (string): %+v", string(data))

	om := s.newOutstandingPublishMessage(topicName, data, retainFlag, qos)

	// Store retained message if necessary
	if retainFlag { // [MQTT-3.3.1-5]
		tops := topics.filter(TopicFilter(topicName))
		if len(tops) == 0 {
			tops = []*Topic{&Topic{topicName, om}}
			topics = append(topics, tops...)
		}
		if len(msg.Data) == 0 {
			tops[0].retainedMessage = nil
		} else {
			tops[0].retainedMessage = om
		}
	}

	// Publish to subscribed sessions
	s.sendToSubscribers(om)

	switch qos {
	case 0: // Do nothing
	case 1: // Send PUBACK
		msg = &messages.Message{messages.PubAck, 0, []byte{}}
		pw := msg.PayloadWriter()
		pw.WriteUint16(packetId)
		logger.Infof("Sending PUBACK for Packet ID %d", packetId)
		msg.Send(s.conn)
	case 2: // Send PUBREC
		s.sendPubRec(packetId)
	}
}

func (s *Session) handlePubAck(msg *messages.Message) {
	pr := msg.PayloadReader(0)
	packetId := pr.GetUint16()
	logger.Infof("Session %d: <-- PUBACK(%d)", s.id, packetId)
	s.lock.Lock()
	defer s.lock.Unlock()
	m, ok := s.unacknowledgedPublishes[packetId]
	if !ok {
		logger.Infof("Session %d: No outstanding PUBLISH for Packet Id %d, ignoring PUBACK", s.id, packetId)
		return
	}
	if m.qos != 1 {
		logger.Infof("Session %d: PUBLISH(%d) has qos %d, ignoring PUBACK", s.id, packetId, m.qos)
		return
	}
	delete(s.unacknowledgedPublishes, packetId)
}

func (s *Session) handlePubRec(msg *messages.Message) {
	pr := msg.PayloadReader(0)
	packetId := pr.GetUint16()
	logger.Infof("Session %d: <-- PUBREC(%d)", s.id, packetId)
	s.lock.Lock()
	_, ok := s.unacknowledgedPublishes[packetId]
	delete(s.unacknowledgedPublishes, packetId)
	s.lock.Unlock()
	if !ok {
		logger.Infof("Session %d: No outstanding PUBLISH for Packet Id %d, ignoring PUBREC", s.id, packetId)
		return
	}

	// send PUBREL
	s.sendPubRel(packetId)
}

func (s *Session) handlePubRel(msg *messages.Message) {
	pr := msg.PayloadReader(0)
	packetId := pr.GetUint16()
	logger.Infof("Session %d: <-- PUBREL(%d)", s.id, packetId)
	s.lock.Lock()
	_, ok := s.unacknowledgedPubRecs[packetId]
	delete(s.unacknowledgedPubRecs, packetId)
	s.lock.Unlock()
	if !ok {
		logger.Infof("Session %d: No outstanding PUBREC for Packet Id %d, ignoring PUBREL", s.id, packetId)
		return
	}

	// send PUBCOMP
	s.sendPubComp(packetId)
}

func (s *Session) handlePubComp(msg *messages.Message) {
	pr := msg.PayloadReader(0)
	packetId := pr.GetUint16()
	logger.Infof("Session %d: <-- PUBCOMP(%d)", s.id, packetId)
	s.lock.Lock()
	defer s.lock.Unlock()
	_, ok := s.unacknowledgedPubRels[packetId]
	if !ok {
		logger.Infof("Session %d: No outstanding PUBREL for Packet Id %d, ignoring PUBCOMP", s.id, packetId)
		return
	}
	delete(s.unacknowledgedPubRels, packetId)
}

func (s *Session) handlePing(msg *messages.Message) {
	logger.Infof("Session %d: <-- PINGREQ", s.id)
	s.sendPingResp()
}

func (s *Session) handleSubscribe(msg *messages.Message) {
	logger.Infof("Session %d: <-- SUBSCRIBE", s.id)
	if msg.Flags != 2 { // [MQTT-3.8.1-1]
		logger.Warningf("Invalid flags %d, closing connection", msg.Flags)
		s.Close()
		return
	}
	pr := msg.PayloadReader(0)
	packetId := pr.GetUint16()
	maxQos := -1
	for !pr.AtEnd() {
		topicFilter := TopicFilter(pr.GetString())
		qos := pr.GetUint8()
		if qos > 2 { // MQTT-3.8.3-4]
			logger.Warningf("Invalid QoS %d, closing connection", qos)
			s.Close()
			return
		}
		if int(qos) > maxQos {
			maxQos = int(qos)
		}

		s.AddSubscription(topicFilter, qos)

	}
	if maxQos == -1 { // [MQTT-3.8.3-3]
		logger.Warningf("SUBSCRIBE without topic filters, closing connection")
		s.Close()
		return
	}

	s.sendSubAck(packetId, maxQos)
}

func (s *Session) handleUnsubscribe(msg *messages.Message) {
	logger.Infof("Session %d: <-- UNSUBSCRIBE", s.id)
	if msg.Flags != 2 { // [MQTT-3.10.1-1]
		logger.Warningf("Invalid flags %d, closing connection", msg.Flags)
		s.Close()
		return
	}
	pr := msg.PayloadReader(0)
	packetId := pr.GetUint16()
	cnt := 0
	for !pr.AtEnd() {
		cnt++
		topicFilter := TopicFilter(pr.GetString())
		s.RemoveSubscription(topicFilter)
	}
	if cnt == 0 { // [MQTT-3.10.3-2]
		logger.Warningf("UNSUBSCRIBE without topic filters, closing connection")
		s.Close()
		return
	}

	msg = &messages.Message{messages.UnsubAck, 0, []byte{}}
	pw := msg.PayloadWriter()
	pw.WriteUint16(packetId)
	msg.Send(s.conn)
}

func (s *Session) handleConnect(msg *messages.Message) {
	var payloadOfs uint16
	var flags byte

	// Check protocol
	if (msg.Data[0] == 0 && msg.Data[1] == 6 && msg.Data[2] == 'M' && msg.Data[3] == 'Q' && msg.Data[4] == 'I' && msg.Data[5] == 's' && msg.Data[6] == 'd' && msg.Data[7] == 'p') {
		// MQTT 3.1
		protocolVersion := msg.Data[8]
		flags = msg.Data[9]
		payloadOfs = 12

		if protocolVersion != 3 {
			logger.Infof("Bad protocol version %d, disconnecting", protocolVersion)
			s.sendConnAck(0x01 /*unacceptable protocol version*/, false)
			s.Close()
			return
		}


	} else if (msg.Data[0] == 0 && msg.Data[1] == 4 && msg.Data[2] == 'M' && msg.Data[3] == 'Q' && msg.Data[4] == 'T' && msg.Data[5] == 'T') {
		// MQTT 3.1.1
		protocolVersion := msg.Data[6]
		flags = msg.Data[7]
		payloadOfs = 10

		if protocolVersion != 4 {
			logger.Infof("Bad protocol version %d, disconnecting", protocolVersion)
			s.sendConnAck(0x01 /*unacceptable protocol version*/, false)
			s.Close()
			return
		}

		if (flags & 1) != 0 {
			logger.Infof("Reserved flag is not 0, disconnecting")
			s.Close()
			return
		}

	} else {
		logger.Info("Unknown protocol, disconnecting")
		s.Close()
		return
	}

	userNameFlag := (flags & 128) > 0
	passwordFlag := (flags & 64) > 0
	willRetain := (flags & 32) > 0
	willQoS := (flags >> 3) & 3
	willFlag := (flags & 4) > 0
	cleanSession := (flags & 2) > 0

	logger.Infof("  userNameFlag = %t", userNameFlag)
	logger.Infof("  passwordFlag = %t", passwordFlag)
	logger.Infof("  willRetain   = %t", willRetain)
	logger.Infof("  willQoS      = %d", willQoS)
	logger.Infof("  willFlag     = %t", willFlag)
	logger.Infof("  cleanSession = %t", cleanSession)

	s.keepAliveDuration = time.Duration(int64(uint16(msg.Data[8])<<8|uint16(msg.Data[9]))) * time.Second
	logger.Infof("KeepAliceSecs = %d", s.keepAliveDuration)

	pr := msg.PayloadReader(payloadOfs)
	clientId := pr.GetString()
	logger.Infof("ClientID: %s", clientId)

	if willFlag {
		willTopic := pr.GetString()
		logger.Infof("WillTopic: %s", willTopic)

		willMessage := pr.GetBytes()
		logger.Infof("WillMessage: %v", willMessage)

		s.will = &will{
			retain: willRetain,
			qos:    willQoS,
			topic:  TopicName(willTopic),
			data:   willMessage,
		}
	}

	if userNameFlag {
		userName := pr.GetString()
		logger.Infof("UserName: %s", userName)
	}
	if passwordFlag {
		password := pr.GetBytes()
		logger.Infof("Password: %s", password)
	}

	s.connected = true
	s.sendConnAck(0x00, true)
}

func (s *Session) handleDisconnect(msg *messages.Message) {
	logger.Infof("Session %d: <-- DISCONNECT", s.id)
	// [MQTT-3.14.1-1]: Only clean disconnect if flags == 0
	if msg.Flags != 0 {
		logger.Warningf("Invalid flags %d, closing connection", msg.Flags)
		s.Close()
	}

	s.will = nil
	s.Close()
}

func (s *Session) checkResend() {
	now := time.Now()
	for _, msg := range s.unacknowledgedPublishes {
		if msg.nextSendTime.Before(now) {
			logger.Infof("Resending PUBLISH(%d)", msg.packetId)
			msg.outstandingMessage.computeNextSendTime()
			msg.dup = true
			msg.toMessage().Send(s.conn)
		}
	}
	for _, msg := range s.unacknowledgedPubRels {
		if msg.nextSendTime.Before(now) {
			logger.Infof("Resending PUBREL(%d)", msg.packetId)
			msg.outstandingMessage.computeNextSendTime()
			msg.toMessage().Send(s.conn)
		}
	}
	for _, msg := range s.unacknowledgedPubRecs {
		if msg.nextSendTime.Before(now) {
			logger.Infof("Resending PUBREC(%d)", msg.packetId)
			msg.outstandingMessage.computeNextSendTime()
			msg.toMessage().Send(s.conn)
		}
	}
}

func (s *Session) Run() {
	defer s.Close()

	msg, _ := messages.ReadMessageWithTimeout(s.conn, 30*time.Second)
	if msg == nil {
		return
	}
	if msg.Type != messages.Connect {
		return
	}
	s.lastMessageReceived = time.Now()

	s.handleConnect(msg)
	if !s.connected {
		logger.Infof("Not connected.")
		return
	}

	ticker := time.NewTicker(1000 * time.Millisecond)
	go func() {
		for range ticker.C {
			s.checkResend()
		}
	}()

	for {
		msg, err := messages.ReadMessageWithTimeout(s.conn, 30*time.Second)
		s.lastMessageReceived = time.Now()
		switch err {
		case messages.ErrEof:
			ticker.Stop()
			return
		case messages.ErrNone:
			break
		case messages.ErrTimeout:
			continue
		default:
			logger.Infof("Unexpected error %v, terminating session", err)
			ticker.Stop()
			return
		}
		switch msg.Type {
		case messages.Publish:
			s.handlePublish(msg)
		case messages.PingReq:
			s.handlePing(msg)
		case messages.Subscribe:
			s.handleSubscribe(msg)
		case messages.Unsubscribe:
			s.handleUnsubscribe(msg)
		case messages.PubAck:
			s.handlePubAck(msg)
		case messages.PubRec:
			s.handlePubRec(msg)
		case messages.PubRel:
			s.handlePubRel(msg)
		case messages.PubComp:
			s.handlePubComp(msg)
		case messages.Disconnect:
			s.handleDisconnect(msg)
		default:
			logger.Infof("Unhandled message: %+v", msg)
		}
	}
}
