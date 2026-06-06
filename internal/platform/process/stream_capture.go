package process

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sync"
)

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
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if mirror != nil {
			_, _ = fmt.Fprintln(mirror, line)
		}
		if c.onLine != nil {
			c.onLine(stream, line)
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, os.ErrClosed) && c.onError != nil {
		c.onError(stream, err)
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
