import { TimePicker, IComboBox, DatePicker, IDatePickerStyles, Stack, Text, TooltipHost, IStackStyles, IconButton, mergeStyleSets } from "@fluentui/react"
import { iconButtonStyles } from "./Statistics"

export function GraphOptionsComponent(props: {duration: string, setDuration: React.Dispatch<React.SetStateAction<string>> , endDate: Date, setEndDate: React.Dispatch<React.SetStateAction<Date>>}): any {    
  const calculatorAddition =  { iconName: "CalculatorAddition" }
  const calculatorSubtract =  { iconName: "CalculatorSubtract" }
  const _increaseDuration = () => {
    let setToDuration = getIncreaseDurationMap().get(props.duration)
    if (setToDuration != undefined) {
      props.setDuration(setToDuration)
    }
  }
  const _decreaseDuration = () => {
    let setToDuration = getDecreaseDurationMap().get(props.duration)
    if (setToDuration != undefined) {
      props.setDuration(setToDuration)
    }
  }
  const durationStyle: Partial<IStackStyles> = {
    root: {
      alignSelf: "flex-start",
      border: "2px",
      marginLeft: "3px",
      marginRight: "3px",
    },
  }
  const dateTimePickerStyles = mergeStyleSets({
    iconContainer: {
      marginLeft: 5,
    },
  })
  const classNames = mergeStyleSets({
    iconContainer: {
      margin: "0px 0px",
      height: 25,
      width: "90px",
    },
  })

  const datePickerStyles: Partial<IDatePickerStyles> = { root: { maxWidth: 145, marginTop: 5, } };
  const timePickerStyles: Partial<IDatePickerStyles> = { root: { maxWidth: 75, marginLeft: 5, marginRight: 5 } };
  const onTimeChange = (event: React.FormEvent<IComboBox>, date: Date) => {        
    var localDate = new Date()
    localDate.setUTCFullYear(props.endDate.getFullYear(), props.endDate.getMonth(), props.endDate.getDate())
    localDate.setUTCHours(date.getHours());
    localDate.setUTCMinutes(date.getMinutes());
    props.setEndDate(localDate)        
    };
  const onDateChange = (date: Date | null | undefined): void => { 
    let localDate: Date = new Date();
    localDate.setUTCFullYear(date!.getFullYear(), date!.getMonth(), date!.getDate())
    localDate.setHours(props.endDate.getHours());
    localDate.setMinutes(props.endDate.getMinutes());
    props.setEndDate(localDate)  
    };
    
  return (
    <Stack horizontal verticalAlign="center">
      <Stack horizontal verticalAlign="center" className={classNames.iconContainer} /* style={{ boxShadow: theme.effects.elevation8 }}*/>
        <TooltipHost>
          <IconButton
            onClick={_decreaseDuration}
            iconProps={calculatorSubtract}
            styles={iconButtonStyles}
          />
        </TooltipHost>
        <TooltipHost>
          <Text styles={durationStyle}>{props.duration}</Text>
        </TooltipHost>
        <TooltipHost >
          <IconButton
            onClick={_increaseDuration}
            iconProps={calculatorAddition}
            styles={iconButtonStyles}
          />
        </TooltipHost>
      </Stack>
      <Stack horizontal verticalAlign="center" className={dateTimePickerStyles.iconContainer}  /* style={{ boxShadow: theme.effects.elevation8 }}*/>
        <DatePicker
          styles={datePickerStyles}
          placeholder="End Date"
          ariaLabel="End Date"
          onSelectDate={onDateChange}
          value={convertToUTC(props.endDate)}
          allowTextInput
        />
        <TimePicker
          styles={timePickerStyles}
          allowFreeform
          placeholder={timeToString(convertToUTC(props.endDate))}
          autoComplete="on"
          onChange={onTimeChange}
          defaultValue={convertToUTC(props.endDate)}
          useComboBoxAsMenuWidth
        />
      </Stack>
    </Stack>
  )
}

function timeToString(date: Date): string {
  var str
  let hourString = date.getHours().toString()
  str = hourString + ":"
  if (hourString.length === 1) {
    str = "0" + hourString + ":"
  }
  let minuteString = date.getMinutes().toString()
  if (minuteString.length === 1) {
    str = str + "0"
  }
  str += minuteString

  return str
}

export const convertToUTC = (date: Date): Date => {
  let localDate = new Date()
  localDate.setFullYear(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate())
  localDate.setHours(date.getUTCHours())
  localDate.setMinutes(date.getUTCMinutes())
  
  return localDate
}

export const convertTimeToHours = (duration: string): string => {
  let durationMap = new Map<string, string>()
  durationMap.set("1d", "24h")
  durationMap.set("2d", "48h")
  durationMap.set("1w", "168h")
  durationMap.set("2w", "336h")
  durationMap.set("4w", "672h")
  durationMap.set("8w", "1344h")
  if (durationMap.has(duration)) {
    return durationMap.get(duration)!
  }
  return duration
}

const getIncreaseDurationMap = (): Map<string, string> => {
  let increaseDurationMap = new Map<string, string>()
  increaseDurationMap.set("1m", "5m")
  increaseDurationMap.set("5m", "10m")
  increaseDurationMap.set("10m", "30m")
  increaseDurationMap.set("30m", "1h")
  increaseDurationMap.set("1h", "2h")
  increaseDurationMap.set("2h", "6h")
  increaseDurationMap.set("6h", "12h")
  increaseDurationMap.set("12h", "1d")
  increaseDurationMap.set("1d", "2d")
  increaseDurationMap.set("2d", "1w")
  increaseDurationMap.set("1w", "2w")
  increaseDurationMap.set("2w", "4w")
  increaseDurationMap.set("4w", "8w")

  return increaseDurationMap
}

const getDecreaseDurationMap = (): Map<string, string> => {
  let decreaseDurationMap = new Map<string, string>()
  decreaseDurationMap.set("8w", "4w")
  decreaseDurationMap.set("4w", "2w")
  decreaseDurationMap.set("2w", "1w")
  decreaseDurationMap.set("1w", "2d")
  decreaseDurationMap.set("2d", "1d")
  decreaseDurationMap.set("1d", "12h")
  decreaseDurationMap.set("12h", "6h")
  decreaseDurationMap.set("6h", "2h")
  decreaseDurationMap.set("2h", "1h")
  decreaseDurationMap.set("1h", "30m")
  decreaseDurationMap.set("30m", "10m")
  decreaseDurationMap.set("10m", "5m")
  decreaseDurationMap.set("5m", "1m")

  return decreaseDurationMap
}

