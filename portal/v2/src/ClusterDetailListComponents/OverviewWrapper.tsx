import { useState, useEffect } from "react"
import { fetchClusterInfo } from "../Request"
import { IClusterCoordinates } from "../App"
import { OverviewComponent } from "./Overview"
import {
  IMessageBarStyles,
  MessageBar,
  MessageBarType,
  Stack,
  CommandBar,
  ICommandBarItemProps,
} from "@fluentui/react"
import { overviewKey } from "../ClusterDetail"

const errorBarStyles: Partial<IMessageBarStyles> = { root: { marginBottom: 15 } }

export function OverviewWrapper(props: {
  clusterName: string
  currentCluster: IClusterCoordinates
  detailPanelSelected: string
  loaded: boolean
}) {
  const [data, setData] = useState<any>([])
  const [error, setError] = useState<Response | null>(null)
  const [fetching, setFetching] = useState("")

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
    setData(newData)
  }

  const controlStyles = {
    root: {
      paddingLeft: 0,
      float: "right",
    },
  }

  const _items: ICommandBarItemProps[] = [
    {
      key: "refresh",
      text: "Refresh",
      iconProps: { iconName: "Refresh" },
      onClick: () => {
        updateData([])
        setFetching("")
      },
    },
  ]

  useEffect(() => {
    const onData = async (result: Response) => {
      if (result.status === 200) {
        updateData(await result.json())
      } else {
        setError(result)
      }
      setFetching(props.currentCluster.name)
    }

    if (
      props.detailPanelSelected.toLowerCase() == overviewKey &&
      fetching === "" &&
      props.loaded &&
      props.currentCluster.name != ""
    ) {
      setFetching("FETCHING")
      fetchClusterInfo(props.currentCluster).then(onData)
    }
  }, [data, props.loaded, props.clusterName])

  return (
    <Stack>
      <Stack.Item grow>{error && errorBar()}</Stack.Item>
      <Stack>
        <CommandBar items={_items} ariaLabel="Refresh" styles={controlStyles} />
        <OverviewComponent
          item={data}
          clusterName={props.currentCluster != null ? props.currentCluster.name : ""}
        />
      </Stack>
    </Stack>
  )
}
