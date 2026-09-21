package recurrence

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/gogi0001/family-tasks/internal/models"
)

var timeRe = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

// Validate — базовая проверка правила. Возвращает ErrInvalid при проблеме.
func Validate(r models.RecurrenceRule) error {
	if !timeRe.MatchString(r.Time) {
		return fmt.Errorf("invalid time %q (want HH:MM)", r.Time)
	}
	switch r.Type {
	case "daily":
		if r.Interval < 0 {
			return errors.New("interval must be >= 0")
		}
	case "weekly":
		if len(r.Weekdays) == 0 {
			return errors.New("weekdays required")
		}
		seen := map[int]bool{}
		for _, d := range r.Weekdays {
			if d < 0 || d > 6 {
				return fmt.Errorf("weekday %d out of range 0..6", d)
			}
			if seen[d] {
				return fmt.Errorf("weekday %d duplicated", d)
			}
			seen[d] = true
		}
	case "monthly":
		if r.DayOfMonth < 1 || r.DayOfMonth > 31 {
			return errors.New("dayOfMonth must be in 1..31")
		}
	default:
		return fmt.Errorf("unknown rule type %q", r.Type)
	}
	return nil
}

// Next возвращает следующий момент после from, в который должно сработать правило.
// Время трактуется в локальной зоне сервера (TZ задаётся через Environment=TZ в unit).
func Next(r models.RecurrenceRule, from time.Time) (time.Time, error) {
	h, m, err := parseHM(r.Time)
	if err != nil {
		return time.Time{}, err
	}

	loc := time.Local
	fromLocal := from.In(loc)

	switch r.Type {
	case "daily":
		interval := r.Interval
		if interval < 1 {
			interval = 1
		}
		c := time.Date(fromLocal.Year(), fromLocal.Month(), fromLocal.Day(), h, m, 0, 0, loc)
		if !c.After(fromLocal) {
			c = c.AddDate(0, 0, interval)
		}
		return c.UTC(), nil

	case "weekly":
		// Ищем ближайший день недели из списка. Сегодня можно, если время не прошло.
		for d := 0; d < 8; d++ {
			cand := fromLocal.AddDate(0, 0, d)
			wd := int(cand.Weekday()) // 0=вс
			if !containsDay(r.Weekdays, wd) {
				continue
			}
			c := time.Date(cand.Year(), cand.Month(), cand.Day(), h, m, 0, 0, loc)
			if c.After(fromLocal) {
				return c.UTC(), nil
			}
		}
		return time.Time{}, errors.New("no next weekday found")

	case "monthly":
		y, mo := fromLocal.Year(), fromLocal.Month()
		for i := 0; i < 14; i++ {
			c := safeDate(y, mo, r.DayOfMonth, h, m, loc)
			if c.After(fromLocal) {
				return c.UTC(), nil
			}
			mo++
			if mo > 12 {
				mo = 1
				y++
			}
		}
		return time.Time{}, errors.New("no next month found")
	}
	return time.Time{}, fmt.Errorf("unknown rule type %q", r.Type)
}

// Describe возвращает человеческое описание правила.
func Describe(r models.RecurrenceRule) string {
	switch r.Type {
	case "daily":
		i := r.Interval
		if i < 1 {
			i = 1
		}
		if i == 1 {
			return "Каждый день в " + r.Time
		}
		return fmt.Sprintf("Каждые %d дня в %s", i, r.Time)
	case "weekly":
		names := []string{"вс", "пн", "вт", "ср", "чт", "пт", "сб"}
		parts := make([]string, 0, len(r.Weekdays))
		for _, d := range r.Weekdays {
			if d >= 0 && d < len(names) {
				parts = append(parts, names[d])
			}
		}
		return "По " + joinComma(parts) + " в " + r.Time
	case "monthly":
		return fmt.Sprintf("%d-го числа в %s", r.DayOfMonth, r.Time)
	}
	return r.Type
}

// --- helpers ---

func parseHM(s string) (int, int, error) {
	m := timeRe.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, fmt.Errorf("invalid time %q", s)
	}
	h, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])
	return h, mi, nil
}

func containsDay(days []int, d int) bool {
	for _, x := range days {
		if x == d {
			return true
		}
	}
	return false
}

// safeDate: если в месяце нет такого дня (например, 31 февраля),
// возвращаем последний день месяца.
func safeDate(y int, mo time.Month, day, h, m int, loc *time.Location) time.Time {
	last := time.Date(y, mo+1, 0, h, m, 0, 0, loc)
	if day > last.Day() {
		return last
	}
	return time.Date(y, mo, day, h, m, 0, 0, loc)
}

func joinComma(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}