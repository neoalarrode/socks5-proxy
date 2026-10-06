package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

const (
	socks5Version = 0x05

	authNone         = 0x00
	authUserPass     = 0x02
	authNoAcceptable = 0xFF

	cmdConnect = 0x01

	atypIPv4   = 0x01
	atypDomain = 0x03
	atypIPv6   = 0x04

	repSuccess          = 0x00
	repGeneralFailure   = 0x01
	repNotAllowed       = 0x02
	repNetUnreachable   = 0x03
	repHostUnreachable  = 0x04
	repConnRefused      = 0x05
	repCmdNotSupported  = 0x07
	repAtypNotSupported = 0x08
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
}

type Server struct {
	cfg      Config
	listener net.Listener
	mu       sync.Mutex
}

func New(cfg Config) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			default:
				log.Printf("accept error: %v", err)
				continue
			}
		}
		go s.handleConn(conn)
	}
}

func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()

	if err := s.negotiate(conn); err != nil {
		log.Printf("[%s] negotiation failed: %v", clientAddr, err)
		return
	}

	if err := s.handleRequest(conn, clientAddr); err != nil {
		log.Printf("[%s] request failed: %v", clientAddr, err)
	}
}

func (s *Server) negotiate(conn net.Conn) error {
	buf := make([]byte, 258)

	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return fmt.Errorf("read greeting: %w", err)
	}

	if buf[0] != socks5Version {
		return fmt.Errorf("unsupported version: %d", buf[0])
	}

	nMethods := int(buf[1])
	if nMethods == 0 {
		return fmt.Errorf("no auth methods offered")
	}

	if _, err := io.ReadFull(conn, buf[:nMethods]); err != nil {
		return fmt.Errorf("read methods: %w", err)
	}

	methods := buf[:nMethods]

	if s.cfg.Username != "" {
		if !containsByte(methods, authUserPass) {
			conn.Write([]byte{socks5Version, authNoAcceptable})
			return fmt.Errorf("client does not support username/password auth")
		}
		conn.Write([]byte{socks5Version, authUserPass})
		return s.authenticateUserPass(conn)
	}

	if containsByte(methods, authNone) {
		conn.Write([]byte{socks5Version, authNone})
		return nil
	}

	conn.Write([]byte{socks5Version, authNoAcceptable})
	return fmt.Errorf("no acceptable auth method")
}

func (s *Server) authenticateUserPass(conn net.Conn) error {
	buf := make([]byte, 513)

	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return fmt.Errorf("read auth header: %w", err)
	}

	if buf[0] != 0x01 {
		return fmt.Errorf("unsupported auth sub-negotiation version: %d", buf[0])
	}

	uLen := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:uLen]); err != nil {
		return fmt.Errorf("read username: %w", err)
	}
	username := string(buf[:uLen])

	if _, err := io.ReadFull(conn, buf[:1]); err != nil {
		return fmt.Errorf("read password length: %w", err)
	}
	pLen := int(buf[0])
	if _, err := io.ReadFull(conn, buf[:pLen]); err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	password := string(buf[:pLen])

	if username == s.cfg.Username && password == s.cfg.Password {
		conn.Write([]byte{0x01, 0x00})
		return nil
	}

	conn.Write([]byte{0x01, 0x01})
	return fmt.Errorf("authentication failed for user %q", username)
}

func (s *Server) handleRequest(conn net.Conn, clientAddr string) error {
	buf := make([]byte, 4)

	if _, err := io.ReadFull(conn, buf); err != nil {
		return fmt.Errorf("read request header: %w", err)
	}

	if buf[0] != socks5Version {
		return fmt.Errorf("unsupported version in request: %d", buf[0])
	}

	cmd := buf[1]
	atyp := buf[3]

	if cmd != cmdConnect {
		s.sendReply(conn, repCmdNotSupported, nil)
		return fmt.Errorf("unsupported command: %d", cmd)
	}

	var targetAddr string

	switch atyp {
	case atypIPv4:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return fmt.Errorf("read IPv4: %w", err)
		}
		targetAddr = net.IP(addr).String()

	case atypDomain:
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return fmt.Errorf("read domain length: %w", err)
		}
		domain := make([]byte, buf[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return fmt.Errorf("read domain: %w", err)
		}
		targetAddr = string(domain)

	case atypIPv6:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return fmt.Errorf("read IPv6: %w", err)
		}
		targetAddr = net.IP(addr).String()

	default:
		s.sendReply(conn, repAtypNotSupported, nil)
		return fmt.Errorf("unsupported address type: %d", atyp)
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return fmt.Errorf("read port: %w", err)
	}
	port := int(portBuf[0])<<8 | int(portBuf[1])

	target := fmt.Sprintf("%s:%d", targetAddr, port)
	log.Printf("[%s] CONNECT %s", clientAddr, target)

	remote, err := net.Dial("tcp", target)
	if err != nil {
		reply := repGeneralFailure
		if opErr, ok := err.(*net.OpError); ok {
			if opErr.Timeout() {
				reply = repHostUnreachable
			}
		}
		s.sendReply(conn, byte(reply), nil)
		return fmt.Errorf("dial %s: %w", target, err)
	}
	defer remote.Close()

	localAddr := remote.LocalAddr().(*net.TCPAddr)
	s.sendReply(conn, repSuccess, localAddr)

	relay(conn, remote)
	return nil
}

func (s *Server) sendReply(conn net.Conn, rep byte, bindAddr *net.TCPAddr) {
	reply := []byte{socks5Version, rep, 0x00}

	if bindAddr != nil {
		ip := bindAddr.IP.To4()
		if ip != nil {
			reply = append(reply, atypIPv4)
			reply = append(reply, ip...)
		} else {
			reply = append(reply, atypIPv6)
			reply = append(reply, bindAddr.IP.To16()...)
		}
		reply = append(reply, byte(bindAddr.Port>>8), byte(bindAddr.Port))
	} else {
		reply = append(reply, atypIPv4, 0, 0, 0, 0, 0, 0)
	}

	conn.Write(reply)
}

func relay(client, remote net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(remote, client)
		remote.(*net.TCPConn).CloseWrite()
	}()

	go func() {
		defer wg.Done()
		io.Copy(client, remote)
		client.(*net.TCPConn).CloseWrite()
	}()

	wg.Wait()
}

func containsByte(slice []byte, b byte) bool {
	for _, v := range slice {
		if v == b {
			return true
		}
	}
	return false
}
