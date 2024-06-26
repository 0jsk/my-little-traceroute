package network

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

// SyscallConn represents a low-level network connection.
type SyscallConn interface {
	Control(func(fd uintptr)) error
}

// UDPConn wraps a *net.UDPConn with additional functionality for traceroute operations.
type UDPConn struct {
	*net.UDPConn
	syscallConn SyscallConn
}

// NewUDPConn creates a new UDP connection bound UDP to the specified local address.
//
// The local address should be in the formay "ip:port". Use ":0" for any available port.
// Returns a pointer to UDPConn and an error if the connection can't be established.
func NewUDPConn(localAddr string) (*UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp4", localAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve local address: %w", err)
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %w", err)
	}

	rawConn, err := conn.SyscallConn()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get syscall conn: %w", err)
	}

	return &UDPConn{
		UDPConn:     conn,
		syscallConn: rawConn,
	}, nil
}

// SetTTL sets the Time to Live (TTL) for outgoing packets.
//
// TTL value determines how many network hops a packet can traverse before being discarded.
// Returns an error if setting TTL fails.
func (c *UDPConn) SetTTL(ttl int) error {
	return c.syscallConn.Control(func(fd uintptr) {
		err := syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)

		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to set TTL: %v\n", err)
		}
	})
}

// SendEmptyPacket sends an empty UDP packet to the specified address.
//
// This function is used to send probe packets in the traceroute process.
// It returns an error if sending the packet fails.
func (c *UDPConn) SendEmptyPacket(addr *net.UDPAddr) error {
	_, err := c.WriteToUDP([]byte{}, addr)

	if err != nil {
		return fmt.Errorf("failed to send UDP packet: %w", err)
	}

	return nil
}

// Close closes the UDP connection and releases associated resources.
//
// It should be called when the connection is no longer needed to prevent resource leaks.
// It returns an error if closing the connection fails.
func (c *UDPConn) Close() error {
	return c.UDPConn.Close()
}
