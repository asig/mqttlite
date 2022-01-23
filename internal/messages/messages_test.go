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
	"testing"
)

func compareBytes(t *testing.T, got, want []byte) {
	if len(got) != len(want) {
		t.Errorf("Got len(got) = %d, want len(got) = %d", len(got), len(want))
	} else {
		eq := true
		for i := 0; i < len(got); i++ {
			if got[i] != want[i] {
				eq = false
			}
		}
		if !eq {
			t.Errorf("Got %v, want %v", got, want)
		}
	}
}

func TestEncodeLength(t *testing.T) {
	tests := []struct {
		desc string
		len  int
		want []byte
	}{
		{
			desc: "Single byte",
			len:  124,
			want: []byte{124},
		},
		{
			desc: "Two byte",
			len:  130,
			want: []byte{0x82, 0x01},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := encodeLength(test.len)
			compareBytes(t, got, test.want)
		})
	}
}

func TestPayloadWriter(t *testing.T) {
	tests := []struct {
		desc string
		f    func(pw *PayloadWriter)
		want []byte
	}{
		{
			desc: "Write string",
			f:    func(pw *PayloadWriter) { pw.WriteString("foo") },
			want: []byte{0, 3, 'f', 'o', 'o'},
		},
		{
			desc: "Write Uint16",
			f:    func(pw *PayloadWriter) { pw.WriteUint16(0xabcd) },
			want: []byte{0xab, 0xcd},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			msg := &Message{PingReq, 0, []byte{}}
			pw := msg.PayloadWriter()
			test.f(pw)
			got := msg.Data
			compareBytes(t, got, test.want)
		})
	}
}
