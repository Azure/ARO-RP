import { Component } from "react"
import React from "react"
import { OverviewWrapper } from "./ClusterDetailListComponents/OverviewWrapper"
import { NodesWrapper } from "./ClusterDetailListComponents/NodesWrapper"
import { MachinesWrapper } from "./ClusterDetailListComponents/MachinesWrapper"
import { MachineSetsWrapper } from "./ClusterDetailListComponents/MachineSetsWrapper"
import { Statistics } from "./ClusterDetailListComponents/Statistics/Statistics"

import { ICluster } from "./App"

interface ClusterDetailComponentProps {
  item: any
  cluster: ICluster | null
  isDataLoaded: boolean
  detailPanelVisible: string
}

export interface IClusterDetails {
  apiServerVisibility: string
  apiServerURL: string
  architectureVersion: string
  consoleLink: string
  createdAt: string
  createdBy: string
  failedProvisioningState: string
  infraId: string
  lastAdminUpdateError: string
  lastModifiedAt: string
  lastModifiedBy: string
  lastProvisioningState: string
  location: string
  name: string
  resourceId: string
  provisioningState: string
  version: string
  installStatus: string
}

interface IClusterDetailComponentState {
  item: IClusterDetails // why both state and props?
  detailPanelSelected: string
}

export class ClusterDetailComponent extends Component<
  ClusterDetailComponentProps,
  IClusterDetailComponentState
> {
  constructor(props: ClusterDetailComponentProps | Readonly<ClusterDetailComponentProps>) {
    super(props)
  }

  public render() {
    interface Map {
      [key: string]: JSX.Element
    }

    const menus: Map = {
      overview: (
        <OverviewWrapper
          clusterName={this.props.item.name}
          currentCluster={this.props.cluster!}
          detailPanelSelected={this.props.detailPanelVisible}
          loaded={this.props.isDataLoaded}
        />
      ),
      nodes: (
        <NodesWrapper
          currentCluster={this.props.cluster!}
          detailPanelSelected={this.props.detailPanelVisible}
          loaded={this.props.isDataLoaded}
        />
      ),
      machines: (
        <MachinesWrapper
          currentCluster={this.props.cluster!}
          detailPanelSelected={this.props.detailPanelVisible}
          loaded={this.props.isDataLoaded}
        />
      ),
      machinesets: (
        <MachineSetsWrapper
          currentCluster={this.props.cluster!}
          detailPanelSelected={this.props.detailPanelVisible}
          loaded={this.props.isDataLoaded}
        />
      ),
      apistatistics: (
        <Statistics
          currentCluster={this.props.cluster!}
          detailPanelSelected={this.props.detailPanelVisible}
          loaded={this.props.isDataLoaded}
          statisticsType={"api"}
        />
      ),
      kcmstatistics: (
        <Statistics
          currentCluster={this.props.cluster!}
          detailPanelSelected={this.props.detailPanelVisible}
          loaded={this.props.isDataLoaded}
          statisticsType={"kcm"}
        />
      ),
      dnsstatistics: (
        <Statistics
          currentCluster={this.props.cluster!}
          detailPanelSelected={this.props.detailPanelVisible}
          loaded={this.props.isDataLoaded}
          statisticsType={"dns"}
        />
      ),
      ingressstatistics: (
        <Statistics
          currentCluster={this.props.cluster!}
          detailPanelSelected={this.props.detailPanelVisible}
          loaded={this.props.isDataLoaded}
          statisticsType={"ingress"}
        />
      ),
    }

    return menus[this.props.detailPanelVisible.toLowerCase()]
  }
}

export const MemoisedClusterDetailListComponent = React.memo(ClusterDetailComponent)
