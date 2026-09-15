package scheduler

import (
	"testing"

	"github.com/vikas/media-sequencer/backend/internal/models"
)

func testState() *models.State {
	return &models.State{
		CycleAnchorMs: 1_000_000,
		Media: []models.Media{
			{ID: "m1", Name: "M1", Type: models.MediaImage, URL: "http://x/1.jpg", DefaultDurationMs: 5000},
			{ID: "m2", Name: "M2", Type: models.MediaImage, URL: "http://x/2.jpg", DefaultDurationMs: 5000},
			{ID: "m3", Name: "M3", Type: models.MediaBlank, DefaultDurationMs: 5000},
		},
		Windows: []models.Window{
			{ID: "w1", Name: "W1", Playlist: []models.PlaylistItem{
				{ID: "i1", MediaID: "m1", DurationMs: 10_000},
				{ID: "i2", MediaID: "m2", DurationMs: 20_000},
				{ID: "i3", MediaID: "m3", DurationMs: 5_000},
			}},
		},
	}
}

func TestResolveWalksThePlaylistInOrder(t *testing.T) {
	st := testState()
	tl := New()
	w := &st.Windows[0]
	anchor := st.CycleAnchorMs

	cases := []struct {
		offset  int64
		wantIdx int
		wantRem int64
	}{
		{0, 0, 10_000},
		{9_999, 0, 1},
		{10_000, 1, 20_000},
		{29_999, 1, 1},
		{30_000, 2, 5_000},
		{34_999, 2, 1},
	}
	for _, c := range cases {
		pb := tl.Resolve(st, w, anchor+c.offset)
		if pb.ItemIndex != c.wantIdx || pb.RemainingMs != c.wantRem {
			t.Fatalf("offset %d: got idx=%d rem=%d, want idx=%d rem=%d",
				c.offset, pb.ItemIndex, pb.RemainingMs, c.wantIdx, c.wantRem)
		}
		if pb.Source != SourceSequence {
			t.Fatalf("offset %d: expected sequence source, got %s", c.offset, pb.Source)
		}
	}
}

func TestPlaylistLoopsWithinTheCycle(t *testing.T) {
	st := testState()
	tl := New()
	w := &st.Windows[0]
	pb := tl.Resolve(st, w, st.CycleAnchorMs+35_000)
	if pb.ItemIndex != 0 {
		t.Fatalf("expected loop back to item 0, got %d", pb.ItemIndex)
	}
	pb = tl.Resolve(st, w, st.CycleAnchorMs+35_000*100+12_000)
	if pb.ItemIndex != 1 {
		t.Fatalf("expected item 1 deep into the cycle, got %d", pb.ItemIndex)
	}
}

func TestCycleRestartsAtFiveHours(t *testing.T) {
	st := testState()
	tl := New()
	w := &st.Windows[0]

	pb := tl.Resolve(st, w, st.CycleAnchorMs+DefaultCycleMs)
	if pb.ItemIndex != 0 || pb.OffsetInCycleMs != 0 {
		t.Fatalf("cycle boundary must restart at item 0 offset 0, got idx=%d offset=%d",
			pb.ItemIndex, pb.OffsetInCycleMs)
	}
	if pb.CycleIndex != 1 {
		t.Fatalf("expected cycle index 1, got %d", pb.CycleIndex)
	}
}

func TestItemIsTruncatedAtTheCycleBoundaryNotPaddedWithBlank(t *testing.T) {
	st := testState()
	tl := New()
	w := &st.Windows[0]

	justBefore := st.CycleAnchorMs + DefaultCycleMs - 1_000
	pb := tl.Resolve(st, w, justBefore)
	if pb.Source != SourceSequence {
		t.Fatalf("tail of the cycle must keep playing the list, got %s", pb.Source)
	}
	if pb.ItemIndex != 0 {
		t.Fatalf("expected item 0 at the tail, got %d", pb.ItemIndex)
	}
	if pb.RemainingMs != 1_000 {
		t.Fatalf("item must be truncated to the cycle boundary, remaining=%d", pb.RemainingMs)
	}
}

func TestSyncOverridesEveryWindowAndThenReleases(t *testing.T) {
	st := testState()
	tl := New()
	w := &st.Windows[0]
	now := st.CycleAnchorMs + 12_000 // mid item 1

	st.ActiveSync = &models.SyncEvent{
		ID: "s1", MediaID: "m2", StartAtMs: now, DurationMs: 6_000, TriggeredAtMs: now,
	}

	during := tl.Resolve(st, w, now+1_000)
	if during.Source != SourceSync || during.MediaID != "m2" {
		t.Fatalf("expected sync override, got source=%s media=%s", during.Source, during.MediaID)
	}
	if during.RemainingMs != 5_000 {
		t.Fatalf("expected 5000ms of sync left, got %d", during.RemainingMs)
	}

	after := tl.Resolve(st, w, now+6_000)
	if after.Source != SourceSequence {
		t.Fatalf("expected release back to sequence, got %s", after.Source)
	}
	if after.ItemIndex != 1 {
		t.Fatalf("expected to rejoin item 1 mid-flight, got %d", after.ItemIndex)
	}
	if after.RemainingMs != 12_000 {
		t.Fatalf("expected 12000ms left of item 1, got %d", after.RemainingMs)
	}
}

func TestEmptyPlaylistIsIdleNotACrash(t *testing.T) {
	st := testState()
	st.Windows[0].Playlist = nil
	pb := New().Resolve(st, &st.Windows[0], st.CycleAnchorMs+5_000)
	if pb.Source != SourceIdle {
		t.Fatalf("expected idle, got %s", pb.Source)
	}
}

func TestClockBehindAnchorDoesNotPanicOrGoNegative(t *testing.T) {
	st := testState()
	pb := New().Resolve(st, &st.Windows[0], st.CycleAnchorMs-7_000)
	if pb.OffsetInCycleMs < 0 || pb.RemainingMs <= 0 {
		t.Fatalf("negative delta mishandled: offset=%d remaining=%d",
			pb.OffsetInCycleMs, pb.RemainingMs)
	}
}
