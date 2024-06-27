package network

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net"
	"testing"
	"time"
)

const (
	testIPAddress    = "192.0.2.1"
	testMessageSize  = 10
	testTimeout      = time.Second
	testEchoID       = 1
	testEchoSeq      = 1
	testEchoData     = "Hello, world!"
	testIPHeaderLen  = 20
	testIPHeaderTTL  = 64
	testIPHeaderPort = 1 // ICMP
)

var (
	testDestIP   = net.IPv4(8, 8, 8, 8)
	testSourceIP = net.IPv4(192, 0, 2, 1)
)

// ICMPPacketConn is an interface that describes methods that we using from icmp.PacketConn.
type ICMPPacketConn interface {
	ReadFrom(b []byte) (int, net.Addr, error)
	SetReadDeadline(t time.Time) error
	Close() error
}

// Ensure MockICMPConn implements ICMPPacketConn
var _ ICMPPacketConn = (*MockICMPConn)(nil)

// MockICMPConn is a mock for the ICMPPacketConn interface
type MockICMPConn struct {
	mock.Mock
}

func (m *MockICMPConn) ReadFrom(b []byte) (int, net.Addr, error) {
	args := m.Called(b)
	return args.Int(0), args.Get(1).(net.Addr), args.Error(2)
}

func (m *MockICMPConn) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockICMPConn) SetReadDeadline(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

// ICMPConn represents an ICMP connection for receiving traceroute responses.
type ICMPConn struct {
	conn ICMPPacketConn
}

func TestNewICMPConn(t *testing.T) {
	conn, err := NewICMPConn()
	defer conn.Close()

	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, conn.conn)
}

func TestICMPConnReadWithTimeout(t *testing.T) {
	mockConn := new(MockICMPConn)
	icmpConn := &ICMPConn{conn: mockConn}

	// Test successful read
	mockConn.On("SetReadDeadline", mock.Anything).Return(nil)
	mockConn.On("ReadFrom", mock.Anything).Return(testMessageSize, &net.IPAddr{IP: net.ParseIP(testIPAddress)}, nil)

	ip, data, err := icmpConn.ReadWithTimeout(testTimeout)
	assert.NoError(t, err)
	assert.Equal(t, testIPAddress, ip.String())
	assert.Len(t, data, testMessageSize)

	// Test timeout
	mockConn.On("ReadFrom", mock.Anything).Return(0, nil, &net.OpError{Err: &timeoutError{}})

	_, _, err = icmpConn.ReadWithTimeout(testTimeout)
	assert.EqualError(t, err, "read timeout")

	mockConn.AssertExpectations(t)
}
