package postgres_test

import "testing"

func TestUserRepositoryCreate(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "Create",
			run:  checkPostgresUserRepo_Create,
		},
		{
			name: "CreateErrors",
			run:  checkPostgresUserRepo_CreateErrors,
		},
		{
			name: "CreateUnexpectedError",
			run:  checkPostgresUserRepo_CreateUnexpectedError,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestUserRepositoryGet(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "GetByEmail",
			run:  checkPostgresUserRepo_GetByEmail,
		},
		{
			name: "GetUser",
			run:  checkPostgresUserRepo_GetUser,
		},
		{
			name: "GetErrors",
			run:  checkPostgresUserRepo_GetErrors,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}
