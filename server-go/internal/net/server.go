// Package netserver accepts TCP sessions, performs the v2 handshake/auth and
// forwards game intents to the world.
package netserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	stdnet "net"
	"strconv"
	"sync"
	"time"

	"github.com/kelaia/vxaos/server/internal/auth"
	"github.com/kelaia/vxaos/server/internal/chargen"
	"github.com/kelaia/vxaos/server/internal/game"
	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/store"
)

var (
	ErrClosed        = errors.New("netserver: session closed")
	ErrSendQueueFull = errors.New("netserver: send queue full")
)

// Options configures the TCP server.
type Options struct {
	Listen         string
	World          *game.World
	Store          *store.Store
	GameVersion    int
	MaxActors      int
	MaxFrame       uint32
	AuthIterations int
	AuthTimeout    time.Duration
	Inactivity     time.Duration
	MaxConnections int
	Chargen        chargen.Options
	Logger         *slog.Logger
}

// Server is the TCP listener and session registry.
type Server struct {
	opts Options
	log  *slog.Logger

	mu       sync.Mutex
	ln       stdnet.Listener
	sessions map[*Session]struct{}
	wg       sync.WaitGroup
	sem      chan struct{}
	closed   bool
}

// New creates a server.
func New(opts Options) (*Server, error) {
	if opts.World == nil || opts.Store == nil {
		return nil, fmt.Errorf("netserver: world and store are required")
	}
	if opts.GameVersion <= 0 || opts.GameVersion > 65535 {
		return nil, fmt.Errorf("netserver: invalid game version %d", opts.GameVersion)
	}
	if opts.MaxActors <= 0 {
		opts.MaxActors = 3
	}
	if opts.MaxFrame == 0 {
		opts.MaxFrame = protocol.MaxFrameSize
	}
	if opts.AuthIterations <= 0 {
		opts.AuthIterations = auth.DefaultIterations
	}
	if opts.AuthTimeout <= 0 {
		opts.AuthTimeout = 30 * time.Second
	}
	if opts.Inactivity <= 0 {
		opts.Inactivity = 5 * time.Minute
	}
	if opts.MaxConnections <= 0 {
		opts.MaxConnections = 500
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Server{
		opts:     opts,
		log:      opts.Logger,
		sessions: make(map[*Session]struct{}),
		sem:      make(chan struct{}, opts.MaxConnections),
	}, nil
}

// ListenAndServe opens opts.Listen and serves until ctx is done.
func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := stdnet.Listen("tcp", s.opts.Listen)
	if err != nil {
		return err
	}
	return s.Serve(ctx, ln)
}

// Serve serves an existing listener. It is used by tests with port 0.
func (s *Server) Serve(ctx context.Context, ln stdnet.Listener) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		ln.Close()
		return ErrClosed
	}
	s.ln = ln
	s.mu.Unlock()
	s.log.Info("listening", "addr", ln.Addr().String())

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			s.closeListener()
			s.closeSessions()
		case <-done:
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				close(done)
				s.wg.Wait()
				return nil
			default:
			}
			if ne, ok := err.(stdnet.Error); ok && ne.Temporary() {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			close(done)
			s.wg.Wait()
			return err
		}
		select {
		case s.sem <- struct{}{}:
		default:
			conn.Close()
			continue
		}
		s.wg.Add(1)
		go func(c stdnet.Conn) {
			defer s.wg.Done()
			defer func() { <-s.sem }()
			s.handle(ctx, c)
		}(conn)
	}
}

// Addr returns the bound address, if serving.
func (s *Server) Addr() stdnet.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln == nil {
		return nil
	}
	return s.ln.Addr()
}

func (s *Server) closeListener() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln != nil {
		s.ln.Close()
	}
}

func (s *Server) closeSessions() {
	s.mu.Lock()
	list := make([]*Session, 0, len(s.sessions))
	for sess := range s.sessions {
		list = append(list, sess)
	}
	s.mu.Unlock()
	for _, sess := range list {
		sess.Close()
	}
}

func (s *Server) addSession(sess *Session) {
	s.mu.Lock()
	s.sessions[sess] = struct{}{}
	s.mu.Unlock()
}

func (s *Server) removeSession(sess *Session) {
	s.mu.Lock()
	delete(s.sessions, sess)
	s.mu.Unlock()
}

func (s *Server) handle(ctx context.Context, conn stdnet.Conn) {
	if tcp, ok := conn.(*stdnet.TCPConn); ok {
		tcp.SetNoDelay(true)
		tcp.SetKeepAlive(true)
	}
	sess := &Session{
		srv:  s,
		conn: conn,
		send: make(chan outbound, 256),
		done: make(chan struct{}),
	}
	s.addSession(sess)
	go sess.writeLoop()
	defer sess.Close()
	sess.run(ctx)
}

type pendingAuth struct {
	Username   string
	Account    *store.Account
	Salt       []byte
	Nonce      []byte
	Iterations int
	Fake       bool
}

type outbound struct {
	op      protocol.ServerOp
	payload []byte
}

// Session is one TCP connection.
type Session struct {
	srv  *Server
	conn stdnet.Conn
	send chan outbound
	done chan struct{}

	closeOnce sync.Once
	writeMu   sync.Mutex

	// Owned by the reader goroutine.
	account       *store.Account
	characters    []*store.Character
	pending       *pendingAuth
	loginNonce    []byte
	authenticated bool
	inGame        bool
	player        *game.Player
}

// Send queues a packet for the writer goroutine.
func (s *Session) Send(op protocol.ServerOp, msg protocol.ServerMessage) error {
	select {
	case <-s.done:
		return ErrClosed
	default:
	}
	payload, err := msg.Encode()
	if err != nil {
		return err
	}
	select {
	case s.send <- outbound{op: op, payload: payload}:
		return nil
	case <-s.done:
		return ErrClosed
	default:
		s.Close()
		return ErrSendQueueFull
	}
}

// Close closes the connection and unregisters the session.
func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
		s.conn.Close()
		s.srv.removeSession(s)
	})
	return nil
}

func (s *Session) writeLoop() {
	for {
		select {
		case <-s.done:
			return
		case out := <-s.send:
			if err := s.writeFrame(out.op, out.payload); err != nil {
				s.Close()
				return
			}
		}
	}
}

func (s *Session) writeFrame(op protocol.ServerOp, payload []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return protocol.WriteFrame(s.conn, uint16(op), payload)
}

func (s *Session) sendDirect(op protocol.ServerOp, msg protocol.ServerMessage) {
	payload, err := msg.Encode()
	if err != nil {
		return
	}
	_ = s.writeFrame(op, payload)
}

func (s *Session) run(ctx context.Context) {
	defer func() {
		if s.inGame && s.player != nil {
			lctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := s.srv.opts.World.Leave(lctx, s.player, true); err != nil && err != game.ErrNotInGame {
				s.srv.log.Warn("leave on disconnect", "err", err)
			}
			cancel()
		}
	}()
	s.conn.SetReadDeadline(time.Now().Add(s.srv.opts.AuthTimeout))
	hello, err := protocol.ReadHello(s.conn)
	if err != nil {
		s.srv.log.Debug("handshake failed", "remote", s.conn.RemoteAddr(), "err", err)
		return
	}
	if hello.ProtocolVersion != protocol.ProtocolVersion || int(hello.GameVersion) != s.srv.opts.GameVersion {
		s.handshakeAck(protocol.StatusVersion, "protocol or game version mismatch")
		return
	}
	if s.handshakeAck(protocol.StatusOK, "") != nil {
		return
	}
	s.srv.log.Debug("handshake ok", "remote", s.conn.RemoteAddr(), "build", hello.Build)

	for {
		deadline := time.Now().Add(s.srv.opts.Inactivity)
		if !s.authenticated {
			deadline = time.Now().Add(s.srv.opts.AuthTimeout)
		}
		s.conn.SetReadDeadline(deadline)
		frame, err := protocol.ReadFrame(s.conn)
		if err != nil {
			return
		}
		if !s.handleFrame(ctx, frame) {
			return
		}
	}
}

func (s *Session) handshakeAck(status uint8, message string) error {
	ack := protocol.HelloAck{
		ProtocolVersion: protocol.ProtocolVersion,
		GameVersion:     uint16(s.srv.opts.GameVersion),
		ServerTime:      time.Now().Unix(),
		MaxFrameSize:    s.srv.opts.MaxFrame,
		Status:          status,
		Message:         message,
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return protocol.WriteHelloAck(s.conn, ack)
}

func (s *Session) handleFrame(ctx context.Context, frame protocol.Frame) bool {
	op := protocol.ClientOp(frame.Op)
	msg, err := protocol.DecodeClient(op, frame.Payload)
	if err != nil {
		s.sendDirect(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeBadRequest, Message: err.Error()})
		return false
	}
	if _, ok := msg.(protocol.Ping); ok {
		s.Send(protocol.ServerPong, protocol.Pong{Timestamp: msg.(protocol.Ping).Timestamp})
		return true
	}
	if !s.authenticated {
		switch m := msg.(type) {
		case protocol.AuthBegin:
			return s.handleAuthBegin(ctx, m)
		case protocol.AuthProof:
			return s.handleAuthProof(ctx, m)
		case protocol.RegisterBegin, protocol.RegisterProof:
			s.sendDirect(protocol.ServerAuthError, protocol.AuthError{Code: protocol.AuthInternal, Message: "registration is disabled in this build"})
			return false
		default:
			s.sendDirect(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeNotAuthenticated, Message: "authenticate first"})
			return false
		}
	}
	if !s.inGame {
		switch m := msg.(type) {
		case protocol.SelectCharacter:
			return s.handleSelectCharacter(ctx, m)
		case protocol.CreateCharacter:
			return s.handleCreateCharacter(ctx, m)
		case protocol.DeleteCharacter:
			return s.handleDeleteCharacter(ctx, m)
		case protocol.LogoutRequest:
			s.Send(protocol.ServerLogout, protocol.Logout{})
			return false
		default:
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeNotInGame, Message: "select a character first"})
			return true
		}
	}
	switch m := msg.(type) {
	case protocol.Move:
		if err := s.srv.opts.World.Move(ctx, s.player, m.Direction); err != nil {
			if errors.Is(err, game.ErrRateLimited) {
				return true
			}
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeBadRequest, Message: err.Error()})
		}
	case protocol.Chat:
		if err := s.srv.opts.World.Chat(ctx, s.player, m.Channel, m.Message, m.Target); err != nil {
			if errors.Is(err, game.ErrRateLimited) {
				return true
			}
			code := protocol.CodeBadRequest
			if errors.Is(err, game.ErrUnsupported) {
				code = protocol.CodeInvalidState
			}
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: code, Message: err.Error()})
		}
	case protocol.EquipItem:
		if err := s.srv.opts.World.Equip(ctx, s.player, m.Slot, m.ItemID); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.SetHotbar:
		if err := s.srv.opts.World.SetHotbar(ctx, s.player, m.Slot, m.Type, m.ItemID); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.SpendPoint:
		if err := s.srv.opts.World.SpendPoint(ctx, s.player, m.ParamID); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.Attack:
		if err := s.srv.opts.World.Attack(ctx, s.player); err != nil {
			if !errors.Is(err, game.ErrRateLimited) {
				s.Send(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeBadRequest, Message: err.Error()})
			}
		}
	case protocol.Target:
		if err := s.srv.opts.World.SetTarget(ctx, s.player, m.Type, m.ID); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeInvalidTarget, Message: err.Error()})
		}
	case protocol.UseItem:
		if err := s.srv.opts.World.UseItem(ctx, s.player, m.ItemID); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.UseSkill:
		if err := s.srv.opts.World.UseSkill(ctx, s.player, m.SkillID); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.UseHotbar:
		if err := s.srv.opts.World.UseHotbar(ctx, s.player, m.Slot); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.EventNext:
		if err := s.srv.opts.World.EventNext(ctx, s.player); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.Choice:
		if err := s.srv.opts.World.EventChoice(ctx, s.player, m.Index); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.ChoiceTeleport:
		if err := s.srv.opts.World.Teleport(ctx, s.player, m.Index); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.ShopBuy:
		if err := s.srv.opts.World.ShopBuy(ctx, s.player, m.Index, m.Amount); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.ShopSell:
		if err := s.srv.opts.World.ShopSell(ctx, s.player, m.ItemID, m.Kind, m.Amount); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.BankItemRequest:
		if err := s.srv.opts.World.BankItem(ctx, s.player, m.ItemID, m.Kind, m.Amount); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.BankGoldRequest:
		if err := s.srv.opts.World.BankGold(ctx, s.player, m.Amount); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.DropItemRequest:
		if err := s.srv.opts.World.DropItem(ctx, s.player, m.ItemID, m.Kind, m.Amount); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.PickupDrop:
		if err := s.srv.opts.World.PickupDrop(ctx, s.player, m.DropID); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.CloseWindow:
		if err := s.srv.opts.World.CloseWindow(ctx, s.player); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.GuildCreate:
		if err := s.srv.opts.World.CreateGuild(ctx, s.player, m.Name, m.Flag); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.GuildOpenRequest:
		if err := s.srv.opts.World.OpenGuild(ctx, s.player); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.GuildSetLeader:
		if err := s.srv.opts.World.SetGuildLeader(ctx, s.player, m.Name); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.GuildSetNotice:
		if err := s.srv.opts.World.SetGuildNotice(ctx, s.player, m.Notice); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.GuildKick:
		if err := s.srv.opts.World.KickGuildMember(ctx, s.player, m.Name); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.GuildInvite:
		if err := s.srv.opts.World.InviteGuildMember(ctx, s.player, m.Name); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.GuildLeave:
		if err := s.srv.opts.World.LeaveGuild(ctx, s.player); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.RequestAccept:
		if err := s.srv.opts.World.AcceptRequest(ctx, s.player); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.RequestDecline:
		if err := s.srv.opts.World.DeclineRequest(ctx, s.player); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.RequestInvite:
		if err := s.srv.opts.World.PlayerRequest(ctx, s.player, m.Type, m.PlayerID); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.PartyLeaveRequest:
		if err := s.srv.opts.World.LeaveParty(ctx, s.player); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.OpenFriends:
		if err := s.srv.opts.World.OpenFriends(ctx, s.player); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.RemoveFriend:
		if err := s.srv.opts.World.RemoveFriend(ctx, s.player, m.Index); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.TradeItemOffer:
		if err := s.srv.opts.World.TradeItem(ctx, s.player, m.ItemID, m.Kind, m.Amount); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.TradeGoldOffer:
		if err := s.srv.opts.World.TradeGold(ctx, s.player, m.Amount); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: equipmentErrorCode(err), Message: err.Error()})
		}
	case protocol.AdminCommand:
		if err := s.srv.opts.World.AdminCommand(ctx, s.player, m); err != nil {
			s.Send(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeBadRequest, Message: err.Error()})
		}
	case protocol.LogoutRequest:
		s.leave(ctx, true)
		s.Send(protocol.ServerLogout, protocol.Logout{})
	case protocol.SelectCharacter:
		s.Send(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeInvalidState, Message: "already in game"})
	default:
		s.Send(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeBadRequest, Message: "unsupported message"})
	}
	return true
}

func (s *Session) handleAuthBegin(ctx context.Context, m protocol.AuthBegin) bool {
	if s.authenticated {
		s.sendDirect(protocol.ServerAuthError, protocol.AuthError{Code: protocol.AuthInternal, Message: "already authenticated"})
		return false
	}
	iter := auth.DefaultIterations
	// A per-session challenge uses the client's stored KDF parameters; the
	// server-level value is the default for new/fake accounts.
	if s.srv.opts.AuthIterations > 0 {
		iter = s.srv.opts.AuthIterations
	}
	acc, err := s.srv.opts.Store.GetAccountByUsername(ctx, m.Username)
	fake := false
	var salt []byte
	if err == sql.ErrNoRows {
		fake = true
		salt, err = auth.RandomBytes(auth.SaltLen)
		if err != nil {
			s.sendDirect(protocol.ServerAuthError, protocol.AuthError{Code: protocol.AuthInternal, Message: "random failure"})
			return false
		}
	} else if err != nil {
		s.srv.log.Error("load account", "err", err)
		s.sendDirect(protocol.ServerAuthError, protocol.AuthError{Code: protocol.AuthInternal, Message: "internal error"})
		return false
	} else {
		if acc.PasswordAlgo != "pbkdf2-sha256" {
			s.sendDirect(protocol.ServerAuthError, protocol.AuthError{Code: protocol.AuthInternal, Message: "unsupported password algorithm"})
			return false
		}
		salt = acc.PasswordSalt
		iter = acc.PasswordIterations
	}
	nonce, err := auth.RandomBytes(auth.NonceLen)
	if err != nil {
		s.sendDirect(protocol.ServerAuthError, protocol.AuthError{Code: protocol.AuthInternal, Message: "random failure"})
		return false
	}
	s.pending = &pendingAuth{
		Username:   m.Username,
		Account:    acc,
		Salt:       salt,
		Nonce:      nonce,
		Iterations: iter,
		Fake:       fake,
	}
	s.srv.log.Debug("auth begin", "username", m.Username, "fake", fake, "iterations", iter)
	return s.Send(protocol.ServerAuthChallenge, protocol.AuthChallenge{
		Salt:       salt,
		KDF:        auth.KDFPBKDF2SHA256,
		Iterations: uint32(iter),
		Nonce:      nonce,
		ServerTime: time.Now().Unix(),
	}) == nil
}

func (s *Session) handleAuthProof(ctx context.Context, m protocol.AuthProof) bool {
	if s.pending == nil {
		s.sendDirect(protocol.ServerAuthError, protocol.AuthError{Code: protocol.AuthInvalidCredentials, Message: "no challenge"})
		return false
	}
	pending := s.pending
	s.pending = nil
	if pending.Fake || pending.Account == nil || len(m.Proof) != auth.ProofLen {
		s.sendDirect(protocol.ServerAuthError, protocol.AuthError{Code: protocol.AuthInvalidCredentials, Message: "invalid credentials"})
		return false
	}
	if !auth.VerifyProof(pending.Account.PasswordVerifier, pending.Username, pending.Nonce, m.ClientNonce, m.Proof) {
		s.srv.log.Debug("auth proof rejected", "account", pending.Username, "proof_len", len(m.Proof), "client_nonce_len", len(m.ClientNonce))
		s.sendDirect(protocol.ServerAuthError, protocol.AuthError{Code: protocol.AuthInvalidCredentials, Message: "invalid credentials"})
		return false
	}
	s.account = pending.Account
	s.loginNonce = append([]byte(nil), pending.Nonce...)
	now := time.Now().Unix()
	accountValue := strconv.FormatInt(s.account.ID, 10)
	if ban, err := s.srv.opts.Store.GetActiveBan(ctx, "account", accountValue, now); err == nil && ban != nil {
		s.sendDirect(protocol.ServerAuthError, protocol.AuthError{Code: protocol.AuthAccountBanned, Message: "conta banida"})
		return false
	}
	if ip := remoteIP(s.conn); ip != "" {
		if ban, err := s.srv.opts.Store.GetActiveBan(ctx, "ip", ip, now); err == nil && ban != nil {
			s.sendDirect(protocol.ServerAuthError, protocol.AuthError{Code: protocol.AuthAccountBanned, Message: "IP banido"})
			return false
		}
	}
	if err := s.srv.opts.Store.UpdateAccountLogin(ctx, s.account.ID, now); err != nil {
		s.srv.log.Warn("update login time", "err", err)
	}
	summaries, err := s.loadActorSummaries(ctx)
	if err != nil {
		s.srv.log.Error("load characters", "err", err)
		s.sendDirect(protocol.ServerAuthError, protocol.AuthError{Code: protocol.AuthInternal, Message: "internal error"})
		return false
	}
	if s.Send(protocol.ServerAuthOK, protocol.AuthOK{
		Group:    uint8(s.account.GroupID),
		VIPUntil: s.account.VIPUntil,
		Actors:   summaries,
	}) != nil {
		return false
	}
	s.authenticated = true
	s.srv.log.Info("auth ok", "account", s.account.Username, "characters", len(summaries))
	return true
}

func (s *Session) loadActorSummaries(ctx context.Context) ([]protocol.ActorSummary, error) {
	chars, err := s.srv.opts.Store.ListCharacters(ctx, s.account.ID)
	if err != nil {
		return nil, err
	}
	s.characters = chars
	out := make([]protocol.ActorSummary, 0, len(chars))
	for _, c := range chars {
		st, err := s.srv.opts.Store.GetCharacterState(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		var equips [protocol.MaxEquips]uint16
		for i := 0; i < len(equips) && i < len(st.Equips); i++ {
			if st.Equips[i] >= 0 && st.Equips[i] <= 65535 {
				equips[i] = uint16(st.Equips[i])
			}
		}
		out = append(out, protocol.ActorSummary{
			Slot:           uint8(c.Slot),
			Name:           c.Name,
			CharacterName:  c.CharacterName,
			CharacterIndex: uint8(c.CharacterIndex),
			FaceName:       c.FaceName,
			FaceIndex:      uint8(c.FaceIndex),
			Sex:            uint8(c.Sex),
			Equips:         equips,
		})
	}
	return out, nil
}

func (s *Session) handleCreateCharacter(ctx context.Context, m protocol.CreateCharacter) bool {
	if s.account == nil || s.optsChargenMissing() {
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeInternal, Message: "character creation unavailable"})
		return true
	}
	for _, c := range s.characters {
		if c.Slot == int(m.Slot) {
			s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeCharacterSlot, Message: "slot already in use"})
			return true
		}
	}
	exists, err := s.srv.opts.Store.CharacterExists(ctx, m.Name)
	if err != nil {
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeInternal, Message: "internal error"})
		return true
	}
	if exists {
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeCharacterName, Message: "name already in use"})
		return true
	}
	maxClass := s.srv.opts.Chargen.MaxDefaultClasses
	if s.account.VIPUntil > time.Now().Unix() && s.srv.opts.Chargen.MaxVIPClasses > 0 {
		maxClass = s.srv.opts.Chargen.MaxVIPClasses
	}
	if maxClass > 0 && int(m.ClassID) > maxClass {
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeCharacterInvalid, Message: "class not available"})
		return true
	}
	var allocated [8]int
	for i, p := range m.Params {
		allocated[i] = int(p)
	}
	c, state, err := chargen.Create(s.srv.opts.Chargen, s.account.ID, int(m.Slot), m.Name, int(m.ClassID), int(m.Sex), int(m.CharacterIndex), allocated)
	if err != nil {
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeCharacterInvalid, Message: err.Error()})
		return true
	}
	if err := s.srv.opts.Store.CreateCharacter(ctx, c); err != nil {
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeCharacterName, Message: "name or slot already in use"})
		return true
	}
	if err := s.srv.opts.Store.SaveCharacter(ctx, c, state); err != nil {
		s.srv.log.Error("save new character", "err", err)
		_ = s.srv.opts.Store.DeleteCharacter(ctx, c.ID)
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeInternal, Message: "internal error"})
		return true
	}
	summaries, err := s.loadActorSummaries(ctx)
	if err != nil {
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeInternal, Message: "internal error"})
		return true
	}
	for _, a := range summaries {
		if a.Slot == m.Slot {
			s.srv.log.Info("character created", "account", s.account.Username, "slot", m.Slot, "name", c.Name)
			s.Send(protocol.ServerCharacterCreated, protocol.CharacterCreated{Actor: a})
			return true
		}
	}
	s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeInternal, Message: "character not found after creation"})
	return true
}

func (s *Session) optsChargenMissing() bool {
	return s.srv.opts.Chargen.Data == nil
}

func (s *Session) handleDeleteCharacter(ctx context.Context, m protocol.DeleteCharacter) bool {
	if s.account == nil {
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeNotAuthenticated, Message: "authenticate first"})
		return true
	}
	var chosen *store.Character
	for _, c := range s.characters {
		if c.Slot == int(m.Slot) {
			chosen = c
			break
		}
	}
	if chosen == nil {
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeCharacterSlot, Message: "unknown character slot"})
		return true
	}
	if !auth.VerifyDeleteProof(s.account.PasswordVerifier, s.account.Username, s.loginNonce, m.ClientNonce, chosen.Name, m.Proof) {
		s.srv.log.Debug("delete proof rejected", "account", s.account.Username, "slot", m.Slot)
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeCharacterInvalid, Message: "invalid password"})
		return true
	}
	online, err := s.srv.opts.World.CharacterOnline(ctx, chosen.ID)
	if err != nil {
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeInternal, Message: "internal error"})
		return true
	}
	if online {
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeInvalidState, Message: "character is online"})
		return true
	}
	if err := s.srv.opts.World.DeleteCharacter(ctx, chosen); err != nil {
		s.srv.log.Error("delete character", "character", chosen.ID, "err", err)
		s.Send(protocol.ServerCharacterError, protocol.CharacterError{Code: protocol.CodeInternal, Message: "internal error"})
		return true
	}
	s.srv.log.Info("character deleted", "account", s.account.Username, "slot", m.Slot, "name", chosen.Name)
	if _, err := s.loadActorSummaries(ctx); err != nil {
		s.srv.log.Warn("reload characters after delete", "err", err)
	}
	s.Send(protocol.ServerCharacterDeleted, protocol.CharacterDeleted{Slot: m.Slot})
	return true
}

func equipmentErrorCode(err error) uint16 {
	switch {
	case errors.Is(err, game.ErrFullInventory):
		return protocol.CodeFullInventory
	case errors.Is(err, game.ErrNotInGame):
		return protocol.CodeNotInGame
	case errors.Is(err, game.ErrInvalidTarget):
		return protocol.CodeInvalidTarget
	default:
		return protocol.CodeBadRequest
	}
}

func (s *Session) handleSelectCharacter(ctx context.Context, m protocol.SelectCharacter) bool {
	var chosen *store.Character
	for _, c := range s.characters {
		if c.Slot == int(m.Slot) {
			chosen = c
			break
		}
	}
	if chosen == nil {
		s.Send(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeBadRequest, Message: "unknown character slot"})
		return true
	}
	c, st, err := s.srv.opts.Store.GetCharacter(ctx, chosen.ID)
	if err != nil {
		s.Send(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeInternal, Message: "character unavailable"})
		return true
	}
	online, err := s.srv.opts.World.CharacterOnline(ctx, c.ID)
	if err != nil {
		s.Send(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeInternal, Message: "world unavailable"})
		return true
	}
	if online {
		s.Send(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeInvalidState, Message: "character already online"})
		return true
	}
	p := &game.Player{Account: s.account, Character: c, State: st, Client: s, IP: remoteIP(s.conn)}
	if err := s.srv.opts.World.Join(ctx, p); err != nil {
		code := protocol.CodeInternal
		if errors.Is(err, game.ErrAlreadyOnline) {
			code = protocol.CodeInvalidState
		}
		s.Send(protocol.ServerError, protocol.ErrorMessage{Code: code, Message: err.Error()})
		return true
	}
	s.player = p
	s.inGame = true
	return true
}

func (s *Session) leave(ctx context.Context, save bool) {
	if s.player == nil || !s.inGame {
		return
	}
	if err := s.srv.opts.World.Leave(ctx, s.player, save); err != nil && !errors.Is(err, game.ErrNotInGame) {
		s.srv.log.Warn("leave", "err", err)
	}
	s.player = nil
	s.inGame = false
}

func remoteIP(conn stdnet.Conn) string {
	if conn == nil {
		return ""
	}
	host, _, err := stdnet.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return conn.RemoteAddr().String()
	}
	return host
}
