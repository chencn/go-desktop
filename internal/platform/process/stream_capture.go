package process

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const maxCapturedLineBytes = 1024 * 1024

type StreamCapture struct {
	oldStdout    *os.File
	oldStderr    *os.File
	stdoutReader *os.File
	stdoutWriter *os.File
	stderrReader *os.File
	stderrWriter *os.File
	onLine       func(stream, line string)
	onError      func(stream string, err error)
	wait         sync.WaitGroup
	closeOnce    sync.Once
}

func NewStreamCapture(onLine func(stream, line string), onError func(stream string, err error)) (*StreamCapture, error) {
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		return nil, err
	}
	capture := &StreamCapture{
		oldStdout:    os.Stdout,
		oldStderr:    os.Stderr,
		stdoutReader: stdoutReader,
		stdoutWriter: stdoutWriter,
		stderrReader: stderrReader,
		stderrWriter: stderrWriter,
		onLine:       onLine,
		onError:      onError,
	}
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	capture.wait.Add(2)
	go capture.copyLines("stdout", stdoutReader, capture.oldStdout)
	go capture.copyLines("stderr", stderrReader, capture.oldStderr)
	return capture, nil
}

func (c *StreamCapture) copyLines(stream string, reader *os.File, mirror *os.File) {
	defer c.wait.Done()
	buffered := bufio.NewReaderSize(reader, 64*1024)
	line := make([]byte, 0, 64*1024)
	truncated := false
	for {
		chunk, err := buffered.ReadSlice('\n')
		if len(chunk) > 0 {
			if len(line) < maxCapturedLineBytes {
				remaining := maxCapturedLineBytes - len(line)
				if len(chunk) > remaining {
					line = append(line, chunk[:remaining]...)
					truncated = true
				} else {
					line = append(line, chunk...)
				}
			} else {
				truncated = true
			}
			if chunk[len(chunk)-1] == '\n' {
				c.emitLine(stream, mirror, line, truncated)
				line = line[:0]
				truncated = false
			}
		}
		if err == nil || errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(err, io.EOF) {
			if len(line) > 0 {
				c.emitLine(stream, mirror, line, truncated)
			}
			return
		}
		if !errors.Is(err, os.ErrClosed) && c.onError != nil {
			c.onError(stream, err)
		}
		return
	}
}

func (c *StreamCapture) emitLine(stream string, mirror *os.File, raw []byte, truncated bool) {
	line := strings.TrimRight(string(raw), "\r\n")
	if truncated {
		line += " ...[truncated]"
	}
	if mirror != nil {
		_, _ = fmt.Fprintln(mirror, line)
	}
	if c.onLine != nil {
		c.onLine(stream, line)
	}
}

func (c *StreamCapture) Close() {
	c.closeOnce.Do(func() {
		if os.Stdout == c.stdoutWriter {
			os.Stdout = c.oldStdout
		}
		if os.Stderr == c.stderrWriter {
			os.Stderr = c.oldStderr
		}
		_ = c.stdoutWriter.Close()
		_ = c.stderrWriter.Close()
		c.wait.Wait()
		_ = c.stdoutReader.Close()
		_ = c.stderrReader.Close()
	})
}
