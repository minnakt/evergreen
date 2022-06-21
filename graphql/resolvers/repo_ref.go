package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/evergreen-ci/evergreen/graphql/generated"
	"github.com/evergreen-ci/evergreen/model"
	restModel "github.com/evergreen-ci/evergreen/rest/model"
)

func (r *repoRefResolver) ValidDefaultLoggers(ctx context.Context, obj *restModel.APIProjectRef) ([]string, error) {
	return model.ValidDefaultLoggers, nil
}

// RepoRef returns generated.RepoRefResolver implementation.
func (r *Resolver) RepoRef() generated.RepoRefResolver { return &repoRefResolver{r} }

type repoRefResolver struct{ *Resolver }