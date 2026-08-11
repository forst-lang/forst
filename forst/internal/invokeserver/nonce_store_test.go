package invokeserver

import (
	"sync"
	"testing"
	"time"
)

func TestNonceStore_consume_singleUseOnly(t *testing.T) {
	store := newNonceStore(time.Second)
	nonce, _, err := store.issue(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !store.consume(nonce, time.Now()) {
		t.Fatal("expected first consume to succeed")
	}
	if store.consume(nonce, time.Now()) {
		t.Fatal("expected replay to fail")
	}
}

func TestNonceStore_consume_expiredNonceRejected(t *testing.T) {
	store := newNonceStore(time.Second)
	issuedAt := time.Now()
	nonce, _, err := store.issue(issuedAt)
	if err != nil {
		t.Fatal(err)
	}
	if store.consume(nonce, issuedAt.Add(store.ttl+time.Nanosecond)) {
		t.Fatal("expected expired nonce to fail")
	}
}

func TestNonceStore_issue_rejectsAtCapacity(t *testing.T) {
	store := newNonceStore(time.Hour)
	now := time.Now()
	for i := 0; i < maxOutstandingNonces; i++ {
		if _, _, err := store.issue(now); err != nil {
			t.Fatalf("issue %d: %v", i, err)
		}
	}
	if _, _, err := store.issue(now); err == nil {
		t.Fatal("expected capacity error")
	}
}

func TestNonceStore_consume_concurrentCallersOnlyOneSucceeds(t *testing.T) {
	store := newNonceStore(time.Second)
	nonce, _, err := store.issue(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	successes := make(chan bool, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			successes <- store.consume(nonce, time.Now())
		}()
	}
	wg.Wait()
	close(successes)
	count := 0
	for ok := range successes {
		if ok {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("successes = %d", count)
	}
}
