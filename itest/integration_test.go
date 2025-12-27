package itest

import (
	"net/http/httptest"
	"testing"

	"github.com/dynoinc/dynovault/inmemory"
	"github.com/stretchr/testify/suite"

	"github.com/dynoinc/dynovault/handler"
)

func TestInMemory(t *testing.T) {
	ts := httptest.NewServer(handler.New(inmemory.New()))
	t.Cleanup(ts.Close)

	suite.Run(t, New(t, ts.URL))
}
