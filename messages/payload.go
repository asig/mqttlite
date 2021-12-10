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

//  ----------------------------------------------
// PayloadReader
//  ----------------------------------------------

type PayloadReader struct {
	msg    *Message
	curPos uint16
}

func (p *PayloadReader) GetUint16() uint16 {
	res := uint16(p.msg.Data[p.curPos])<<8 | uint16(p.msg.Data[p.curPos+1])
	p.curPos += 2
	return res
}

func (p *PayloadReader) GetUint8() uint8 {
	res := uint8(p.msg.Data[p.curPos])
	p.curPos += 1
	return res
}

func (p *PayloadReader) GetBytes() []byte {
	l := p.GetUint16()
	res := p.msg.Data[p.curPos : p.curPos+l]
	p.curPos += l
	return res
}

func (p *PayloadReader) GetString() string {
	return string(p.GetBytes())
}

func (p *PayloadReader) GetCurPos() uint16 {
	return p.curPos
}

func (p *PayloadReader) AtEnd() bool {
	return p.curPos >= uint16(len(p.msg.Data))
}

//  ----------------------------------------------
// PayloadWriter
//  ----------------------------------------------

type PayloadWriter struct {
	msg *Message
}

func (p *PayloadWriter) WriteUint16(v uint16) {
	p.msg.Data = append(p.msg.Data, []byte{byte(v >> 8), byte(v & 255)}...)
}

func (p *PayloadWriter) WriteUint8(v uint8) {
	p.msg.Data = append(p.msg.Data, v)
}

func (p *PayloadWriter) WriteBytes(b []byte) {
	p.msg.Data = append(p.msg.Data, b...)
}

func (p *PayloadWriter) WriteString(s string) {
	p.WriteUint16(uint16(len(s)))
	p.WriteBytes([]byte(s))
}
