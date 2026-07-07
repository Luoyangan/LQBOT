// Package permission provides role-based access control for commands.
package permission

import (
	"github.com/Luoyangan/LQBOT/internal/contract"
	"github.com/Luoyangan/LQBOT/internal/log"
)

// Permission levels (ordered from most to least restrictive).
const (
	LevelOwner  = "owner"  // 群主
	LevelAdmin  = "admin"  // 群管理员及以上
	LevelMember = "member" // 所有群成员
	LevelPublic = "public" // 所有人（包括非群聊场景）
)

// levelRank maps permission level names to numeric rank (lower = more restrictive).
var levelRank = map[string]int{
	LevelOwner:  0,
	LevelAdmin:  1,
	LevelMember: 2,
	LevelPublic: 3,
}

// Checker checks command permissions against config mapping.
type Checker struct {
	logger      *log.Logger
	permissions map[string]string // command name → required permission level
}

// NewChecker creates a permission checker.
// permissions maps command names to required permission levels.
// Commands not in the map default to "public" (no restriction).
func NewChecker(permissions map[string]string, logger *log.Logger) *Checker {
	normalized := make(map[string]string, len(permissions))
	for cmd, level := range permissions {
		if _, ok := levelRank[level]; !ok {
			level = LevelPublic
		}
		normalized[cmd] = level
	}
	return &Checker{
		logger:      logger,
		permissions: normalized,
	}
}

// Check verifies that the user has permission to execute the command.
// Returns true if allowed, false if denied.
func (c *Checker) Check(cmd *contract.Command, ctx contract.EventContext) bool {
	required, exists := c.permissions[cmd.Name]
	if !exists {
		// Also check aliases
		for _, alias := range cmd.Aliases {
			if req, ok := c.permissions[alias]; ok {
				required = req
				exists = true
				break
			}
		}
	}
	if !exists {
		required = cmd.Permission
		if required == "" {
			return true // no restriction
		}
	}

	requiredRank, ok := levelRank[required]
	if !ok {
		return true // unknown level = no restriction
	}
	if requiredRank >= levelRank[LevelPublic] {
		return true // public = no restriction
	}

	userRole := ctx.Role()
	userRank, hasRole := levelRank[userRole]
	if !hasRole {
		// Non-group context (channel/C2C) — only allow public commands
		return requiredRank >= levelRank[LevelPublic]
	}

	if userRank > requiredRank {
		c.logger.Warn("permission denied",
			"command", cmd.Name,
			"user_role", userRole,
			"required", required,
			"user", ctx.AuthorID(),
		)
		return false
	}

	return true
}
