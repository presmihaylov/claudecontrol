package core

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

func NewID(prefix string) string {
	if prefix == "" || strings.TrimSpace(prefix) == "" {
		panic("Prefix cannot be empty")
	}

	entropy := rand.New(rand.NewSource(time.Now().UnixNano()))
	id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)

	return fmt.Sprintf("%s_%s", strings.ToLower(strings.TrimSpace(prefix)), id.String())
}