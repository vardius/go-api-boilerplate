import React, {Ref} from 'react';

export default function withForwardRef<BaseProps>(WrappedComponent: React.ComponentType<BaseProps>) {
  function forwardRef(props: BaseProps, ref: Ref<HTMLElement>) {
    return <WrappedComponent {...props} forwardedRef={ref}/>;
  }

  forwardRef.displayName = `withForwardRef(${WrappedComponent.displayName || WrappedComponent.name || 'Component'})`;

  return React.forwardRef(forwardRef);
}
