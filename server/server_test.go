/*
 * Copyright (c) 2018 Andreas Signer <asigner@gmail.com>
 *
 * This file is part of mqttlite.
 *
 * kosmos-cp1 is free software: you can redistribute it and/or
 * modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * kosmos-cp1 is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with mqttlite.  If not, see <http://www.gnu.org/licenses/>.
 */
package server

import (
	"github.com/asig/go-logging/logging"

	"fmt"
	"net"
	"reflect"
	"testing"
	"time"
)

type fakeConn struct {
}

func (c *fakeConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (c *fakeConn) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (c *fakeConn) Close() error {
	return nil
}

func (c *fakeConn) LocalAddr() net.Addr {
	return nil
}

func (c *fakeConn) RemoteAddr() net.Addr {
	return nil
}

func (c *fakeConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *fakeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *fakeConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func init() {
	logger = logging.Get("test")
}

func TestSplit(t *testing.T) {
	tests := []struct {
		desc string
		s    string
		want []string
	}{
		{
			desc: "Single topic",
			s:    "a",
			want: []string{"a"},
		},
		{
			desc: "Empty start topic",
			s:    "/a",
			want: []string{"", "a"},
		},
		{
			desc: "Empty end topic",
			s:    "a/",
			want: []string{"a", ""},
		},
		{
			desc: "Multiple topics",
			s:    "a/b/c",
			want: []string{"a", "b", "c"},
		},
		{
			desc: "Empty topics in between",
			s:    "a//c",
			want: []string{"a", "", "c"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := split(test.s)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("split(%q): got %q, want %q", test.s, got, test.want)
			}
		})
	}
}

func TestTopicFilterMatches(t *testing.T) {
	tests := []struct {
		desc   string
		filter TopicFilter
		name   TopicName
		want   bool
	}{
		{
			desc:   "No wildcards",
			filter: TopicFilter("sport/tennis/player1"),
			name:   TopicName("sport/tennis/player1"),
			want:   true,
		},
		{
			desc:   "No wildcards 2",
			filter: TopicFilter("sport/tennis/player1"),
			name:   TopicName("sport/tennis/player2"),
			want:   false,
		},
		{
			desc:   "Multi-level wildcard 1",
			filter: TopicFilter("sport/tennis/player1/#"),
			name:   TopicName("sport/tennis/player1"),
			want:   true,
		},
		{
			desc:   "Multi-level wildcard 2",
			filter: TopicFilter("sport/tennis/player1/#"),
			name:   TopicName("sport/tennis/player1/ranking"),
			want:   true,
		},
		{
			desc:   "Multi-level wildcard 3",
			filter: TopicFilter("sport/tennis/player1/#"),
			name:   TopicName("sport/tennis/player1/score/wimbledon"),
			want:   true,
		},
		{
			desc:   "Multi-level wildcard 4",
			filter: TopicFilter("sport/tennis/player1/#"),
			name:   TopicName("sport/tennis/player2/score/wimbledon"),
			want:   false,
		},
		{
			desc:   "System topics 1",
			filter: TopicFilter("#"),
			name:   TopicName("$SYS"),
			want:   false,
		},
		{
			desc:   "System topics 2",
			filter: TopicFilter("#"),
			name:   TopicName("$SYS/foo"),
			want:   false,
		},
		{
			desc:   "System topics 3",
			filter: TopicFilter("+/monitor/Clients"),
			name:   TopicName("$SYS/monitor/Clients"),
			want:   false,
		},
		{
			desc:   "System topics 4",
			filter: TopicFilter("$SYS/#"),
			name:   TopicName("$SYS"),
			want:   true,
		},
		{
			desc:   "System topics 5",
			filter: TopicFilter("$SYS/monitor/+"),
			name:   TopicName("$SYS/monitor/Clients"),
			want:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := test.filter.matches(test.name)
			if got != test.want {
				t.Errorf("%q.matches(%q): got %t, want %t", test.filter, test.name, got, test.want)
			}
		})
	}
}

func TestSessionlistRemoveDead(t *testing.T) {
	sessions := SessionList{}
	sessions.sessions = []*Session{
		&Session{
			id:                  23,
			keepAliveDuration:   100 * time.Second,
			lastMessageReceived: time.Now().Add(-50 * time.Second),
			conn:                &fakeConn{},
		},
		&Session{
			id:                  42,
			keepAliveDuration:   100 * time.Second,
			lastMessageReceived: time.Now().Add(-150 * time.Second),
			conn:                &fakeConn{},
		},
	}
	fmt.Printf("*********************1\n")
	sessions.RemoveDead()
	fmt.Printf("*********************2\n")
	got := len(sessions.sessions)
	fmt.Printf("*********************3\n")
	if got != 1 {
		t.Errorf("len(sessions.sessions): Got %d, want %d", got, 1)

		if sessions.sessions[0].id != 23 {
			t.Errorf("len(sessions.sessions): Got %d, want %d", sessions.sessions[0].id, 23)
		}
	}
}
