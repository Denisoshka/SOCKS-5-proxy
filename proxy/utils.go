package proxy

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

const (
	RequiredSocksVersion = byte(0x05)
	RequiredAuthMethod   = byte(0x00)

	AuthMethodNoAuthenticationRequired = byte(0x00)
	AuthMethodGSSAPI                   = byte(0x01)
	AuthMethodUsernameOrPassword       = byte(0x02)
	AuthNoAcceptableMethods            = byte(0xFF)
	//0x03–0x7F: methods assigned by IANA[17]
	//			0x03: Challenge–Handshake Authentication Protocol
	//			0x04: Unassigned
	//			0x05: Challenge–Response Authentication Method
	//			0x06: Secure Sockets Layer
	//			0x07: NDS Authentication
	//			0x08: Multi-Authentication Framework
	//			0x09: JSON Parameter Block
	//			0x0A–0x7F: Unassigned
	//0x80–0xFE: methods reserved for private use

	REPLYSucceeded                     = byte(0x00)
	REPLYGeneralSOCKSServerFailure     = byte(0x01)
	REPLYConnectionNotAllowedByRuleset = byte(0x02)
	REPLYNetworkUnreachable            = byte(0x03)
	REPLYHostUnreachable               = byte(0x04)
	REPLYConnectionRefused             = byte(0x05)
	REPLYTTLExpired                    = byte(0x06)
	REPLYCommandNotSupported           = byte(0x07)
	REPLYAddressTypeNotSupported       = byte(0x08)

	CMDConnect      = byte(0x01)
	CMDBind         = byte(0x02)
	CMDUdpAssociate = byte(0x03)

	ATYPIPV4Address = byte(0x01)
	ATYPDomainName  = byte(0x03)
	ATYPIPV6Address = byte(0x04)
)

var LOG *logrus.Logger

type CustomTextFormatter struct{}

func (f *CustomTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	message := fmt.Sprintf(
		"%s %5.5s %s\n",
		entry.Time.Format("2006-01-02 15:04:05.000"), // Date-time
		strings.ToUpper(entry.Level.String()),        // Log level
		entry.Message,                                // Log message
	)

	return []byte(message), nil
}

func init() {
	file, err := os.OpenFile(
		"proxy.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		panic(err)
	}
	LOG = &logrus.Logger{
		Out:       file,
		Level:     logrus.TraceLevel,
		Formatter: &CustomTextFormatter{},
	}
}
