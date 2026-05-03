package logger

import (
	"log"
)

func Init(prefix string) {
	if prefix != "" {
		log.SetPrefix(prefix + " ")
	}
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
}

func Infof(format string, args ...interface{}) {
	log.Printf("INFO: "+format, args...)
}

func Errorf(format string, args ...interface{}) {
	log.Printf("ERROR: "+format, args...)
}

func Fatalf(format string, args ...interface{}) {
	log.Fatalf("FATAL: "+format, args...)
}
