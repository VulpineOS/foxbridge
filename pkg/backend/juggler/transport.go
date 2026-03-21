package juggler

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

// Transport is the interface for sending/receiving Juggler messages.
type Transport interface {
	Send(msg *Message) error
	Receive() (*Message, error)
	Close() error
}

// PipeTransport communicates with Firefox via file descriptors.
// Messages are JSON terminated by a null byte (\x00), matching
// nsRemoteDebuggingPipe.cpp's framing protocol.
type PipeTransport struct {
	reader  *bufio.Reader
	writer  io.WriteCloser
	readFD  *os.File
	writeFD *os.File
	writeMu sync.Mutex
}

// NewPipeTransport creates a transport using the given read/write file descriptors.
func NewPipeTransport(readFD, writeFD *os.File) *PipeTransport {
	return &PipeTransport{
		reader:  bufio.NewReaderSize(readFD, 1<<16),
		writer:  writeFD,
		readFD:  readFD,
		writeFD: writeFD,
	}
}

// Send marshals the message to JSON and writes it followed by a null byte.
func (t *PipeTransport) Send(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if _, err := t.writer.Write(append(data, 0)); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	return nil
}

// Receive reads until a null byte and unmarshals the JSON message.
func (t *PipeTransport) Receive() (*Message, error) {
	data, err := t.reader.ReadBytes(0)
	if err != nil {
		return nil, fmt.Errorf("read message: %w", err)
	}

	// Strip the null byte
	if len(data) > 0 {
		data = data[:len(data)-1]
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}
	return &msg, nil
}

// Close closes both file descriptors.
func (t *PipeTransport) Close() error {
	readErr := t.readFD.Close()
	writeErr := t.writeFD.Close()
	return errors.Join(readErr, writeErr)
}
