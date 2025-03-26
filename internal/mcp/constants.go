package mcp

// Access levels for users and teams
const (
	// AccessLevelEnvironmentAdmin represents the environment administrator access level
	AccessLevelEnvironmentAdmin = "environment_administrator"
	// AccessLevelHelpdeskUser represents the helpdesk user access level
	AccessLevelHelpdeskUser = "helpdesk_user"
	// AccessLevelStandardUser represents the standard user access level
	AccessLevelStandardUser = "standard_user"
	// AccessLevelReadonlyUser represents the readonly user access level
	AccessLevelReadonlyUser = "readonly_user"
	// AccessLevelOperatorUser represents the operator user access level
	AccessLevelOperatorUser = "operator_user"
)

// User roles
const (
	// UserRoleAdmin represents an admin user role
	UserRoleAdmin = "admin"
	// UserRoleUser represents a regular user role
	UserRoleUser = "user"
	// UserRoleEdgeAdmin represents an edge admin user role
	UserRoleEdgeAdmin = "edge_admin"
)

// MIME types
const (
	// MIMETypeJSON represents the JSON MIME type
	MIMETypeJSON = "application/json"
)

// Resource URIs
const (
	// ResourceURIEnvironments is the URI for environments resource
	ResourceURIEnvironments = "portainer://environments"
	// ResourceURIUsers is the URI for users resource
	ResourceURIUsers = "portainer://users"
	// ResourceURITeams is the URI for teams resource
	ResourceURITeams = "portainer://teams"
	// ResourceURIAccessGroups is the URI for access groups resource
	ResourceURIAccessGroups = "portainer://access-groups"
	// ResourceURITags is the URI for tags resource
	ResourceURITags = "portainer://tags"
	// ResourceURIStacks is the URI for stacks resource
	ResourceURIStacks = "portainer://stacks"
	// ResourceURISettings is the URI for settings resource
	ResourceURISettings = "portainer://settings"
)

// All available access levels
var AllAccessLevels = []string{
	AccessLevelEnvironmentAdmin,
	AccessLevelHelpdeskUser,
	AccessLevelStandardUser,
	AccessLevelReadonlyUser,
	AccessLevelOperatorUser,
}

// All available user roles
var AllUserRoles = []string{
	UserRoleAdmin,
	UserRoleUser,
	UserRoleEdgeAdmin,
}

// IsValidAccessLevel checks if a given string is a valid access level
func IsValidAccessLevel(access string) bool {
	for _, level := range AllAccessLevels {
		if access == level {
			return true
		}
	}
	return false
}

// IsValidUserRole checks if a given string is a valid user role
func IsValidUserRole(role string) bool {
	for _, r := range AllUserRoles {
		if role == r {
			return true
		}
	}
	return false
}