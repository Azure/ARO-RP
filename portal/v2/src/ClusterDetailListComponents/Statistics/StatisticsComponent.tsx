import { useEffect, useState } from "react"
import { Stack, StackItem, IStackProps} from '@fluentui/react';
import { Spinner, SpinnerSize } from '@fluentui/react/lib/Spinner';
import { ILineChartPoints, LineChart, ILineChartDataPoint } from '@fluentui/react-charting';
import { IChartProps } from '@fluentui/react-charting';
import { DefaultPalette} from '@fluentui/react/lib/Styling';
import { IMetrics} from './StatisticsWrapper';

export function StatisticsComponent(props: {
  metrics: IMetrics[],
  clusterName: any,
  duration: string,
  height: number,
  width: number,
  endDate: Date,
}) {
  const width = props.width
  const height = props.height
  const [points, setPoints] = useState<ILineChartPoints[]>([])
  const [data, setData] = useState<IChartProps>({})
  const [spinnerVisible, setSpinnerVisible] = useState<boolean>(true)
  const timeFormat = '%H:%M'

  const colors: string[] = [
    DefaultPalette.blue,
    DefaultPalette.black,
    DefaultPalette.redDark,
    DefaultPalette.blueDark,
    DefaultPalette.yellow,
    DefaultPalette.green,
    DefaultPalette.greenLight,
    DefaultPalette.purple,
  ]

  const _onLegendClickHandler = (selectedLegend: string | null | string[]): void => {
    if (selectedLegend !== null) {
      console.log(`Selected legend - ${selectedLegend}`);
    }
  };

  function _styledExample(): JSX.Element {   
    useEffect(() => {
      if (props.metrics.length > 0) {
        const newPoints: ILineChartPoints[] = []
        props.metrics.forEach((metric, i) => {          
          var dataPoints: ILineChartDataPoint[] = []
          metric.MetricValue.forEach(metricValue => {
            var data: ILineChartDataPoint = {
              x: new Date(metricValue.timestamp), y: metricValue.value
            }
            dataPoints.push(data)
          })

          var lineChartPoint: ILineChartPoints = {
            legend: metric.Name,
            onLegendClick: _onLegendClickHandler,
            data: dataPoints,
            color: colors[i]
          }
          newPoints.push(lineChartPoint)
        })
        setPoints(newPoints)    
      }
    },[props.metrics])

    useEffect(() => {   
      setData({
        chartTitle: 'Line Chart',
        lineChartData: points,
      });
      (points.length > 0) ? setSpinnerVisible(false) : setSpinnerVisible(true)
    }, [points])
    
    useEffect(() => {   
      setSpinnerVisible(true)
    }, [props.duration, props.endDate])

    const rootStyle = { width: `${width}px`, height: `${height}px` };
    const tokens = {
      sectionStack: {
        childrenGap: 10,
      },
      spinnerStack: {
        childrenGap: 20,
      },
    };
    const rowProps: IStackProps = { horizontal: false, verticalAlign: 'center' };
    
    return (
    <Stack>
      <StackItem>
        {
          spinnerVisible
          ?
          <Stack  {...rowProps} tokens={tokens.spinnerStack}>
            <StackItem> 
              <Spinner size={SpinnerSize.large} />
            </StackItem>
            <div style={rootStyle}>
              <LineChart
                data={data}
                strokeWidth={2}
                tickFormat={timeFormat}
                height={height}
                legendProps={{ canSelectMultipleLegends: true, allowFocusOnLegends: true }}
              />
            </div>
          </Stack>
          :
          <div style={rootStyle}>
            <LineChart
              data={data}
              strokeWidth={2}
              tickFormat={timeFormat}
              height={height}
              width={width}
              legendProps={{ canSelectMultipleLegends: true, allowFocusOnLegends: true }}
            />
          </div>
        }
      </StackItem>
    </Stack>
    );
  }
  return (
    <>
      <div>{_styledExample()}</div>
    </>
  );
}