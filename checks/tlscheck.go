package checks

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/shaardie/mondane/database"
)

type TLSCheck struct {
	check database.TLSCheck
}

func NewTLSCheck(check database.TLSCheck) TLSCheck {
	return TLSCheck{check: check}
}

func (TLSCheck) Type() string {
	return "tls"
}

func (tc TLSCheck) ID() uint {
	return tc.check.ID
}

func (tc TLSCheck) Check(t time.Time) (interface{}, error) {
	r := &database.TLSResult{
		Time:       t,
		TLSCheckID: tc.check.ID,
	}
	conn, err := tls.Dial("tcp", fmt.Sprintf("%v:%v", tc.check.Host, tc.check.Port), nil)
	if err != nil {
		r.DialError = err.Error()
		return r, nil
	}
	r.Success = true
	defer conn.Close()
	state := conn.ConnectionState()
	r.TLSVersion = state.Version
	r.Expiry = getCertExpiry(&state)
	r.CipherSuite = state.CipherSuite
	return r, nil
}

func getCertExpiry(state *tls.ConnectionState) time.Time {
	earliest := time.Time{}
	for _, cert := range state.PeerCertificates {
		if (earliest.IsZero() || cert.NotAfter.Before(earliest)) && !cert.NotAfter.IsZero() {
			earliest = cert.NotAfter
		}
	}
	return earliest
}
