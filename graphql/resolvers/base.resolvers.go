package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/evergreen-ci/evergreen"
	gql "github.com/evergreen-ci/evergreen/graphql"
	gqlModel "github.com/evergreen-ci/evergreen/graphql/model"
	"github.com/evergreen-ci/evergreen/model"
	"github.com/evergreen-ci/evergreen/rest/data"
	"github.com/evergreen-ci/gimlet"
)

// Mutation returns graphql1.MutationResolver implementation.
func (r *Resolver) Mutation() gql.MutationResolver { return &mutationResolver{r} }

// Query returns graphql1.QueryResolver implementation.
func (r *Resolver) Query() gql.QueryResolver { return &queryResolver{r} }

type Resolver struct {
	sc data.Connector
}
type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

// New injects resources into the resolvers, such as the data connector
func New(apiURL string) gql.Config {
	c := gql.Config{
		Resolvers: &Resolver{
			sc: &data.DBConnector{URL: apiURL},
		},
	}
	c.Directives.RequireSuperUser = func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
		user := gimlet.GetUser(ctx)
		if user == nil {
			return nil, gql.Forbidden.Send(ctx, "user not logged in")
		}
		opts := gimlet.PermissionOpts{
			Resource:      evergreen.SuperUserPermissionsID,
			ResourceType:  evergreen.SuperUserResourceType,
			Permission:    evergreen.PermissionAdminSettings,
			RequiredLevel: evergreen.AdminSettingsEdit.Value,
		}
		if user.HasPermission(opts) {
			return next(ctx)
		}
		return nil, gql.Forbidden.Send(ctx, fmt.Sprintf("user %s does not have permission to access this resolver", user.Username()))
	}
	c.Directives.RequireProjectAccess = func(ctx context.Context, obj interface{}, next graphql.Resolver, access gqlModel.ProjectSettingsAccess) (res interface{}, err error) {
		var permissionLevel int
		if access == gqlModel.ProjectSettingsAccessEdit {
			permissionLevel = evergreen.ProjectSettingsEdit.Value
		} else if access == gqlModel.ProjectSettingsAccessView {
			permissionLevel = evergreen.ProjectSettingsView.Value
		} else {
			return nil, gql.Forbidden.Send(ctx, "Permission not specified")
		}

		args, isStringMap := obj.(map[string]interface{})
		if !isStringMap {
			return nil, gql.ResourceNotFound.Send(ctx, "Project not specified")
		}

		if id, hasId := args["id"].(string); hasId {
			return gql.HasProjectPermission(ctx, id, next, permissionLevel)
		} else if projectId, hasProjectId := args["projectId"].(string); hasProjectId {
			return gql.HasProjectPermission(ctx, projectId, next, permissionLevel)
		} else if identifier, hasIdentifier := args["identifier"].(string); hasIdentifier {
			pid, err := model.GetIdForProject(identifier)
			if err != nil {
				return nil, gql.ResourceNotFound.Send(ctx, fmt.Sprintf("Could not find project with identifier: %s", identifier))
			}
			return gql.HasProjectPermission(ctx, pid, next, permissionLevel)
		}
		return nil, gql.ResourceNotFound.Send(ctx, "Could not find project")
	}
	return c
}
