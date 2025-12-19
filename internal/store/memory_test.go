package store

import (
	"math/big"
	"sync"
	"testing"

	"payment-sim/internal/domain"
)

func TestMemoryStore_SaveAndGet(t *testing.T) {
	store := NewMemoryStore()
	amount := big.NewRat(100, 1)
	payment := domain.NewPayment("P001", amount, "USD", "M001")

	// Save
	err := store.Save(payment)
	if err != nil {
		t.Errorf("Save() error = %v", err)
	}

	// Get
	got, err := store.Get("P001")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if got.ID != "P001" {
		t.Errorf("Get() ID = %v, want P001", got.ID)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.Get("NONEXISTENT")
	if err != domain.ErrPaymentNotFound {
		t.Errorf("Get() error = %v, want ErrPaymentNotFound", err)
	}
}

func TestMemoryStore_List(t *testing.T) {
	store := NewMemoryStore()
	amount := big.NewRat(100, 1)

	// Add payments in non-sorted order
	store.Save(domain.NewPayment("P003", amount, "USD", "M001"))
	store.Save(domain.NewPayment("P001", amount, "USD", "M001"))
	store.Save(domain.NewPayment("P002", amount, "USD", "M001"))

	list, err := store.List()
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("List() length = %v, want 3", len(list))
	}

	// Check sorted order
	expected := []string{"P001", "P002", "P003"}
	for i, p := range list {
		if p.ID != expected[i] {
			t.Errorf("List()[%d].ID = %v, want %v", i, p.ID, expected[i])
		}
	}
}

func TestMemoryStore_Exists(t *testing.T) {
	store := NewMemoryStore()
	amount := big.NewRat(100, 1)
	store.Save(domain.NewPayment("P001", amount, "USD", "M001"))

	if !store.Exists("P001") {
		t.Error("Exists() = false, want true")
	}
	if store.Exists("P002") {
		t.Error("Exists() = true, want false")
	}
}

func TestMemoryStore_BatchIDs(t *testing.T) {
	store := NewMemoryStore()

	store.RecordBatchID("BATCH001")
	store.RecordBatchID("BATCH002")

	if !store.BatchIDExists("BATCH001") {
		t.Error("BatchIDExists() = false, want true")
	}
	if store.BatchIDExists("BATCH003") {
		t.Error("BatchIDExists() = true, want false")
	}

	ids := store.GetBatchIDs()
	if len(ids) != 2 {
		t.Errorf("GetBatchIDs() length = %v, want 2", len(ids))
	}
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()
	amount := big.NewRat(100, 1)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			payment := domain.NewPayment(
				string(rune('A'+id%26))+string(rune('0'+id%10)),
				amount, "USD", "M001",
			)
			store.Save(payment)
		}(i)
	}
	wg.Wait()

	// Should not panic and should have stored some payments
	list, _ := store.List()
	if len(list) == 0 {
		t.Error("No payments stored after concurrent access")
	}
}

func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore()
	amount := big.NewRat(100, 1)
	payment := domain.NewPayment("P001", amount, "USD", "M001")
	store.Save(payment)

	// Update the payment
	payment.TransitionTo(domain.StateAuthorized, "AUTHORIZE", "")
	store.Save(payment)

	got, _ := store.Get("P001")
	if got.State != domain.StateAuthorized {
		t.Errorf("State = %v, want AUTHORIZED", got.State)
	}
}
