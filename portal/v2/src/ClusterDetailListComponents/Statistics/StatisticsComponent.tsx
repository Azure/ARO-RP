import { useEffect, useState } from "react"
import { Stack, StackItem, IStackProps } from "@fluentui/react"
import { Spinner, SpinnerSize } from "@fluentui/react/lib/Spinner"
import {
  ILineChartPoints,
  ILegendsProps,
  IChartProps,
  LineChart,
  ILineChartDataPoint,
  ILineChartProps,
} from "@fluentui/react-charting"
import { DefaultPalette } from "@fluentui/react/lib/Styling"
import { IMetrics } from "./StatisticsWrapper"
import { convertToUTC } from "./GraphOptionsComponent"

export function StatisticsComponent(props: {
  metrics: IMetrics[]
  clusterName: any
  duration: string
  height: number
  width: number
  endDate: Date
  fetchStatus: string
}) {
  const width = props.width
  const height = props.height
  const [points, setPoints] = useState<ILineChartPoints[]>([])
  const [data, setData] = useState<IChartProps>({})
  const [spinnerVisible, setSpinnerVisible] = useState<boolean>(true)
  const timeFormat = "%H:%M"

  const colors: string[] = [
    DefaultPalette.blue,
    DefaultPalette.blueLight,
    DefaultPalette.blueDark,
    DefaultPalette.blueMid,
    DefaultPalette.black,
    DefaultPalette.red,
    DefaultPalette.redDark,
    DefaultPalette.yellow,
    DefaultPalette.yellowDark,
    DefaultPalette.yellowLight,
    DefaultPalette.green,
    DefaultPalette.greenLight,
    DefaultPalette.greenDark,
    DefaultPalette.purple,
    DefaultPalette.purpleLight,
    DefaultPalette.purpleDark,
    DefaultPalette.orange,
    DefaultPalette.orangeLight,
    DefaultPalette.orangeLighter,
    DefaultPalette.magenta,
    DefaultPalette.magentaDark,
    DefaultPalette.magentaLight,
    DefaultPalette.themePrimary,
    DefaultPalette.neutralPrimary,
    DefaultPalette.neutralDark,
    DefaultPalette.neutralSecondary,
    DefaultPalette.neutralTertiary,
    DefaultPalette.teal,
    DefaultPalette.tealDark,
    DefaultPalette.tealLight,
    DefaultPalette.accent,
    DefaultPalette.themeDarker,
    DefaultPalette.themeDarkAlt,
    DefaultPalette.themeDark,
    DefaultPalette.themeLight,
    DefaultPalette.themeLighter,
    DefaultPalette.themeLighterAlt,
    DefaultPalette.themePrimary,
    DefaultPalette.themeSecondary,
    DefaultPalette.themeTertiary,
  ]

  function StatisticsHelperComponent(): JSX.Element {
    useEffect(() => {
      if (props.fetchStatus === "error") {
        setSpinnerVisible(false)
        return
      }
      const newPoints: ILineChartPoints[] = []
      props.metrics.forEach((metric, i) => {
        var dataPoints: ILineChartDataPoint[] = []
        metric.MetricValue.forEach((metricValue) => {
          let timeStamp = new Date(metricValue.timestamp)
          let metricsTime = convertToUTC(timeStamp)
          var data: ILineChartDataPoint = {
            x: metricsTime,
            y: metricValue.value,
          }
          dataPoints.push(data)
        })
        var lineChartPoint: ILineChartPoints = {
          legend: metric.Name,
          data: dataPoints,
          color: colors[i],
        }
        newPoints.push(lineChartPoint)
      })
      setPoints(newPoints)
    }, [props.metrics, props.fetchStatus])

    useEffect(() => {
      setData({
        chartTitle: "Line Chart",
        lineChartData: points,
      })
      props.fetchStatus === "success" ? setSpinnerVisible(false) : setSpinnerVisible(true)
    }, [points])

    useEffect(() => {
      setSpinnerVisible(true)
    }, [props.duration, props.endDate])

    const rootStyle = { width: `${width}px`, height: `${height}px` }
    const tokens = {
      sectionStack: {
        childrenGap: 10,
      },
      spinnerStack: {
        childrenGap: 20,
      },
    }
    const rowProps: IStackProps = { horizontal: false, verticalAlign: "center" }
    const legendProps: Partial<ILegendsProps> = {
      canSelectMultipleLegends: true,
      allowFocusOnLegends: true,
    }
    let lineChartProps: ILineChartProps = {
      data: data,
      strokeWidth: 2,
      tickFormat: timeFormat,
      height: height,
      width: width,
      legendProps: legendProps,
    }

    const renderLineChart = (lineChartProps: ILineChartProps) => {
      return (
        <div style={rootStyle}>
          <LineChart {...lineChartProps} />
        </div>
      )
    }

    return (
      <Stack>
        <StackItem>
          {spinnerVisible ? (
            <Stack {...rowProps} tokens={tokens.spinnerStack}>
              <StackItem>
                <Spinner size={SpinnerSize.large} />
              </StackItem>
              {renderLineChart(lineChartProps)}
            </Stack>
          ) : (
            renderLineChart(lineChartProps)
          )}
        </StackItem>
      </Stack>
    )
  }
  return (
    <>
      <div>{StatisticsHelperComponent()}</div>
    </>
  )
}
