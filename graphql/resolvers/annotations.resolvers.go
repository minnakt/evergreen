package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"strconv"

	gql "github.com/evergreen-ci/evergreen/graphql"
	gqlModel "github.com/evergreen-ci/evergreen/graphql/model"
	"github.com/evergreen-ci/evergreen/model"
	"github.com/evergreen-ci/evergreen/model/annotations"
	"github.com/evergreen-ci/evergreen/model/task"
	restModel "github.com/evergreen-ci/evergreen/rest/model"
	"github.com/evergreen-ci/evergreen/thirdparty"
	"github.com/evergreen-ci/evergreen/util"
)

func (r *annotationResolver) WebhookConfigured(ctx context.Context, obj *restModel.APITaskAnnotation) (bool, error) {
	t, err := task.FindOneId(*obj.TaskId)
	if err != nil {
		return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("error finding task: %s", err.Error()))
	}
	if t == nil {
		return false, gql.ResourceNotFound.Send(ctx, "error finding task for the task annotation")
	}
	_, ok, _ := model.IsWebhookConfigured(t.Project, t.Version)
	return ok, nil
}

func (r *issueLinkResolver) JiraTicket(ctx context.Context, obj *restModel.APIIssueLink) (*thirdparty.JiraTicket, error) {
	return restModel.GetJiraTicketFromURL(*obj.URL)
}

// AddAnnotationIssue adds to the annotation for that taskID/execution.
// If isIssue is set, it adds to Issues, otherwise it adds to Suspected Issues.
func (r *mutationResolver) AddAnnotationIssue(ctx context.Context, taskID string, execution int, apiIssue restModel.APIIssueLink, isIssue bool) (bool, error) {
	usr := gql.MustHaveUser(ctx)
	issue := restModel.APIIssueLinkToService(apiIssue)
	if err := util.CheckURL(issue.URL); err != nil {
		return false, gql.InputValidationError.Send(ctx, fmt.Sprintf("issue does not have valid URL: %s", err.Error()))
	}
	if isIssue {
		if err := annotations.AddIssueToAnnotation(taskID, execution, *issue, usr.Username()); err != nil {
			return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("couldn't add issue: %s", err.Error()))
		}
		return true, nil
	} else {
		if err := annotations.AddSuspectedIssueToAnnotation(taskID, execution, *issue, usr.Username()); err != nil {
			return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("couldn't add suspected issue: %s", err.Error()))
		}
		return true, nil
	}
}

// EditAnnotationNote updates the note for the annotation, assuming it hasn't been updated in the meantime.
func (r *mutationResolver) EditAnnotationNote(ctx context.Context, taskID string, execution int, originalMessage string, newMessage string) (bool, error) {
	usr := gql.MustHaveUser(ctx)
	if err := annotations.UpdateAnnotationNote(taskID, execution, originalMessage, newMessage, usr.Username()); err != nil {
		return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("couldn't update note: %s", err.Error()))
	}
	return true, nil
}

// MoveAnnotationIssue moves an issue for the annotation. If isIssue is set, it removes the issue from Issues and adds it
// to Suspected Issues, otherwise vice versa.
func (r *mutationResolver) MoveAnnotationIssue(ctx context.Context, taskID string, execution int, apiIssue restModel.APIIssueLink, isIssue bool) (bool, error) {
	usr := gql.MustHaveUser(ctx)
	issue := restModel.APIIssueLinkToService(apiIssue)
	if isIssue {
		if err := annotations.MoveIssueToSuspectedIssue(taskID, execution, *issue, usr.Username()); err != nil {
			return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("couldn't move issue to suspected issues: %s", err.Error()))
		}
		return true, nil
	} else {
		if err := annotations.MoveSuspectedIssueToIssue(taskID, execution, *issue, usr.Username()); err != nil {
			return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("couldn't move issue to suspected issues: %s", err.Error()))
		}
		return true, nil
	}
}

// RemoveAnnotationIssue adds to the annotation for that taskID/execution.
// If isIssue is set, it adds to Issues, otherwise it adds to Suspected Issues.
func (r *mutationResolver) RemoveAnnotationIssue(ctx context.Context, taskID string, execution int, apiIssue restModel.APIIssueLink, isIssue bool) (bool, error) {
	issue := restModel.APIIssueLinkToService(apiIssue)
	if isIssue {
		if err := annotations.RemoveIssueFromAnnotation(taskID, execution, *issue); err != nil {
			return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("couldn't delete issue: %s", err.Error()))
		}
		return true, nil
	} else {
		if err := annotations.RemoveSuspectedIssueFromAnnotation(taskID, execution, *issue); err != nil {
			return false, gql.InternalServerError.Send(ctx, fmt.Sprintf("couldn't delete suspected issue: %s", err.Error()))
		}
		return true, nil
	}
}

func (r *queryResolver) BbGetCreatedTickets(ctx context.Context, taskID string) ([]*thirdparty.JiraTicket, error) {
	createdTickets, err := gql.BBGetCreatedTicketsPointers(taskID)
	if err != nil {
		return nil, err
	}

	return createdTickets, nil
}

func (r *queryResolver) BuildBaron(ctx context.Context, taskID string, execution int) (*gqlModel.BuildBaron, error) {
	execString := strconv.Itoa(execution)

	searchReturnInfo, bbConfig, err := model.GetSearchReturnInfo(taskID, execString)
	if err != nil {
		return nil, gql.InternalServerError.Send(ctx, err.Error())
	}

	return &gqlModel.BuildBaron{
		SearchReturnInfo:        searchReturnInfo,
		BuildBaronConfigured:    bbConfig.ProjectFound && bbConfig.SearchConfigured,
		BbTicketCreationDefined: bbConfig.TicketCreationDefined,
	}, nil
}

func (r *ticketFieldsResolver) AssignedTeam(ctx context.Context, obj *thirdparty.TicketFields) (*string, error) {
	if obj.AssignedTeam == nil {
		return nil, nil
	}
	if len(obj.AssignedTeam) != 0 {
		return &obj.AssignedTeam[0].Value, nil
	}
	return nil, nil
}

func (r *ticketFieldsResolver) AssigneeDisplayName(ctx context.Context, obj *thirdparty.TicketFields) (*string, error) {
	if obj.Assignee == nil {
		return nil, nil
	}
	return &obj.Assignee.DisplayName, nil
}

func (r *ticketFieldsResolver) ResolutionName(ctx context.Context, obj *thirdparty.TicketFields) (*string, error) {
	if obj.Resolution == nil {
		return nil, nil
	}
	return &obj.Resolution.Name, nil
}

// Annotation returns AnnotationResolver implementation.
func (r *Resolver) Annotation() gql.AnnotationResolver { return &annotationResolver{r} }

// IssueLink returns IssueLinkResolver implementation.
func (r *Resolver) IssueLink() gql.IssueLinkResolver { return &issueLinkResolver{r} }

// TicketFields returns graphql1.TicketFieldsResolver implementation.
func (r *Resolver) TicketFields() gql.TicketFieldsResolver { return &ticketFieldsResolver{r} }

type annotationResolver struct{ *Resolver }
type issueLinkResolver struct{ *Resolver }
type ticketFieldsResolver struct{ *Resolver }
