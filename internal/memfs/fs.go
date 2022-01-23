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
package memfs

import (
	"errors"
	"net/http"
	"os"
	"time"
)

//
// FileSystem (implements http.FileSystem)
//

type FileSystem map[string][]byte

func (fs FileSystem) Open(name string) (http.File, error) {
	content, found := fs[name]
	if !found {
		return nil, errors.New("No such file")
	}
	f := &File{
		at:   0,
		name: name,
		data: content,
		fs:   fs,
	}
	return f, nil
}

func (fs FileSystem) Add(name string, content []byte) {
	fs[name] = content
}

//
// File (implements http.File)
//

type File struct {
	at   int64
	name string
	data []byte
	fs   FileSystem
}

func (f *File) Close() error {
	return nil
}

func (f *File) Stat() (os.FileInfo, error) {
	return &FileInfo{
		name: f.name,
		size:  int64(len(f.data)),
	}, nil
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	res := make([]os.FileInfo, len(f.fs))
	i := 0
	for name, content := range f.fs {
		res[i] = &FileInfo{
			name: name,
			size: int64(len(content)),
		}
		i++
	}
	return res, nil
}

func (f *File) Read(b []byte) (int, error) {
	i := 0
	for f.at < int64(len(f.data)) && i < len(b) {
		b[i] = f.data[f.at]
		i++
		f.at++
	}
	return i, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		f.at = offset
	case 1:
		f.at += offset
	case 2:
		f.at = int64(len(f.data)) + offset
	}
	return f.at, nil
}

//
// FileInfo (implements os.FileInfo)
//

type FileInfo struct {
	name string
	size int64
}

func (f *FileInfo) Name() string {
	return f.name
}

func (f *FileInfo) Size() int64 {
	return f.size
}

func (f *FileInfo) Mode() os.FileMode {
	return os.ModeTemporary
}

func (s *FileInfo) ModTime() time.Time {
	return time.Time{}
}

func (s *FileInfo) IsDir() bool {
	return false
}

func (s *FileInfo) Sys() interface{} {
	return nil
}
