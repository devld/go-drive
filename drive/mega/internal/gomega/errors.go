package gomega

import (
	"errors"
	"fmt"
)

var (
	// General errors
	EINTERNAL  = errors.New("Internal error occurred")
	EARGS      = errors.New("Invalid arguments")
	EAGAIN     = errors.New("Try again")
	ERATELIMIT = errors.New("Rate limit reached")
	EBADRESP   = errors.New("Bad response from server")

	// Upload errors
	EFAILED  = errors.New("The upload failed. Please restart it from scratch")
	ETOOMANY = errors.New("Too many concurrent IP addresses are accessing this upload target URL")
	ERANGE   = errors.New("The upload file packet is out of range or not starting and ending on a chunk boundary")
	EEXPIRED = errors.New("The upload target URL you are trying to access has expired. Please request a fresh one")

	// Filesystem/Account errors
	ENOENT              = errors.New("Object (typically, node or user) not found")
	ECIRCULAR           = errors.New("Circular linkage attempted")
	EACCESS             = errors.New("Access violation")
	EEXIST              = errors.New("Trying to create an object that already exists")
	EINCOMPLETE         = errors.New("Trying to access an incomplete resource")
	EKEY                = errors.New("A decryption operation failed")
	ESID                = errors.New("Invalid or expired user session, please relogin")
	EBLOCKED            = errors.New("User blocked")
	EOVERQUOTA          = errors.New("Request over quota")
	ETEMPUNAVAIL        = errors.New("Resource temporarily not available, please try again later")
	EMACMISMATCH        = errors.New("MAC verification failed")
	EBADATTR            = errors.New("Bad node attribute")
	ETOOMANYCONNECTIONS = errors.New("Too many connections on this resource")
	EWRITE              = errors.New("File could not be written to (or failed post-write integrity check)")
	EREAD               = errors.New("File could not be read from (or changed unexpectedly during reading)")
	EAPPKEY             = errors.New("Invalid or missing application key")
	ESSL                = errors.New("SSL verification failed")
	EGOINGOVERQUOTA     = errors.New("Not enough quota")
	EMFAREQUIRED        = errors.New("Multi-factor authentication required")

	// Config errors
	EWORKER_LIMIT_EXCEEDED = errors.New("Maximum worker limit exceeded")
)

type ErrorMsg int

func parseError(errno ErrorMsg) error {
	switch errno {
	case 0:
		return nil
	case -1:
		return EINTERNAL
	case -2:
		return EARGS
	case -3:
		return EAGAIN
	case -4:
		return ERATELIMIT
	case -5:
		return EFAILED
	case -6:
		return ETOOMANY
	case -7:
		return ERANGE
	case -8:
		return EEXPIRED
	case -9:
		return ENOENT
	case -10:
		return ECIRCULAR
	case -11:
		return EACCESS
	case -12:
		return EEXIST
	case -13:
		return EINCOMPLETE
	case -14:
		return EKEY
	case -15:
		return ESID
	case -16:
		return EBLOCKED
	case -17:
		return EOVERQUOTA
	case -18:
		return ETEMPUNAVAIL
	case -19:
		return ETOOMANYCONNECTIONS
	case -20:
		return EWRITE
	case -21:
		return EREAD
	case -22:
		return EAPPKEY
	case -23:
		return ESSL
	case -24:
		return EGOINGOVERQUOTA
	case -26:
		return EMFAREQUIRED
	default:
		return fmt.Errorf("Unknown mega error %d", errno)
	}
}
