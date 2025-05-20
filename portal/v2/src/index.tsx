import { createRoot } from "react-dom/client"
import App from "./App"
import { mergeStyles } from "@fluentui/react"
import { createBrowserRouter, RouterProvider } from "react-router"

import { initializeIcons } from "@fluentui/font-icons-mdl2"
initializeIcons()

// Inject some global styles
mergeStyles({
  ":global(body,html,#root)": {
    margin: 0,
    padding: 0,
    height: "100vh",
  },
})

const router = createBrowserRouter([
  {
    path: "/*",
    element: <App />,
  },
])

createRoot(document.getElementById("root")!).render(<RouterProvider router={router} />)
