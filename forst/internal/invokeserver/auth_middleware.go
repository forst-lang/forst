package invokeserver

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authEnabled() {
			next.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path
		if path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		peerKey := s.peerKey(r)
		now := time.Now()
		if s.backoff != nil && !s.backoff.Allow(peerKey, now) {
			s.sendAuthError(w, r)
			return
		}
		if s.cfg.network() == transportUnix {
			if conn := connFromContext(r.Context()); conn != nil {
				if !verifyPeerUID(s.peerReader, conn, currentUID()) {
					s.recordAuthFailure(peerKey, now)
					s.sendAuthError(w, r)
					return
				}
			}
		} else if !isLoopbackRemoteAddr(r.RemoteAddr) {
			s.recordAuthFailure(peerKey, now)
			s.sendAuthError(w, r)
			return
		}
		if path == "/invoke/challenge" {
			next.ServeHTTP(w, r)
			return
		}
		if !s.verifyProof(r, peerKey, now) {
			s.sendAuthError(w, r)
			return
		}
		if s.backoff != nil {
			s.backoff.Reset(peerKey)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) verifyProof(r *http.Request, peerKey string, now time.Time) bool {
	if s.auth == nil || s.nonces == nil {
		return false
	}
	nonce := strings.TrimSpace(r.Header.Get(HeaderInvokeNonce))
	if nonce == "" {
		s.recordAuthFailure(peerKey, now)
		return false
	}
	if !s.nonces.consume(nonce, now) {
		s.recordAuthFailure(peerKey, now)
		return false
	}
	genHeader := strings.TrimSpace(r.Header.Get(HeaderInvokeGeneration))
	if genHeader == "" {
		s.recordAuthFailure(peerKey, now)
		return false
	}
	clientGen, err := strconv.ParseUint(genHeader, 10, 64)
	if err != nil {
		s.recordAuthFailure(peerKey, now)
		return false
	}
	serverGen := s.auth.currentGeneration()
	if clientGen != serverGen {
		s.recordAuthFailure(peerKey, now)
		return false
	}
	_, token := s.auth.snapshot()
	proof := strings.TrimSpace(r.Header.Get(HeaderInvokeProof))
	if proof == "" || strings.TrimSpace(r.Header.Get(HeaderInvokeToken)) != "" {
		s.recordAuthFailure(peerKey, now)
		return false
	}
	if !verifyInvokeProof(token, serverGen, nonce, proof) {
		s.recordAuthFailure(peerKey, now)
		return false
	}
	return true
}

func (s *Server) recordAuthFailure(peerKey string, now time.Time) {
	if s.backoff != nil {
		s.backoff.RecordFailure(peerKey, now)
	}
}

func (s *Server) sendAuthError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	s.sendJSON(w, r, Response{Success: false, Error: "unauthorized"})
}

func (s *Server) peerKey(r *http.Request) string {
	if s.cfg.network() == transportUnix {
		if conn := connFromContext(r.Context()); conn != nil {
			if creds, ok := s.peerReader.PeerCredentials(conn); ok && creds.PID > 0 {
				return "uds:" + strconv.Itoa(creds.PID)
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return "tcp:" + host
}

func isLoopbackRemoteAddr(remote string) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return false
	}
	return isLoopbackHost(host)
}

type connContextKey struct{}

func connFromContext(ctx context.Context) net.Conn {
	conn, _ := ctx.Value(connContextKey{}).(net.Conn)
	return conn
}
