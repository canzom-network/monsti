// This file is part of Monsti, a web content management system.
// Copyright 2012-2013 Christian Neumann
//
// Monsti is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option) any
// later version.
//
// Monsti is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR
// A PARTICULAR PURPOSE.  See the GNU Affero General Public License for more
// details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Monsti.  If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"fmt"
)

// SessionPool holds sessions to be used to access services.
type SessionPool struct {
	// Size is the maximum number of sessions to cache.
	Size int
	// InfoPath is the path to the Info service to be used.
	InfoPath string
	data     chan *DataClient
	info     chan *InfoClient
}

// NewSessionPool returns a new session pool.
func NewSessionPool(size int, infoPath string) *SessionPool {
	pool := &SessionPool{Size: size, InfoPath: infoPath}
	pool.data = make(chan *DataClient, size)
	pool.info = make(chan *InfoClient, size)
	return pool
}

// New returns a session from the pool.
func (s *SessionPool) New() (*Session, error) {
	session := &Session{pool: s}
	select {
	case session.info = <-s.info:
	default:
		info, err := NewInfoConnection(s.InfoPath)
		if err != nil {
			return nil, fmt.Errorf("Could not create Info client: %v", err)
		}
		session.info = info
	}
	return session, nil
}

// Free puts a session back to the pool.
func (s *SessionPool) Free(session *Session) {
	select {
	case s.data <- session.data:
	default:
	}
	select {
	case s.info <- session.info:
	default:
	}
}

// Session holds connections to the services.
type Session struct {
	info *InfoClient
	data *DataClient
	pool *SessionPool
}

// Info returns an InfoClient.
func (s *Session) Info() *InfoClient {
	return s.info
}

// Data returns a DataClient.
func (s *Session) Data() *DataClient {
	if s.data != nil {
		return s.data
	}
	select {
	case s.data = <-s.pool.data:
	default:
		data, err := s.info.FindDataService()
		s.data = data
		if err != nil {
			s.data = new(DataClient)
			s.data.Error = err
		}
	}
	return s.data
}
