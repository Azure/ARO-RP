import { IconButton, TooltipHost } from "@fluentui/react"
import { RequestKubeconfig } from "./Request"
import { MutableRefObject, useEffect, useLayoutEffect } from "react"
import { useState } from "react"
import { useRef } from "react"
import { forwardRef } from "react"
import { parse as parseContentDisposition } from "content-disposition"

type ToolIconsProps = {
  csrfToken: MutableRefObject<string>
  resourceId: string
  version: number
  sshBox: any
}

type FileDownload = {
  name: string
  content: string
}

export const ToolIcons = forwardRef<any, ToolIconsProps>(
  ({ csrfToken, resourceId, version, sshBox }) => {
    const [data, setData] = useState<FileDownload>({ name: "", content: "" })
    const [error, setError] = useState<Response | null>(null)
    const [fetching, setFetching] = useState("DONE")
    const buttonRef = useRef<HTMLAnchorElement | null>(null)

    useEffect(() => {
      const onData = async (result: Response) => {
        if (result.status === 200) {
          const blob = await result.blob()
          const fileDownloadUrl = URL.createObjectURL(blob)
          const filename = parseContentDisposition(result.headers.get("content-disposition") || "")
            .parameters.filename
          setData({ content: fileDownloadUrl, name: filename })
        } else {
          setError(result)
        }
        setFetching("DONE")
      }

      if (fetching === "") {
        setFetching("FETCHING")
        RequestKubeconfig(csrfToken.current, resourceId).then(onData)
      }
    }, [fetching, error, data, resourceId, csrfToken])

    const _onCopyResourceID = (resourceId: any) => {
      navigator.clipboard.writeText(resourceId)
    }

    const _onSSHClick = (resourceId: any) => {
      const modal = sshBox
      if (modal && modal.current) {
        modal.current.LoadSSH(resourceId)
      }
    }

    useLayoutEffect(() => {
      if (data.content && buttonRef && buttonRef.current) {
        buttonRef.current.href = data.content
        buttonRef.current.download = data.name
        buttonRef.current.click()
        URL.revokeObjectURL(data.content)
        data.content = ""
      }
    }, [data])

    return (
      <>
        <TooltipHost content={`Copy Resource ID`}>
          <IconButton
            iconProps={{ iconName: "Copy" }}
            aria-label="Copy Resource ID"
            onClick={_onCopyResourceID.bind({}, resourceId)}
          />
        </TooltipHost>
        <TooltipHost content={`Prometheus`}>
          <IconButton
            iconProps={{ iconName: "BIDashboard" }}
            aria-label="Prometheus"
            href={resourceId + (+version >= 4.11 ? `/prometheus` : `/prometheus/graph`)}
          />
        </TooltipHost>
        <TooltipHost content={`SSH`}>
          <IconButton
            iconProps={{ iconName: "CommandPrompt" }}
            aria-label="SSH"
            onClick={() => _onSSHClick(resourceId)}
          />
        </TooltipHost>
        <TooltipHost content={`Download Kubeconfig`}>
          <IconButton
            iconProps={{ iconName: "kubernetes-svg" }}
            disabled={fetching === "FETCHING"}
            aria-label="Download Kubeconfig"
            onClick={() => setFetching("")}
          />
          <a style={{ display: "none" }} ref={buttonRef} href={"#"}>
            dl
          </a>
        </TooltipHost>
      </>
    )
  }
)

ToolIcons.displayName = "toolicons"
