package network

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net"
	"testing"
	"time"
)

type MockSyscallConn struct {
	mock.Mock
}

func (m *MockSyscallConn) Control(f func(fd uintptr)) error {
	args := m.Called(f)
	return args.Error(0)
}

func TestNewUDPConn(t *testing.T) {
	conn, err := NewUDPConn(":0")
	defer conn.Close()

	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, conn.UDPConn)
	assert.NotNil(t, conn.syscallConn)
}

func TestUDPConnSetTTL(t *testing.T) {
	mockSyscallConn := new(MockSyscallConn)
	mockSyscallConn.On("Control", mock.AnythingOfType("func(uintptr)")).Return(nil)

	conn := &UDPConn{
		syscallConn: mockSyscallConn,
	}

	err := conn.SetTTL(64)
	assert.NoError(t, err)
	mockSyscallConn.AssertCalled(t, "Control", mock.AnythingOfType("func(uintptr)"))
}

func TestUDPConnSendEmptyPacket(t *testing.T) {
	serverAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	assert.NoError(t, err)

	serverConn, err := net.ListenUDP("udp4", serverAddr)
	assert.NoError(t, err)
	defer serverConn.Close()

	received := make(chan struct{})
	go func() {
		buf := make([]byte, 1024)
		// TODO: try without 1 *
		serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, _, err := serverConn.ReadFromUDP(buf)
		assert.NoError(t, err)
		assert.Equal(t, 0, n)
		close(received)
	}()

	clientConn, err := NewUDPConn(":0")
	assert.NoError(t, err)
	defer clientConn.Close()

	err = clientConn.SendEmptyPacket(serverConn.LocalAddr().(*net.UDPAddr))
	assert.NoError(t, err)

	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for packet")
	}
}

func TestUDPConnIntegration(t *testing.T) {
	conn, err := NewUDPConn(":0")
	assert.NoError(t, err)
	defer conn.Close()

	err = conn.SetTTL(64)
	assert.NoError(t, err)

	nonExistentAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:61337")
	assert.NoError(t, err)

	err = conn.SendEmptyPacket(nonExistentAddr)
	assert.NoError(t, err)
}
