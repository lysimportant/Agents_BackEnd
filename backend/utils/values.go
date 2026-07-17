package utils

import (
	"strings"

	"collector-backend/models"
	"collector-backend/permissions"
)

func ParseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func SanitizeFileName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '_'
		}
	}, name)
	if name == "" {
		return "file"
	}
	if len(name) > 40 {
		return name[:40]
	}
	return name
}

func IsAdmin(user models.User) bool {
	return permissions.IsAdministratorRoleCode(user.RoleCode)
}

func IsSuperAdmin(user models.User) bool {
	return permissions.IsSuperAdminRoleCode(user.RoleCode)
}
