import React from "react"
import { Navigate, Route, Routes } from "react-router"

import { OverviewWrapper } from "./ClusterDetailListComponents/OverviewWrapper"
import { NodesWrapper } from "./ClusterDetailListComponents/NodesWrapper"
import { MachinesWrapper } from "./ClusterDetailListComponents/MachinesWrapper"
import { MachineSetsWrapper } from "./ClusterDetailListComponents/MachineSetsWrapper"
import { Statistics } from "./ClusterDetailListComponents/Statistics/Statistics"
import { ClusterOperatorsWrapper } from "./ClusterDetailListComponents/ClusterOperatorsWrapper"

import { IClusterCoordinates } from "./App"
import {
  apiStatisticsKey,
  clusterOperatorsKey,
  dnsStatisticsKey,
  ingressStatisticsKey,
  kcmStatisticsKey,
  machineSetsKey,
  machinesKey,
  nodesKey,
  overviewKey,
} from "./ClusterDetail"

interface ClusterDetailComponentProps {
  item: IClusterDetails
  cluster: IClusterCoordinates | null
  isDataLoaded: boolean
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

export interface WrapperProps {
  currentCluster: IClusterCoordinates | null
  detailPanelSelected: string
  loaded: boolean
}

export function ClusterDetailComponent(props: ClusterDetailComponentProps) {
  return (
    <Routes>
      <Route path="" element={<Navigate to="overview" />} />
      <Route
        path="overview"
        element={
          <OverviewWrapper
            clusterName={props.cluster?.name!}
            currentCluster={props.cluster!}
            detailPanelSelected={overviewKey}
            loaded={props.isDataLoaded}
          />
        }
      />
      <Route
        path="nodes"
        element={
          <NodesWrapper
            currentCluster={props.cluster!}
            detailPanelSelected={nodesKey}
            loaded={props.isDataLoaded}
          />
        }
      />
      <Route
        path="machines"
        element={
          <MachinesWrapper
            currentCluster={props.cluster!}
            detailPanelSelected={machinesKey}
            loaded={props.isDataLoaded}
          />
        }
      />
      <Route
        path="machinesets"
        element={
          <MachineSetsWrapper
            currentCluster={props.cluster!}
            detailPanelSelected={machineSetsKey}
            loaded={props.isDataLoaded}
          />
        }
      />
      <Route
        path="apistatistics"
        element={
          <Statistics
            currentCluster={props.cluster!}
            detailPanelSelected={apiStatisticsKey}
            loaded={props.isDataLoaded}
            statisticsType="api"
          />
        }
      />
      <Route
        path="kcmstatistics"
        element={
          <Statistics
            currentCluster={props.cluster!}
            detailPanelSelected={kcmStatisticsKey}
            loaded={props.isDataLoaded}
            statisticsType="kcm"
          />
        }
      />
      <Route
        path="dnsstatistics"
        element={
          <Statistics
            currentCluster={props.cluster!}
            detailPanelSelected={dnsStatisticsKey}
            loaded={props.isDataLoaded}
            statisticsType="dns"
          />
        }
      />
      <Route
        path="ingressstatistics"
        element={
          <Statistics
            currentCluster={props.cluster!}
            detailPanelSelected={ingressStatisticsKey}
            loaded={props.isDataLoaded}
            statisticsType="ingress"
          />
        }
      />
      <Route
        path="clusteroperators"
        element={
          <ClusterOperatorsWrapper
            currentCluster={props.cluster!}
            detailPanelSelected={clusterOperatorsKey}
            loaded={props.isDataLoaded}
          />
        }
      />
    </Routes>
  )
}

export const MemoisedClusterDetailListComponent = React.memo(ClusterDetailComponent)
