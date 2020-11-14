package islelib

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCastSpell(test *testing.T) {
	assert.Equal(test, "Avada Kedavra", CastSpell())
}
