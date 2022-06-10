package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/evergreen-ci/evergreen"
	gql "github.com/evergreen-ci/evergreen/graphql"
	gqlModel "github.com/evergreen-ci/evergreen/graphql/model"
	"github.com/evergreen-ci/evergreen/model"
	"github.com/evergreen-ci/evergreen/model/build"
	"github.com/evergreen-ci/evergreen/model/event"
	"github.com/evergreen-ci/evergreen/model/task"
	"github.com/evergreen-ci/evergreen/model/user"
	"github.com/evergreen-ci/evergreen/rest/data"
	restModel "github.com/evergreen-ci/evergreen/rest/model"
	"github.com/evergreen-ci/gimlet"
	"github.com/evergreen-ci/utility"
)

func (r *mutationResolver) ClearMySubscriptions(ctx context.Context) (int, error) {
	usr := gql.MustHaveUser(ctx)
	username := usr.Username()
	subs, err := event.FindSubscriptionsByOwner(username, event.OwnerTypePerson)
	if err != nil {
		return 0, gql.InternalServerError.Send(ctx, fmt.Sprintf("Error retrieving subscriptions %s", err.Error()))
	}
	subIDs := gql.RemoveGeneralSubscriptions(usr, subs)
	err = data.DeleteSubscriptions(username, subIDs)
	if err != nil {
		return 0, gql.InternalServerError.Send(ctx, fmt.Sprintf("Error deleting subscriptions %s", err.Error()))
	}
	return len(subIDs), nil
}

func (r *mutationResolver) CreatePublicKey(ctx context.Context, publicKeyInput gqlModel.PublicKeyInput) ([]*restModel.APIPubKey, error) {
	err := gql.SavePublicKey(ctx, publicKeyInput)
	if err != nil {
		return nil, err
	}
	myPublicKeys := gql.GetMyPublicKeys(ctx)
	return myPublicKeys, nil
}

func (r *mutationResolver) RemovePublicKey(ctx context.Context, keyName string) ([]*restModel.APIPubKey, error) {
	if !gql.DoesPublicKeyNameAlreadyExist(ctx, keyName) {
		return nil, gql.InputValidationError.Send(ctx, fmt.Sprintf("Error deleting public key. Provided key name, %s, does not exist.", keyName))
	}
	err := gql.MustHaveUser(ctx).DeletePublicKey(keyName)
	if err != nil {
		return nil, gql.InternalServerError.Send(ctx, fmt.Sprintf("Error deleting public key: %s", err.Error()))
	}
	myPublicKeys := gql.GetMyPublicKeys(ctx)
	return myPublicKeys, nil
}

func (r *mutationResolver) SaveSubscription(ctx context.Context, subscription restModel.APISubscription) (bool, error) {
	usr := gql.MustHaveUser(ctx)
	username := usr.Username()
	idType, id, err := gql.GetResourceTypeAndIdFromSubscriptionSelectors(ctx, subscription.Selectors)
	if err != nil {
		return false, err
	}
	switch idType {
	case "task":
		t, taskErr := task.FindOneId(id)
		if taskErr != nil {
			return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("error finding task by id %s: %s", id, taskErr.Error()))
		}
		if t == nil {
			return false, gql.ResourceNotFound.Send(ctx, fmt.Sprintf("cannot find task with id %s", id))
		}
	case "build":
		b, buildErr := build.FindOneId(id)
		if buildErr != nil {
			return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("error finding build by id %s: %s", id, buildErr.Error()))
		}
		if b == nil {
			return false, gql.ResourceNotFound.Send(ctx, fmt.Sprintf("cannot find build with id %s", id))
		}
	case "version":
		v, versionErr := model.VersionFindOneId(id)
		if versionErr != nil {
			return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("error finding version by id %s: %s", id, versionErr.Error()))
		}
		if v == nil {
			return false, gql.ResourceNotFound.Send(ctx, fmt.Sprintf("cannot find version with id %s", id))
		}
	case "project":
		p, projectErr := data.FindProjectById(id, false, false)
		if projectErr != nil {
			return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("error finding project by id %s: %s", id, projectErr.Error()))
		}
		if p == nil {
			return false, gql.ResourceNotFound.Send(ctx, fmt.Sprintf("cannot find project with id %s", id))
		}
	default:
		return false, gql.InputValidationError.Send(ctx, "Selectors do not indicate a target version, build, project, or task ID")
	}
	err = data.SaveSubscriptions(username, []restModel.APISubscription{subscription}, false)
	if err != nil {
		return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("error saving subscription: %s", err.Error()))
	}
	return true, nil
}

func (r *mutationResolver) UpdatePublicKey(ctx context.Context, targetKeyName string, updateInfo gqlModel.PublicKeyInput) ([]*restModel.APIPubKey, error) {
	if !gql.DoesPublicKeyNameAlreadyExist(ctx, targetKeyName) {
		return nil, gql.InputValidationError.Send(ctx, fmt.Sprintf("Error updating public key. The target key name, %s, does not exist.", targetKeyName))
	}
	if updateInfo.Name != targetKeyName && gql.DoesPublicKeyNameAlreadyExist(ctx, updateInfo.Name) {
		return nil, gql.InputValidationError.Send(ctx, fmt.Sprintf("Error updating public key. The updated key name, %s, already exists.", targetKeyName))
	}
	err := gql.VerifyPublicKey(ctx, updateInfo)
	if err != nil {
		return nil, err
	}
	usr := gql.MustHaveUser(ctx)
	err = usr.UpdatePublicKey(targetKeyName, updateInfo.Name, updateInfo.Key)
	if err != nil {
		return nil, gql.InternalServerError.Send(ctx, fmt.Sprintf("Error updating public key, %s: %s", targetKeyName, err.Error()))
	}
	myPublicKeys := gql.GetMyPublicKeys(ctx)
	return myPublicKeys, nil
}

func (r *mutationResolver) UpdateUserSettings(ctx context.Context, userSettings *restModel.APIUserSettings) (bool, error) {
	usr := gql.MustHaveUser(ctx)

	updatedUserSettings, err := restModel.UpdateUserSettings(ctx, usr, *userSettings)
	if err != nil {
		return false, gql.InternalServerError.Send(ctx, err.Error())
	}
	err = data.UpdateSettings(usr, *updatedUserSettings)
	if err != nil {
		return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("Error saving userSettings : %s", err.Error()))
	}
	return true, nil
}

func (r *permissionsResolver) CanCreateProject(ctx context.Context, obj *gqlModel.Permissions) (bool, error) {
	usr, err := user.FindOneById(obj.UserID)
	if err != nil {
		return false, gql.ResourceNotFound.Send(ctx, "user not found")
	}
	return usr.HasPermission(gimlet.PermissionOpts{
		Resource:      evergreen.SuperUserPermissionsID,
		ResourceType:  evergreen.SuperUserResourceType,
		Permission:    evergreen.PermissionProjectCreate,
		RequiredLevel: evergreen.ProjectCreate.Value,
	}), nil
}

func (r *queryResolver) MyPublicKeys(ctx context.Context) ([]*restModel.APIPubKey, error) {
	publicKeys := gql.GetMyPublicKeys(ctx)
	return publicKeys, nil
}

func (r *queryResolver) User(ctx context.Context, userID *string) (*restModel.APIDBUser, error) {
	usr := gql.MustHaveUser(ctx)
	var err error
	if userID != nil {
		usr, err = user.FindOneById(*userID)
		if err != nil {
			return nil, gql.ResourceNotFound.Send(ctx, fmt.Sprintf("Error getting user from user ID: %s", err.Error()))
		}
		if usr == nil {
			return nil, gql.ResourceNotFound.Send(ctx, "Could not find user from user ID")
		}
	}
	displayName := usr.DisplayName()
	username := usr.Username()
	email := usr.Email()
	user := restModel.APIDBUser{
		DisplayName:  &displayName,
		UserID:       &username,
		EmailAddress: &email,
	}
	return &user, nil
}

func (r *queryResolver) UserConfig(ctx context.Context) (*gqlModel.UserConfig, error) {
	usr := gql.MustHaveUser(ctx)
	settings := evergreen.GetEnvironment().Settings()
	config := &gqlModel.UserConfig{
		User:          usr.Username(),
		APIKey:        usr.GetAPIKey(),
		UIServerHost:  settings.Ui.Url,
		APIServerHost: settings.ApiUrl + "/api",
	}
	return config, nil
}

func (r *queryResolver) UserSettings(ctx context.Context) (*restModel.APIUserSettings, error) {
	usr := gql.MustHaveUser(ctx)
	userSettings := restModel.APIUserSettings{}
	err := userSettings.BuildFromService(usr.Settings)
	if err != nil {
		return nil, gql.InternalServerError.Send(ctx, err.Error())
	}
	return &userSettings, nil
}

func (r *userResolver) Permissions(ctx context.Context, obj *restModel.APIDBUser) (*gqlModel.Permissions, error) {
	return &gqlModel.Permissions{UserID: utility.FromStringPtr(obj.UserID)}, nil
}

// Permissions returns gql.PermissionsResolver implementation.
func (r *Resolver) Permissions() gql.PermissionsResolver { return &permissionsResolver{r} }

// User returns gql.UserResolver implementation.
func (r *Resolver) User() gql.UserResolver { return &userResolver{r} }

type permissionsResolver struct{ *Resolver }
type userResolver struct{ *Resolver }
