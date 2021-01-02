package access

import "testing"

func TestRole_String(t *testing.T) {
	tests := []struct {
		name string
		r    Role
		want string
	}{
		{"USER", RoleUser, "USER"},
		{"ADMIN", RoleAdmin, "ADMIN"},
		{"SUPER_ADMIN", RoleSuperAdmin, "SUPER_ADMIN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
