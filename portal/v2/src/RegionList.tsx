import { Component } from "react"
import { Nav, INavLinkGroup, INavStyles, IRenderGroupHeaderProps, IRenderFunction, Stack, Text } from "@fluentui/react"

const navStyles: Partial<INavStyles> = {
  chevronIcon: {
    visibility: 'hidden'
  }
};

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
  return (<h3>{props?.name}</h3>);
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
      <Stack>
        <Nav onRenderGroupHeader={_onRenderGroupHeader} styles={navStyles} groups={navGroups}  />
        <Text>{this.props.item.regions.length == 0 && "No extra regions within this cloud"}</Text>
      </Stack>
    )
  }
}
