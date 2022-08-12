package nighthackbot

import (
	"testing"
	"time"
)

func TestScheduleExpression(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2022-08-12T16:00:00Z")
	parsed, err := ParseScheduleExpression("friday 18:00")
	if err != nil {
		t.Fatal(err)
	}
	next := parsed.GetNextOccurence(now)
	if next.Weekday() != time.Friday {
		t.Fatal("expected friday")
	}
	if next.Format(time.RFC3339) != "2022-08-12T18:00:00Z" {
		t.Fatalf("expected 2022-08-12T18:00:00Z, got %s", next.Format(time.RFC3339))
	}

}

func TestScheduleExpressionSameDayAfter(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2022-08-12T19:00:00Z")
	parsed, err := ParseScheduleExpression("friday 18:00")
	if err != nil {
		t.Fatal(err)
	}
	next := parsed.GetNextOccurence(now)
	if next.Weekday() != time.Friday {
		t.Fatal("expected friday")
	}
	if next.Format(time.RFC3339) != "2022-08-19T18:00:00Z" {
		t.Fatalf("expected 2022-08-19T18:00:00Z, got %s", next.Format(time.RFC3339))
	}

}

func TestScheduleExpressionEveryday(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2022-08-12T19:00:00Z")
	parsed, err := ParseScheduleExpression("everyday 18:00")
	if err != nil {
		t.Fatal(err)
	}
	next := parsed.GetNextOccurence(now)
	if next.Weekday() != time.Saturday {
		t.Fatal("expected Saturday")
	}
	if next.Format(time.RFC3339) != "2022-08-13T18:00:00Z" {
		t.Fatalf("expected 2022-08-13T18:00:00Z, got %s", next.Format(time.RFC3339))
	}

}
