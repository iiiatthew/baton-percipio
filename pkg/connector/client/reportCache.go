package client

// StatusesStore is a type alias for an in-memory cache of grant-related data.
// It is used by the client to store the results of the learning activity report.
// It holds a nested map where the outer key is a course ID and the inner map links user IDs to their completion status.
// This structure organizes the report data for efficient lookups when building grants for a specific course.
// Instances are created by the `New` client function and populated by the `Load` method.
type StatusesStore map[string]map[string]string

// Load method processes a learning activity report and populates the StatusesStore cache.
// It implements the data hydration for the in-memory grant cache.
// The method iterates through each row of the report, creating a nested map of course IDs to user IDs to their normalized statuses.
// Which transforms the flat report data into a structured cache for fast, resource-specific grant lookups.
// This implementation processes the entire report at once to build the cache in memory.
func (r StatusesStore) Load(report *Report) error {
	for _, row := range *report {
		found, ok := r[row.ContentUUID]
		if !ok {
			found = make(map[string]string)
		}

		found[row.UserUUID] = toStatus(row.Status)
		r[row.ContentUUID] = found
	}

	return nil
}

// Get method retrieves all user-to-status relationships for a given course ID from the cache.
// It implements the lookup functionality for the grant cache.
// The method takes a course ID and returns the corresponding map of user IDs to their statuses.
// Which provides the grant builder with the necessary data to create grants for a specific course resource.
// This implementation returns `nil` if the course ID is not found in the cache.
func (r StatusesStore) Get(courseId string) map[string]string {
	found, ok := r[courseId]
	if !ok {
		return nil
	}
	return found
}

// toStatus function normalizes a Percipio status string into a connector-compatible status.
// It implements the status mapping required for creating grants.
// The function uses a switch statement to convert Percipio's status terms (e.g., "Started") into the statuses used by the connector (e.g., "in_progress").
// Which ensures that the grant entitlements are consistent and understood by the Baton system.
// This implementation defaults to "unknown" for any status that is not explicitly mapped.
func toStatus(status string) string {
	switch status {
	case "Started":
		return "in_progress"
	case "Completed":
		return "completed"
	default:
		return "unknown"
	}
}
