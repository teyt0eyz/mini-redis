package server

import (
	"fmt"
	"net"
	"sync"

	"mini-redis/internal/persistence"
)

type Server struct {
	addr     string
	listener net.Listener
	wg       sync.WaitGroup
	quit     chan struct{}
}

func New(addr string) *Server {
	return &Server{
		addr: addr,
		quit: make(chan struct{}),
	}
}

func (s *Server) Start() error {
	if err := persistence.Replay("data/appendonly.aof", handle); err != nil {
		fmt.Println("AOF replay error:", err)
	}
	if err := persistence.Open("data/appendonly.aof"); err != nil {
		return err
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = ln
	fmt.Println("[Server] Listening on", s.addr)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return nil
			default:
				fmt.Println("Accept error:", err)
				continue
			}
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			handleConnection(conn)
		}()
	}
}

func (s *Server) Stop() {
	close(s.quit)
	s.listener.Close()
	s.wg.Wait()
	persistence.Close()
}
