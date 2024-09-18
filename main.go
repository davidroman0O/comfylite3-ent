package main

import (
	"context"
	"fmt"
	"log"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/davidroman0O/comfylite3"
	"github.com/davidroman0O/comfylite3-ent/ent"
	"github.com/davidroman0O/comfylite3-ent/ent/user"
)

func main() {

	comfy, err := comfylite3.New(
		comfylite3.WithPath("./ent.db"),
	)
	if err != nil {
		log.Fatalf("failed creating ComfyDB: %v", err)
	}
	defer comfy.Close()

	db := comfylite3.OpenDB(
		comfy,
		comfylite3.WithOption("_fk=1"),
		comfylite3.WithOption("cache=shared"),
		comfylite3.WithOption("mode=rwc"),
		comfylite3.WithForeignKeys(),
	)

	client := ent.NewClient(ent.Driver(sql.OpenDB(dialect.SQLite, db)))
	defer client.Close()

	ctx := context.Background()

	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	// Create (Insert new users)
	users, err := client.User.CreateBulk(
		client.User.Create().SetName("Alice").SetAge(30).SetEmail("alice@example.com"),
		client.User.Create().SetName("Bob").SetAge(32).SetEmail("bob@example.com"),
		client.User.Create().SetName("Charlie").SetAge(35).SetEmail("charlie@example.com"),
	).Save(ctx)
	if err != nil {
		log.Fatalf("failed creating users: %v", err)
	}
	fmt.Printf("Created %d users\n", len(users))

	// Read (Query users)
	filteredUsers, err := client.User.Query().
		Where(
			user.AgeGT(30),
			user.NameContains("a"),
		).
		Order(ent.Desc(user.FieldAge)).
		All(ctx)
	if err != nil {
		log.Fatalf("failed querying users: %v", err)
	}
	fmt.Println("Filtered users:")
	for _, u := range filteredUsers {
		fmt.Printf("  User: %s, Age: %d\n", u.Name, u.Age)
	}

	// Update (Modify a user's details)
	updatedUser, err := client.User.UpdateOneID(users[0].ID).
		SetAge(31).
		SetEmail("alice_new@example.com").
		Save(ctx)
	if err != nil {
		log.Fatalf("failed updating user: %v", err)
	}
	fmt.Printf("Updated user: %s, New Age: %d, New Email: %s\n", updatedUser.Name, updatedUser.Age, updatedUser.Email)

	// Pagination (List users in pages)
	pageSize := 2
	for i := 0; ; i++ {
		pagedUsers, err := client.User.Query().
			Limit(pageSize).
			Offset(i * pageSize).
			All(ctx)
		if err != nil {
			log.Fatalf("failed querying users: %v", err)
		}
		if len(pagedUsers) == 0 {
			break
		}
		fmt.Printf("Page %d:\n", i+1)
		for _, u := range pagedUsers {
			fmt.Printf("  User: %s\n", u.Name)
		}
	}

	// Aggregation (Calculate average age)
	avgAge, err := client.User.Query().
		Aggregate(func(s *sql.Selector) string {
			return "AVG(age)"
		}).
		Float64(ctx)
	if err != nil {
		log.Fatalf("failed to calculate average age: %v", err)
	}
	fmt.Printf("Average age: %.2f\n", avgAge)

	// Transactions (Perform multiple operations)
	err = WithTx(ctx, client, func(tx *ent.Tx) error {
		// Create a new user within the transaction
		newUser, err := tx.User.Create().
			SetName("David").
			SetAge(28).
			SetEmail("david@example.com").
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed creating user: %w", err)
		}

		// Update another user within the transaction
		_, err = tx.User.UpdateOneID(users[1].ID).
			SetAge(33).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed updating user: %w", err)
		}

		fmt.Printf("Created user in transaction: %s\n", newUser.Name)
		return nil
	})
	if err != nil {
		log.Fatalf("failed to run transaction: %v", err)
	}

	// Delete (Remove a user)
	err = client.User.DeleteOneID(users[2].ID).Exec(ctx)
	if err != nil {
		log.Fatalf("failed deleting user: %v", err)
	}
	fmt.Printf("Deleted user: %s\n", users[2].Name)

	// Verify all users after operations
	allUsers, err := client.User.Query().All(ctx)
	if err != nil {
		log.Fatalf("failed querying all users: %v", err)
	}
	fmt.Println("All users after operations:")
	for _, u := range allUsers {
		fmt.Printf("  User: %s, Age: %d, Email: %s\n", u.Name, u.Age, u.Email)
	}
}

// WithTx wraps the given function in a transaction.
func WithTx(ctx context.Context, client *ent.Client, fn func(tx *ent.Tx) error) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if v := recover(); v != nil {
			tx.Rollback()
			panic(v)
		}
	}()
	if err := fn(tx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = fmt.Errorf("rolling back transaction: %w", rerr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}
