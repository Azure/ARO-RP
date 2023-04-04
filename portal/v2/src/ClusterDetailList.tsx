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
    switch (this.props.detailPanelVisible.toLowerCase()) {
      case "overview": {
        return (
          <OverviewWrapper
            clusterName={this.props.item.name}
            currentCluster={this.props.cluster!}
            detailPanelSelected={this.props.detailPanelVisible}
            loaded={this.props.isDataLoaded}
          />
        )
      }
      case "nodes": {
        return (
          <NodesWrapper
            currentCluster={this.props.cluster!}
            detailPanelSelected={this.props.detailPanelVisible}
            loaded={this.props.isDataLoaded}
          />
        )
      }
      case "machines": {
        return (
          <MachinesWrapper
            currentCluster={this.props.cluster!}
            detailPanelSelected={this.props.detailPanelVisible}
            loaded={this.props.isDataLoaded}
          />
        )
      }
      case "machinesets": {
        return (
          <MachineSetsWrapper
            currentCluster={this.props.cluster!}
            detailPanelSelected={this.props.detailPanelVisible}
            loaded={this.props.isDataLoaded}
          />
        )
      }
      case "apistatistics": {
        return (
          <Statistics
            currentCluster={this.props.cluster!}
            detailPanelSelected={this.props.detailPanelVisible}
            loaded={this.props.isDataLoaded}
            statisticsType={"api"}
          />
        )
      }
      case "kcmstatistics": {
        return (
          <Statistics
            currentCluster={this.props.cluster!}
            detailPanelSelected={this.props.detailPanelVisible}
            loaded={this.props.isDataLoaded}
            statisticsType={"kcm"}
          />
        )
      }
      case "dnsstatistics": {
        return (
          <Statistics
            currentCluster={this.props.cluster!}
            detailPanelSelected={this.props.detailPanelVisible}
            loaded={this.props.isDataLoaded}
            statisticsType={"dns"}
          />
        )
      }
      case "ingressstatistics": {
        return (
          <Statistics
            currentCluster={this.props.cluster!}
            detailPanelSelected={this.props.detailPanelVisible}
            loaded={this.props.isDataLoaded}
            statisticsType={"ingress"}
          />
        )
      }
    }
  }
}

export const MemoisedClusterDetailListComponent = React.memo(ClusterDetailComponent)
