package sets

import "time"

const DEFAULT_POLL_TIME = time.Second * 10
const DEFAULT_TIMEOUT_DURATION = time.Minute * 20

var DEFAULT_MAINTENANCE_SETS = map[string]MaintenanceSet{}
