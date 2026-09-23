/*
©AngelaMos | 2026
server.go

Telnet honeypot service accepting terminal client connections

Emulates a Linux Telnet login service with basic command shell.
Logs every credential attempt, command, and session interaction to the
event bus. Implements Telnet protocol negotiation (IAC commands) for
compatibility with real Telnet clients.

This is a NEW feature not present in the source project.
*/

package telnetd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/CarterPerez-dev/miragenet/internal/config"
	"github.com/CarterPerez-dev/miragenet/internal/event"
	"github.com/CarterPerez-dev/miragenet/internal/ratelimit"
	"github.com/CarterPerez-dev/miragenet/internal/session"
	"github.com/CarterPerez-dev/miragenet/pkg/types"
)

// Telnet protocol constants (RFC 854)
const (
	IAC  = 255 // Interpret As Command
	DONT = 254 // Don't
	DO   = 253 // Do
	WONT = 252 // Won't
	WILL = 251 // Will
	SE   = 240 // Subnegotiation End
	SB   = 250 // Subnegotiation Begin
)

// Telnet options (RFC 855, 1073, etc.)
const (
	ECHO              = 1  // Echo
	SUPPRESS_GO_AHEAD = 3  // Suppress Go Ahead
	TERMINAL_TYPE     = 24 // Terminal Type
	WINDOW_SIZE       = 31 // Window Size
	TERMINAL_SPEED    = 32 // Terminal Speed
	LINEMODE          = 34 // Line Mode
)

type TelnetService struct {
	cfg     *config.Config
	bus     *event.Bus
	logger  zerolog.Logger
	tracker *session.Tracker
	limiter *ratelimit.IPLimiter
}

func New(
	cfg *config.Config,
	bus *event.Bus,
	logger *zerolog.Logger,
	tracker *session.Tracker,
	limiter *ratelimit.IPLimiter,
) *TelnetService {
	return &TelnetService{
		cfg:     cfg,
		bus:     bus,
		logger:  logger.With().Str("service", "telnet").Logger(),
		tracker: tracker,
		limiter: limiter,
	}
}

func (s *TelnetService) Name() string { return "telnet" }

func (s *TelnetService) Start(ctx context.Context) error {
	addr := s.cfg.Addr(s.cfg.Telnet.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("telnet listen %s: %w", addr, err)
	}

	s.logger.Info().
		Str("addr", addr).
		Msg("telnet honeypot listening")

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for ctx.Err() == nil {
		conn, err := listener.Accept()
		if err != nil {
			s.logger.Debug().Err(err).Msg("accept failed")
			continue
		}

		go s.handleConnection(ctx, conn)
	}

	return nil
}

func (s *TelnetService) handleConnection(ctx context.Context, conn net.Conn) {
	defer func() { _ = conn.Close() }()

	srcIP, srcPort := types.RemoteAddr(conn)
	if !s.limiter.Allow(srcIP) {
		return
	}

	sess := s.tracker.Start(
		s.cfg.Sensor.ID, types.ServiceTelnet,
		srcIP, srcPort, s.cfg.Telnet.Port,
	)
	defer s.tracker.End(sess.ID)

	s.publishConnect(sess, srcIP, srcPort)
	defer s.publishDisconnect(sess, srcIP, srcPort)

	tc := newTelnetConn(conn, s.cfg.Telnet.Hostname)

	// Send initial Telnet negotiations
	tc.negotiate()

	// Send banner
	tc.writeLine(s.cfg.Telnet.Banner)
	tc.writeLine("")

	// Login phase
	username, password, err := tc.readLogin()
	if err != nil {
		return
	}

	s.publishAuth(sess.ID, srcIP, username, password)
	s.tracker.SetLogin(sess.ID, true, username, "")

	// Send login success message
	tc.writeLine(fmt.Sprintf("Last login: %s from %s", time.Now().Format("Mon Jan 2 15:04:05 2006"), srcIP))
	tc.writeLine("")

	// Shell phase
	for {
		if ctx.Err() != nil {
			return
		}

		tc.writePrompt()

		cmd, err := tc.readCommand()
		if err != nil {
			return
		}

		if cmd == "" {
			continue
		}

		s.publishCommand(sess.ID, srcIP, cmd)

		// Execute command
		output := tc.executeCommand(cmd)
		if output != "" {
			tc.writeLine(output)
		}

		// Handle exit commands
		if cmd == "exit" || cmd == "logout" || cmd == "quit" {
			tc.writeLine("logout")
			return
		}
	}
}

type telnetConn struct {
	conn     net.Conn
	reader   *bufio.Reader
	hostname string
	username string
}

func newTelnetConn(conn net.Conn, hostname string) *telnetConn {
	return &telnetConn{
		conn:     conn,
		reader:   bufio.NewReader(conn),
		hostname: hostname,
	}
}

func (tc *telnetConn) negotiate() {
	// Tell client we WILL echo (server-side echo)
	tc.sendCommand(IAC, WILL, ECHO)
	// Tell client we WILL suppress go-ahead
	tc.sendCommand(IAC, WILL, SUPPRESS_GO_AHEAD)
	// Ask client to send terminal type
	tc.sendCommand(IAC, DO, TERMINAL_TYPE)
	// Ask client for window size
	tc.sendCommand(IAC, DO, WINDOW_SIZE)
}

func (tc *telnetConn) sendCommand(bytes ...byte) {
	_, _ = tc.conn.Write(bytes)
}

func (tc *telnetConn) writeLine(s string) {
	_, _ = tc.conn.Write([]byte(s + "\r\n"))
}

func (tc *telnetConn) writePrompt() {
	if tc.username == "" {
		_, _ = tc.conn.Write([]byte(config.TelnetPrompt))
	} else {
		prompt := fmt.Sprintf("%s@%s:~$ ", tc.username, tc.hostname)
		_, _ = tc.conn.Write([]byte(prompt))
	}
}

func (tc *telnetConn) readLogin() (string, string, error) {
	// Read username
	tc.writePrompt()
	username, err := tc.readLine()
	if err != nil {
		return "", "", err
	}
	tc.username = username

	// Read password (send prompt without echo)
	_, _ = tc.conn.Write([]byte("Password: "))
	password, err := tc.readLine()
	if err != nil {
		return "", "", err
	}

	return username, password, nil
}

func (tc *telnetConn) readCommand() (string, error) {
	line, err := tc.readLine()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (tc *telnetConn) readLine() (string, error) {
	var line []byte

	for {
		b, err := tc.reader.ReadByte()
		if err != nil {
			return "", err
		}

		// Handle Telnet IAC commands
		if b == IAC {
			if err := tc.handleIAC(); err != nil {
				return "", err
			}
			continue
		}

		// Handle line terminators
		if b == '\r' {
			// Check for \r\n sequence
			next, err := tc.reader.ReadByte()
			if err == nil && next != '\n' {
				_ = tc.reader.UnreadByte()
			}
			break
		}
		if b == '\n' {
			break
		}

		// Handle backspace/delete
		if b == 127 || b == 8 {
			if len(line) > 0 {
				line = line[:len(line)-1]
				// Echo backspace sequence
				tc.sendCommand(8, ' ', 8)
			}
			continue
		}

		// Echo character back to client
		tc.sendCommand(b)

		line = append(line, b)
	}

	return string(line), nil
}

func (tc *telnetConn) handleIAC() error {
	cmd, err := tc.reader.ReadByte()
	if err != nil {
		return err
	}

	switch cmd {
	case IAC:
		// Escaped IAC (255 255 means literal 255)
		return nil
	case DO, DONT, WILL, WONT:
		// Read option byte
		opt, err := tc.reader.ReadByte()
		if err != nil {
			return err
		}

		// Respond to options
		switch cmd {
		case DO:
			// Client wants us to do something, decline most options
			if opt == ECHO || opt == SUPPRESS_GO_AHEAD {
				tc.sendCommand(IAC, WILL, opt)
			} else {
				tc.sendCommand(IAC, WONT, opt)
			}
		case DONT:
			// Client doesn't want us to do something, acknowledge
			tc.sendCommand(IAC, WONT, opt)
		case WILL:
			// Client will do something, acknowledge
			tc.sendCommand(IAC, DO, opt)
		case WONT:
			// Client won't do something, acknowledge
			tc.sendCommand(IAC, DONT, opt)
		}
	case SB:
		// Subnegotiation - read until SE
		for {
			b, err := tc.reader.ReadByte()
			if err != nil {
				return err
			}
			if b == IAC {
				next, err := tc.reader.ReadByte()
				if err != nil {
					return err
				}
				if next == SE {
					break
				}
			}
		}
	}

	return nil
}

func (tc *telnetConn) executeCommand(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}

	command := parts[0]

	switch command {
	case "ls", "dir":
		return "bin  boot  dev  etc  home  lib  lib64  media  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var"
	case "pwd":
		return "/home/" + tc.username
	case "whoami":
		return tc.username
	case "id":
		return fmt.Sprintf("uid=1000(%s) gid=1000(%s) groups=1000(%s)", tc.username, tc.username, tc.username)
	case "uname":
		if len(parts) > 1 && parts[1] == "-a" {
			return "Linux " + tc.hostname + " 5.15.0-105-generic #115-Ubuntu SMP Mon Apr 15 09:52:04 UTC 2024 x86_64 x86_64 x86_64 GNU/Linux"
		}
		return "Linux"
	case "hostname":
		return tc.hostname
	case "cat":
		if len(parts) > 1 {
			return fmt.Sprintf("cat: %s: No such file or directory", parts[1])
		}
		return "cat: missing operand"
	case "cd":
		return "" // Silent success
	case "echo":
		if len(parts) > 1 {
			return strings.Join(parts[1:], " ")
		}
		return ""
	case "help":
		return "Available commands: ls, pwd, whoami, id, uname, hostname, cat, cd, echo, help, exit"
	case "exit", "logout", "quit":
		return "" // Handled by caller
	default:
		return fmt.Sprintf("%s: command not found", command)
	}
}

func (s *TelnetService) publishConnect(
	sess *types.Session,
	srcIP string,
	srcPort int,
) {
	s.bus.Publish(config.TopicConnect, &types.Event{
		ID:            uuid.Must(uuid.NewV7()).String(),
		SessionID:     sess.ID,
		SensorID:      s.cfg.Sensor.ID,
		Timestamp:     time.Now().UTC(),
		ServiceType:   types.ServiceTelnet,
		EventType:     types.EventConnect,
		SourceIP:      srcIP,
		SourcePort:    srcPort,
		DestPort:      s.cfg.Telnet.Port,
		Protocol:      types.ProtocolTCP,
		SchemaVersion: config.SchemaVersion,
	})
}

func (s *TelnetService) publishDisconnect(
	sess *types.Session,
	srcIP string,
	srcPort int,
) {
	s.bus.Publish(config.TopicDisconnect, &types.Event{
		ID:            uuid.Must(uuid.NewV7()).String(),
		SessionID:     sess.ID,
		SensorID:      s.cfg.Sensor.ID,
		Timestamp:     time.Now().UTC(),
		ServiceType:   types.ServiceTelnet,
		EventType:     types.EventDisconnect,
		SourceIP:      srcIP,
		SourcePort:    srcPort,
		DestPort:      s.cfg.Telnet.Port,
		Protocol:      types.ProtocolTCP,
		SchemaVersion: config.SchemaVersion,
	})
}

func (s *TelnetService) publishAuth(
	sessionID string,
	srcIP string,
	username string,
	password string,
) {
	serviceData, _ := json.Marshal(map[string]string{
		"username":    username,
		"password":    password,
		"auth_method": "password",
	})

	s.bus.Publish(config.TopicAuth, &types.Event{
		ID:            uuid.Must(uuid.NewV7()).String(),
		SessionID:     sessionID,
		SensorID:      s.cfg.Sensor.ID,
		Timestamp:     time.Now().UTC(),
		ServiceType:   types.ServiceTelnet,
		EventType:     types.EventLoginSuccess,
		SourceIP:      srcIP,
		Protocol:      types.ProtocolTCP,
		SchemaVersion: config.SchemaVersion,
		ServiceData:   serviceData,
	})
}

func (s *TelnetService) publishCommand(
	sessionID string,
	srcIP string,
	cmd string,
) {
	serviceData, _ := json.Marshal(map[string]string{
		"command": cmd,
	})

	s.bus.Publish(config.TopicCommand, &types.Event{
		ID:            uuid.Must(uuid.NewV7()).String(),
		SessionID:     sessionID,
		SensorID:      s.cfg.Sensor.ID,
		Timestamp:     time.Now().UTC(),
		ServiceType:   types.ServiceTelnet,
		EventType:     types.EventCommand,
		SourceIP:      srcIP,
		Protocol:      types.ProtocolTCP,
		SchemaVersion: config.SchemaVersion,
		ServiceData:   serviceData,
	})
}
