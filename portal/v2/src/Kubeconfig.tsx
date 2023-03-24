import { IconButton, TooltipHost } from "@fluentui/react"
import { AxiosResponse } from "axios"
import { RequestKubeconfig } from "./Request"
import { MutableRefObject, useEffect, useLayoutEffect } from "react"
import { useState } from "react"
import { useRef } from "react"
import { forwardRef } from "react"
import { parse as parseContentDisposition } from "content-disposition"

type KubeconfigButtonProps = {
  csrfToken: MutableRefObject<string>
  resourceId: string
}

type FileDownload = {
  name: string
  content: string
}

export const KubeconfigButton = forwardRef<any, KubeconfigButtonProps>(
  ({ csrfToken, resourceId }) => {
    const [data, setData] = useState<FileDownload>({ name: "", content: "" })
    const [error, setError] = useState<AxiosResponse | null>(null)
    const [fetching, setFetching] = useState("DONE")
    const buttonRef = useRef<HTMLAnchorElement | null>(null)

    useEffect(() => {
      const onData = (result: AxiosResponse | null) => {
        if (result?.status === 200) {
          const blob = new Blob([result.request.response])
          const fileDownloadUrl = URL.createObjectURL(blob)
          const filename = parseContentDisposition(result.headers["content-disposition"] || '').parameters
            .filename
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

KubeconfigButton.displayName = "kubeconfigbutton"
