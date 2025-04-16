package mcp

import "slices"

// Tool names as defined in the YAML file
const (
	ToolCreateEnvironmentGroup             = "createEnvironmentGroup"
	ToolListEnvironmentGroups              = "listEnvironmentGroups"
	ToolUpdateEnvironmentGroup             = "updateEnvironmentGroup"
	ToolCreateAccessGroup                  = "createAccessGroup"
	ToolListAccessGroups                   = "listAccessGroups"
	ToolUpdateAccessGroup                  = "updateAccessGroup"
	ToolAddEnvironmentToAccessGroup        = "addEnvironmentToAccessGroup"
	ToolRemoveEnvironmentFromAccessGroup   = "removeEnvironmentFromAccessGroup"
	ToolListEnvironments                   = "listEnvironments"
	ToolUpdateEnvironment                  = "updateEnvironment"
	ToolGetStackFile                       = "getStackFile"
	ToolCreateStack                        = "createStack"
	ToolListStacks                         = "listStacks"
	ToolUpdateStack                        = "updateStack"
	ToolCreateEnvironmentTag               = "createEnvironmentTag"
	ToolListEnvironmentTags                = "listEnvironmentTags"
	ToolCreateTeam                         = "createTeam"
	ToolListTeams                          = "listTeams"
	ToolUpdateTeamName                     = "updateTeamName"
	ToolUpdateTeamMembers                  = "updateTeamMembers"
	ToolListUsers                          = "listUsers"
	ToolUpdateUserRole                     = "updateUserRole"
	ToolGetSettings                        = "getSettings"
	ToolUpdateAccessGroupName              = "updateAccessGroupName"
	ToolUpdateAccessGroupUserAccesses      = "updateAccessGroupUserAccesses"
	ToolUpdateAccessGroupTeamAccesses      = "updateAccessGroupTeamAccesses"
	ToolUpdateEnvironmentTags              = "updateEnvironmentTags"
	ToolUpdateEnvironmentUserAccesses      = "updateEnvironmentUserAccesses"
	ToolUpdateEnvironmentTeamAccesses      = "updateEnvironmentTeamAccesses"
	ToolUpdateEnvironmentGroupName         = "updateEnvironmentGroupName"
	ToolUpdateEnvironmentGroupEnvironments = "updateEnvironmentGroupEnvironments"
	ToolUpdateEnvironmentGroupTags         = "updateEnvironmentGroupTags"
	ToolDockerProxy                        = "dockerProxy"
	ToolKubernetesProxy                    = "kubernetesProxy"
)

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

// isValidAccessLevel checks if a given string is a valid access level
func isValidAccessLevel(access string) bool {
	return slices.Contains(AllAccessLevels, access)
}

// isValidUserRole checks if a given string is a valid user role
func isValidUserRole(role string) bool {
	return slices.Contains(AllUserRoles, role)
}
