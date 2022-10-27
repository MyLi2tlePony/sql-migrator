package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger interface {
	Error(string)
	Info(string)
}

type logger struct {
}

func New() Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	return &logger{}
}

func (l *logger) Error(msg string) {
	log.Error().Msg(msg)
}

func (l *logger) Info(msg string) {
	log.Info().Msg(msg)
}
