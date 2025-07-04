package app

const STATIC_URL string = "/static/"
const STATIC_ROOT string = "static/"

const GRAPH_MINUTES int = 60
const POLL_INTERVAL_SECONDS int = 60
const METRIC_ARRAY_SIZE int = (60 / POLL_INTERVAL_SECONDS * GRAPH_MINUTES)

var version string = "undefined"
