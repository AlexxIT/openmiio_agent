package serial

import (
	"fmt"
	"golang.org/x/sys/unix"
)

type Serial struct {
	fd int
}

func Open(port string, baudRate uint32, hardware bool) (*Serial, error) {
	fd, err := unix.Open(port, unix.O_RDWR|unix.O_NOCTTY|unix.O_NDELAY, 0)
	if err != nil {
		return nil, err
	}

	term, err := unix.IoctlGetTermios(fd, unix.TCGETS2)
	if err != nil {
		return nil, err
	}

	// clear previous values
	for i := 0; i < len(term.Cc); i++ {
		term.Cc[i] = 0
	}

	// Block reads until at least one char is available (no timeout)
	term.Cc[unix.VTIME] = 30    // 3 sec
	term.Cc[unix.VMIN] = 0      // 0 byte
	term.Cc[unix.VSTART] = 0x11 // XON
	term.Cc[unix.VSTOP] = 0x13  // XOFF

	// clear previous values
	term.Cflag = 0

	// 0x800018B2
	term.Cflag |= unix.CREAD  // 0x80
	term.Cflag |= unix.CLOCAL // 0x800
	term.Cflag |= unix.CS8    // 0x30 (databits)

	switch baudRate {
	case 38400:
		term.Cflag |= unix.B38400
	case 115200:
		term.Cflag |= unix.B115200 // 0x1002 (baudrate)
	default:
		return nil, fmt.Errorf("unsupported baud rate: %d", baudRate)
	}

	term.Ispeed = baudRate
	term.Ospeed = baudRate

	if hardware {
		term.Cflag |= unix.CRTSCTS // 0x80000000
	}

	if err = unix.IoctlSetTermios(fd, unix.TCSETS2, term); err != nil {
		return nil, err
	}

	if err = unix.SetNonblock(fd, false); err != nil {
		return nil, err
	}

	if err = unix.IoctlSetInt(fd, unix.TIOCEXCL, 0); err != nil {
		return nil, err
	}

	return &Serial{fd: fd}, nil
}

func (s *Serial) Read(p []byte) (n int, err error) {
	return unix.Read(s.fd, p)
}

func (s *Serial) Write(p []byte) (n int, err error) {
	return unix.Write(s.fd, p)
}

func (s *Serial) Close() (err error) {
	_ = unix.IoctlSetInt(s.fd, unix.TIOCNXCL, 0)
	return unix.Close(s.fd)
}
