// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package strftime formats times with the conversion specifications of
// POSIX strftime() in the POSIX locale, plus the GNU extensions date
// scripts use.
//
// A specification is % [flags] [width] [E|O] conversion. Flags: '0' pads
// with zeros, '+' pads with zeros and signs a year wider than four digits
// (two for %C) (POSIX); '-' does not pad, '_' pads with spaces, '^' upper
// cases, '#' swaps the case (GNU). E and O are accepted and ignored (no
// alternative representations in the POSIX locale). An unknown
// specification is copied as is.
//
// Conversions (POSIX): a A b B c C d D e F g G h H I j m M n p r R s S t
// T u U V w W x X y Y z Z %. GNU: N (nanoseconds), P (am/pm), k and l
// (space-padded hours), :z (+hh:mm).
package strftime

import (
	"strconv"
	"strings"
	"time"
)

// MaxWidth bounds a field width: "%99999999999d" pads to MaxWidth.
const MaxWidth = 1024

// Format returns t formatted by layout.
func Format(t time.Time, layout string) string {
	var b strings.Builder
	for i := 0; i < len(layout); i++ {
		c := layout[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		start := i
		i++
		var flag byte
		for i < len(layout) && strings.IndexByte("-_0^#+", layout[i]) >= 0 {
			flag = layout[i]
			i++
		}
		width := 0
		for i < len(layout) && layout[i] >= '0' && layout[i] <= '9' {
			// Clamped: a width is a pad, not a buffer size to honour.
			width = min(width*10+int(layout[i]-'0'), MaxWidth)
			i++
		}
		if i < len(layout) && (layout[i] == 'E' || layout[i] == 'O') {
			i++
		}
		colon := false
		if i+1 < len(layout) && layout[i] == ':' && layout[i+1] == 'z' {
			colon = true
			i++
		}
		if i >= len(layout) {
			b.WriteString(layout[start:])
			break
		}
		s, ok := conv(t, layout[i], flag, width, colon)
		if !ok {
			b.WriteString(layout[start : i+1])
			continue
		}
		b.WriteString(s)
	}
	return b.String()
}

var weekdays = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
var months = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}

// conv renders one conversion.
func conv(t time.Time, c, flag byte, width int, colon bool) (string, bool) {
	num := func(v int64, digits int, pad byte) string { return number(v, digits, pad, flag, width) }
	str := func(s string) string { return text(s, flag, width) }
	hour12 := t.Hour() % 12
	if hour12 == 0 {
		hour12 = 12
	}
	yday := t.YearDay() - 1
	wday := int(t.Weekday())
	switch c {
	case 'a':
		return str(weekdays[wday][:3]), true
	case 'A':
		return str(weekdays[wday]), true
	case 'b', 'h':
		return str(months[t.Month()-1][:3]), true
	case 'B':
		return str(months[t.Month()-1]), true
	case 'c':
		return str(Format(t, "%a %b %e %H:%M:%S %Y")), true
	case 'C':
		return year(int64(t.Year())/100, 2, 2, flag, width), true
	case 'd':
		return num(int64(t.Day()), 2, '0'), true
	case 'D', 'x':
		return str(Format(t, "%m/%d/%y")), true
	case 'e':
		return num(int64(t.Day()), 2, ' '), true
	case 'F':
		// POSIX: with a width x, the year as %Y with a width of x-6
		// (x below 6 counts as 6).
		w := 0
		if width > 0 {
			w = max(width, 6) - 6
		}
		return year(int64(t.Year()), 1, 4, flag, w) + Format(t, "-%m-%d"), true
	case 'g':
		y, _ := t.ISOWeek()
		return num(int64(y%100), 2, '0'), true
	case 'G':
		y, _ := t.ISOWeek()
		return year(int64(y), 1, 4, flag, width), true
	case 'H':
		return num(int64(t.Hour()), 2, '0'), true
	case 'I':
		return num(int64(hour12), 2, '0'), true
	case 'j':
		return num(int64(yday+1), 3, '0'), true
	case 'k':
		return num(int64(t.Hour()), 2, ' '), true
	case 'l':
		return num(int64(hour12), 2, ' '), true
	case 'm':
		return num(int64(t.Month()), 2, '0'), true
	case 'M':
		return num(int64(t.Minute()), 2, '0'), true
	case 'n':
		return "\n", true
	case 'N':
		return num(int64(t.Nanosecond()), 9, '0'), true
	case 'p', 'P':
		s := "AM"
		if t.Hour() >= 12 {
			s = "PM"
		}
		if c == 'P' {
			s = strings.ToLower(s)
		}
		if flag == '#' {
			return text(strings.ToLower(s), 0, width), true
		}
		return str(s), true
	case 'r':
		return str(Format(t, "%I:%M:%S %p")), true
	case 'R':
		return str(Format(t, "%H:%M")), true
	case 's':
		return num(t.Unix(), 1, '0'), true
	case 'S':
		return num(int64(t.Second()), 2, '0'), true
	case 't':
		return "\t", true
	case 'T', 'X':
		return str(Format(t, "%H:%M:%S")), true
	case 'u':
		u := wday
		if u == 0 {
			u = 7
		}
		return num(int64(u), 1, '0'), true
	case 'U':
		return num(int64((yday+7-wday)/7), 2, '0'), true
	case 'V':
		_, w := t.ISOWeek()
		return num(int64(w), 2, '0'), true
	case 'w':
		return num(int64(wday), 1, '0'), true
	case 'W':
		return num(int64((yday+7-(wday+6)%7)/7), 2, '0'), true
	case 'y':
		return num(int64(t.Year()%100), 2, '0'), true
	case 'Y':
		return year(int64(t.Year()), 1, 4, flag, width), true
	case 'z':
		_, off := t.Zone()
		sign := byte('+')
		if off < 0 {
			sign, off = '-', -off
		}
		s := string(sign) + pad2(off/3600)
		if colon {
			s += ":"
		}
		return str(s + pad2(off%3600/60)), true
	case 'Z':
		name, _ := t.Zone()
		if flag == '#' {
			return text(strings.ToLower(name), 0, width), true
		}
		return str(name), true
	case '%':
		return "%", true
	}
	return "", false
}

func pad2(v int) string {
	if v < 10 {
		return "0" + strconv.Itoa(v)
	}
	return strconv.Itoa(v)
}

// number renders v with its default digits and pad, as the flag and the
// width change them.
func number(v int64, digits int, pad, flag byte, width int) string {
	switch flag {
	case '-':
		pad, digits = 0, 1
	case '_':
		pad = ' '
	case '0', '+':
		pad = '0'
	}
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	s := strconv.FormatInt(v, 10)
	w := max(width, digits) - len(sign) // the sign counts in the width
	switch {
	case pad == '0' && len(s) < w:
		s = strings.Repeat("0", w-len(s)) + s // after the sign
	case pad == ' ' && len(s) < w:
		return strings.Repeat(" ", w-len(s)) + sign + s
	}
	return sign + s
}

// year renders a year (or century, signLimit 2): POSIX's '+' pads with
// zeros to the width and signs a value that needs more than signLimit
// digits; otherwise as a number of at least minDigits.
func year(v int64, minDigits, signLimit int, flag byte, width int) string {
	if flag != '+' || v < 0 {
		return number(v, minDigits, '0', flag, width)
	}
	padded := number(v, minDigits, '0', '0', width)
	if len(padded) <= signLimit {
		return padded
	}
	return "+" + number(v, minDigits, '0', '0', width-1)
}

// text renders a name with the case flags and a width (space padded, or
// zero padded with '0').
func text(s string, flag byte, width int) string {
	switch flag {
	case '^', '#': // '#' swaps: names are capitalised, so upper case
		s = strings.ToUpper(s)
	}
	if len(s) < width {
		p := " "
		if flag == '0' {
			p = "0"
		}
		s = strings.Repeat(p, width-len(s)) + s
	}
	return s
}
