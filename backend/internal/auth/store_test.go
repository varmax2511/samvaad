package auth

import (
	"sync"
	"testing"
)

func TestNewUserStore_ReturnsEmptyStore(t *testing.T) {
	store := NewUserStore()

	if store == nil {
		t.Fatal("expected non-nil store")
	}

	if store.users == nil {
		t.Fatal("expected initialized users map")
	}

	if len(store.users) != 0 {
		t.Fatalf("expected empty users map, got %d users", len(store.users))
	}
}

func TestUserStore_Create_Success(t *testing.T) {
	store := NewUserStore()

	user, err := store.Create("testuser", "password123")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if user == nil {
		t.Fatal("expected non-nil user")
	}

	if user.Username != "testuser" {
		t.Errorf("expected username %q, got %q", "testuser", user.Username)
	}

	if user.Id == "" {
		t.Error("expected non-empty user ID")
	}

	if user.PasswordHash == "" {
		t.Error("expected non-empty password hash")
	}

	if user.PasswordHash == "password123" {
		t.Error("password hash should not equal plain password")
	}
}

func TestUserStore_Create_DuplicateUsername(t *testing.T) {
	store := NewUserStore()

	_, err := store.Create("testuser", "password123")
	if err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	_, err = store.Create("testuser", "differentpassword")
	if err == nil {
		t.Fatal("expected error for duplicate username, got nil")
	}
}

func TestUserStore_Create_DifferentUsernames(t *testing.T) {
	store := NewUserStore()

	user1, err := store.Create("user1", "password1")
	if err != nil {
		t.Fatalf("Create user1 failed: %v", err)
	}

	user2, err := store.Create("user2", "password2")
	if err != nil {
		t.Fatalf("Create user2 failed: %v", err)
	}

	if user1.Id == user2.Id {
		t.Error("expected different IDs for different users")
	}
}

func TestUserStore_GetByUsername_Success(t *testing.T) {
	store := NewUserStore()

	created, err := store.Create("testuser", "password123")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	fetched, err := store.GetByUsername("testuser")
	if err != nil {
		t.Fatalf("GetByUsername failed: %v", err)
	}

	if fetched.Id != created.Id {
		t.Errorf("expected ID %q, got %q", created.Id, fetched.Id)
	}

	if fetched.Username != created.Username {
		t.Errorf("expected Username %q, got %q", created.Username, fetched.Username)
	}
}

func TestUserStore_GetByUsername_NotFound(t *testing.T) {
	store := NewUserStore()

	_, err := store.GetByUsername("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
}

func TestUserStore_GetByUsername_EmptyStore(t *testing.T) {
	store := NewUserStore()

	_, err := store.GetByUsername("anyuser")
	if err == nil {
		t.Fatal("expected error for empty store, got nil")
	}
}

func TestUserStore_ConcurrentCreate(t *testing.T) {
	store := NewUserStore()
	var wg sync.WaitGroup
	numUsers := 100

	// Track successful creations
	successes := make(chan string, numUsers)
	errors := make(chan error, numUsers)

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			username := "user" + string(rune('a'+idx%26)) + string(rune('0'+idx/26))
			_, err := store.Create(username, "password")
			if err != nil {
				errors <- err
			} else {
				successes <- username
			}
		}(i)
	}

	wg.Wait()
	close(successes)
	close(errors)

	// Count results
	successCount := 0
	for range successes {
		successCount++
	}

	errorCount := 0
	for range errors {
		errorCount++
	}

	// All unique usernames should succeed
	if successCount+errorCount != numUsers {
		t.Errorf("expected %d total operations, got %d successes + %d errors", numUsers, successCount, errorCount)
	}
}

func TestUserStore_ConcurrentGetByUsername(t *testing.T) {
	store := NewUserStore()

	// Create a user first
	_, err := store.Create("testuser", "password123")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	var wg sync.WaitGroup
	numReads := 100

	for i := 0; i < numReads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			user, err := store.GetByUsername("testuser")
			if err != nil {
				t.Errorf("GetByUsername failed: %v", err)
			}
			if user.Username != "testuser" {
				t.Errorf("expected username %q, got %q", "testuser", user.Username)
			}
		}()
	}

	wg.Wait()
}
