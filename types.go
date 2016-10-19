package twilio

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ttacon/libphonenumber"
)

type PhoneNumber string

var ErrEmptyNumber = errors.New("twilio: The provided phone number was empty")

// NewPhoneNumber parses the given value as a phone number or returns an error
// if it cannot be parsed as one. If a phone number does not begin with a plus
// sign, we assume it's a US national number. Numbers are stored in E.164
// format.
func NewPhoneNumber(pn string) (PhoneNumber, error) {
	if len(pn) == 0 {
		return "", ErrEmptyNumber
	}
	num, err := libphonenumber.Parse(pn, "US")
	// Add some better error messages - the ones in libphonenumber are generic
	switch {
	case err == libphonenumber.ErrNotANumber:
		return "", fmt.Errorf("twilio: Invalid phone number: %s", pn)
	case err == libphonenumber.ErrInvalidCountryCode:
		return "", fmt.Errorf("twilio: Invalid country code for number: %s", pn)
	case err != nil:
		return "", err
	}
	return PhoneNumber(libphonenumber.Format(num, libphonenumber.E164)), nil
}

// Friendly returns a friendly international representation of the phone
// number, for example, "+14105554092" is returned as "+1 410-555-4092". If the
// phone number is not in E.164 format, we try to parse it as a US number. If
// we cannot parse it as a US number, it is returned as is.
func (pn PhoneNumber) Friendly() string {
	num, err := libphonenumber.Parse(string(pn), "US")
	if err != nil {
		return string(pn)
	}
	return libphonenumber.Format(num, libphonenumber.INTERNATIONAL)
}

// Local returns a friendly national representation of the phone number, for
// example, "+14105554092" is returned as "(410) 555-4092". If the phone number
// is not in E.164 format, we try to parse it as a US number. If we cannot
// parse it as a US number, it is returned as is.
func (pn PhoneNumber) Local() string {
	num, err := libphonenumber.Parse(string(pn), "US")
	if err != nil {
		return string(pn)
	}
	return libphonenumber.Format(num, libphonenumber.NATIONAL)
}

// A uintStr is sent back from Twilio as a str, but should be parsed as a uint.
type uintStr uint

type Segments uintStr
type NumMedia uintStr

func (seg *uintStr) UnmarshalJSON(b []byte) error {
	s := new(string)
	if err := json.Unmarshal(b, s); err != nil {
		return err
	}
	u, err := strconv.ParseUint(*s, 10, 64)
	if err != nil {
		return err
	}
	*seg = uintStr(u)
	return nil
}

func (seg *Segments) UnmarshalJSON(b []byte) (err error) {
	u := new(uintStr)
	if err = json.Unmarshal(b, u); err != nil {
		return
	}
	*seg = Segments(*u)
	return
}

func (n *NumMedia) UnmarshalJSON(b []byte) (err error) {
	u := new(uintStr)
	if err = json.Unmarshal(b, u); err != nil {
		return
	}
	*n = NumMedia(*u)
	return
}

// TwilioTime can parse a timestamp returned in the Twilio API and turn it into
// a valid Go Time struct.
type TwilioTime struct {
	Time  time.Time
	Valid bool
}

// NewTwilioTime returns a TwilioTime instance. val should be formatted using
// the TimeLayout.
func NewTwilioTime(val string) *TwilioTime {
	t, err := time.Parse(TimeLayout, val)
	if err == nil {
		return &TwilioTime{Time: t, Valid: true}
	} else {
		return &TwilioTime{}
	}
}

// The reference time, as it appears in the Twilio API.
const TimeLayout = "Mon, 2 Jan 2006 15:04:05 -0700"

func (t *TwilioTime) UnmarshalJSON(b []byte) error {
	s := new(string)
	if err := json.Unmarshal(b, s); err != nil {
		return err
	}
	if s == nil || *s == "null" || *s == "" {
		t.Valid = false
		return nil
	}
	tim, err := time.Parse(TimeLayout, *s)
	if err != nil {
		return err
	}
	*t = TwilioTime{Time: tim, Valid: true}
	return nil
}

func (tt *TwilioTime) MarshalJSON() ([]byte, error) {
	if tt.Valid == false {
		return []byte("null"), nil
	}
	b, err := json.Marshal(tt.Time)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

var symbols = map[string]string{
	"USD": "$",
	"GBP": "£",
	"JPY": "¥",
	"MXN": "$",
	"CHF": "CHF",
	"CAD": "$",
	"CNY": "¥",
	"SGD": "$",
	"EUR": "€",
}

// Price flips the sign of the amount and prints it with a currency symbol for
// the given unit.
func price(unit string, amount string) string {
	if len(amount) == 0 {
		return amount
	}
	if amount[0] == '-' {
		amount = amount[1:]
	} else {
		amount = "-" + amount
	}
	for strings.Contains(amount, ".") && strings.HasSuffix(amount, "0") {
		amount = amount[:len(amount)-1]
	}
	if strings.HasSuffix(amount, ".") {
		amount = amount[:len(amount)-1]
	}
	unit = strings.ToUpper(unit)
	if sym, ok := symbols[unit]; ok {
		return sym + amount
	} else {
		if unit == "" {
			return amount
		}
		return unit + " " + amount
	}
}

type TwilioDuration time.Duration

func (td *TwilioDuration) UnmarshalJSON(b []byte) error {
	s := new(string)
	if err := json.Unmarshal(b, s); err != nil {
		return err
	}
	i, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		return err
	}
	*td = TwilioDuration(i) * TwilioDuration(time.Second)
	return nil
}

type AnsweredBy string

const AnsweredByHuman = AnsweredBy("human")
const AnsweredByMachine = AnsweredBy("machine")

type NullAnsweredBy struct {
	Valid      bool
	AnsweredBy AnsweredBy
}

// The status of the message (accepted, queued, etc).
// For more information , see https://www.twilio.com/docs/api/rest/message
type Status string

func (s Status) Friendly() string {
	switch s {
	case StatusInProgress:
		return "In Progress"
	case StatusNoAnswer:
		return "No Answer"
	default:
		return strings.Title(string(s))
	}
}

const StatusAccepted = Status("accepted")
const StatusDelivered = Status("delivered")
const StatusReceiving = Status("receiving")
const StatusReceived = Status("received")
const StatusSending = Status("sending")
const StatusSent = Status("sent")
const StatusUndelivered = Status("undelivered")

// Call statuses
const StatusBusy = Status("busy")
const StatusCanceled = Status("canceled")
const StatusCompleted = Status("completed")
const StatusInProgress = Status("in-progress")
const StatusNoAnswer = Status("no-answer")
const StatusRinging = Status("ringing")

// Shared
const StatusFailed = Status("failed")
const StatusQueued = Status("queued")
