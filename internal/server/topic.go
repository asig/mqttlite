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
	"strings"
)

var (
	topics TopicList
)

type TopicName string
type TopicFilter string
type Topic struct {
	name            TopicName
	retainedMessage *outstandingPublishMessage
}
type TopicList []*Topic

func split(s string) []string {
	var res []string
	start := 0
	for {
		i := start
		for i < len(s) && s[i] != '/' {
			i++
		}
		if i == len(s) {
			break
		}
		res = append(res, s[start:i])
		start = i + 1
	}
	res = append(res, s[start:])
	return res
}

func (f *TopicFilter) matches(t TopicName) bool {
	filterParts := split(string(*f))
	nameParts := split(string(t))

	if (strings.HasPrefix(filterParts[0], "+") || strings.HasPrefix(filterParts[0], "#")) && strings.HasPrefix(nameParts[0], "$") {
		// [MQTT-4.7.2-1]
		return false
	}

	for i := 0; i < len(filterParts); i++ {
		p := filterParts[i]
		if p == "#" {
			return true
		}
		if len(nameParts) < i {
			// not enough name parts
			return false
		}
		if p != "+" && p != nameParts[i] {
			return false
		}
	}
	return true
}

func (l *TopicList) filter(filter TopicFilter) TopicList {
	var res TopicList
	for _, topic := range topics {
		if filter.matches(topic.name) {
			res = append(res, topic)
		}
	}
	return res
}
