import React, {PureComponent} from 'react';
import styles from './Box.module.scss';

type SPACING_OPTIONS = 0 | 1 | 2 | 3 | 4 | 5 | 'auto';

export type Breakpoint = 'xs' | 'sm' | 'md' | 'lg' | 'xl';
export type BreakpointsMap<T> = { [K in Breakpoint]?: T; };
export type SpacingProp = SPACING_OPTIONS | BreakpointsMap<SPACING_OPTIONS>;

type OverflowOptions = 'auto' | 'hidden' | 'scroll' | 'visible';
type OverflowProp = OverflowOptions | BreakpointsMap<OverflowOptions>;
type PaddingOptions = 0 | 1 | 2 | 3 | 4 | 5;
type PaddingProp = PaddingOptions | BreakpointsMap<PaddingOptions>;
type TextAlignOptions = 'center' | 'justify' | 'left' | 'right';
type TextAlignProp = TextAlignOptions | BreakpointsMap<TextAlignOptions>;
type DisplayOptions =
  | 'block'
  | 'flex'
  | 'inline'
  | 'inline-block'
  | 'inline-flex'
  | 'none'
  | 'table'
  | 'table-cell'
  | 'table-row';
type DisplayOptionsProp = DisplayOptions | BreakpointsMap<DisplayOptions>;
type FlexAlignContentOptions = 'around' | 'between' | 'center' | 'end' | 'start' | 'stretch';
type FlexAlignContentProp = FlexAlignContentOptions | BreakpointsMap<FlexAlignContentOptions>;
type FlexAlignItemsOptions = 'baseline' | 'center' | 'end' | 'start' | 'stretch';
type FlexAlignItemsProp = FlexAlignItemsOptions | BreakpointsMap<FlexAlignItemsOptions>;
type FlexAlignSelfOptions = 'baseline' | 'center' | 'end' | 'start' | 'stretch';
type FlexAlignSelfProp = FlexAlignSelfOptions | BreakpointsMap<FlexAlignSelfOptions>;
type FlexDirectionOptions = 'column' | 'column-reverse' | 'row' | 'row-reverse';
type FlexDirectionProp = FlexDirectionOptions | BreakpointsMap<FlexDirectionOptions>;
type FlexJustifyContentOptions = 'around' | 'between' | 'center' | 'end' | 'start';
type FlexJustifyContentProp = FlexJustifyContentOptions | BreakpointsMap<FlexJustifyContentOptions>;
type BorderRadiusOptions = 'circle' | 'none' | 'top' | 'right' | 'bottom' | 'left' | boolean;
type OrderOptions = 'first' | 'last' | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;
type NoBorderOptions = 'top' | 'right' | 'bottom' | 'left' | boolean;

export type BoxProps = {
  element?: React.ElementType,
  children?: React.ReactNode,
  forwardedRef?: React.Ref<any>,
  bgColor?:
    | 'primary'
    | 'secondary'
    | 'success'
    | 'danger'
    | 'warning'
    | 'info'
    | 'dark'
    | 'light'
    | 'white'
    | 'transparent'
    | 'overlay',
  borderRadius?: BorderRadiusOptions | BreakpointsMap<BorderRadiusOptions>,
  noBorder?: NoBorderOptions | BreakpointsMap<NoBorderOptions>,
  className?: string,
  cursor?:
    | 'auto'
    | 'crosshair'
    | 'default'
    | 'grab'
    | 'help'
    | 'move'
    | 'text'
    | 'pointer'
    | 'progress'
    | 'wait'
    | 'not-allowed'
    | 'no-drop'
    | 'e-resize'
    | 'w-resize'
    | 'n-resize'
    | 'ne-resize'
    | 'nw-resize'
    | 's-resize'
    | 'se-resize'
    | 'sw-resize',
  display?: DisplayOptionsProp,
  flexAlignContent?: FlexAlignContentProp,
  flexAlignItems?: FlexAlignItemsProp,
  flexAlignSelf?: FlexAlignSelfProp,
  flexBasis?: 0 | 100 | 'auto' | BreakpointsMap<0 | 100 | 'auto'>,
  flexDirection?: FlexDirectionProp,
  flexGrow?: 0 | 1 | BreakpointsMap<0 | 1>,
  flexJustifyContent?: FlexJustifyContentProp,
  flexShrink?: 0 | 1 | BreakpointsMap<0 | 1>,
  flexWrap?: 'nowrap' | 'wrap-reverse' | 'wrap' | BreakpointsMap<'nowrap' | 'wrap-reverse' | 'wrap'>,
  margin?: SpacingProp,
  marginX?: SpacingProp,
  marginY?: SpacingProp,
  marginTop?: SpacingProp,
  marginRight?: SpacingProp,
  marginBottom?: SpacingProp,
  marginLeft?: SpacingProp,
  order?: OrderOptions | BreakpointsMap<OrderOptions>,
  overflow?: OverflowProp,
  overflowX?: OverflowProp,
  overflowY?: OverflowProp,
  padding?: PaddingProp,
  paddingX?: PaddingProp,
  paddingY?: PaddingProp,
  paddingTop?: PaddingProp,
  paddingRight?: PaddingProp,
  paddingBottom?: PaddingProp,
  paddingLeft?: PaddingProp,
  position?:
    | 'absolute-top'
    | 'absolute-top-right'
    | 'absolute-top-left'
    | 'absolute-top-center'
    | 'absolute-bottom'
    | 'absolute-bottom-right'
    | 'absolute-bottom-left'
    | 'absolute-bottom-center'
    | 'absolute-left'
    | 'absolute-left-center'
    | 'absolute-right'
    | 'absolute-right-center'
    | 'absolute-center'
    | 'fixed-top'
    | 'fixed-top-right'
    | 'fixed-top-left'
    | 'fixed-top-center'
    | 'fixed-bottom'
    | 'fixed-bottom-right'
    | 'fixed-bottom-left'
    | 'fixed-bottom-center'
    | 'fixed-left'
    | 'fixed-right'
    | 'fixed-center'
    | 'relative'
    | 'sticky-bottom'
    | 'sticky-top',
  textAlign?: TextAlignProp,
  textStyle?: 'body' | 'muted' | 'dark' | 'white' | 'link' | 'primary' | 'success' | 'info' | 'warning' | 'danger',
  verticalAlign?: 'baseline' | 'top' | 'middle' | 'bottom' | 'text-bottom' | 'text-top',
  visibleScrollbar?: boolean,
  columns?: 0 | 1 | 2 | 3 | 4 | 5 | 6 | BreakpointsMap<0 | 1 | 2 | 3 | 4 | 5 | 6>,
};

export default class Box extends PureComponent<BoxProps> {
  render() {
    const {
      element = 'div',
      className = '',
      forwardedRef,
      children,
      bgColor,
      borderRadius,
      noBorder,
      cursor,
      display,
      margin,
      marginX,
      marginY,
      marginTop,
      marginRight,
      marginBottom,
      marginLeft,
      padding,
      paddingX,
      paddingY,
      paddingTop,
      paddingRight,
      paddingBottom,
      paddingLeft,
      flexAlignContent,
      flexAlignItems,
      flexAlignSelf,
      flexBasis,
      flexDirection,
      flexGrow,
      flexJustifyContent,
      flexShrink,
      flexWrap,
      order,
      overflow,
      overflowX,
      overflowY,
      position,
      textAlign,
      textStyle,
      verticalAlign,
      visibleScrollbar,
      columns,
      ...attributes
    } = this.props;

    let scssClasses = [];

    scssClasses.push(className);

    // margin
    scssClasses.push(mapPropToClass('m', margin));
    scssClasses.push(mapPropToClass('mx', marginX));
    scssClasses.push(mapPropToClass('my', marginY));
    scssClasses.push(mapPropToClass('mt', marginTop));
    scssClasses.push(mapPropToClass('mr', marginRight));
    scssClasses.push(mapPropToClass('mb', marginBottom));
    scssClasses.push(mapPropToClass('ml', marginLeft));

    // padding
    scssClasses.push(mapPropToClass('p', padding));
    scssClasses.push(mapPropToClass('px', paddingX));
    scssClasses.push(mapPropToClass('py', paddingY));
    scssClasses.push(mapPropToClass('pt', paddingTop));
    scssClasses.push(mapPropToClass('pr', paddingRight));
    scssClasses.push(mapPropToClass('pb', paddingBottom));
    scssClasses.push(mapPropToClass('pl', paddingLeft));

    // align
    scssClasses.push(mapPropToClass('align', verticalAlign));
    scssClasses.push(mapPropToClass('align-content', flexAlignContent));
    scssClasses.push(mapPropToClass('align-items', flexAlignItems));
    scssClasses.push(mapPropToClass('align-self', flexAlignSelf));
    scssClasses.push(mapPropToClass('flex-basis', flexBasis));
    scssClasses.push(mapPropToClass('flex', flexDirection));
    scssClasses.push(mapPropToClass('flex-grow', flexGrow));
    scssClasses.push(mapPropToClass('justify-content', flexJustifyContent));
    scssClasses.push(mapPropToClass('flex-shrink', flexShrink));
    scssClasses.push(mapPropToClass('flex', flexWrap));

    scssClasses.push(mapPropToClass('bg', bgColor));
    scssClasses.push(mapPropToClass('rounded', borderRadius));
    scssClasses.push(mapPropToClass('no-border', noBorder));
    scssClasses.push(mapPropToClass('d', display));
    scssClasses.push(mapPropToClass('order', order));
    scssClasses.push(mapPropToClass('overflow', overflow));
    scssClasses.push(mapPropToClass('overflow-x', overflowX));
    scssClasses.push(mapPropToClass('overflow-y', overflowY));
    scssClasses.push(mapPropToClass('text', textAlign));
    scssClasses.push(mapPropToClass('text', textStyle));
    scssClasses.push(mapPropToClass('columns', columns));
    scssClasses.push(mapPropToClass('cursor', cursor));
    scssClasses.push(mapPropToClass('visible-scrollbar', visibleScrollbar));

    scssClasses.push(mapPropToClass('position', position));
    scssClasses.push(mapPropToClass('cursor', cursor));
    scssClasses.push(mapPropToClass('visible-scrollbar', visibleScrollbar));

    if (position === 'fixed-top') {
      scssClasses.push(styles[position]);
    }
    if (position === 'fixed-bottom') {
      scssClasses.push(styles[position]);
    }
    if (position === 'sticky-top') {
      scssClasses.push(styles[position]);
    }

    scssClasses = scssClasses.filter(notEmpty);

    return React.createElement(
      element,
      scssClasses.length > 0
        ? {...attributes, className: scssClasses.join(' '), ref: forwardedRef}
        : {...attributes, ref: forwardedRef},
      children,
    );
  }
}

function notEmpty<T>(value: T | null | undefined): value is T {
  return value !== null && value !== undefined;
}

function mapPropToClass(prefix: string, prop?: BreakpointsMap<number | string | boolean> | number | string | boolean): string {
  if (!prop && typeof prop !== 'number') {
    return '';
  }

  if (typeof prop === 'boolean') {
    return styles[`${prefix}`];
  }

  if (typeof prop === 'object') {
    if (Array.isArray(prop)) {
      const propStyles = prop.map((singleProp) => styles[`${prefix}-${singleProp}`]);

      return propStyles.join(' ');
    }

    const classes = [];
    for (const breakpoint in prop) {
      if (prop.hasOwnProperty(breakpoint)) {
        // @ts-ignore
        const value = prop[breakpoint];

        if (typeof value === 'boolean') {
          if (value) {
            classes.push(breakpoint === 'xs'
              ? styles[`${prefix}`]
              : styles[`${prefix}-${breakpoint}`]);
          }
        } else if (typeof value === 'object') {
          if (Array.isArray(value)) {
            value.map((singleValue) =>
              classes.push(breakpoint === 'xs'
                ? styles[`${prefix}-${singleValue}`]
                : styles[`${prefix}-${breakpoint}-${singleValue}`])
            );
          }
        } else {
          classes.push(
            breakpoint === 'xs'
              ? styles[`${prefix}-${value}`]
              : styles[`${prefix}-${breakpoint}-${value}`]
          );
        }
      }
    }

    return classes.join(' ');
  }

  return styles[`${prefix}-${prop}`];
}
