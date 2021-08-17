import { DefaultButton } from '@fluentui/react/lib/Button';
import { Panel, PanelType } from '@fluentui/react/lib/Panel';
import { useBoolean } from '@fluentui/react-hooks';
import React, { useState, useImperativeHandle, useEffect, Component, useRef, forwardRef, MutableRefObject } from "react"
import { IMessageBarStyles, MessageBar, MessageBarType } from '@fluentui/react';
import { AxiosResponse } from 'axios';
import { FetchClusterInfo } from './Request';

// does the controller need props?
type ClusterDetailPanelProps = {
  csrfToken: MutableRefObject<string>
  name : any
  subscription : any
  resourceGroup : any
  loaded : string
}

// interface IClusterDetails {
//   key: string
//   name: string
//   subscription: string
//   resourceGroup : string
//   id: string
//   version: string
//   createdDate: string
//   provisionedBy: string
//   lastModified: string
//   state: string
//   failed: string
//   consoleLink: string
// }

interface ClusterDetailComponentProps {
  item: MutableRefObject<any> // probably bad... maybe should be specific interface of details?
  ref: MutableRefObject<any>
//  csrfToken: MutableRefObject<string>
}

interface IClusterDetailComponentState {
  item: MutableRefObject<any> // ... experiment - probably bad... maybe should be specific interface of details?
  modalOpen: boolean
}


class ClusterDetailComponent extends Component<ClusterDetailComponentProps, IClusterDetailComponentState> {
//export const ClusterDetailComponent = forwardRef<any, ClusterDetailProps>(({ csrfToken }, ref) => {
  public render() {
   // const {item} = this.state

    return (
      <p>Test data</p>
    );
  }
};

const errorBarStyles: Partial<IMessageBarStyles> = {root: {marginBottom: 15}}

export const ClusterDetailPanel = forwardRef<any, ClusterDetailPanelProps>(({csrfToken, name, subscription, resourceGroup, loaded}, ref) => {

/*export function ClusterDetailPanel(props: {
  csrfToken: MutableRefObject<string>
  name : any
  subscription : any
  resourceGroup : any
  loaded : string

}) {*/
  const [data, setData] = useState<any>([])
  const [error, setError] = useState<AxiosResponse | null>(null)
  const state = useRef<ClusterDetailComponent>(null)
  const [fetching, setFetching] = useState("")

  const [resourceID, setResourceID] = useState("")

  const [isOpen, { setTrue: openPanel, setFalse: dismissPanel }] = useBoolean(false); // panel controls

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

  useImperativeHandle(ref, () => ({
    LoadClusterDetailPanel: (item: any) => {
      name = item.name
      subscription = item.subscription
      resourceGroup = item.resourceGroup
      setResourceID(item.name)
      openPanel()
      updateData([])
      setFetching("")
    },
  }))

  // updateData - updates the state of the component
  // can be used if we want a refresh button.
  // api/clusterdetail returns a single item.
  const updateData = (newData: any) => {
    setData(newData)
    if (state && state.current) {
      state.current.setState({item: newData})
    }
  }

  useEffect(() => {
    const onData = (result: AxiosResponse | null) => {
      if (result?.status === 200) {
        setData(result)
      } else {
        setError(result)
      }
      setFetching("DONE")
    }

    if (fetching === "" && loaded === "DONE" && subscription !== "" && resourceGroup !== "" && name !== "") {
      setFetching("FETCHING")
      FetchClusterInfo(subscription, resourceGroup, name).then(onData)
    }
  }, [data, fetching, setFetching, loaded]) // props.loaded should be tied to the cluster list - or not used.. this component doesn't care.

  return (  
    <Panel
      isLightDismiss
      isOpen={isOpen}
      type={PanelType.large}
      onDismiss={dismissPanel}
      closeButtonAriaLabel="Close"
      headerText={resourceID}
    > 
    {error && errorBar()}
    <ClusterDetailComponent 
     item={data}
     ref={state} // why do we need ref here?
     //csrfToken={props.csrfToken} // probably don't need this? we already have fetched the data.
    />
  </Panel>
  )
})
