package mysql

import "testing"

func TestPoolManagerInvalidateMissingPoolIsNoop(t *testing.T) {
	manager := NewPoolManager()
	if err := manager.Invalidate(999); err != nil {
		t.Fatalf("Invalidate() error = %v", err)
	}
}

func TestPoolManagerCloseAllEmptyIsNoop(t *testing.T) {
	manager := NewPoolManager()
	if err := manager.CloseAll(); err != nil {
		t.Fatalf("CloseAll() error = %v", err)
	}
}
