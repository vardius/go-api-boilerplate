import React, { Component, ReactNode } from "react";
import {
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  CloseButton,
} from "@chakra-ui/core";

export interface Props {
  children: ReactNode;
}

type State = {
  readonly error: Error | null;
  readonly errorInfo?: Object;
};

export default class ErrorBoundary extends Component<Props, State> {
  readonly state: State = {
    error: null,
  };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  componentDidCatch(error: Error, errorInfo: Object): void {
    this.setState({ error, errorInfo });
  }

  render() {
    if (this.state.error) {
      return (
        <Alert status="error">
          <AlertIcon />
          <AlertTitle mr={2}>Something went wrong.</AlertTitle>
          <AlertDescription>Please refresh page or try again</AlertDescription>
          <CloseButton position="absolute" right="8px" top="8px" />
        </Alert>
      );
    }

    return this.props.children;
  }
}
