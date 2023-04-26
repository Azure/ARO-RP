import { useState, useEffect } from "react"
import { AxiosResponse } from "axios"
import { ICluster } from "../../App"
import { StatisticsComponent } from "./StatisticsComponent"
import { fetchStatistics } from "../../Request"
import {
  apiStatisticsKey,
  dnsStatisticsKey,
  ingressStatisticsKey,
  kcmStatisticsKey,
} from "../../ClusterDetail"
import { IMessageBarStyles, MessageBar, MessageBarType, Stack } from "@fluentui/react"

export interface IMetricValue {
  timestamp: Date
  value: number
}

export interface IMetrics {
  Name: string
  MetricValue: IMetricValue[]
}

export function StatisticsWrapper(props: {
  currentCluster: ICluster
  detailPanelSelected: string
  loaded: boolean
  statisticsName: string
  duration: string
  endDate: Date
  graphHeight: number
  graphWidth: number
}) {
  const [error, setError] = useState<AxiosResponse | null>(null)
  const [metrics, setMetrics] = useState<IMetrics[]>([])
  const [fetching, setFetching] = useState("")
  const [localDuration, setLocalDuration] = useState(props.duration)
  const [localEndDate, setLocalEndDate] = useState(props.endDate)
  const errorBarStyles: Partial<IMessageBarStyles> = { root: { marginBottom: 15 } }
  const statisticsKeys = [
    apiStatisticsKey,
    dnsStatisticsKey,
    ingressStatisticsKey,
    kcmStatisticsKey,
  ]

  const errorBar = (): any => {
    return (
      <MessageBar
        messageBarType={MessageBarType.error}
        isMultiline={false}
        onDismiss={() => setError(null)}
        dismissButtonAriaLabel="Close"
        styles={errorBarStyles}>
        {error?.statusText}
      </MessageBar>
    )
  }

  // updateData - updates the state of the component
  // can be used if we want a refresh button.
  // api/clusterdetail returns a single item.
  const updateData = (newData: any) => {
    const metrics: IMetrics[] = []
    newData.forEach((element: { metricname: any; metricvalue: IMetricValue[] }) => {
      const metric: IMetrics = {
        Name: element.metricname,
        MetricValue: element.metricvalue,
      }
      metrics.push(metric)
    })
    setMetrics(metrics)
  }

  useEffect(() => {
    const onData = (result: AxiosResponse | null) => {
      if (result?.status === 200) {
        setFetching("success")
        updateData(result.data)
        setError(null)
      } else {
        setError(result)
        setFetching("error")
      }
    }

    if (
      statisticsKeys.includes(props.detailPanelSelected.toLowerCase()) &&
      (fetching === "" || localDuration != props.duration || localEndDate != props.endDate) &&
      props.loaded &&
      props.currentCluster.name != ""
    ) {
      setLocalDuration(props.duration)
      setLocalEndDate(props.endDate)
      setFetching("FETCHING")
      fetchStatistics(
        props.currentCluster,
        props.statisticsName,
        props.duration,
        props.endDate
      ).then(onData)
    }
  }, [props.loaded, props.detailPanelSelected, props.duration, props.endDate])

  return (
    <Stack>
      <Stack.Item grow>{error && errorBar()}</Stack.Item>
      <Stack>
        <StatisticsComponent
          metrics={metrics}
          fetchStatus={fetching}
          duration={props.duration}
          clusterName={props.currentCluster != null ? props.currentCluster.name : ""}
          height={props.graphHeight}
          width={props.graphWidth}
          endDate={props.endDate}
        />
      </Stack>
    </Stack>
  )
}
