package model

import (
	"context"

	"github.com/evergreen-ci/evergreen/db"
	"github.com/evergreen-ci/evergreen/db/mgo/bson"
	"github.com/evergreen-ci/evergreen/model/patch"
	"github.com/evergreen-ci/evergreen/util"
	"github.com/mongodb/anser/bsonutil"
	adb "github.com/mongodb/anser/db"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

const (
	ParserProjectCollection = "parser_projects"
)

var (
	ParserProjectIdKey                = bsonutil.MustHaveTag(ParserProject{}, "Id")
	ParserProjectEnabledKey           = bsonutil.MustHaveTag(ParserProject{}, "Enabled")
	ParserProjectStepbackKey          = bsonutil.MustHaveTag(ParserProject{}, "Stepback")
	ParserProjectPreErrorFailsTaskKey = bsonutil.MustHaveTag(ParserProject{}, "PreErrorFailsTask")
	ParserProjectOomTracker           = bsonutil.MustHaveTag(ParserProject{}, "OomTracker")
	ParserProjectBatchTimeKey         = bsonutil.MustHaveTag(ParserProject{}, "BatchTime")
	ParserProjectOwnerKey             = bsonutil.MustHaveTag(ParserProject{}, "Owner")
	ParserProjectRepoKey              = bsonutil.MustHaveTag(ParserProject{}, "Repo")
	ParserProjectRemotePathKey        = bsonutil.MustHaveTag(ParserProject{}, "RemotePath")
	ParserProjectBranchKey            = bsonutil.MustHaveTag(ParserProject{}, "Branch")
	ParserProjectIdentifierKey        = bsonutil.MustHaveTag(ParserProject{}, "Identifier")
	ParserProjectDisplayNameKey       = bsonutil.MustHaveTag(ParserProject{}, "DisplayName")
	ParserProjectCommandTypeKey       = bsonutil.MustHaveTag(ParserProject{}, "CommandType")
	ParserProjectIgnoreKey            = bsonutil.MustHaveTag(ParserProject{}, "Ignore")
	ParserProjectParametersKey        = bsonutil.MustHaveTag(ParserProject{}, "Parameters")
	ParserProjectPreKey               = bsonutil.MustHaveTag(ParserProject{}, "Pre")
	ParserProjectPostKey              = bsonutil.MustHaveTag(ParserProject{}, "Post")
	ParserProjectEarlyTerminationKey  = bsonutil.MustHaveTag(ParserProject{}, "EarlyTermination")
	ParserProjectTimeoutKey           = bsonutil.MustHaveTag(ParserProject{}, "Timeout")
	ParserProjectCallbackTimeoutKey   = bsonutil.MustHaveTag(ParserProject{}, "CallbackTimeout")
	ParserProjectModulesKey           = bsonutil.MustHaveTag(ParserProject{}, "Modules")
	ParserProjectContainersKey        = bsonutil.MustHaveTag(ParserProject{}, "Containers")
	ParserProjectBuildVariantsKey     = bsonutil.MustHaveTag(ParserProject{}, "BuildVariants")
	ParserProjectFunctionsKey         = bsonutil.MustHaveTag(ParserProject{}, "Functions")
	ParserProjectTaskGroupsKey        = bsonutil.MustHaveTag(ParserProject{}, "TaskGroups")
	ParserProjectTasksKey             = bsonutil.MustHaveTag(ParserProject{}, "Tasks")
	ParserProjectExecTimeoutSecsKey   = bsonutil.MustHaveTag(ParserProject{}, "ExecTimeoutSecs")
	ParserProjectLoggersKey           = bsonutil.MustHaveTag(ParserProject{}, "Loggers")
	ParserProjectAxesKey              = bsonutil.MustHaveTag(ParserProject{}, "Axes")
	ParserProjectCreateTimeKey        = bsonutil.MustHaveTag(ParserProject{}, "CreateTime")
)

// ParserProjectFindOneById returns the parser project from the DB for the
// given ID.
func parserProjectFindOneById(id string) (*ParserProject, error) {
	pp, err := parserProjectFindOne(parserProjectById(id))
	if err != nil {
		return nil, err
	}
	if pp != nil && pp.Functions == nil {
		pp.Functions = map[string]*YAMLCommandSet{}
	}
	return pp, nil
}

// parserProjectFindOne finds a parser project in the DB with a given query.
func parserProjectFindOne(query db.Q) (*ParserProject, error) {
	project := &ParserProject{}
	err := db.FindOneQ(ParserProjectCollection, query, project)
	if adb.ResultsNotFound(err) {
		return nil, nil
	}
	return project, err
}

// parserProjectById returns a DB query to find a parser project by ID.
func parserProjectById(id string) db.Q {
	return db.Query(bson.M{ParserProjectIdKey: id})
}

// parserProjectUpsertOne updates one parser project in the DB.
func parserProjectUpsertOne(query interface{}, update interface{}) error {
	_, err := db.Upsert(
		ParserProjectCollection,
		query,
		update,
	)

	return err
}

func FindParametersForVersion(ctx context.Context, v *Version) ([]patch.Parameter, error) {
	pp, err := GetParserProjectStorage(v.ProjectStorageMethod).FindOneByIDWithFields(ctx, v.Id, ParserProjectParametersKey)
	if err != nil {
		return nil, errors.Wrap(err, "finding parser project")
	}
	return pp.GetParameters(), nil
}

func FindExpansionsForVariant(ctx context.Context, v *Version, variant string) (util.Expansions, error) {
	pp, err := GetParserProjectStorage(v.ProjectStorageMethod).FindOneByIDWithFields(ctx, v.Id, ParserProjectBuildVariantsKey, ParserProjectAxesKey)
	if err != nil {
		return nil, errors.Wrap(err, "finding parser project")
	}

	bvs, errs := GetVariantsWithMatrices(nil, pp.Axes, pp.BuildVariants)
	if len(errs) > 0 {
		catcher := grip.NewBasicCatcher()
		catcher.Extend(errs)
		return nil, catcher.Resolve()
	}
	for _, bv := range bvs {
		if bv.Name == variant {
			return bv.Expansions, nil
		}
	}
	return nil, errors.New("could not find variant")
}

// ParserProjectDBStorage implements the ParserProjectStorage interface to
// access parser projects stored in the DB.
type ParserProjectDBStorage struct{}

// FindOneByID finds a parser project from the DB by its ID. This ignores the
// context parameter.
func (s ParserProjectDBStorage) FindOneByID(_ context.Context, id string) (*ParserProject, error) {
	return parserProjectFindOneById(id)
}

// FindOneByIDWithFields returns the parser project from the DB with only the
// requested fields populated. This may be more efficient than fetching the
// entire parser project. This ignores the context parameter.
func (s ParserProjectDBStorage) FindOneByIDWithFields(_ context.Context, id string, fields ...string) (*ParserProject, error) {
	return parserProjectFindOne(parserProjectById(id).WithFields(fields...))
}

// UpsertOne replaces a parser project in the DB if one exists with the same ID.
// Otherwise, if it does not exist yet, it inserts a new parser project.
func (s ParserProjectDBStorage) UpsertOne(ctx context.Context, pp *ParserProject) error {
	return parserProjectUpsertOne(bson.M{ParserProjectIdKey: pp.Id}, pp)
}
