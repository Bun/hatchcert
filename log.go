package hatchcert

import (
	"bytes"
	"fmt"
	"os"

	"github.com/go-acme/lego/v3/log"
)

type LegoOutput struct {
	Previous log.StdLogger
	Output   bytes.Buffer
}

func (l *LegoOutput) Emit() {
	os.Stderr.Write(l.Output.Bytes())
}

func (l *LegoOutput) write(pfx, msg string) {
	l.Output.Write([]byte(pfx + msg + "\n"))
}

func (l *LegoOutput) Fatal(args ...interface{}) {
	l.write("fatal: ", fmt.Sprint(args...))
}

func (l *LegoOutput) Fatalln(args ...interface{}) {
	l.write("fatal: ", fmt.Sprintln(args...))
}

func (l *LegoOutput) Fatalf(format string, args ...interface{}) {
	l.write("fatal: ", fmt.Sprintf(format, args...))
}

func (l *LegoOutput) Print(args ...interface{}) {
	l.write("", fmt.Sprint(args...))
}

func (l *LegoOutput) Println(args ...interface{}) {
	l.write("", fmt.Sprintln(args...))
}

func (l *LegoOutput) Printf(format string, args ...interface{}) {
	l.write("", fmt.Sprintf(format, args...))
}

func (l *LegoOutput) Warnf(format string, args ...interface{}) {
	l.write("warning: ", fmt.Sprintf(format, args...))
}

func (l *LegoOutput) Infof(format string, args ...interface{}) {
	l.write("info: ", fmt.Sprintf(format, args...))
}

func (l *LegoOutput) Restore() {
	if l.Previous != nil {
		log.Logger = l.Previous
		l.Previous = nil
	}
}

func InterceptOutput() *LegoOutput {
	o := &LegoOutput{Previous: log.Logger}
	log.Logger = o
	return o
}
