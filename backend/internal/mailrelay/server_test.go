package mailrelay

import (
	"context"
	"errors"
	"log"
	"strings"
	"testing"
)

type recordingCatchAllAccessManager struct {
	reservations []CatchAllUsageReservation
	releases     []CatchAllUsageReservation
}

func (m *recordingCatchAllAccessManager) Reserve(ctx context.Context, userID int64, count int64) (CatchAllUsageReservation, error) {
	reservation := CatchAllUsageReservation{
		UserID:       userID,
		Count:        count,
		ConsumedMode: "quantity",
		UsageDate:    "2026-03-13",
	}
	m.reservations = append(m.reservations, reservation)
	return reservation, nil
}

func (m *recordingCatchAllAccessManager) Release(ctx context.Context, reservation CatchAllUsageReservation) error {
	m.releases = append(m.releases, reservation)
	return nil
}

type staticForwarder struct {
	err   error
	calls []ForwardRequest
}

func (f *staticForwarder) Forward(ctx context.Context, request ForwardRequest) error {
	f.calls = append(f.calls, request)
	return f.err
}

// TestSMTPDataReservesCatchAllOncePerForwardGroup verifies that multiple
// aliases routed to the same target only reserve one usage unit for that owner.
func TestSMTPDataReservesCatchAllOncePerForwardGroup(t *testing.T) {
	manager := &recordingCatchAllAccessManager{}
	forwarder := &staticForwarder{}
	session := &smtpSession{
		accessManager:  manager,
		forwarder:      forwarder,
		logger:         log.Default(),
		forwardTimeout: defaultForwardTimeout,
		recipients: []ResolvedRecipient{
			{
				OriginalRecipient: "one@alice.linuxdo.space",
				TargetEmail:       "target@example.com",
				RouteOwnerUserID:  11,
				UsedCatchAll:      true,
			},
			{
				OriginalRecipient: "two@alice.linuxdo.space",
				TargetEmail:       "target@example.com",
				RouteOwnerUserID:  11,
				UsedCatchAll:      true,
			},
		},
	}

	if err := session.Data(strings.NewReader("Subject: test\r\n\r\nbody")); err != nil {
		t.Fatalf("smtp data should succeed, got %v", err)
	}
	if len(manager.reservations) != 1 {
		t.Fatalf("expected one reservation for one final forward group, got %d", len(manager.reservations))
	}
	if len(manager.releases) != 0 {
		t.Fatalf("expected no release on successful forward, got %d", len(manager.releases))
	}
	if len(forwarder.calls) != 1 {
		t.Fatalf("expected one forward call, got %d", len(forwarder.calls))
	}
}

// TestSMTPDataReleasesCatchAllReservationOnForwardFailure verifies that quota
// is rolled back when the upstream forward fails after reservation.
func TestSMTPDataReleasesCatchAllReservationOnForwardFailure(t *testing.T) {
	manager := &recordingCatchAllAccessManager{}
	forwarder := &staticForwarder{err: errors.New("smtp upstream failed")}
	session := &smtpSession{
		accessManager:  manager,
		forwarder:      forwarder,
		logger:         log.Default(),
		forwardTimeout: defaultForwardTimeout,
		recipients: []ResolvedRecipient{
			{
				OriginalRecipient: "one@alice.linuxdo.space",
				TargetEmail:       "target@example.com",
				RouteOwnerUserID:  11,
				UsedCatchAll:      true,
			},
		},
	}

	if err := session.Data(strings.NewReader("Subject: test\r\n\r\nbody")); err == nil {
		t.Fatalf("expected smtp data to fail when the upstream forward fails")
	}
	if len(manager.reservations) != 1 {
		t.Fatalf("expected one reservation before the forward attempt, got %d", len(manager.reservations))
	}
	if len(manager.releases) != 1 {
		t.Fatalf("expected one release after forward failure, got %d", len(manager.releases))
	}
}
