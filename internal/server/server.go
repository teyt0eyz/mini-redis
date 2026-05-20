package server

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
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
	aofPath := os.Getenv("AOF_PATH")
	if aofPath == "" {
		aofPath = "data/appendonly.aof"
	}
	if err := os.MkdirAll(filepath.Dir(aofPath), 0755); err != nil {
		return err
	}
	if err := persistence.Replay(aofPath, handle); err != nil {
		fmt.Println("AOF replay error:", err)
	}
	if err := persistence.Open(aofPath); err != nil {
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
