package frontend

import "fmt"

type PrometheusQuery struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Query       string `json:"query"`
}

func GetDownsizeCondition1Query(memSizeFactor int64) PrometheusQuery {
	// The current average memory usage for the last 2 weeks must fit
	// in the target provisioned memory. Specifically, the current
	// average memory usage must be less than 60% of the target
	// provisioned memory. In the query below, replace the variable
	// Mem_Size_Factor with its calculated value and run this query
	// to check if this condition is met or not.
	query :=

		fmt.Sprintf(
			`clamp_max(
				(
				(sum(node_memory_MemTotal_bytes{instance=~".*master.*"}) / 1073741824)
				-
				(sum(avg_over_time(node_memory_MemAvailable_bytes{instance=~".*master.*"}[2w:])) / 1073741824)
				)
			<
				(
				%d * (sum(node_memory_MemTotal_bytes{instance=~".*master.*"}) / 1073741824) * 0.6
				),
			1
			)
			or
			vector(0)`,
			memSizeFactor,
		)

	return PrometheusQuery{
		Name:        "downsizeCondition1",
		Description: "Executes the downsize assessment condition 1",
		Query:       query,
	}
}

func GetDownsizeCondition2Query(cpuSizeFactor int64) PrometheusQuery {
	// The current average CPU usage for the last 2 weeks must fit
	// in the target provisioned CPU. Specifically, the current average
	// CPU usage must be less than 60% of the overall target provisioned CPU.
	// In the query below, replace the variable CPU_Size_Factor with its
	// calculated value and run this query to check if this condition is
	// met or not.
	query :=

		fmt.Sprintf(
			`clamp_max(
				(
					avg_over_time(
					sum(
					100
					-
					(avg by (instance) (rate(node_cpu_seconds_total{instance=~".*master.*",mode="idle"}[1m])) * 100)
					)[2w:]
					)
				)
				<
				((%d * 100) * 0.6),
				1
				)
				or
				vector(0)`,
			cpuSizeFactor,
		)

	return PrometheusQuery{
		Name:        "downsizeCondition2",
		Description: "Executes the downsize assessment condition 2",
		Query:       query,
	}
}

func GetDownsizeCondition3Query(memSizeFactor int64) PrometheusQuery {
	// If the average memory usage over 2 weeks is close[2] to 60% of the target
	// provisioned memory, then the estimated total memory available[3] trend
	// should not be downwards[4]. This helps avoid the case where the sustained
	// memory usage over 2 weeks is under 60% of the target provisioned memory
	// but close enough to exceed it sooner due to memory usage trending upwards.
	// In the query below, replace the variable Mem_Size_Factor with its calculated
	// value and run this query to check if this condition is met or not.
	query :=

		fmt.Sprintf(
			`clamp_max(
				(
				  (
					(
					  (sum(node_memory_MemTotal_bytes{instance=~".*master.*"}) / 1073741824)
					 -
					  (sum(avg_over_time(node_memory_MemAvailable_bytes{instance=~".*master.*"}[2w:])) / 1073741824)
					)
				   /
					((%d * (sum(node_memory_MemTotal_bytes{instance=~".*master.*"}) / 1073741824)) * 0.6)
				  )
				 *
				  100
				)
			   >=
				80
			  and
			   (sum(deriv(node_memory_MemAvailable_bytes{instance=~".*master.*"}[2w:])) > 0),
			  1
			 )
			or
			 vector(0)`,
			memSizeFactor,
		)

	return PrometheusQuery{
		Name:        "downsizeCondition3",
		Description: "Executes the downsize assessment condition 3",
		Query:       query,
	}
}

func GetDownsizeCondition4Query(cpuSizeFactor int64) PrometheusQuery {
	// If the average memory usage over 2 weeks is close[2] to 60% of the target
	// provisioned memory, then the estimated total memory available[3] trend
	// should not be downwards[4]. This helps avoid the case where the sustained
	// memory usage over 2 weeks is under 60% of the target provisioned memory
	// but close enough to exceed it sooner due to memory usage trending upwards.
	// In the query below, replace the variable Mem_Size_Factor with its calculated
	// value and run this query to check if this condition is met or not.
	query :=

		fmt.Sprintf(
			`clamp_max((
				(
					(
					  avg_over_time(
						sum(
							100
						  -
							(avg by (instance) (rate(node_cpu_seconds_total{instance=~".*master.*",mode="idle"}[1m])) * 100)
						)[2w:]
					  )
					)
				  /
					(%d * 100 * 0.6)
				)
			  *
				100
			)
		  >=
			80
		and
			((
			  deriv(
				sum(
					100
				  -
					(avg by (instance) (rate(node_cpu_seconds_total{instance=~".*master.*",mode="idle"}[1m])) * 100)
				)[2w:]
			  )
			)
		  <
			0), 1) or vector(0)`,
			cpuSizeFactor,
		)

	return PrometheusQuery{
		Name:        "downsizeCondition4",
		Description: "Executes the downsize assessment condition 4",
		Query:       query,
	}
}

func GetFiringAlertsQuery() PrometheusQuery {
	query :=
		`
ALERTS{alertstate="firing"}
  and
ALERTS{alertstate="firing"} offset 1h
`

	return PrometheusQuery{
		Name:        "firingAlerts",
		Description: "Firing alerts since the last hour",
		Query:       query,
	}
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
