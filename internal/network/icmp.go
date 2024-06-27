package network

import (
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"net"
	"time"
)

const (
	// AllInterfaces is the IP address representing all available network interfaces.
	AllInterfaces = "0.0.0.0"

	// MaxPacketSize is the maximum size of an ICMP packet (Ethernet MTU).
	MaxPacketSize = 1500
)

// ICMPConn represents an ICMP connection for receiving traceroute responses.
type ICMPConn struct {
	conn *icmp.PacketConn
}

// NewICMPConn creates a new ICMP connection for listening to ICMP messages.
//
// Returns a pointer to ICMPConn and an error if the connection can't be established.
func NewICMPConn() (*ICMPConn, error) {
	conn, err := icmp.ListenPacket("ipv4:icmp", AllInterfaces)
	if err != nil {
		return nil, fmt.Errorf("failed to create ICMP connection: %w", err)
	}

	return &ICMPConn{
		conn: conn,
	}, nil
}

// ReadWithTimeout reads an ICMP message from the connection with a specified timeout.
//
// Returns the source IP address of the message, the raw message data, and an error if any.
// If no messages is received within the timeout period, it return a timeout error.
func (c *ICMPConn) ReadWithTimeout(timeout time.Duration) (net.IP, []byte, error) {
	err := c.conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	buffer := make([]byte, MaxPacketSize)
	n, peer, err := c.conn.ReadFrom(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, nil, fmt.Errorf("read timeout")
		}

		return nil, nil, fmt.Errorf("failed to read ICMP message: %w", err)
	}

	return peer.(*net.IPAddr).IP, buffer[:n], nil
}

// ParseICMPMessage parses the raw ICMP message and extracts info.
//
// Returns the type of ICMP message, the embedded IP header, and an error if parsing fails.
func ParseICMPMessage(msg []byte) (ipv4.ICMPType, *ipv4.Header, error) {
	m, err := icmp.ParseMessage(1, msg)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse ICMP message: %w", err)
	}

	icmpType, ok := m.Type.(ipv4.ICMPType)
	if !ok {
		return 0, nil, fmt.Errorf("unexpected ICMP message type: %v", m.Type)
	}

	switch icmpType {
	case ipv4.ICMPTypeTimeExceeded:
		return parseTimeExceededMessage(m)
	case ipv4.ICMPTypeEchoReply:
		return icmpType, nil, nil
	default:
		return icmpType, nil, fmt.Errorf("unexpected ICMP message type: %v", icmpType)
	}
}

// parseTimeExceededMessage handles specific case of a Time Exceeded ICMP message.
func parseTimeExceededMessage(m *icmp.Message) (ipv4.ICMPType, *ipv4.Header, error) {
	body, ok := m.Body.(*icmp.TimeExceeded)
	if !ok {
		return ipv4.ICMPTypeTimeExceeded, nil, fmt.Errorf("invalid TimeExceeded message body")
	}

	header, err := ipv4.ParseHeader(body.Data)
	if err != nil {
		return ipv4.ICMPTypeTimeExceeded, nil, fmt.Errorf("failed to parse IP header: %w", err)
	}

	return ipv4.ICMPTypeTimeExceeded, header, nil
}

// Close closes the ICMP connection.
//
// Return an error if closing connection fails.
func (c *ICMPConn) Close() error {
	return c.conn.Close()
}
