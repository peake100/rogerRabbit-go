package amqp_test

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

// init sets up our logger for testing.
func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro

	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}