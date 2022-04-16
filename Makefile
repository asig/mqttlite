#
# Copyright (c) 2022 Andreas Signer <asigner@gmail.com>
#
# This file is part of mqttlite.
#
# mqttlite is free software: you can redistribute it and/or
# modify it under the terms of the GNU General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# mqttlite is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with mqttlite.  If not, see <http://www.gnu.org/licenses/>.
#

.PHONY: clean

RESOURCES = \
    $(wildcard web/**/*)

all: internal/memfs/content.go
	go build

clean:
	@rm -f \
      mqttlite \
      bin2go \
      internal/memfs/content.go

run:    all
	./mqttlite

internal/memfs/content.go: bin2go $(RESOURCES)
	echo $(RESOURCES)

bin2go:	tools/bin2go/bin2go.go
	go build tools/bin2go/bin2go.go

