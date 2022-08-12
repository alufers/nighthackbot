package nighthackbot

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ScheduleExpression struct {
	Leafs []*ScheduleExpressionLeaf
}

func ParseScheduleExpression(src string) (*ScheduleExpression, error) {
	segments := strings.Split(src, ",")
	if len(segments) == 0 {
		return nil, fmt.Errorf("empty schedule expression")
	}
	se := &ScheduleExpression{}
	for _, segment := range segments {
		leaf, err := ParseScheduleExpressionLeaf(strings.TrimSpace(segment))
		if err != nil {
			return nil, err
		}
		se.Leafs = append(se.Leafs, &leaf)
	}
	return se, nil
}

func (s *ScheduleExpression) String() string {
	leafs := []string{}
	for _, leaf := range s.Leafs {
		leafs = append(leafs, leaf.String())
	}
	return strings.Join(leafs, ", ")
}

func (se *ScheduleExpression) GetNextOccurence(now time.Time) time.Time {
	// return the earliest occurence of the leafs
	var result time.Time
	for _, leaf := range se.Leafs {
		t := leaf.GetNextOccurence(now)
		if result.IsZero() || t.Before(result) {
			result = t
		}
	}
	return result
}

type WeekdayMask uint

const (
	WeekdayMonday = WeekdayMask(1 << iota)
	WeekdayTuesday
	WeekdayWednesday
	WeekdayThursday
	WeekdayFriday
	WeekdaySaturday
	WeekdaySunday
)
const AllWeekdays = WeekdayMonday | WeekdayTuesday | WeekdayWednesday | WeekdayThursday | WeekdayFriday | WeekdaySaturday | WeekdaySunday

var WeekDayNames = map[string]WeekdayMask{
	"monday":    WeekdayMonday,
	"tuesday":   WeekdayTuesday,
	"wednesday": WeekdayWednesday,
	"thursday":  WeekdayThursday,
	"friday":    WeekdayFriday,
	"saturday":  WeekdaySaturday,
	"sunday":    WeekdaySunday,
	"everyday":  AllWeekdays,
}

func timeWeekdayToMask(t time.Weekday) WeekdayMask {
	switch t {
	case time.Monday:
		return WeekdayMonday
	case time.Tuesday:
		return WeekdayTuesday
	case time.Wednesday:
		return WeekdayWednesday
	case time.Thursday:
		return WeekdayThursday
	case time.Friday:
		return WeekdayFriday
	case time.Saturday:
		return WeekdaySaturday
	case time.Sunday:
		return WeekdaySunday
	}
	return 0
}

type ScheduleExpressionLeaf struct {
	WeekdayMask WeekdayMask
	Hour        int
	Minute      int
}

func (se *ScheduleExpressionLeaf) String() string {
	dayNames := []string{}

	for weekday, mask := range WeekDayNames {
		if se.WeekdayMask&mask != 0 {
			dayNames = append(dayNames, weekday)
		}
	}
	if se.WeekdayMask&AllWeekdays != 0 {
		dayNames = []string{"everyday"}
	}
	return fmt.Sprintf("%s %02d:%02d", strings.Join(dayNames, " "), se.Hour, se.Minute)
}

func (se *ScheduleExpressionLeaf) GetNextOccurence(now time.Time) time.Time {

	t, _ := time.Parse("2006/01/02", now.Format("2006/01/02"))
	t = t.Add(time.Hour * time.Duration(se.Hour))
	t = t.Add(time.Minute * time.Duration(se.Minute))

	for i := 0; i < 8; i++ {
		if se.WeekdayMask&timeWeekdayToMask(t.Weekday()) != 0 && t.After(now) {
			return t
		}
		t = t.Add(time.Hour * 24)
	}
	panic("unreachable")
}

func ParseScheduleExpressionLeaf(src string) (ScheduleExpressionLeaf, error) {
	se := ScheduleExpressionLeaf{}
	parts := strings.Split(src, " ")
	if len(parts) < 2 {
		return se, fmt.Errorf("expected day name and hour, only one part found: '%s'", src)
	}
	for _, part := range parts[:len(parts)-1] {
		part = strings.ToLower(part)
		if weekday, ok := WeekDayNames[part]; ok {
			se.WeekdayMask |= weekday
		} else {
			allowedNames := []string{}
			for name := range WeekDayNames {
				allowedNames = append(allowedNames, name)
			}
			return se, fmt.Errorf("invalid day name: '%s', allowed names are: %v", src, strings.Join(allowedNames, ", "))
		}
	}

	timeParts := strings.Split(parts[len(parts)-1], ":")
	if len(timeParts) != 2 {
		return se, fmt.Errorf("expected hour and minute, found: '%s'", src)
	}
	hour, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return se, fmt.Errorf("failed to parse hour: '%s'", src)
	}
	minute, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return se, fmt.Errorf("failed to parse minute: '%s'", src)
	}
	if hour < 0 || hour > 23 {
		return se, fmt.Errorf("invalid hour: '%s'", src)
	}
	if minute < 0 || minute > 59 {
		return se, fmt.Errorf("invalid minute: '%s'", src)
	}
	se.Hour = hour
	se.Minute = minute
	return se, nil
}
