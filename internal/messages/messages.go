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
package messages

import (
	"io"
	"net"
	"time"
)

type MessageType uint8

const (
	Connect     MessageType = 1
	ConnAck     MessageType = 2
	Publish     MessageType = 3
	PubAck      MessageType = 4
	PubRec      MessageType = 5
	PubRel      MessageType = 6
	PubComp     MessageType = 7
	Subscribe   MessageType = 8
	SubAck      MessageType = 9
	Unsubscribe MessageType = 10
	UnsubAck    MessageType = 11
	PingReq     MessageType = 12
	PingResp    MessageType = 13
	Disconnect  MessageType = 14
)

type Message struct {
	Type  MessageType
	Flags uint8
	Data  []byte
}

type Error uint8

const (
	ErrNone Error = iota
	ErrEof
	ErrTimeout
	ErrMalformedRemainingLength
	ErrOther
)

func encodeLength(l int) []byte {
	var buf []byte

	for {
		b := byte(l & 0x7f)
		l = l >> 7
		if l > 0 {
			b |= 128
		}
		buf = append(buf, b)
		if l == 0 {
			break
		}
	}
	return buf
}

func (msg *Message) Send(conn net.Conn) {
	var buf []byte
	buf = append(buf, byte(msg.Type<<4)|byte(msg.Flags))
	buf = append(buf, encodeLength(len(msg.Data))...)
	buf = append(buf, msg.Data...)
	conn.Write(buf)
}

func readBytes(conn net.Conn, timeout time.Duration, b []byte) Error {
	conn.SetReadDeadline(time.Now().Add(timeout))
	pos := 0
	remaining := len(b)
	for {
		buf := make([]byte, remaining)
		read, err := conn.Read(buf)
		if e, ok := err.(net.Error); ok && e.Timeout() {
			return ErrTimeout
		} else if err == io.EOF {
			return ErrEof
		}
		for i := 0; i < read; i++ {
			b[pos] = buf[i]
			pos++
		}
		if pos == len(b) {
			return ErrNone
		}
		remaining -= read
	}
}

func readByte(conn net.Conn, timeout time.Duration) (byte, Error) {
	b := make([]byte, 1)
	err := readBytes(conn, timeout, b)
	if err != ErrNone {
		return 0, err
	}
	return b[0], err
}

func ReadMessageWithTimeout(conn net.Conn, timeout time.Duration) (*Message, Error) {
	var b byte
	var err Error

	msg := &Message{}

	if b, err = readByte(conn, timeout); err != ErrNone {
		return nil, err
	}
	msg.Type = MessageType(b) >> 4
	msg.Flags = b & 0xf

	var l uint64 = 0
	shift := 0
	for {
		if b, err = readByte(conn, timeout); err != ErrNone {
			return nil, err
		}
		cont := (b & 128) > 0
		b = b & 0x7f
		l = (uint64(b) << shift) | l
		if !cont {
			break
		}
		shift += 7
		if shift > 21 {
			return nil, ErrMalformedRemainingLength
		}
	}
	msg.Data = make([]byte, l)
	if err = readBytes(conn, timeout, msg.Data); err != ErrNone {
		return nil, err
	}
	return msg, ErrNone
}

func (msg *Message) PayloadReader(pos uint16) *PayloadReader {
	return &PayloadReader{msg, pos}
}

func (msg *Message) PayloadWriter() *PayloadWriter {
	return &PayloadWriter{msg}
}
