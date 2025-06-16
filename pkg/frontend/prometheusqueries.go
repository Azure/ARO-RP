package frontend

import "fmt"

type PrometheusQuery struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Query       string `json:"query"`
}

func GetDownsizeMetricAvailabilityQueries() []PrometheusQuery {
	queries := []PrometheusQuery{}

	// This query retrieves the percentage of available metric data points collected for
	// The total number of metric data points can be calculated as;
	//   duration / prometheus scrape interval
	// Duration is for 2 weeks which is 14d * 24h * 60m * 60s. The default prometheus scrape
	// interval is 15s for node-exporter, see cluster's prometheus config for more information
	queryFmt :=
		`
min(
	count_over_time(
	  %s{instance=~".*master.*"}[2w]
	)
  )
  /
  (
	(14 * 24 * 60 * 60) / 15
  )
  * 100
`
	queries = append(queries, PrometheusQuery{
		Name:        "node_memory_MemTotal_bytes",
		Query:       fmt.Sprintf(queryFmt, "node_memory_MemTotal_bytes"),
		Description: "Gets the percentage available data points of node_memory_MemTotal_bytes",
	})
	queries = append(queries, PrometheusQuery{
		Name:        "node_memory_MemAvailable_bytes",
		Query:       fmt.Sprintf(queryFmt, "node_memory_MemAvailable_bytes"),
		Description: "Gets the percentage available data points of node_memory_MemAvailable_bytes",
	})
	queries = append(queries, PrometheusQuery{
		Name:        "node_cpu_seconds_total",
		Query:       fmt.Sprintf(queryFmt, "node_cpu_seconds_total"),
		Description: "Gets the percentage available data points of node_cpu_seconds_total",
	})
	return queries
}
