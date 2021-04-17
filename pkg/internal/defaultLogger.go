package internal

import (
	_ "code.cloudfoundry.org/go-diodes" // import for lockless writing
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"os"
	"time"
)

// CreateDefaultLogger creates the default logger used by the amqp and roger packages
func CreateDefaultLogger(level zerolog.Level) zerolog.Logger {
	wr := diode.NewWriter(os.Stdout, 1000, 10*time.Millisecond, func(missed int) {
		_, _ = fmt.Printf("Logger Dropped %d messages", missed)
	})
	return zerolog.New(zerolog.ConsoleWriter{Out: wr}).Level(level).With().Timestamp().Logger()
}
