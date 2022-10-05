import { Component } from "react"
import { Nav, INavLinkGroup, INavStyles, IRenderGroupHeaderProps, IRenderFunction, IFontStyles, IStyle } from "@fluentui/react"

const navStyles: Partial<INavStyles> = {
  chevronIcon: {
    visibility: 'hidden'
  }
};

const headerStyle: Partial<IStyle> = {
  h3: {
    "font-family": '"Segoe UI","Segoe UI Web (West European)", "Segoe UI", -apple-system, BlinkMacSystemFont, Roboto, "Helvetica Neue", sans-serif;',
    "font-weight": "600",
    "font-size": "24px",
    "line-height": "32px",
  }
}



interface RegionComponentProps {
  item: IRegionList
}

interface IRegionComponentState {
  item: IRegion // why both state and props?
}

export interface IRegionList {
  regions: IRegion[]
}

export interface IRegion {
  name: string
  url: string
}

const _onRenderGroupHeader: IRenderFunction<IRenderGroupHeaderProps> = (props): JSX.Element => {
  return (<h3 className="">{props?.name}</h3>);
};

export class RegionComponent extends Component<
  RegionComponentProps,
  IRegionComponentState
> {
  constructor(props: RegionComponentProps | Readonly<RegionComponentProps>) {
    super(props)
  }

  public render() {
    var navGroups: INavLinkGroup[] = [
      {
        name: 'Regions',
        links: this.props.item.regions
      }
    ]
    return (
      <Nav onRenderGroupHeader={_onRenderGroupHeader} styles={navStyles} groups={navGroups}  />
    )
  }
}
