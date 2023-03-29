import { useState, useEffect, useRef } from "react"
import { AxiosResponse } from 'axios';
import { fetchClusterOperators } from '../Request';
import { IMessageBarStyles, MessageBar, MessageBarType, Stack } from '@fluentui/react';
import { clusterOperatorsKey } from "../ClusterDetail";
import { ClusterOperatorListComponent } from "./ClusterOperatorList";
import { WrapperProps } from "../ClusterDetailList";

export interface ICondition {
  status: string,
  reason: string,
  lastUpdated: string,
  message: string
}

export interface IClusterOperator {
  name: string,
  available: string,
  progressing: string,
  degraded: string,
  conditions?: ICondition[],
}

export function ClusterOperatorsWrapper(props: WrapperProps) {
  const [operators, setOperators] = useState<IClusterOperator[]>([])
  const [error, setError] = useState<AxiosResponse | null>(null)
  const state = useRef<ClusterOperatorListComponent>(null)
  
  const [fetching, setFetching] = useState("")

  const errorBarStyles: Partial<IMessageBarStyles> = { root: { marginBottom: 15 } }

  const errorBar = (): any => {
    return (
      <MessageBar
        messageBarType={MessageBarType.error}
        isMultiline={false}
        onDismiss={() => setError(null)}
        dismissButtonAriaLabel="Close"
        styles={errorBarStyles}
      >
        {error?.statusText}
      </MessageBar>
    )
  }

  // updateData - updates the state of the component
  // can be used if we want a refresh button.
  // api/clusterdetail returns a single item.
  const updateData = (newData: any) => {
    setOperators(newData)
    const clusterOperatorList: IClusterOperator[] = []
    if (state && state.current) {
      newData.operators.forEach((element: { name: any;
                                        available: any;
                                        progressing: any;
                                        degraded: any;
                                        conditions: ICondition[]}) => {
        const clusterOp: IClusterOperator = {
          name: element.name,
          available: element.available,
          progressing: element.progressing,
          degraded: element.degraded
        }
        clusterOp.conditions = []
        element.conditions.forEach((condition: ICondition) => {
          clusterOp.conditions!.push(condition)
        });
        clusterOperatorList.push(clusterOp)
      });
      state.current.setState({ clusterOperators: clusterOperatorList })
    }
  }

  useEffect(() => {
    const onData = (result: AxiosResponse | null) => {
      if (result?.status === 200) {
        updateData(result.data)
      } else {
        setError(result)
      }
      if(props.currentCluster) {
        setFetching(props.currentCluster.name)
      }
    }

    if (props.detailPanelSelected.toLowerCase() == clusterOperatorsKey && 
        fetching === "" &&
        props.loaded &&
        props.currentCluster) {
      setFetching("FETCHING")
      fetchClusterOperators(props.currentCluster).then(onData)
    }
  }, [operators, props.loaded, props.detailPanelSelected])
  
  return (
    <Stack>
      <Stack.Item grow>{error && errorBar()}</Stack.Item>
      <Stack>
        <ClusterOperatorListComponent clusterOperators={operators} ref={state} clusterName={props.currentCluster != null ? props.currentCluster.name : ""} />
      </Stack>
    </Stack>   
  )
}
