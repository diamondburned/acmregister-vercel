package verifyemail

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/diamondburned/acmregister/acmregister"
	"github.com/diamondburned/arikawa/v3/discord"
)

func init() { rand.Seed(time.Now().UnixNano()) }

// PINStore describes an interface that stores the state for verifying PINs over
// email.
type PINStore interface {
	acmregister.ContainsContext

	// GeneratePIN generates a new PIN that's assigned to the given email.
	//
	// TODO: invalidate the old PIN if there's already an existing one.
	GeneratePIN(discord.GuildID, discord.UserID) (PIN, error)
	// ValidatePIN validates the email associated with the given PIN. PINStores
	// should use its underlying SubmissionStore for this.
	ValidatePIN(discord.GuildID, discord.UserID, PIN) (*acmregister.MemberMetadata, error)
}

// PINDigits is the number of digits in the PIN code.
const PINDigits = 4

const pinf = "%04d"

// PIN describes a PIN code.
type PIN int

// InvalidPIN is an invalid PIN to be used as a placeholder.
const InvalidPIN PIN = 0000

// String formats a PIN code from 0001 to 9999. Anything else is invalid.
func (pin PIN) String() string {
	if pin <= InvalidPIN || pin > 9999 {
		return "<invalid PIN>"
	}
	return fmt.Sprintf(pinf, pin)
}

// Format always formats the PIN as a 4-digit number without checking.
func (pin PIN) Format() string {
	return fmt.Sprintf(pinf, pin)
}

var maxPIN = int(math.Pow10(PINDigits) - 1)

// GeneratePIN generates a random PIN code for use.
func GeneratePIN() PIN {
	return PIN(rand.Intn(maxPIN) + 1)
}
