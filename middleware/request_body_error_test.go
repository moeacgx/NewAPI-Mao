package middleware

import (
	"io"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
)

func TestInvalidRequestStatusDistinguishesBodyLimit(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "oversized body", err: common.ErrRequestBodyTooLarge, want: http.StatusRequestEntityTooLarge},
		{name: "interrupted read", err: io.ErrUnexpectedEOF, want: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, invalidRequestStatus(test.err))
		})
	}
}
