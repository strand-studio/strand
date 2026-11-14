package fanout

import (
	"path/filepath"
	"testing"

	"strand/internal/store"
)

func TestMessagingAlertTargets_RespectsDefaults(t *testing.T) {
	dir := t.TempDir()
	st, err := store.New(filepath.Join(dir, "fanout.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	chAll := &store.MessagingChannel{
		ID:            "ch-all",
		UserID:        "u1",
		Name:          "all",
		PlatformType:  "telegram",
		Enabled:       true,
		AlertsEnabled: true,
	}
	chSel := &store.MessagingChannel{
		ID:            "ch-sel",
		UserID:        "u1",
		Name:          "sel",
		PlatformType:  "discord",
		Enabled:       true,
		AlertsEnabled: true,
	}
	if err := st.Messaging().CreateChannel(chAll); err != nil {
		t.Fatal(err)
	}
	if err := st.Messaging().CreateChannel(chSel); err != nil {
		t.Fatal(err)
	}

	a := New(st, nil)
	all := a.messagingAlertTargets()
	if len(all) != 2 {
		t.Fatalf("expected 2 without selection, got %d", len(all))
	}

	d := &store.ResourceDefaults{UserID: "u1"}
	d.SetAlertIDs([]string{"ch-sel"})
	if err := st.Resources().SaveDefaults(d); err != nil {
		t.Fatal(err)
	}
	filtered := a.messagingAlertTargets()
	if len(filtered) != 1 || filtered[0].ID != "ch-sel" {
		t.Fatalf("expected only ch-sel, got %#v", filtered)
	}
}
