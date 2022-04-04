package event

import (
	"time"

	"github.com/evergreen-ci/evergreen/db"
	"github.com/mongodb/anser/bsonutil"
	adb "github.com/mongodb/anser/db"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
)

// unprocessedEvents returns a bson.M query to fetch all unprocessed events
func unprocessedEvents() bson.M {
	return bson.M{
		processedAtKey: bson.M{
			"$eq": time.Time{},
		},
	}
}

func ResourceTypeKeyIs(key string) bson.M {
	return bson.M{
		resourceTypeKey: key,
	}
}

// === DB Logic ===

// Find takes a collection storing events and a query, generated
// by one of the query functions, and returns a slice of events.
func Find(coll string, query db.Q) ([]EventLogEntry, error) {
	events := []EventLogEntry{}
	err := db.FindAllQ(coll, query, &events)
	if err != nil || adb.ResultsNotFound(err) {
		return nil, errors.WithStack(err)
	}

	return events, nil
}

func FindPaginated(hostID, hostTag, coll string, limit, page int) ([]EventLogEntry, int, error) {
	query := MostRecentHostEvents(hostID, hostTag, limit)
	events := []EventLogEntry{}
	skip := page * limit
	if skip > 0 {
		query = query.Skip(skip)
	}

	err := db.FindAllQ(coll, query, &events)
	if err != nil || adb.ResultsNotFound(err) {
		return nil, 0, errors.WithStack(err)
	}
	count, err := db.CountQ(coll, query)

	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to fetch number of host events")
	}

	return events, count, nil
}

// FindUnprocessedEvents returns all unprocessed events in AllLogCollection.
// Events are considered unprocessed if their "processed_at" time IsZero
func FindUnprocessedEvents(limit int) ([]EventLogEntry, error) {
	out := []EventLogEntry{}
	query := db.Query(unprocessedEvents()).Sort([]string{TimestampKey})
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := db.FindAllQ(AllLogCollection, query, &out)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch unprocessed events")
	}

	return out, nil
}

// FindByID finds a single event matching the given event ID.
func FindByID(eventID string) (*EventLogEntry, error) {
	query := bson.M{
		idKey: eventID,
	}

	var e EventLogEntry
	if err := db.FindOneQ(AllLogCollection, db.Query(query), &e); err != nil {
		if adb.ResultsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "finding event by ID")
	}
	return &e, nil
}

func FindLastProcessedEvent() (*EventLogEntry, error) {
	q := db.Query(bson.M{
		processedAtKey: bson.M{
			"$gt": time.Time{},
		},
	}).Sort([]string{"-" + processedAtKey})

	e := EventLogEntry{}
	if err := db.FindOneQ(AllLogCollection, q, &e); err != nil {
		if adb.ResultsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to fetch most recently processed event")
	}

	return &e, nil
}

func CountUnprocessedEvents() (int, error) {
	q := db.Query(unprocessedEvents())

	n, err := db.CountQ(AllLogCollection, q)
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch number of unprocessed events")
	}

	return n, nil
}

// === Queries ===

// Host Events
func MostRecentHostEvents(id string, tag string, n int) db.Q {
	filter := ResourceTypeKeyIs(ResourceTypeHost)
	if tag != "" {
		filter[ResourceIdKey] = bson.M{"$in": []string{id, tag}}
	} else {
		filter[ResourceIdKey] = id
	}

	return db.Query(filter).Sort([]string{"-" + TimestampKey}).Limit(n)
}

// Task Events
func TaskEventsForId(id string) db.Q {
	filter := ResourceTypeKeyIs(ResourceTypeTask)
	filter[ResourceIdKey] = id

	return db.Query(filter)
}

func MostRecentTaskEvents(id string, n int) db.Q {
	return TaskEventsForId(id).Sort([]string{"-" + TimestampKey}).Limit(n)
}

func TaskEventsInOrder(id string) db.Q {
	return TaskEventsForId(id).Sort([]string{TimestampKey})
}

// Distro Events

// FindLatestPrimaryDistroEvents return the most recent non-AMI events for the distro.
func FindLatestPrimaryDistroEvents(id string, n int) ([]EventLogEntry, error) {
	events := []EventLogEntry{}
	err := db.Aggregate(AllLogCollection, latestDistroEventsPipeline(id, n, false), &events)
	if err != nil {
		return nil, err
	}
	return events, err
}

// FindLatestAMIModifiedDistroEvent returns the most recent AMI event. Returns an empty struct if nothing exists.
func FindLatestAMIModifiedDistroEvent(id string) (EventLogEntry, error) {
	events := []EventLogEntry{}
	res := EventLogEntry{}
	err := db.Aggregate(AllLogCollection, latestDistroEventsPipeline(id, 1, true), &events)
	if err != nil {
		return res, err
	}
	if len(events) > 0 {
		res = events[0]
	}
	return res, nil
}

func latestDistroEventsPipeline(id string, n int, amiOnly bool) []bson.M {
	// We use two different match stages to use the most efficient index.
	resourceFilter := ResourceTypeKeyIs(ResourceTypeDistro)
	resourceFilter[ResourceIdKey] = id
	var eventFilter = bson.M{}
	if amiOnly {
		eventFilter[TypeKey] = EventDistroAMIModfied
	} else {
		eventFilter[TypeKey] = bson.M{"$ne": EventDistroAMIModfied}
	}
	return []bson.M{
		{"$match": resourceFilter},
		{"$sort": bson.M{TimestampKey: -1}},
		{"$match": eventFilter},
		{"$limit": n},
	}
}

// Scheduler Events
func SchedulerEventsForId(distroID string) db.Q {
	filter := ResourceTypeKeyIs(ResourceTypeScheduler)
	filter[ResourceIdKey] = distroID

	return db.Query(filter)
}

func RecentSchedulerEvents(distroId string, n int) db.Q {
	return SchedulerEventsForId(distroId).Sort([]string{"-" + TimestampKey}).Limit(n)
}

// Admin Events
// RecentAdminEvents returns the N most recent admin events
func RecentAdminEvents(n int) db.Q {
	filter := ResourceTypeKeyIs(ResourceTypeAdmin)
	filter[ResourceIdKey] = ""

	return db.Query(filter).Sort([]string{"-" + TimestampKey}).Limit(n)
}

func ByGuid(guid string) db.Q {
	return db.Query(bson.M{
		bsonutil.GetDottedKeyName(DataKey, "guid"): guid,
	})
}

func AdminEventsBefore(before time.Time, n int) db.Q {
	filter := ResourceTypeKeyIs(ResourceTypeAdmin)
	filter[ResourceIdKey] = ""
	filter[TimestampKey] = bson.M{
		"$lt": before,
	}

	return db.Query(filter).Sort([]string{"-" + TimestampKey}).Limit(n)
}

func FindAllByResourceID(resourceID string) ([]EventLogEntry, error) {
	return Find(AllLogCollection, db.Query(bson.M{ResourceIdKey: resourceID}))
}

// Pod events

// MostRecentPodEvents creates a query to find the n most recent pod events for
// the given pod ID.
func MostRecentPodEvents(id string, n int) db.Q {
	filter := ResourceTypeKeyIs(ResourceTypePod)
	filter[ResourceIdKey] = id

	return db.Query(filter).Sort([]string{"-" + TimestampKey}).Limit(n)
}
