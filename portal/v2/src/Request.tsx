import urlJoin from "url-join"
import { IClusterCoordinates } from "./App"
import { convertTimeToHours } from "./ClusterDetailListComponents/Statistics/GraphOptionsComponent"

const OnError = (err: Response): Response => {
  if (err.status === 403) {
    var href = "/api/login"
    if (document.location.pathname !== "/") {
      href += "?redirect_uri=" + document.location.pathname
    }
    document.location.href = href
    return err
  } else {
    return err
  }
}

const doFetch = async (url: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
  const result = fetch(url, init)

  try {
    const g = await result
    if (!g.ok) {
      return OnError(g)
    }
    return g
  } catch (e: any) {
    console.error(e)
    return result
  }
}

export const fetchClusters = async (): Promise<Response> => {
  return doFetch("/api/clusters")
}

export const fetchClusterInfo = async (cluster: IClusterCoordinates): Promise<Response> => {
  return doFetch(urlJoin("/", "api", cluster.subscription, cluster.resourceGroup, cluster.name))
}

export const fetchInfo = async (): Promise<Response> => {
  return doFetch("/api/info")
}

export const fetchNodes = async (cluster: IClusterCoordinates): Promise<Response> => {
  return doFetch(
    urlJoin("/", "api", cluster.subscription, cluster.resourceGroup, cluster.name, "nodes")
  )
}

export const fetchMachines = async (cluster: IClusterCoordinates): Promise<Response> => {
  return doFetch(
    urlJoin("/", "api", cluster.subscription, cluster.resourceGroup, cluster.name, "machines")
  )
}

export const fetchMachineSets = async (cluster: IClusterCoordinates): Promise<Response> => {
  return doFetch(
    urlJoin("/", "api", cluster.subscription, cluster.resourceGroup, cluster.name, "machine-sets")
  )
}

export const fetchClusterOperators = async (cluster: IClusterCoordinates): Promise<Response> => {
  return doFetch(
    urlJoin(
      "/",
      "api",
      cluster.subscription,
      cluster.resourceGroup,
      cluster.name,
      "clusteroperators"
    )
  )
}

export const fetchRegions = async (): Promise<Response> => {
  return doFetch("/api/regions")
}

export const ProcessLogOut = async (): Promise<any> => {
  try {
    const result = await doFetch("/api/logout", { method: "POST" })
    return result
  } catch (e: any) {
    const err = e.response as Response
    console.log(err)
  }
  document.location.href = "/api/login"
}

export const RequestKubeconfig = async (
  csrfToken: string,
  resourceID: string
): Promise<Response> => {
  return doFetch(urlJoin("/", resourceID, "kubeconfig", "new"), {
    method: "POST",
    headers: {
      "X-CSRF-Token": csrfToken,
    },
  })
}

export const RequestSSH = async (
  csrfToken: string,
  machine: string,
  resourceID: string
): Promise<Response> => {
  return doFetch(urlJoin("/", resourceID, "ssh", "new"), {
    method: "POST",
    body: JSON.stringify({ master: machine }),
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": csrfToken,
    },
  })
}

export const fetchStatistics = async (
  cluster: IClusterCoordinates,
  statisticsName: string,
  duration: string,
  endDate: Date
): Promise<Response> => {
  duration = convertTimeToHours(duration)
  let endDateJSON = endDate.toJSON()

  const url = new URL(
    urlJoin(
      "/",
      "api",
      cluster.subscription,
      cluster.resourceGroup,
      cluster.name,
      "statistics",
      statisticsName
    )
  )
  url.searchParams.append("duration", duration)
  url.searchParams.append("endtime", endDateJSON)

  return doFetch(url)
}
