package cloudflare

import (
	"encoding/json"
	"testing"
)

// TestParseEmailRoutingDNSRecordsAcceptsEnablementObject verifies that the
// Email Routing enable endpoint can return a status object instead of DNS
// records and should still be treated as a successful no-op for callers that
// only need to ensure the feature is enabled.
func TestParseEmailRoutingDNSRecordsAcceptsEnablementObject(t *testing.T) {
	raw := json.RawMessage(`{
		"id": "9a1e91c12c5575164bf31d0988fd2954",
		"tag": "9a1e91c12c5575164bf31d0988fd2954",
		"name": "linuxdo.space",
		"enabled": true,
		"created": "2026-03-09T16:40:17.046665Z",
		"modified": "2026-03-10T15:42:13.790082Z",
		"skip_wizard": true,
		"support_subaddress": false,
		"synced": true,
		"admin_locked": false,
		"status": "ready"
	}`)

	records, err := parseEmailRoutingDNSRecords(raw)
	if err != nil {
		t.Fatalf("parse email routing enablement object: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected no dns records from enablement object, got %+v", records)
	}
}

// TestParseEmailRoutingDNSRecordsAcceptsNamespaceRecordList verifies that the
// namespace DNS inspection payload still extracts the required MX and SPF
// records from Cloudflare's `records` field.
func TestParseEmailRoutingDNSRecordsAcceptsNamespaceRecordList(t *testing.T) {
	raw := json.RawMessage(`{
		"errors": [
			{
				"code": "mx.missing"
			}
		],
		"records": [
			{
				"name": "alice.linuxdo.space",
				"content": "route1.mx.cloudflare.net.",
				"type": "MX",
				"priority": 60,
				"ttl": 1
			},
			{
				"name": "alice.linuxdo.space",
				"content": "\"v=spf1 include:_spf.mx.cloudflare.net ~all\"",
				"type": "TXT",
				"ttl": 1
			}
		]
	}`)

	records, err := parseEmailRoutingDNSRecords(raw)
	if err != nil {
		t.Fatalf("parse namespace email routing records: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected two dns records, got %d", len(records))
	}
	if records[0].Name != "alice.linuxdo.space" || records[0].Type != "MX" {
		t.Fatalf("unexpected first record: %+v", records[0])
	}
	if records[1].Type != "TXT" {
		t.Fatalf("unexpected second record: %+v", records[1])
	}
}
